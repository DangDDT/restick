package httpx

import (
	"net/http"
	"time"
)

// admin_keys.go manages read-only client API keys. The raw key is returned
// only by create and rotate.

func (a *admin) listKeys(w http.ResponseWriter, r *http.Request) {
	items, err := a.auth.ListKeys(r.Context())
	if err != nil {
		writeKeyError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *admin) createKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      string     `json:"name"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	key, err := a.auth.CreateKey(r.Context(), body.Name, utcExpiry(body.ExpiresAt))
	if err != nil {
		writeKeyError(w, err)
		return
	}
	publicJSON(w, http.StatusCreated, key)
}

func (a *admin) rotateKey(w http.ResponseWriter, r *http.Request) {
	key, err := a.auth.RotateKey(r.Context(), r.PathValue("id"))
	if err != nil {
		writeKeyError(w, err)
		return
	}
	publicJSON(w, http.StatusCreated, key)
}

func (a *admin) keyPacks(w http.ResponseWriter, r *http.Request) {
	ids, err := a.auth.KeyPacks(r.Context(), r.PathValue("id"))
	if err != nil {
		writeKeyError(w, err)
		return
	}
	writePackIDs(w, ids)
}

func (a *admin) setKeyPacks(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PackIDs *[]string `json:"pack_ids"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.PackIDs == nil || *body.PackIDs == nil {
		publicError(w, http.StatusBadRequest, "bad_request", "pack_ids is required")
		return
	}
	id := r.PathValue("id")
	if err := a.auth.SetKeyPacks(r.Context(), id, *body.PackIDs); err != nil {
		writeKeyError(w, err)
		return
	}
	a.keyPacks(w, r)
}

// writePackIDs renders the grant set; an empty set is [], never null.
func writePackIDs(w http.ResponseWriter, ids []string) {
	if ids == nil {
		ids = []string{}
	}
	publicJSON(w, http.StatusOK, map[string]any{"pack_ids": ids})
}

func (a *admin) disableKey(w http.ResponseWriter, r *http.Request) {
	if err := a.auth.DisableKey(r.Context(), r.PathValue("id")); err != nil {
		writeKeyError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

// utcExpiry normalizes an optional RFC3339 expiry to UTC second precision.
func utcExpiry(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	utc := value.UTC().Truncate(time.Second)
	return &utc
}
