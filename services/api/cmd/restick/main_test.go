package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eykh/restick/api_svc/internal/infrastructure/config"
	"github.com/eykh/restick/api_svc/internal/jobs"
	"github.com/eykh/restick/api_svc/internal/storage"
	"github.com/google/uuid"
)

// testConfig builds the runtime configuration against a real data dir and the
// actual T9 static export / canonical OpenAPI documents.
func testConfig(t *testing.T, dataDir string) config.FactoryConfig {
	t.Helper()
	return config.FactoryConfig{
		DataDir:           dataDir,
		HTTPAddr:          freeAddr(t),
		PublicBaseURL:     "https://restick.example",
		AdminPasswordFile: writePassword(t),
		ProcessTimeout:    10 * time.Second,
		WebDir:            repoPath(t, "clients", "web-next", "out"),
		OpenAPIPath:       repoPath(t, "contracts", "openapi", "v1.yaml"),
	}
}

func repoPath(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{"..", "..", "..", ".."}, parts...)...)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("repository path %s: %v", path, err)
	}
	return path
}

func writePassword(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "admin_password")
	if err := os.WriteFile(path, []byte("owner-password\n"), 0o600); err != nil {
		t.Fatalf("write password file: %v", err)
	}
	return path
}

func freeAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

type testServer struct {
	baseURL string
	cancel  context.CancelFunc
	errCh   chan error
}

func startTestServer(t *testing.T, cfg config.FactoryConfig, processor jobs.Processor) *testServer {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- serveWith(ctx, cfg, processor) }()
	deadline := time.Now().Add(20 * time.Second)
	client := &http.Client{Timeout: time.Second}
	healthyURL := "http://" + cfg.HTTPAddr + "/healthz"
	for {
		select {
		case err := <-errCh:
			cancel()
			t.Fatalf("serve exited during startup: %v", err)
		default:
		}
		response, err := client.Get(healthyURL)
		if err == nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return &testServer{baseURL: "http://" + cfg.HTTPAddr, cancel: cancel, errCh: errCh}
			}
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("server did not become healthy at %s", healthyURL)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (s *testServer) stop(t *testing.T) {
	t.Helper()
	s.cancel()
	select {
	case err := <-s.errCh:
		if err != nil {
			t.Fatalf("serve shutdown: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("serve did not stop")
	}
}

func get(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	client := &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	response, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatalf("read %s: %v", url, err)
	}
	return response, string(body)
}

func TestStartupRejectsUnwritableDataAndMissingBootstrapSecret(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Run("unwritable data dir", func(t *testing.T) {
			dataDir := t.TempDir()
			cfg := testConfig(t, dataDir)
			if err := os.Chmod(dataDir, 0o500); err != nil {
				t.Fatalf("make data dir read-only: %v", err)
			}
			t.Cleanup(func() { _ = os.Chmod(dataDir, 0o700) })
			if err := serveWith(context.Background(), cfg, jobs.PythonProcessor{}); err == nil {
				t.Fatal("serve accepted a data dir it cannot write")
			}
		})
	}

	t.Run("missing bootstrap secret leaves no database", func(t *testing.T) {
		dataDir := t.TempDir()
		cfg := testConfig(t, dataDir)
		cfg.AdminPasswordFile = ""
		err := serveWith(context.Background(), cfg, jobs.PythonProcessor{})
		if err == nil {
			t.Fatal("serve accepted a fresh data dir without a bootstrap secret")
		}
		if _, statErr := os.Stat(filepath.Join(dataDir, storage.DBFileName)); !os.IsNotExist(statErr) {
			t.Fatalf("startup created %s without a bootstrap secret (stat: %v)", storage.DBFileName, statErr)
		}
	})

	t.Run("partial restore tree is never initialised", func(t *testing.T) {
		dataDir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dataDir, "sources"), 0o755); err != nil {
			t.Fatalf("prepare partial tree: %v", err)
		}
		cfg := testConfig(t, dataDir)
		err := serveWith(context.Background(), cfg, jobs.PythonProcessor{})
		if err == nil {
			t.Fatal("serve initialised a data dir with resource directories but no database")
		}
		if _, statErr := os.Stat(filepath.Join(dataDir, storage.DBFileName)); !os.IsNotExist(statErr) {
			t.Fatalf("startup created %s on a partial restore tree (stat: %v)", storage.DBFileName, statErr)
		}
		if info, statErr := os.Stat(filepath.Join(dataDir, "sources")); statErr != nil || !info.IsDir() {
			t.Fatalf("partial tree was modified: %v", statErr)
		}
	})
}

func TestPartialRestoreRejectsAnimationDirectories(t *testing.T) {
	for _, name := range []string{"clips", "animations"} {
		t.Run(name, func(t *testing.T) {
			dataDir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dataDir, name), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := checkPartialRestore(dataDir); err == nil {
				t.Fatalf("accepted partial restore with %s but no database", name)
			}
		})
	}
}

func TestHealthcheckUsesRunningServer(t *testing.T) {
	cfg := testConfig(t, t.TempDir())
	server := startTestServer(t, cfg, jobs.PythonProcessor{})

	// serve holds the exclusive app lock while healthcheck runs; success proves
	// the probe bypasses the lock and does not need its own database handle.
	if err := healthcheck(cfg); err != nil {
		server.cancel()
		t.Fatalf("healthcheck against the running server: %v", err)
	}
	server.stop(t)
	if err := healthcheck(cfg); err == nil {
		t.Fatal("healthcheck passed after the server stopped")
	}
}

type blockingProcessor struct {
	started   chan struct{}
	cancelled chan struct{}
	release   chan struct{}
	once      sync.Once
}

func newBlockingProcessor() *blockingProcessor {
	return &blockingProcessor{
		started:   make(chan struct{}),
		cancelled: make(chan struct{}),
		release:   make(chan struct{}),
	}
}

func (p *blockingProcessor) Process(ctx context.Context, _ jobs.JobRequest) (jobs.JobResult, error) {
	p.once.Do(func() { close(p.started) })
	<-ctx.Done()
	close(p.cancelled)
	<-p.release
	return jobs.JobResult{}, ctx.Err()
}

func TestShutdownCancelsChildWithoutPublishing(t *testing.T) {
	dataDir := t.TempDir()
	jobID := seedQueuedJob(t, dataDir)
	cfg := testConfig(t, dataDir)
	processor := newBlockingProcessor()

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- serveWith(ctx, cfg, processor) }()

	select {
	case <-processor.started:
	case err := <-errCh:
		cancel()
		t.Fatalf("serve exited before running the queued job: %v", err)
	case <-time.After(20 * time.Second):
		cancel()
		t.Fatal("queued job never reached the processor")
	}

	cancel()
	select {
	case <-processor.cancelled:
	case <-time.After(10 * time.Second):
		t.Fatal("shutdown did not cancel the child")
	}

	// serve must keep the database and lock open until the child has exited.
	time.Sleep(100 * time.Millisecond)
	select {
	case err := <-errCh:
		t.Fatalf("serve returned before the child exited: %v", err)
	default:
	}
	close(processor.release)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("graceful shutdown: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("serve did not stop after the child exited")
	}

	db, err := storage.Open(context.Background(), dataDir)
	if err != nil {
		t.Fatalf("reopen database: %v", err)
	}
	defer db.Close()
	var status, code string
	if err := db.QueryRow(`SELECT status, error_code FROM processing_jobs WHERE id = ?`, jobID).
		Scan(&status, &code); err != nil {
		t.Fatalf("read job: %v", err)
	}
	if status != "interrupted" || code != "shutdown" {
		t.Fatalf("job state = %s/%s, want interrupted/shutdown", status, code)
	}
	var versions int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pack_versions`).Scan(&versions); err != nil {
		t.Fatalf("count versions: %v", err)
	}
	if versions != 0 {
		t.Fatalf("shutdown published %d pack versions", versions)
	}
}

func TestStartupKeepsExistingOwnerWithoutBootstrapSecret(t *testing.T) {
	dataDir := t.TempDir()
	cfg := testConfig(t, dataDir)
	server := startTestServer(t, cfg, jobs.PythonProcessor{})
	server.stop(t)

	before := ownerHash(t, dataDir)
	if err := os.Remove(cfg.AdminPasswordFile); err != nil {
		t.Fatalf("remove bootstrap secret: %v", err)
	}
	cfg.AdminPasswordFile = ""
	cfg.HTTPAddr = freeAddr(t)
	server = startTestServer(t, cfg, jobs.PythonProcessor{})
	server.stop(t)
	if after := ownerHash(t, dataDir); after != before {
		t.Fatal("restart without the bootstrap secret changed the owner password hash")
	}
}

func ownerHash(t *testing.T, dataDir string) string {
	t.Helper()
	db, err := storage.Open(context.Background(), dataDir)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	var hash string
	if err := db.QueryRow(`SELECT password_hash FROM owner WHERE id = 1`).Scan(&hash); err != nil {
		t.Fatalf("read owner hash: %v", err)
	}
	return hash
}

func TestUnknownAPIPathIsNotHTML(t *testing.T) {
	cfg := testConfig(t, t.TempDir())
	server := startTestServer(t, cfg, jobs.PythonProcessor{})
	defer server.stop(t)

	for _, path := range []string{"/api/v1/nope", "/v1/nope"} {
		response, body := get(t, server.baseURL+path)
		if response.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want 404", path, response.StatusCode)
		}
		if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
			t.Errorf("GET %s content type = %q, want JSON", path, contentType)
		}
		if strings.Contains(strings.ToLower(body), "<html") || !strings.Contains(body, `"code":"not_found"`) {
			t.Errorf("GET %s body = %q, want JSON not_found", path, body)
		}
	}
}

func TestStaticRouteReload(t *testing.T) {
	cfg := testConfig(t, t.TempDir())
	server := startTestServer(t, cfg, jobs.PythonProcessor{})
	defer server.stop(t)

	response, _ := get(t, server.baseURL+"/")
	if response.StatusCode != http.StatusTemporaryRedirect || response.Header.Get("Location") != "/factory" {
		t.Fatalf("GET / = %d Location %q, want 307 /factory", response.StatusCode, response.Header.Get("Location"))
	}

	for _, path := range []string{"/factory", "/library", "/characters", "/packs", "/keys", "/api-docs", "/packs/detail?id=missing"} {
		response, body := get(t, server.baseURL+path)
		if response.StatusCode != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", path, response.StatusCode)
		}
		if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/html") || !strings.Contains(body, "<!DOCTYPE html>") {
			t.Errorf("GET %s did not serve the exported document", path)
		}
	}

	response, body := get(t, server.baseURL+"/openapi.yaml")
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "openapi: 3.1.0") {
		t.Fatalf("GET /openapi.yaml = %d, canonical document not served", response.StatusCode)
	}
}

// seedQueuedJob prepares a real queued job (pack, source file and request) so
// the production runner claims it at startup.
func seedQueuedJob(t *testing.T, dataDir string) string {
	t.Helper()
	ctx := context.Background()
	db, err := storage.Open(ctx, dataDir)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer db.Close()

	packID := uuid.NewString()
	sourceID := uuid.NewString()
	jobID := uuid.NewString()
	if _, err := db.Exec(`INSERT INTO packs (id, slug, name, author) VALUES (?, ?, ?, ?)`,
		packID, "shutdown-pack", "Shutdown Pack", "Tester"); err != nil {
		t.Fatalf("insert pack: %v", err)
	}
	sourceDir := filepath.Join(dataDir, "sources", sourceID)
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "original.png"), []byte("sheet"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	request := fmt.Sprintf(`{"schema_version":1,"job_id":%q,"pack_id":%q,"version":1,"source":{"id":%q}}`,
		jobID, packID, sourceID)
	if _, err := db.Exec(`INSERT INTO processing_jobs (id, pack_id, version, status, request_json)
		VALUES (?, ?, 1, 'queued', ?)`, jobID, packID, request); err != nil {
		t.Fatalf("insert job: %v", err)
	}
	var version int
	if err := db.QueryRow(`SELECT COUNT(*) FROM processing_jobs`).Scan(&version); err != nil || version != 1 {
		t.Fatalf("job fixture count = %d err %v", version, err)
	}
	return jobID
}
