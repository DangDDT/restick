package studio

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// decodeDraft builds a draft the way the admin HTTP layer does, from JSON, so
// the optional reference and label wire fields are exercised before they are
// named in Go.
func decodeDraft(t *testing.T, body string) Draft {
	t.Helper()
	var draft Draft
	if err := json.Unmarshal([]byte(body), &draft); err != nil {
		t.Fatalf("decode draft: %v", err)
	}
	return draft
}

func wireObject(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("decode wire: %v", err)
	}
	return wire
}

func draftWire(t *testing.T, draft Draft) map[string]any {
	t.Helper()
	raw, err := json.Marshal(draft)
	if err != nil {
		t.Fatalf("encode draft: %v", err)
	}
	return wireObject(t, raw)
}

// draftWithReference round-trips a draft through JSON to add the selected
// reference, so fixture updates stay valid whatever the Go field is named.
func draftWithReference(t *testing.T, draft Draft, assetID string) Draft {
	t.Helper()
	wire := draftWire(t, draft)
	wire["reference_asset_id"] = assetID
	raw, err := json.Marshal(wire)
	if err != nil {
		t.Fatalf("encode draft: %v", err)
	}
	return decodeDraft(t, string(raw))
}

func draftReferenceID(t *testing.T, draft Draft) string {
	t.Helper()
	id, _ := draftWire(t, draft)["reference_asset_id"].(string)
	return id
}

func snapshotWire(t *testing.T, snapshot Snapshot) map[string]any {
	t.Helper()
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("encode snapshot: %v", err)
	}
	return wireObject(t, raw)
}

func snapshotReferenceID(t *testing.T, snapshot Snapshot) (string, bool) {
	t.Helper()
	character, ok := snapshotWire(t, snapshot)["character"].(map[string]any)
	if !ok {
		return "", false
	}
	id, _ := character["reference_asset_id"].(string)
	return id, true
}

func cellLabels(t *testing.T, payload map[string]any) []string {
	t.Helper()
	raw, ok := payload["cells"].([]any)
	if !ok {
		t.Fatalf("payload has no cells array: %v", payload)
	}
	labels := make([]string, 0, len(raw))
	for _, entry := range raw {
		cell, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("cell = %#v, want an object", entry)
		}
		label, _ := cell["label"].(string)
		labels = append(labels, label)
	}
	return labels
}

func snapshotLabels(t *testing.T, snapshot Snapshot) []string {
	t.Helper()
	return cellLabels(t, snapshotWire(t, snapshot))
}

func draftLabels(t *testing.T, draft Draft) []string {
	t.Helper()
	return cellLabels(t, draftWire(t, draft))
}

func lineContaining(t *testing.T, text, needle string) string {
	t.Helper()
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	t.Fatalf("prompt has no line containing %q:\n%s", needle, text)
	return ""
}

func TestCompileReferenceMembership(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	refA := mustReferenceAsset(t, service, 0x01)
	refB := mustReferenceAsset(t, service, 0x02)
	refC := mustReferenceAsset(t, service, 0x03)

	charA, err := service.CreateCharacter(ctx, CharacterDraft{
		Name: "Nhân vật A", ReferenceAssetIDs: []string{refA.ID, refB.ID},
	})
	if err != nil {
		t.Fatalf("CreateCharacter A: %v", err)
	}
	if _, err := service.CreateCharacter(ctx, CharacterDraft{
		Name: "Nhân vật B", ReferenceAssetIDs: []string{refC.ID},
	}); err != nil {
		t.Fatalf("CreateCharacter B: %v", err)
	}

	draft := func(characterID, referenceID string) Draft {
		payload := map[string]any{
			"name": "Bộ tham chiếu", "rows": 1, "columns": 1, "outline_width": 4,
			"cells": []map[string]any{{"action": "waving"}},
		}
		if characterID != "" {
			payload["character_id"] = characterID
		}
		if referenceID != "" {
			payload["reference_asset_id"] = referenceID
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode draft: %v", err)
		}
		return decodeDraft(t, string(raw))
	}

	compiled, err := service.Compile(ctx, draft(charA.ID, refB.ID))
	if err != nil {
		t.Fatalf("Compile with a character reference: %v", err)
	}
	if compiled.Snapshot.SchemaVersion != 1 {
		t.Fatalf("snapshot schema_version = %d, want 1", compiled.Snapshot.SchemaVersion)
	}
	if got, ok := snapshotReferenceID(t, compiled.Snapshot); !ok || got != refB.ID {
		t.Fatalf("snapshot reference = %q (present %v), want %s", got, ok, refB.ID)
	}
	// The selected image is sent by the operator, never linked from the prompt.
	if strings.Contains(compiled.Text, refB.ID) || strings.Contains(compiled.Text, "/api/") {
		t.Fatalf("prompt leaks a private asset id or URL:\n%s", compiled.Text)
	}
	if !strings.Contains(strings.ToLower(compiled.Text), "reference image") {
		t.Fatalf("character prompt does not ask for the selected image:\n%s", compiled.Text)
	}

	saved, err := service.SaveDesign(ctx, draft(charA.ID, refB.ID))
	if err != nil {
		t.Fatalf("SaveDesign with a character reference: %v", err)
	}
	if got, ok := snapshotReferenceID(t, saved.Snapshot); !ok || got != refB.ID {
		t.Fatalf("saved snapshot reference = %q (present %v), want %s", got, ok, refB.ID)
	}
	if got := draftReferenceID(t, saved.Draft); got != refB.ID {
		t.Fatalf("saved draft reference = %q, want %s", got, refB.ID)
	}

	invalid := []struct {
		name  string
		draft Draft
	}{
		{"character without a reference", draft(charA.ID, "")},
		{"reference of another character", draft(charA.ID, refC.ID)},
		{"unknown reference", draft(charA.ID, "no-such-asset")},
		{"reference without a character", draft("", refA.ID)},
	}
	for _, testCase := range invalid {
		if _, err := service.Compile(ctx, testCase.draft); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: Compile = %v, want ErrInvalid", testCase.name, err)
		}
		if _, err := service.SaveDesign(ctx, testCase.draft); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: SaveDesign = %v, want ErrInvalid", testCase.name, err)
		}
	}

	plain, err := service.Compile(ctx, draft("", ""))
	if err != nil {
		t.Fatalf("Compile without character and reference: %v", err)
	}
	if _, ok := snapshotWire(t, plain.Snapshot)["character"]; ok {
		t.Fatalf("no-character snapshot carries a character block: %v", snapshotWire(t, plain.Snapshot))
	}
	if strings.Contains(strings.ToLower(plain.Text), "reference image") {
		t.Fatalf("no-character prompt asks for a reference image:\n%s", plain.Text)
	}
}

func TestFrozenReferenceSurvivesCharacterEdit(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	refA := mustReferenceAsset(t, service, 0x01)
	refB := mustReferenceAsset(t, service, 0x02)
	character, err := service.CreateCharacter(ctx, CharacterDraft{
		Name: "Bé Mực", Description: "a small ink-drop character",
		IdentityNotes:     "one white hair streak",
		ReferenceAssetIDs: []string{refA.ID, refB.ID},
	})
	if err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}

	draft := draftWithReference(t, sheetDraft(nil, nil), refA.ID)
	draft.CharacterID = character.ID
	saved, err := service.SaveDesign(ctx, draft)
	if err != nil {
		t.Fatalf("SaveDesign: %v", err)
	}
	if got, ok := snapshotReferenceID(t, saved.Snapshot); !ok || got != refA.ID {
		t.Fatalf("saved snapshot reference = %q (present %v), want %s", got, ok, refA.ID)
	}

	// The operator detaches refA from the profile after the design was saved.
	if _, err := service.UpdateCharacter(ctx, character.ID, CharacterDraft{
		Name: "Bé Mực", Description: "a small ink-drop character",
		IdentityNotes:     "one white hair streak",
		ReferenceAssetIDs: []string{refB.ID},
	}); err != nil {
		t.Fatalf("UpdateCharacter: %v", err)
	}

	reloaded, err := service.GetDesign(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetDesign: %v", err)
	}
	if reloaded.Prompt != saved.Prompt {
		t.Fatalf("saved prompt changed after the profile edit:\nbefore: %s\nafter:  %s", saved.Prompt, reloaded.Prompt)
	}
	if got, ok := snapshotReferenceID(t, reloaded.Snapshot); !ok || got != refA.ID {
		t.Fatalf("snapshot reference = %q (present %v), want the frozen %s", got, ok, refA.ID)
	}
	if got := draftReferenceID(t, reloaded.Draft); got != refA.ID {
		t.Fatalf("saved draft reference = %q, want %s", got, refA.ID)
	}
	_, file, err := service.OpenReferenceAsset(ctx, refA.ID)
	if err != nil {
		t.Fatalf("frozen reference became unreadable: %v", err)
	}
	file.Close()

	// The old draft cannot be recompiled until the operator picks a current image.
	if _, err := service.Compile(ctx, reloaded.Draft); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Compile of the frozen draft = %v, want ErrInvalid", err)
	}
	recompiled, err := service.Compile(ctx, draftWithReference(t, reloaded.Draft, refB.ID))
	if err != nil {
		t.Fatalf("Compile with the new reference: %v", err)
	}
	if got, ok := snapshotReferenceID(t, recompiled.Snapshot); !ok || got != refB.ID {
		t.Fatalf("recompiled reference = %q (present %v), want %s", got, ok, refB.ID)
	}
}

func TestThemedLabelsRoundTrip(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	type dish struct{ label, description string }
	dishes := []dish{
		{"Bún bò Huế", "a deep bowl of beef noodle soup with herbs"},
		{"Phở Hà Nội", "a plate of Hanoi pho with sliced beef"},
		{"Bánh xèo miền Tây", "a crispy sizzling crepe with shrimp and bean sprouts"},
	}
	cells := make([]map[string]any, 0, len(dishes))
	for _, item := range dishes {
		cells = append(cells, map[string]any{
			"action": "steaming", "free_text": item.description, "label": item.label,
		})
	}
	raw, err := json.Marshal(map[string]any{
		"name": "Bộ ẩm thực", "rows": 1, "columns": 3, "outline_width": 4,
		"free_text": "Street food trio, each sticker its own dish.",
		"cells":     cells,
	})
	if err != nil {
		t.Fatalf("encode draft: %v", err)
	}
	draft := decodeDraft(t, string(raw))

	compiled, err := service.Compile(ctx, draft)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	want := []string{dishes[0].label, dishes[1].label, dishes[2].label}
	if got := snapshotLabels(t, compiled.Snapshot); !reflect.DeepEqual(got, want) {
		t.Fatalf("snapshot labels = %#v, want %#v", got, want)
	}
	for _, item := range dishes {
		if !strings.Contains(compiled.Text, strconv.Quote(item.label)) {
			t.Errorf("prompt lacks the quoted label %q:\n%s", item.label, compiled.Text)
		}
		if strings.Contains(compiled.Text, strconv.Quote(item.description)) {
			t.Errorf("prompt renders the drawing description as visible text %q:\n%s", item.description, compiled.Text)
		}
		if count := strings.Count(compiled.Text, item.description); count != 1 {
			t.Errorf("description %q appears %d times, want one drawing instruction", item.description, count)
		}
		line := lineContaining(t, compiled.Text, item.description)
		if !strings.Contains(line, strconv.Quote(item.label)) {
			t.Errorf("label %q is not tied to its own cell: %q", item.label, line)
		}
	}
	lower := strings.ToLower(compiled.Text)
	if strings.Contains(lower, "same subject") {
		t.Fatalf("no-character prompt still forces one subject:\n%s", compiled.Text)
	}
	if strings.Contains(lower, "text, captions") {
		t.Fatalf("global text ban contradicts the explicit labels:\n%s", compiled.Text)
	}
	if !strings.Contains(lower, "label") {
		t.Fatalf("prompt does not explain the visible labels:\n%s", compiled.Text)
	}

	saved, err := service.SaveDesign(ctx, draft)
	if err != nil {
		t.Fatalf("SaveDesign: %v", err)
	}
	stored, err := service.GetDesign(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetDesign: %v", err)
	}
	if got := snapshotLabels(t, stored.Snapshot); !reflect.DeepEqual(got, want) {
		t.Fatalf("stored snapshot labels = %#v, want %#v", got, want)
	}
	if got := draftLabels(t, stored.Draft); !reflect.DeepEqual(got, want) {
		t.Fatalf("stored draft labels = %#v, want %#v", got, want)
	}
	for _, item := range dishes {
		if !strings.Contains(stored.Prompt, strconv.Quote(item.label)) {
			t.Fatalf("stored prompt lost label %q:\n%s", item.label, stored.Prompt)
		}
	}

	suggested, err := service.Suggest(ctx, stored.Draft, rand.New(rand.NewSource(7)))
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if got := draftLabels(t, suggested); !reflect.DeepEqual(got, want) {
		t.Fatalf("suggested labels = %#v, want %#v", got, want)
	}
	if suggested.CharacterID != "" {
		t.Fatalf("suggestion invented a character: %q", suggested.CharacterID)
	}
	if got := draftReferenceID(t, suggested); got != "" {
		t.Fatalf("suggestion invented a reference: %q", got)
	}
}

func TestMixedLabelsRemainExplicit(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	const rawLabel = "  Sen \"nở\"\nBuổi sáng  "
	const label = "Sen \"nở\"\nBuổi sáng"
	const firstDescription = "a lotus opening at dawn"
	const secondDescription = "a closed lotus bud"
	raw, err := json.Marshal(map[string]any{
		"rows": 1, "columns": 2, "outline_width": 4,
		"cells": []map[string]any{
			{"action": "blooming", "free_text": firstDescription, "label": rawLabel},
			{"action": "resting", "free_text": secondDescription},
		},
	})
	if err != nil {
		t.Fatalf("encode draft: %v", err)
	}

	compiled, err := service.Compile(ctx, decodeDraft(t, string(raw)))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	labels := snapshotLabels(t, compiled.Snapshot)
	if len(labels) != 2 || labels[0] != label || labels[1] != "" {
		t.Fatalf("snapshot labels = %#v, want [%q \"\"]", labels, label)
	}
	first := lineContaining(t, compiled.Text, firstDescription)
	if !strings.Contains(first, strconv.Quote(label)) {
		t.Fatalf("first cell does not quote its escaped label:\n%s", first)
	}
	second := lineContaining(t, compiled.Text, secondDescription)
	if !strings.Contains(strings.ToLower(second), "no text") {
		t.Fatalf("second cell is not told to leave out text: %q", second)
	}
	if strings.Contains(compiled.Text, strconv.Quote(firstDescription)) ||
		strings.Contains(compiled.Text, strconv.Quote(secondDescription)) {
		t.Fatalf("a drawing description was rendered as visible text:\n%s", compiled.Text)
	}
}

func TestLabelOnlyCellRejected(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	for name, cell := range map[string]map[string]any{
		"label only":  {"label": "Phở Hà Nội"},
		"blank label": {"label": "   "},
	} {
		raw, err := json.Marshal(map[string]any{
			"name": "Bộ lỗi", "rows": 1, "columns": 1, "outline_width": 4,
			"cells": []map[string]any{cell},
		})
		if err != nil {
			t.Fatalf("%s: encode draft: %v", name, err)
		}
		draft := decodeDraft(t, string(raw))
		if _, err := service.Compile(ctx, draft); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: Compile = %v, want ErrInvalid", name, err)
		}
		if _, err := service.SaveDesign(ctx, draft); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: SaveDesign = %v, want ErrInvalid", name, err)
		}
	}
}

func TestSuggestPreservesReferenceAndLabels(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	ref := mustReferenceAsset(t, service, 0x01)
	character, err := service.CreateCharacter(ctx, CharacterDraft{
		Name: "Bé Mực", ReferenceAssetIDs: []string{ref.ID},
	})
	if err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}
	empty, err := service.CreateCharacter(ctx, CharacterDraft{Name: "Chưa có ảnh"})
	if err != nil {
		t.Fatalf("CreateCharacter empty: %v", err)
	}
	group := mustGroup(t, service, "Phong cách", 10)
	style := mustFragment(t, service, group.ID, "flat colour", ScopePack)
	mustFragment(t, service, group.ID, "hatched shadows", ScopePack)

	payload := func(characterID string) map[string]any {
		item := map[string]any{
			"name": "Bộ gợi ý", "character_id": characterID,
			"group_ids": []string{group.ID}, "fragment_ids": []string{style.ID},
			"pinned_fragment_ids": []string{style.ID},
			"rows":                1, "columns": 2, "outline_width": 4,
			"cells": []map[string]any{
				{"action": "waving", "free_text": "a noodle bowl", "label": "Bún bò Huế"},
				{"action": "sitting", "free_text": "a pho plate", "label": "Phở Hà Nội"},
			},
		}
		return item
	}
	withReference := payload(character.ID)
	withReference["reference_asset_id"] = ref.ID
	raw, err := json.Marshal(withReference)
	if err != nil {
		t.Fatalf("encode draft: %v", err)
	}
	draft := decodeDraft(t, string(raw))

	suggested, err := service.Suggest(ctx, draft, rand.New(rand.NewSource(11)))
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if got := draftReferenceID(t, suggested); got != ref.ID {
		t.Fatalf("suggested reference = %q, want %s", got, ref.ID)
	}
	want := []string{"Bún bò Huế", "Phở Hà Nội"}
	if got := draftLabels(t, suggested); !reflect.DeepEqual(got, want) {
		t.Fatalf("suggested labels = %#v, want %#v", got, want)
	}
	if len(suggested.PinnedFragmentIDs) != 1 || suggested.PinnedFragmentIDs[0] != style.ID {
		t.Fatalf("suggestion dropped the pin: %v", suggested.PinnedFragmentIDs)
	}

	// Suggest never compiles, so an unchosen reference must not block it.
	raw, err = json.Marshal(payload(empty.ID))
	if err != nil {
		t.Fatalf("encode draft: %v", err)
	}
	withoutReference, err := service.Suggest(ctx, decodeDraft(t, string(raw)), rand.New(rand.NewSource(11)))
	if err != nil {
		t.Fatalf("Suggest without a chosen reference: %v", err)
	}
	if withoutReference.CharacterID != empty.ID {
		t.Fatalf("suggestion changed the character to %q", withoutReference.CharacterID)
	}
	if got := draftReferenceID(t, withoutReference); got != "" {
		t.Fatalf("suggestion invented a reference: %q", got)
	}
	if got := draftLabels(t, withoutReference); !reflect.DeepEqual(got, want) {
		t.Fatalf("suggested labels = %#v, want %#v", got, want)
	}
}
