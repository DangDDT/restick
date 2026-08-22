package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/eykh/restick/ftp_svc/internal/application"
	"github.com/eykh/restick/ftp_svc/internal/domain/objectkey"
)

type Server struct {
	objects *application.ObjectService
	auth    Auth
	origin  string
}

func NewServer(objects *application.ObjectService, auth Auth, origin string) *Server {
	return &Server{objects: objects, auth: auth, origin: origin}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /internal/v1/objects", s.listInternal)
	mux.HandleFunc("PUT /internal/v1/objects/{key...}", s.putInternal)
	mux.HandleFunc("POST /internal/v1/bundles", s.buildBundle)
	mux.HandleFunc("GET /v1/objects/{key...}", s.getObject)
	mux.HandleFunc("PUT /v1/objects/{key...}", s.putUpload)
	mux.HandleFunc("GET /v1/bundles/{packID}/v/{version}", s.getBundle)
	return s.cors(mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "ftp_svc"})
}

func (s *Server) listInternal(w http.ResponseWriter, r *http.Request) {
	if !s.auth.Internal(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	prefix := strings.TrimSpace(r.URL.Query().Get("prefix"))
	if prefix == "" || strings.Contains(prefix, "..") {
		http.Error(w, "invalid prefix", http.StatusBadRequest)
		return
	}
	keys, err := s.objects.List(r.Context(), prefix)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"keys": keys})
}

func (s *Server) putInternal(w http.ResponseWriter, r *http.Request) {
	if !s.auth.Internal(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	key := r.PathValue("key")
	if err := objectkey.ValidateProcessed(key); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.putBytes(w, r, key)
}

func (s *Server) putUpload(w http.ResponseWriter, r *http.Request) {
	principal, err := s.auth.JWT(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	key := r.PathValue("key")
	if err := objectkey.ValidateUpload(key, principal.UserID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.putBytes(w, r, key)
}

func (s *Server) putBytes(w http.ResponseWriter, r *http.Request, key string) {
	data, err := io.ReadAll(io.LimitReader(r.Body, objectkey.MaxObjectBytes+1))
	if err != nil {
		http.Error(w, "read failed", http.StatusBadRequest)
		return
	}
	if err := s.objects.Put(r.Context(), key, data, r.Header.Get("Content-Type")); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, objectkey.ErrTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"key": key})
}

func (s *Server) getObject(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if !strings.HasPrefix(key, "packs/") && !s.auth.APIKey(r) && !s.auth.Internal(r) {
		if _, err := s.auth.JWT(r); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}
	s.streamKey(w, r, key)
}

func (s *Server) getBundle(w http.ResponseWriter, r *http.Request) {
	version, _ := strconv.Atoi(r.PathValue("version"))
	s.streamKey(w, r, objectkey.BundleKey(r.PathValue("packID"), version))
}

func (s *Server) streamKey(w http.ResponseWriter, r *http.Request, key string) {
	data, contentType, err := s.objects.Get(r.Context(), key)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(data)
}

func (s *Server) buildBundle(w http.ResponseWriter, r *http.Request) {
	if !s.auth.Internal(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		PackID  string   `json:"pack_id"`
		Version int      `json:"version"`
		Keys    []string `json:"keys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	key, err := s.objects.BuildBundle(r.Context(), body.PackID, body.Version, body.Keys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"bundle_key": key})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", s.origin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Api-Key, X-Internal-Token")
			w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
