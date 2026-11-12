package studio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

// decodeCharacterDraft builds a draft the way the admin HTTP layer does, from
// JSON, so this regression test does not compile against the new field.
func decodeCharacterDraft(t *testing.T, body string) CharacterDraft {
	t.Helper()
	var draft CharacterDraft
	if err := json.Unmarshal([]byte(body), &draft); err != nil {
		t.Fatalf("decode character draft: %v", err)
	}
	return draft
}

// characterWire marshals a character exactly as the admin API would and returns
// its JSON object.
func characterWire(t *testing.T, item Character) map[string]any {
	t.Helper()
	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("encode character: %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("decode character wire: %v", err)
	}
	return wire
}

func characterReferenceIDs(t *testing.T, item Character) []string {
	t.Helper()
	wire := characterWire(t, item)
	raw, ok := wire["reference_asset_ids"]
	if !ok {
		t.Fatalf("character JSON has no reference_asset_ids: %v", wire)
	}
	list, ok := raw.([]any)
	if !ok {
		t.Fatalf("reference_asset_ids = %#v, want an array", raw)
	}
	ids := make([]string, 0, len(list))
	for _, value := range list {
		id, ok := value.(string)
		if !ok {
			t.Fatalf("reference_asset_ids entry = %#v, want a string", value)
		}
		ids = append(ids, id)
	}
	return ids
}

func mustReferenceAsset(t *testing.T, service *Service, suffix byte) ReferenceAsset {
	t.Helper()
	raw := append(pngBytes(t), suffix)
	asset, err := service.SaveReferenceAsset(context.Background(), bytes.NewReader(raw), "reference.png")
	if err != nil {
		t.Fatalf("SaveReferenceAsset: %v", err)
	}
	return asset
}

func TestCharacterReferenceUpdateIsAtomic(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	refA := mustReferenceAsset(t, service, 0x01)
	refB := mustReferenceAsset(t, service, 0x02)

	character, err := service.CreateCharacter(ctx, decodeCharacterDraft(t, `{
		"name": "Bé Mực",
		"reference_asset_ids": ["`+refB.ID+`", "`+refA.ID+`"]
	}`))
	if err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}
	if got := characterReferenceIDs(t, character); !reflect.DeepEqual(got, []string{refB.ID, refA.ID}) {
		t.Fatalf("created references = %v, want [%s %s]", got, refB.ID, refA.ID)
	}

	_, err = service.UpdateCharacter(ctx, character.ID, decodeCharacterDraft(t, `{
		"name": "Renamed",
		"reference_asset_ids": ["`+refA.ID+`", "missing"]
	}`))
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("update with an unknown asset = %v, want ErrInvalid", err)
	}

	stored, err := service.GetCharacter(ctx, character.ID)
	if err != nil {
		t.Fatalf("GetCharacter: %v", err)
	}
	if stored.Name != "Bé Mực" {
		t.Fatalf("failed update changed the name to %q", stored.Name)
	}
	if got := characterReferenceIDs(t, stored); !reflect.DeepEqual(got, []string{refB.ID, refA.ID}) {
		t.Fatalf("failed update changed references to %v", got)
	}
	for _, asset := range []ReferenceAsset{refA, refB} {
		_, file, err := service.OpenReferenceAsset(ctx, asset.ID)
		if err != nil {
			t.Fatalf("asset %s became unreadable: %v", asset.ID, err)
		}
		file.Close()
	}

	updated, err := service.UpdateCharacter(ctx, character.ID, decodeCharacterDraft(t, `{
		"name": "Renamed",
		"reference_asset_ids": ["`+refA.ID+`", "`+refA.ID+`", " "]
	}`))
	if err != nil {
		t.Fatalf("valid update: %v", err)
	}
	if updated.Name != "Renamed" {
		t.Fatalf("updated name = %q", updated.Name)
	}
	if got := characterReferenceIDs(t, updated); !reflect.DeepEqual(got, []string{refA.ID}) {
		t.Fatalf("normalized references = %v, want [%s]", got, refA.ID)
	}
	_, file, err := service.OpenReferenceAsset(ctx, refB.ID)
	if err != nil {
		t.Fatalf("detached reference bytes disappeared: %v", err)
	}
	file.Close()
}

func TestCharacterReferenceListsStayIndependent(t *testing.T) {
	service, _ := newTestService(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	refA := mustReferenceAsset(t, service, 0x01)
	refB := mustReferenceAsset(t, service, 0x02)
	refC := mustReferenceAsset(t, service, 0x03)

	first, err := service.CreateCharacter(ctx, decodeCharacterDraft(t, `{
		"name": "A first", "reference_asset_ids": ["`+refB.ID+`", "`+refA.ID+`"]
	}`))
	if err != nil {
		t.Fatalf("CreateCharacter A: %v", err)
	}
	second, err := service.CreateCharacter(ctx, decodeCharacterDraft(t, `{
		"name": "B second", "reference_asset_ids": ["`+refC.ID+`"]
	}`))
	if err != nil {
		t.Fatalf("CreateCharacter B: %v", err)
	}
	third, err := service.CreateCharacter(ctx, decodeCharacterDraft(t, `{"name": "C empty"}`))
	if err != nil {
		t.Fatalf("CreateCharacter C: %v", err)
	}

	want := map[string][]string{
		first.ID:  {refB.ID, refA.ID},
		second.ID: {refC.ID},
		third.ID:  {},
	}
	list, err := service.ListCharacters(ctx)
	if err != nil {
		t.Fatalf("ListCharacters: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("ListCharacters returned %d characters", len(list))
	}
	for _, item := range list {
		if got := characterReferenceIDs(t, item); !reflect.DeepEqual(got, want[item.ID]) {
			t.Fatalf("listed %q references = %v, want %v", item.Name, got, want[item.ID])
		}
		got, err := service.GetCharacter(ctx, item.ID)
		if err != nil {
			t.Fatalf("GetCharacter(%s): %v", item.ID, err)
		}
		if ids := characterReferenceIDs(t, got); !reflect.DeepEqual(ids, want[item.ID]) {
			t.Fatalf("got %q references = %v, want %v", got.Name, ids, want[item.ID])
		}
	}
}
