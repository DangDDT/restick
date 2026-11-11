package httpx

import (
	"embed"
	"net/http"
)

// docs.html is the Swagger UI shell; it always loads /openapi.yaml, whose
// runtime copy comes from contracts/openapi/v1.yaml.
//
//go:embed docs/docs.html
var swaggerFiles embed.FS

func serveOpenAPI(w http.ResponseWriter, data []byte, err error) {
	if err != nil {
		publicError(w, http.StatusInternalServerError, "internal", "openapi unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func serveDocs(w http.ResponseWriter) {
	data, err := swaggerFiles.ReadFile("docs/docs.html")
	if err != nil {
		publicError(w, http.StatusInternalServerError, "internal", "docs unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
