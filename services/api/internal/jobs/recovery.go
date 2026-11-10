package jobs

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
)

// Recover is the idempotent startup step: jobs left running by a previous
// process become interrupted, queued jobs are left for the worker and files no
// committed version references are removed. It never deletes committed data.
func (r *Runner) Recover(ctx context.Context) error {
	r.recoverOnce.Do(func() { r.recoverErr = r.recover(ctx) })
	return r.recoverErr
}

func (r *Runner) recover(ctx context.Context) error {
	// A dead process is never evidence of completion.
	if _, err := r.db.ExecContext(ctx, `UPDATE processing_jobs SET status = 'interrupted',
		error_code = ?, error_message = ?,
		finished_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE status = 'running'`,
		CodeStartupRecovery, "the server stopped while this job was running"); err != nil {
		return fmt.Errorf("jobs: interrupt stale jobs: %w", err)
	}
	r.cleanupOrphans(ctx)
	return nil
}

// cleanupOrphans removes the files of terminal jobs whose version never
// committed: the crash window between the file rename and the database commit
// plus failed attempts. Unreferenced files are never public, so a cleanup
// failure must not block startup and is ignored.
func (r *Runner) cleanupOrphans(ctx context.Context) {
	rows, err := r.db.QueryContext(ctx, `SELECT j.id, j.pack_id, j.version FROM processing_jobs j
		WHERE j.status IN ('failed', 'interrupted')
		AND NOT EXISTS (
			SELECT 1 FROM pack_versions v WHERE v.pack_id = j.pack_id AND v.version = j.version
		)`)
	if err != nil {
		return
	}
	type orphan struct {
		jobID   string
		packID  string
		version int
	}
	var orphans []orphan
	for rows.Next() {
		var item orphan
		if err := rows.Scan(&item.jobID, &item.packID, &item.version); err != nil {
			rows.Close()
			return
		}
		orphans = append(orphans, item)
	}
	rows.Close()
	if rows.Err() != nil {
		return
	}
	for _, item := range orphans {
		version := fmt.Sprintf("v%d", item.version)
		r.removeCommitted(path.Join("packs", item.packID, version, "pack.json"), true)
		r.removeCommitted(path.Join("bundles", item.packID, version+".zip"), false)
		// WorkDir creates the directory, so a missing work dir needs no special
		// case: it is created and removed in one step.
		if workDir, err := r.files.WorkDir(item.jobID); err == nil {
			_ = os.RemoveAll(workDir)
		}
	}
}

// removeCommitted resolves a data-root-relative file through the store's
// allowlist before removing it, so a cleanup path can never leave the root.
func (r *Runner) removeCommitted(relative string, removeDir bool) {
	file, err := r.files.OpenCommitted(relative)
	if err != nil {
		return
	}
	name := file.Name()
	file.Close()
	if removeDir {
		_ = os.RemoveAll(filepath.Dir(name))
		return
	}
	_ = os.Remove(name)
}
