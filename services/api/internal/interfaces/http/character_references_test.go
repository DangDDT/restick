package httpx

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/eykh/restick/api_svc/internal/studio"
)

// referenceIDsFromWire reads the array contract from a decoded admin response.
func referenceIDsFromWire(t *testing.T, payload map[string]any) []string {
	t.Helper()
	raw, ok := payload["reference_asset_ids"]
	if !ok {
		t.Fatalf("character JSON has no reference_asset_ids: %v", payload)
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

func TestAdminCharacterReferenceCollection(t *testing.T) {
	f := newAdminFixture(t)
	f.login()

	rec := f.upload("/api/v1/admin/reference-assets", "file", "a.png", testPNG(t, 5, 6))
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload A = %d: %s", rec.Code, rec.Body.String())
	}
	var refA studio.ReferenceAsset
	decodeInto(t, rec, &refA)
	rec = f.upload("/api/v1/admin/reference-assets", "file", "b.png", testPNG(t, 7, 8))
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload B = %d: %s", rec.Code, rec.Body.String())
	}
	var refB studio.ReferenceAsset
	decodeInto(t, rec, &refB)

	rec = f.mutate(http.MethodPost, "/api/v1/admin/characters", map[string]any{
		"name": "Mèo Mun", "description": "black cat",
		"reference_asset_ids": []string{refA.ID, refB.ID},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create character = %d: %s", rec.Code, rec.Body.String())
	}
	created := decodeMap(t, rec)
	if _, singular := created["reference_asset_id"]; singular {
		t.Fatalf("single reference field still encoded: %v", created)
	}
	if ids := referenceIDsFromWire(t, created); !reflect.DeepEqual(ids, []string{refA.ID, refB.ID}) {
		t.Fatalf("created references = %v", ids)
	}
	characterID, _ := created["id"].(string)
	if characterID == "" {
		t.Fatalf("created character has no id: %v", created)
	}

	rec = f.get("/api/v1/admin/characters/" + characterID)
	if rec.Code != http.StatusOK {
		t.Fatalf("get character = %d: %s", rec.Code, rec.Body.String())
	}
	if ids := referenceIDsFromWire(t, decodeMap(t, rec)); !reflect.DeepEqual(ids, []string{refA.ID, refB.ID}) {
		t.Fatalf("got references = %v", ids)
	}

	rec = f.get("/api/v1/admin/characters")
	if rec.Code != http.StatusOK {
		t.Fatalf("list characters = %d: %s", rec.Code, rec.Body.String())
	}
	listed, _ := decodeMap(t, rec)["items"].([]any)
	if len(listed) != 1 {
		t.Fatalf("listed characters = %v", listed)
	}
	item, _ := listed[0].(map[string]any)
	if ids := referenceIDsFromWire(t, item); !reflect.DeepEqual(ids, []string{refA.ID, refB.ID}) {
		t.Fatalf("listed references = %v", ids)
	}

	// An explicit empty list must encode as [] rather than null.
	rec = f.mutate(http.MethodPut, "/api/v1/admin/characters/"+characterID, map[string]any{
		"name": "Mèo Mun", "reference_asset_ids": []string{},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("update to empty = %d: %s", rec.Code, rec.Body.String())
	}
	if ids := referenceIDsFromWire(t, decodeMap(t, rec)); ids == nil || len(ids) != 0 {
		t.Fatalf("empty update references = %#v, want []", ids)
	}

	// An unknown asset rejects the whole update and keeps the stored profile.
	rec = f.mutate(http.MethodPut, "/api/v1/admin/characters/"+characterID, map[string]any{
		"name": "Renamed", "reference_asset_ids": []string{refA.ID, "missing"},
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("update with unknown asset = %d: %s", rec.Code, rec.Body.String())
	}
	rec = f.get("/api/v1/admin/characters/" + characterID)
	if rec.Code != http.StatusOK {
		t.Fatalf("get after rejected update = %d", rec.Code)
	}
	stored := decodeMap(t, rec)
	if stored["name"] != "Mèo Mun" {
		t.Fatalf("rejected update changed the name to %v", stored["name"])
	}
	if ids := referenceIDsFromWire(t, stored); ids == nil || len(ids) != 0 {
		t.Fatalf("rejected update changed references to %#v", ids)
	}

	// Reference bytes stay owner-session only.
	key, err := f.auth.CreateKey(f.t.Context(), "sdk", nil)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	if rec := f.request(http.MethodGet, "/api/v1/admin/reference-assets/"+refA.ID, nil,
		map[string]string{"X-Api-Key": key.Raw}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("client key preview = %d, want 401", rec.Code)
	}
	if rec := f.request(http.MethodGet, "/api/v1/admin/reference-assets/"+refA.ID, nil, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous preview = %d, want 401", rec.Code)
	}
}
