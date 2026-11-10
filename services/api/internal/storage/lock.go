package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// lockFileName is the exclusive app/maintenance lock inside the data dir.
const lockFileName = ".restick.lock"

// ErrAppLocked reports that another restick process holds the data dir lock.
var ErrAppLocked = errors.New("storage: data dir is locked by another restick process")

// AcquireAppLock holds the exclusive app/maintenance lock on dataDir until
// release is called. Serve, backup, restore and set-password all take it so two
// writers can never share one SQLite file.
func AcquireAppLock(dataDir string) (release func() error, err error) {
	dir, err := prepareDataDir(dataDir)
	if err != nil {
		return nil, err
	}
	lockPath := filepath.Join(dir, lockFileName)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("storage: open app lock %s: %w", lockPath, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("%w: %s: %v", ErrAppLocked, lockPath, err)
	}
	// Best effort: the pid inside the lock file is a diagnostic, not the lock.
	_ = f.Truncate(0)
	_, _ = f.WriteAt([]byte(strconv.Itoa(os.Getpid())+"\n"), 0)
	_ = f.Sync()

	var once sync.Once
	var releaseErr error
	return func() error {
		once.Do(func() {
			releaseErr = errors.Join(syscall.Flock(int(f.Fd()), syscall.LOCK_UN), f.Close())
		})
		return releaseErr
	}, nil
}

func prepareDataDir(dataDir string) (string, error) {
	if strings.TrimSpace(dataDir) == "" {
		return "", errors.New("storage: data dir is required")
	}
	dir, err := filepath.Abs(dataDir)
	if err != nil {
		return "", fmt.Errorf("storage: resolve data dir %s: %w", dataDir, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("storage: prepare data dir %s: %w", dir, err)
	}
	return dir, nil
}
