package httpx

import (
	"embed"
	"net/http"
)

//go:embed docs/docs.html docs/openapi.yaml
var swaggerFiles embed.FS

func (s *Server) openAPI(w http.ResponseWriter, _ *http.Request) {
	data, err := swaggerFiles.ReadFile("docs/openapi.yaml")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", "openapi missing")
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) swaggerUI(w http.ResponseWriter, _ *http.Request) {
	data, err := swaggerFiles.ReadFile("docs/docs.html")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", "docs missing")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
