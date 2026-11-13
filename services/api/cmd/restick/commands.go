package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eykh/restick/api_svc/internal/animations"
	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/auth"
	"github.com/eykh/restick/api_svc/internal/infrastructure/config"
	httpx "github.com/eykh/restick/api_svc/internal/interfaces/http"
	"github.com/eykh/restick/api_svc/internal/jobs"
	"github.com/eykh/restick/api_svc/internal/maintenance"
	"github.com/eykh/restick/api_svc/internal/packs"
	"github.com/eykh/restick/api_svc/internal/storage"
	"github.com/eykh/restick/api_svc/internal/studio"
)

// serve starts the production runtime with the packaged Python child.
func serve(ctx context.Context, cfg config.FactoryConfig) error {
	return serveWith(ctx, cfg, jobs.PythonProcessor{})
}

// serveWith wires the single factory runtime. Startup order is fixed:
// app lock -> partial-restore guard -> database/schema -> owner bootstrap
// (only when absent) -> seed -> recovery -> HTTP + job runner + clip worker. On
// shutdown both children are cancelled and waited for before the database and
// lock close.
func serveWith(ctx context.Context, cfg config.FactoryConfig, processor jobs.Processor) error {
	release, err := storage.AcquireAppLock(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("restick: acquire data dir: %w", err)
	}
	// Deferred before the database handle so the DB closes first, then the lock.
	defer release()

	if err := checkPartialRestore(cfg.DataDir); err != nil {
		return err
	}
	exists, err := databaseExists(cfg.DataDir)
	if err != nil {
		return err
	}
	var bootstrapPassword []byte
	if !exists {
		// Refuse before creating a database when a fresh data dir has no secret.
		bootstrapPassword, err = readPasswordFile(cfg.AdminPasswordFile)
		if err != nil {
			return fmt.Errorf("restick: fresh data dir %s: %w", cfg.DataDir, err)
		}
	}

	db, err := storage.Open(ctx, cfg.DataDir)
	if err != nil {
		return err
	}
	defer db.Close()

	files, err := assets.NewStore(cfg.DataDir)
	if err != nil {
		return err
	}
	authSvc := auth.NewService(db)
	hasOwner, err := authSvc.HasOwner(ctx)
	if err != nil {
		return err
	}
	if !hasOwner {
		if bootstrapPassword == nil {
			bootstrapPassword, err = readPasswordFile(cfg.AdminPasswordFile)
			if err != nil {
				return fmt.Errorf("restick: database has no owner: %w", err)
			}
		}
		if err := authSvc.BootstrapOwner(ctx, bootstrapPassword); err != nil {
			return fmt.Errorf("restick: bootstrap owner: %w", err)
		}
		bootstrapPassword = nil
	}
	// An existing owner is never touched: no secret is read and no hash changes.

	studioSvc := studio.NewService(db, files)
	if err := studioSvc.Seed(ctx); err != nil {
		return err
	}
	animationSvc := animations.NewService(db, files)
	runner := jobs.NewRunner(db, files, processor, cfg.ProcessTimeout, jobs.WithAnimations(animationSvc))
	if err := runner.Recover(ctx); err != nil {
		return err
	}
	// Before HTTP starts: clip recovery prunes clips no row references yet.
	if err := animationSvc.Recover(ctx); err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	handler := httpx.NewHandler(httpx.Dependencies{
		Auth:              authSvc,
		Studio:            studioSvc,
		Packs:             packs.NewService(db),
		Jobs:              runner,
		Assets:            files,
		Animations:        animationSvc,
		PublicBaseURL:     cfg.PublicBaseURL,
		TrustedProxyCIDRs: cfg.TrustedProxyCIDRs,
		WebDir:            cfg.WebDir,
		OpenAPIPath:       cfg.OpenAPIPath,
		Ready:             readiness(db, cfg.DataDir),
	})

	runnerErr := make(chan error, 1)
	clipErr := make(chan error, 1)
	httpErr := make(chan error, 1)
	go func() { runnerErr <- runner.Run(runCtx) }()
	go func() {
		clipErr <- animationSvc.RunWorker(runCtx, animations.PythonClipProcessor{}, runner.ChildLock())
	}()
	go func() { httpErr <- httpx.Listen(runCtx, cfg.HTTPAddr, handler) }()
	slog.Info("restick listening", "http", cfg.HTTPAddr, "data", cfg.DataDir)

	// Whichever side stops first cancels the other; a parent cancel is a
	// graceful shutdown. Both are awaited before the deferred DB/lock close.
	var serveErr error
	remaining := 3
	parentDone := ctx.Done()
	for remaining > 0 {
		select {
		case err := <-runnerErr:
			remaining--
			cancel()
			if err != nil && serveErr == nil {
				serveErr = fmt.Errorf("restick: job runner: %w", err)
			}
		case err := <-clipErr:
			remaining--
			cancel()
			if err != nil && serveErr == nil {
				serveErr = fmt.Errorf("restick: clip worker: %w", err)
			}
		case err := <-httpErr:
			remaining--
			cancel()
			if err != nil && serveErr == nil {
				serveErr = fmt.Errorf("restick: http server: %w", err)
			}
		case <-parentDone:
			cancel()
			parentDone = nil
		}
	}
	return serveErr
}

// checkPartialRestore refuses a data dir that has resource directories or a
// restore staging tree but no database. It never deletes or initializes that
// tree: a failed restore must be retried into another new empty directory.
func checkPartialRestore(dataDir string) error {
	exists, err := databaseExists(dataDir)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return fmt.Errorf("restick: inspect data dir %s: %w", dataDir, err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == storage.LockFileName {
			continue
		}
		if name == "sources" || name == "work" || name == "packs" || name == "bundles" ||
			name == "clips" || name == "animations" ||
			strings.HasPrefix(name, ".restick-restore-") {
			return fmt.Errorf("restick: data dir %s holds %s but no %s; retry the restore into a new empty data dir instead of initialising on top of the partial tree",
				dataDir, name, storage.DBFileName)
		}
	}
	return nil
}

func databaseExists(dataDir string) (bool, error) {
	_, err := os.Stat(filepath.Join(dataDir, storage.DBFileName))
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("restick: inspect data dir %s: %w", dataDir, err)
	}
}

// readiness is the /healthz service state: the database answers and the data
// root accepts writes. The error stays inside the server.
func readiness(db *sql.DB, dataDir string) func(context.Context) error {
	return func(ctx context.Context) error {
		if err := db.PingContext(ctx); err != nil {
			return err
		}
		probe, err := os.CreateTemp(dataDir, ".restick-health-")
		if err != nil {
			return err
		}
		name := probe.Name()
		defer os.Remove(name)
		if _, err := probe.Write([]byte("ok")); err != nil {
			probe.Close()
			return err
		}
		return probe.Close()
	}
}

// healthcheck probes the running server over loopback HTTP only. It never takes
// the app lock or opens the database, so it is safe while serve holds both.
func healthcheck(cfg config.FactoryConfig) error {
	url := loopbackURL(cfg.HTTPAddr)
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("restick: healthcheck %s: %w", url, err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("restick: healthcheck %s: status %d", url, response.StatusCode)
	}
	return nil
}

func loopbackURL(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr + "/healthz"
	}
	return "http://" + net.JoinHostPort("127.0.0.1", port) + "/healthz"
}

// setPassword resets the owner password and every admin session under the
// exclusive app lock, so it must run while the server is stopped.
func setPassword(ctx context.Context, dataDir, passwordFile string) error {
	password, err := readPasswordFile(passwordFile)
	if err != nil {
		return err
	}
	release, err := storage.AcquireAppLock(dataDir)
	if err != nil {
		return fmt.Errorf("restick: stop the service first: %w", err)
	}
	defer release()
	exists, err := databaseExists(dataDir)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("restick: %s has no %s", dataDir, storage.DBFileName)
	}
	db, err := storage.Open(ctx, dataDir)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := auth.NewService(db).SetPassword(ctx, password); err != nil {
		return fmt.Errorf("restick: set password: %w", err)
	}
	return nil
}

// backup snapshots a stopped data dir. Backup itself takes the app lock.
func backup(ctx context.Context, dataDir, output string) error {
	if err := maintenance.Backup(ctx, dataDir, output); err != nil {
		if errors.Is(err, storage.ErrAppLocked) {
			return fmt.Errorf("restick: stop the service before backup: %w", err)
		}
		return err
	}
	return nil
}

// restore rebuilds a new data dir from a backup. Restore takes the app lock and
// requires an empty target plus the new owner password.
func restore(ctx context.Context, dataDir, inputDir, passwordFile string) error {
	if err := maintenance.Restore(ctx, inputDir, dataDir, passwordFile); err != nil {
		if errors.Is(err, storage.ErrAppLocked) {
			return fmt.Errorf("restick: stop the service before restore: %w", err)
		}
		return err
	}
	return nil
}

// readPasswordFile trims surrounding whitespace (the normal secret-file form)
// and rejects a missing or blank file before any state changes.
func readPasswordFile(path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("password file is required")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read password file: %w", err)
	}
	password := bytes.TrimSpace(body)
	if len(password) == 0 {
		return nil, errors.New("password file is empty after trimming whitespace")
	}
	return password, nil
}
