// Package jobs owns the durable processing queue. Enqueue freezes a design,
// catalog and source snapshot and reserves a strictly increasing pack version;
// one worker runs the packaged image processor at a time and publishes the
// result through packs, which only commits while the job is running and the
// pack is not revoked.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/eykh/restick/api_svc/internal/studio"
)

const (
	// RequestSchemaVersion is the persisted processing_jobs.request_json schema.
	RequestSchemaVersion = 1
	// DefaultTimeout bounds one processor child.
	DefaultTimeout = 300 * time.Second
	// MaxResultBytes is the largest accepted processor result document.
	MaxResultBytes = 1 << 20
)

// Status is the lifecycle state of a processing job.
type Status string

const (
	StatusQueued      Status = "queued"
	StatusRunning     Status = "running"
	StatusSucceeded   Status = "succeeded"
	StatusFailed      Status = "failed"
	StatusInterrupted Status = "interrupted"
)

// Error codes stored in processing_jobs.error_code.
const (
	CodeTimeout         = "timeout"
	CodeProcessorFailed = "processor_failed"
	CodeResultInvalid   = "result_invalid"
	CodeStageFailed     = "stage_failed"
	CodeCommitFailed    = "commit_failed"
	CodePublishFailed   = "publish_failed"
	CodeShutdown        = "shutdown"
	CodeStartupRecovery = "startup_recovery"
	CodePackRevoked     = "pack_revoked"
)

var (
	ErrNotFound    = errors.New("jobs: job not found")
	ErrInvalid     = errors.New("jobs: invalid job")
	ErrJobActive   = errors.New("jobs: pack already has an active job")
	ErrPackRevoked = errors.New("jobs: pack is revoked")
	ErrNoSource    = errors.New("jobs: pack has no source sheet")
	// ErrAnimationBusy means a clip of the pack is queued or running.
	ErrAnimationBusy = errors.New("jobs: an animation clip is still processing")
	// ErrNoChanges means the animated set equals the current version's.
	ErrNoChanges = errors.New("jobs: no animation changes to publish")

	errInvalidResult = errors.New("jobs: processor result does not match the design")
)

// KindPublish marks a Go-only job that republishes the current version with
// the enabled animated drafts. An empty kind is a sheet job.
const KindPublish = "publish"

// JobRequest is the frozen input of one processing run. It is written at
// enqueue time, so later design or catalog edits never change a queued job.
type JobRequest struct {
	SchemaVersion int             `json:"schema_version"`
	Kind          string          `json:"kind,omitempty"`
	JobID         string          `json:"job_id"`
	PackID        string          `json:"pack_id"`
	Version       int             `json:"version"`
	DesignID      string          `json:"design_id,omitempty"`
	Design        studio.Snapshot `json:"design"`
	Source        SourceRef       `json:"source"`
	Catalog       CatalogRef      `json:"catalog"`
	Rows          int             `json:"rows"`
	Columns       int             `json:"columns"`
	OutlineWidth  int             `json:"outline_width"`
	OutlineColor  string          `json:"outline_color"`

	// Publish jobs only: the version they copy statics from and the animated
	// drafts checked at enqueue, so the job publishes exactly what was approved.
	BaseVersion int                    `json:"base_version,omitempty"`
	Animations  []PublishableAnimation `json:"animations,omitempty"`

	// SheetPath and WorkDir are resolved by the runner at execution time and
	// never persisted: the data root can move between backup and restore.
	SheetPath string `json:"-"`
	WorkDir   string `json:"-"`
}

// PublishableAnimation is a ready, enabled, non-stale animated draft. Path is
// data-root relative (animations/<id>.webp); Checksum is its SHA256 hex.
type PublishableAnimation struct {
	StickerIndex int    `json:"sticker_index"`
	Path         string `json:"path"`
	Checksum     string `json:"checksum_sha256"`
	Frames       int    `json:"frames"`
	DurationMS   int    `json:"duration_ms"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

// AnimationSource is the animated draft store a publish job reads. It is
// satisfied by animations.Service; jobs never imports that package.
type AnimationSource interface {
	Busy(ctx context.Context, packID string) (bool, error)
	Publishable(ctx context.Context, packID string) ([]PublishableAnimation, error)
}

// SourceRef is the frozen source sheet metadata of a job.
type SourceRef struct {
	ID             string `json:"id"`
	Filename       string `json:"filename"`
	ChecksumSHA256 string `json:"checksum_sha256"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	ByteSize       int64  `json:"byte_size"`
}

// CatalogRef is the mutable pack metadata frozen into the version manifest.
type CatalogRef struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Author      string   `json:"author"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// JobResult is what a Processor returns for one request. The runner validates
// it before anything is staged or published.
type JobResult struct {
	Stickers    []ResultSticker `json:"stickers"`
	Diagnostics string          `json:"diagnostics,omitempty"`
	// SuggestedThemeColor is a validated #RRGGBB, or "" when none was usable.
	SuggestedThemeColor string `json:"suggested_theme_color,omitempty"`
}

// ResultSticker is one processed cell. File and ThumbFile are flat basenames
// inside the job output directory, in row-major sort order.
type ResultSticker struct {
	File      string `json:"file"`
	ThumbFile string `json:"thumb_file"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

// Job is a persisted queue row. ResultJSON is the publication summary written
// when the version committed; StartedAt and FinishedAt are zero until set.
type Job struct {
	ID           string
	PackID       string
	Version      int
	Status       Status
	Request      JobRequest
	ResultJSON   json.RawMessage
	ErrorCode    string
	ErrorMessage string
	Diagnostics  string
	CreatedAt    time.Time
	StartedAt    time.Time
	FinishedAt   time.Time
}
