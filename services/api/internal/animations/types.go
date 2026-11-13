// Package animations keeps the per-sticker animated drafts of a pack: the
// uploaded clip, its processing status on the clip queue and the resulting
// animated WebP, until a publish job copies enabled drafts into a version.
package animations

import "errors"

// Status is the clip queue state of one draft.
type Status string

const (
	StatusQueued      Status = "queued"
	StatusRunning     Status = "running"
	StatusReady       Status = "ready"
	StatusFailed      Status = "failed"
	StatusInterrupted Status = "interrupted"
)

// Errors returned to the admin HTTP layer; codes are mapped there.
var (
	ErrNotFound  = errors.New("animations: not found")
	ErrInvalid   = errors.New("animations: invalid input")
	ErrJobActive = errors.New("animations: a job is active for this pack")
	ErrNotReady  = errors.New("animations: animation is not ready")
)

// Animation is one draft. Stale is computed on read: the static sticker at
// StickerIndex in the current version no longer has StaticChecksum.
type Animation struct {
	ID               string
	PackID           string
	StickerIndex     int
	StaticChecksum   string
	MotionText       string
	ClipByteSize     int64
	Status           Status
	UseAnimated      bool
	AnimatedPath     string
	AnimatedChecksum string
	Frames           int
	FPS              int
	DurationMS       int
	Quality          int
	Width            int
	Height           int
	AnimatedByteSize int64
	ErrorCode        string
	ErrorMessage     string
	Stale            bool
}
