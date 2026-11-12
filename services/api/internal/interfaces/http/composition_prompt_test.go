package httpx

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/eykh/restick/api_svc/internal/studio"
)

func cellLabelsFromWire(t *testing.T, container map[string]any) []string {
	t.Helper()
	raw, ok := container["cells"].([]any)
	if !ok {
		t.Fatalf("payload has no cells array: %v", container)
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

func snapshotLabelsFromWire(t *testing.T, payload map[string]any) []string {
	t.Helper()
	snapshot, ok := payload["snapshot"].(map[string]any)
	if !ok {
		t.Fatalf("payload has no snapshot: %v", payload)
	}
	return cellLabelsFromWire(t, snapshot)
}

func draftLabelsFromWire(t *testing.T, payload map[string]any) []string {
	t.Helper()
	draft, ok := payload["draft"].(map[string]any)
	if !ok {
		t.Fatalf("payload has no draft: %v", payload)
	}
	return cellLabelsFromWire(t, draft)
}

func snapshotReferenceFromWire(t *testing.T, payload map[string]any) string {
	t.Helper()
	snapshot, ok := payload["snapshot"].(map[string]any)
	if !ok {
		t.Fatalf("payload has no snapshot: %v", payload)
	}
	character, ok := snapshot["character"].(map[string]any)
	if !ok {
		t.Fatalf("snapshot has no character: %v", snapshot)
	}
	id, _ := character["reference_asset_id"].(string)
	return id
}

func TestAdminCompositionReferencesAndLabels(t *testing.T) {
	f := newAdminFixture(t)
	f.login()

	upload := func(filename string) studio.ReferenceAsset {
		rec := f.upload("/api/v1/admin/reference-assets", "file", filename, testPNG(t, 5, 5))
		if rec.Code != http.StatusCreated {
			t.Fatalf("upload %s = %d: %s", filename, rec.Code, rec.Body.String())
		}
		var asset studio.ReferenceAsset
		decodeInto(t, rec, &asset)
		return asset
	}
	refA := upload("a.png")
	refB := upload("b.png")
	refC := upload("c.png")

	createCharacter := func(name string, ids ...string) string {
		rec := f.mutate(http.MethodPost, "/api/v1/admin/characters", map[string]any{
			"name": name, "description": "a small character", "reference_asset_ids": ids,
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("create character %s = %d: %s", name, rec.Code, rec.Body.String())
		}
		id, _ := decodeMap(t, rec)["id"].(string)
		if id == "" {
			t.Fatalf("created character %s has no id", name)
		}
		return id
	}
	charA := createCharacter("Nhân vật A", refA.ID, refB.ID)
	createCharacter("Nhân vật B", refC.ID)

	labels := []string{"Bún bò Huế", "Phở Hà Nội", "Bánh xèo miền Tây"}
	descriptions := []string{"a beef noodle bowl", "a pho plate", "a crispy crepe"}
	foodDraft := func(characterID, referenceID string) map[string]any {
		cells := make([]map[string]any, 0, len(labels))
		for index, label := range labels {
			cells = append(cells, map[string]any{
				"action": "steaming", "free_text": descriptions[index], "label": label,
			})
		}
		payload := map[string]any{
			"name": "Bộ ẩm thực", "rows": 1, "columns": 3, "outline_width": 4, "cells": cells,
		}
		if characterID != "" {
			payload["character_id"] = characterID
		}
		if referenceID != "" {
			payload["reference_asset_id"] = referenceID
		}
		return payload
	}

	rec := f.mutate(http.MethodPost, "/api/v1/admin/designs", foodDraft("", ""))
	if rec.Code != http.StatusCreated {
		t.Fatalf("save valid food design = %d: %s", rec.Code, rec.Body.String())
	}
	foodDesignID, _ := decodeMap(t, rec)["id"].(string)
	if foodDesignID == "" {
		t.Fatalf("saved food design has no id: %s", rec.Body.String())
	}

	invalid := []struct {
		name    string
		payload map[string]any
	}{
		{"character without a reference", foodDraft(charA, "")},
		{"reference of another character", foodDraft(charA, refC.ID)},
		{"unknown reference", foodDraft(charA, "no-such-asset")},
		{"reference without a character", foodDraft("", refA.ID)},
	}
	for _, testCase := range invalid {
		for _, route := range []struct{ method, target string }{
			{http.MethodPost, "/api/v1/admin/designs/compile"},
			{http.MethodPost, "/api/v1/admin/designs"},
			{http.MethodPut, "/api/v1/admin/designs/" + foodDesignID},
		} {
			rec := f.mutate(route.method, route.target, testCase.payload)
			if rec.Code != http.StatusUnprocessableEntity {
				t.Errorf("%s %s with %s = %d, want 422: %s",
					route.method, route.target, testCase.name, rec.Code, rec.Body.String())
			}
		}
	}

	// A valid design keeps the three per-cell labels and the selected image.
	valid := foodDraft(charA, refB.ID)
	rec = f.mutate(http.MethodPost, "/api/v1/admin/designs/compile", valid)
	if rec.Code != http.StatusOK {
		t.Fatalf("compile valid design = %d: %s", rec.Code, rec.Body.String())
	}
	compiled := decodeMap(t, rec)
	if got := snapshotLabelsFromWire(t, compiled); !reflect.DeepEqual(got, labels) {
		t.Fatalf("compiled labels = %#v, want %#v", got, labels)
	}
	if got := snapshotReferenceFromWire(t, compiled); got != refB.ID {
		t.Fatalf("compiled reference = %q, want %s", got, refB.ID)
	}
	prompt, _ := compiled["compiled_prompt"].(string)
	for _, label := range labels {
		if !strings.Contains(prompt, strconv.Quote(label)) {
			t.Fatalf("compiled prompt lacks the quoted label %q:\n%s", label, prompt)
		}
	}

	rec = f.mutate(http.MethodPost, "/api/v1/admin/designs", valid)
	if rec.Code != http.StatusCreated {
		t.Fatalf("save valid design = %d: %s", rec.Code, rec.Body.String())
	}
	created := decodeMap(t, rec)
	designID, _ := created["id"].(string)
	for name, got := range map[string][]string{
		"snapshot": snapshotLabelsFromWire(t, created),
		"draft":    draftLabelsFromWire(t, created),
	} {
		if !reflect.DeepEqual(got, labels) {
			t.Fatalf("saved %s labels = %#v, want %#v", name, got, labels)
		}
	}
	if got := snapshotReferenceFromWire(t, created); got != refB.ID {
		t.Fatalf("saved reference = %q, want %s", got, refB.ID)
	}

	// Update and reload keep the persisted values.
	rec = f.mutate(http.MethodPut, "/api/v1/admin/designs/"+designID, valid)
	if rec.Code != http.StatusOK {
		t.Fatalf("update design = %d: %s", rec.Code, rec.Body.String())
	}
	rec = f.get("/api/v1/admin/designs/" + designID)
	if rec.Code != http.StatusOK {
		t.Fatalf("get design = %d: %s", rec.Code, rec.Body.String())
	}
	reloaded := decodeMap(t, rec)
	if got := snapshotLabelsFromWire(t, reloaded); !reflect.DeepEqual(got, labels) {
		t.Fatalf("reloaded snapshot labels = %#v, want %#v", got, labels)
	}
	if got := draftLabelsFromWire(t, reloaded); !reflect.DeepEqual(got, labels) {
		t.Fatalf("reloaded draft labels = %#v, want %#v", got, labels)
	}
	if got := snapshotReferenceFromWire(t, reloaded); got != refB.ID {
		t.Fatalf("reloaded reference = %q, want %s", got, refB.ID)
	}
}
