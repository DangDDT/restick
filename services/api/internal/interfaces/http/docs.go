package httpx

import (
	"embed"
	"net/http"
)

// docs.html is the Swagger UI shell; it always loads /openapi.yaml. The bundled
// docs/openapi.yaml is the legacy copy used by the retired NewServer and is
// deleted with it at cutover — contracts/openapi/v1.yaml is the only source of
// truth and NewHandler serves the runtime copy named by Dependencies.
//
//go:embed docs/docs.html docs/openapi.yaml
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

// legacy openAPI/swaggerUI methods keep the retired community server working
// until T11 removes it.
func (s *Server) openAPI(w http.ResponseWriter, _ *http.Request) {
	data, err := swaggerFiles.ReadFile("docs/openapi.yaml")
	serveOpenAPI(w, data, err)
}

func (s *Server) swaggerUI(w http.ResponseWriter, _ *http.Request) {
	serveDocs(w)
}
