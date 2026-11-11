package httpx

import (
	"context"
	"net/http"
	"net/netip"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/auth"
	"github.com/eykh/restick/api_svc/internal/jobs"
	"github.com/eykh/restick/api_svc/internal/packs"
	"github.com/eykh/restick/api_svc/internal/studio"
)

// Dependencies are the reviewed factory services served by NewHandler. WebDir
// and OpenAPIPath are runtime copies: contracts/openapi/v1.yaml is the only
// source of truth and the packaged image copies it beside the exported UI.
type Dependencies struct {
	Auth              *auth.Service
	Studio            *studio.Service
	Packs             *packs.Service
	Jobs              *jobs.Runner
	Assets            *assets.Store
	PublicBaseURL     string
	TrustedProxyCIDRs []netip.Prefix
	WebDir            string
	OpenAPIPath       string
}

// NewHandler serves the SDK read API, the owner admin API, the exported UI,
// docs and health from one mux. It replaces the legacy community server.
func NewHandler(deps Dependencies) http.Handler {
	return newHandler(deps, time.Now)
}

func newHandler(deps Dependencies, now func() time.Time) http.Handler {
	public := &publicAPI{
		catalog: deps.Packs,
		files:   deps.Assets,
		auth:    deps.Auth,
		baseURL: strings.TrimRight(deps.PublicBaseURL, "/"),
	}
	owner := &admin{
		auth:    deps.Auth,
		studio:  deps.Studio,
		packs:   deps.Packs,
		jobs:    deps.Jobs,
		assets:  deps.Assets,
		baseURL: strings.TrimRight(deps.PublicBaseURL, "/"),
		trusted: deps.TrustedProxyCIDRs,
		limiter: newLoginLimiter(now),
		now:     now,
	}

	mux := http.NewServeMux()
	public.mount(mux)
	owner.mount(mux)
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("GET /openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		data, err := os.ReadFile(deps.OpenAPIPath)
		serveOpenAPI(w, data, err)
	})
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, _ *http.Request) { serveDocs(w) })
	mux.HandleFunc("GET /docs/", func(w http.ResponseWriter, _ *http.Request) { serveDocs(w) })
	// Unknown API routes must answer JSON, never the exported HTML shell.
	mux.HandleFunc("/api/", notFound)
	mux.HandleFunc("/v1/", notFound)
	mux.HandleFunc("/", staticHandler(deps.WebDir))
	return mux
}

func health(w http.ResponseWriter, _ *http.Request) {
	publicJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "api_svc"})
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	publicError(w, http.StatusNotFound, "not_found", "route not found")
}

// staticHandler serves the exported UI without directory listing and without
// falling back to HTML for missing API paths (handled before by notFound).
func staticHandler(webDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if webDir == "" {
			publicError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			publicError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		clean := path.Clean("/" + r.URL.Path)
		if clean == "/" {
			http.Redirect(w, r, "/factory", http.StatusTemporaryRedirect)
			return
		}
		file := filepath.Join(webDir, filepath.FromSlash(strings.TrimPrefix(clean, "/")))
		switch {
		case strings.HasSuffix(r.URL.Path, "/"):
			file = filepath.Join(file, "index.html")
		case isDir(file):
			file = filepath.Join(file, "index.html")
		case !exists(file):
			if !exists(file + ".html") {
				http.NotFound(w, r)
				return
			}
			file += ".html"
		}
		http.ServeFile(w, r, file)
	}
}

func exists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

func isDir(file string) bool {
	info, err := os.Stat(file)
	return err == nil && info.IsDir()
}

func Listen(ctx context.Context, addr string, handler http.Handler) error {
	server := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
