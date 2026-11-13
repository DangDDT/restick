package animations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/eykh/restick/api_svc/internal/jobs"
)

const (
	// pythonExecutable and clipJobModule are the fixed child argv. Nothing from
	// an HTTP request becomes the executable or an argv element: only
	// server-generated work paths are appended.
	pythonExecutable = "python"
	clipJobModule    = "media_factory.interfaces.clip_job"

	clipRequestSchema = "restick.clip-request.v1"
	clipResultSchema  = "restick.clip-result.v1"
	maxStderrSize     = 4 << 10
	maxMessageChars   = 1000
	maxDiagnostics    = 4000
)

// ClipRequest is one clip run. OutputDir must be <work dir>/output: the
// request and result documents are written next to it, in the work dir.
type ClipRequest struct {
	AnimationID  string
	PackID       string
	StickerIndex int
	ClipPath     string
	OutputDir    string
	FPS          int
	Quality      int
	OutlineWidth int
	OutlineColor string
}

// ClipResult is the child's claim about the animated file it wrote below
// OutputDir; the worker re-verifies the file before trusting it.
type ClipResult struct {
	Filename    string
	Frames      int
	FPS         int
	DurationMS  int
	ByteSize    int
	Quality     int
	Width       int
	Height      int
	Diagnostics string
}

// ClipProcessor is the scheduling boundary around one clip child; tests
// substitute a fake while exercising the queue.
type ClipProcessor interface {
	ProcessClip(ctx context.Context, r ClipRequest) (ClipResult, error)
}

// PythonClipProcessor runs media_factory.interfaces.clip_job without a shell.
type PythonClipProcessor struct{}

type clipDocument struct {
	Schema       string `json:"schema"`
	JobID        string `json:"job_id"`
	PackID       string `json:"pack_id"`
	StickerIndex int    `json:"sticker_index"`
	ClipPath     string `json:"clip_path"`
	OutputDir    string `json:"output_dir"`
	FPS          int    `json:"fps"`
	Quality      int    `json:"quality"`
	OutlineWidth int    `json:"outline_width"`
	OutlineColor string `json:"outline_color"`
}

type clipResultDocument struct {
	Schema    string `json:"schema"`
	JobID     string `json:"job_id"`
	PackID    string `json:"pack_id"`
	Animation *struct {
		Filename   string `json:"filename"`
		Frames     int    `json:"frames"`
		FPS        int    `json:"fps"`
		DurationMS int    `json:"duration_ms"`
		ByteSize   int    `json:"byte_size"`
		Quality    int    `json:"quality"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
	} `json:"animation"`
	Diagnostics json.RawMessage `json:"diagnostics"`
	Error       *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ProcessClip writes the request, runs the child and reads the bounded result.
// A non-zero exit is always a failure, even when a result file exists.
func (PythonClipProcessor) ProcessClip(ctx context.Context, r ClipRequest) (ClipResult, error) {
	if r.OutputDir == "" || r.ClipPath == "" {
		return ClipResult{}, errors.New("animations: clip request paths were not resolved")
	}
	payload, err := json.Marshal(clipDocument{
		Schema: clipRequestSchema, JobID: r.AnimationID, PackID: r.PackID, StickerIndex: r.StickerIndex,
		ClipPath: r.ClipPath, OutputDir: r.OutputDir, FPS: r.FPS, Quality: r.Quality,
		OutlineWidth: r.OutlineWidth, OutlineColor: r.OutlineColor,
	})
	if err != nil {
		return ClipResult{}, fmt.Errorf("animations: encode clip request: %w", err)
	}
	workDir := filepath.Dir(r.OutputDir)
	requestPath := filepath.Join(workDir, "request.json")
	resultPath := filepath.Join(workDir, "result.json")
	if err := writeNoFollow(requestPath, payload); err != nil {
		return ClipResult{}, err
	}
	// A result left by an earlier attempt must never be read as this run's.
	if err := os.Remove(resultPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return ClipResult{}, fmt.Errorf("animations: clear stale result: %w", err)
	}

	stderr := &boundedBuffer{limit: maxStderrSize}
	cmd := exec.CommandContext(ctx, pythonExecutable, "-m", clipJobModule,
		"--request", requestPath, "--result", resultPath)
	cmd.Stdout = io.Discard
	cmd.Stderr = stderr
	runErr := cmd.Run()
	if err := ctx.Err(); err != nil {
		return ClipResult{}, err
	}
	document, readErr := readClipResult(resultPath, r)
	if runErr != nil {
		if readErr == nil && document.Error != nil {
			return ClipResult{}, &jobs.ProcessError{Code: document.Error.Code, Message: document.Error.Message}
		}
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = runErr.Error()
		}
		return ClipResult{}, &jobs.ProcessError{Message: truncate(message, maxMessageChars)}
	}
	if readErr != nil {
		return ClipResult{}, readErr
	}
	if document.Error != nil {
		return ClipResult{}, &jobs.ProcessError{Code: document.Error.Code, Message: document.Error.Message}
	}
	if document.Animation == nil {
		return ClipResult{}, errors.New("animations: clip result has no animation")
	}
	a := document.Animation
	return ClipResult{
		Filename: a.Filename, Frames: a.Frames, FPS: a.FPS, DurationMS: a.DurationMS, ByteSize: a.ByteSize,
		Quality: a.Quality, Width: a.Width, Height: a.Height, Diagnostics: diagnosticsText(document.Diagnostics),
	}, nil
}

// readClipResult reads at most jobs.MaxResultBytes and checks the schema and
// that the result belongs to this request.
func readClipResult(p string, r ClipRequest) (clipResultDocument, error) {
	file, err := os.OpenFile(p, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return clipResultDocument{}, fmt.Errorf("animations: open clip result: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, jobs.MaxResultBytes+1))
	if err != nil {
		return clipResultDocument{}, fmt.Errorf("animations: read clip result: %w", err)
	}
	if len(data) > jobs.MaxResultBytes {
		return clipResultDocument{}, errors.New("animations: clip result exceeds 1 MiB")
	}
	var document clipResultDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return clipResultDocument{}, fmt.Errorf("animations: decode clip result: %w", err)
	}
	if document.Schema != clipResultSchema {
		return clipResultDocument{}, fmt.Errorf("animations: unexpected clip result schema %q", document.Schema)
	}
	if document.JobID != r.AnimationID || document.PackID != r.PackID {
		return clipResultDocument{}, errors.New("animations: clip result identity does not match the request")
	}
	return document, nil
}

func diagnosticsText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil || compact.Len() > maxDiagnostics {
		return ""
	}
	return compact.String()
}

func writeNoFollow(p string, data []byte) error {
	file, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("animations: create clip request: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("animations: write clip request: %w", err)
	}
	return file.Close()
}

func truncate(s string, limit int) string {
	if len(s) > limit {
		return s[:limit]
	}
	return s
}

type boundedBuffer struct {
	buf   bytes.Buffer
	limit int
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if remaining := b.limit - b.buf.Len(); remaining > 0 {
		b.buf.Write(p[:min(len(p), remaining)])
	}
	return len(p), nil
}

func (b *boundedBuffer) String() string { return b.buf.String() }
