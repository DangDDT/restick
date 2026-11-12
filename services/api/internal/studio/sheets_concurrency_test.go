package studio

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eykh/restick/api_svc/internal/storage"
)

// Pause each real SQLite insert after SaveSheet's initial duplicate lookup.
type sheetInsertBarrier struct {
	base    driver.Driver
	dsn     string
	arrived chan struct{}
	release chan struct{}
}

func (b *sheetInsertBarrier) Driver() driver.Driver { return b.base }

func (b *sheetInsertBarrier) Connect(context.Context) (driver.Conn, error) {
	conn, err := b.base.Open(b.dsn)
	if err != nil {
		return nil, err
	}
	return &sheetInsertBarrierConn{Conn: conn, barrier: b}, nil
}

type sheetInsertBarrierConn struct {
	driver.Conn
	barrier *sheetInsertBarrier
}

func (c *sheetInsertBarrierConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if strings.Contains(query, "INSERT INTO design_sheets") {
		c.barrier.arrived <- struct{}{}
		select {
		case <-c.barrier.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return c.Conn.(driver.ExecerContext).ExecContext(ctx, query, args)
}

func TestSaveSheetConcurrentDuplicateAtLimit(t *testing.T) {
	service, db, dir, design := sheetTestService(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for i := range 19 {
		if _, err := service.SaveSheet(ctx, design.ID, bytes.NewReader(pngVariant(t, i)), "a.png", "gemini", ""); err != nil {
			t.Fatalf("seed sheet %d: %v", i, err)
		}
	}

	barrier := &sheetInsertBarrier{
		base: db.Driver(), dsn: filepath.Join(dir, storage.DBFileName) + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)",
		arrived: make(chan struct{}, 2), release: make(chan struct{}),
	}
	concurrentDB := sql.OpenDB(barrier)
	defer concurrentDB.Close()
	concurrentDB.SetMaxOpenConns(2)
	service = NewService(concurrentDB, service.files)
	data := pngVariant(t, 999)
	ids := make([]string, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sheet, err := service.SaveSheet(ctx, design.ID, bytes.NewReader(data), "a.png", "gemini", "")
			ids[i], errs[i] = sheet.ID, err
		}()
	}
	// Both lookups must miss before either upload can claim the last slot.
	for range 2 {
		select {
		case <-barrier.arrived:
		case <-ctx.Done():
			wg.Wait()
			t.Fatalf("uploads did not reach insert barrier: %v, results %v", ctx.Err(), errs)
		}
	}
	close(barrier.release)
	wg.Wait()
	if errs[0] != nil || errs[1] != nil || ids[0] == "" || ids[0] != ids[1] {
		t.Errorf("errors = %v, ids = %v; want the same winning sheet", errs, ids)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM design_sheets WHERE design_id = ?`, design.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 20 {
		t.Errorf("design_sheets rows = %d, want 20", count)
	}
	if n := sourceDirs(t, dir); n != 20 {
		t.Errorf("sources has %d dirs, want 20", n)
	}
}
