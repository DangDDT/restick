package jobs

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

	"github.com/eykh/restick/api_svc/internal/packs"
)

// Processor is the scheduling boundary around one image processing run. The
// production implementation execs the packaged Python machine job; tests may
// substitute their own while exercising the queue.
type Processor interface {
	Process(ctx context.Context, request JobRequest) (JobResult, error)
}

const (
	// pythonExecutable and machineJobModule are the fixed child argv from spec
	// §7.1. Nothing from an HTTP request ever becomes the executable or an argv
	// element: only server-generated job paths are appended.
	pythonExecutable = "python"
	machineJobModule = "media_factory.interfaces.machine_job"

	requestSchema = "restick.job.v1"
	resultSchema  = "restick.job-result.v1"
	maxStderrSize = 4 << 10
)

// PythonProcessor runs media_factory.interfaces.machine_job without a shell.
type PythonProcessor struct{}

// Process writes the request document, runs the child and reads the bounded
// result. A non-zero exit is always a failure, even when a result file exists.
func (PythonProcessor) Process(ctx context.Context, request JobRequest) (JobResult, error) {
	if request.WorkDir == "" || request.SheetPath == "" {
		return JobResult{}, errors.New("jobs: request paths were not resolved")
	}
	document := machineRequest{
		Schema:       requestSchema,
		JobID:        request.JobID,
		PackID:       request.PackID,
		Version:      request.Version,
		SheetPath:    request.SheetPath,
		OutputDir:    filepath.Join(request.WorkDir, "output"),
		Rows:         request.Rows,
		Columns:      request.Columns,
		OutlineWidth: request.OutlineWidth,
		OutlineColor: request.OutlineColor,
	}
	payload, err := json.Marshal(document)
	if err != nil {
		return JobResult{}, fmt.Errorf("jobs: encode processor request: %w", err)
	}
	requestPath := filepath.Join(request.WorkDir, "request.json")
	resultPath := filepath.Join(request.WorkDir, "result.json")
	if err := writeFile(requestPath, payload); err != nil {
		return JobResult{}, err
	}
	// A result left by an earlier attempt must never be read as this run's.
	if err := os.Remove(resultPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return JobResult{}, fmt.Errorf("jobs: clear stale result: %w", err)
	}

	stderr := &boundedBuffer{limit: maxStderrSize}
	cmd := exec.CommandContext(ctx, pythonExecutable, "-m", machineJobModule,
		"--request", requestPath, "--result", resultPath)
	cmd.Stdout = io.Discard
	cmd.Stderr = stderr
	runErr := cmd.Run()
	if err := ctx.Err(); err != nil {
		return JobResult{}, err
	}
	if runErr != nil {
		if failure := readFailure(resultPath, request); failure != nil {
			return JobResult{}, failure
		}
		return JobResult{}, &ProcessError{Message: stderrMessage(stderr, runErr)}
	}
	return readSuccess(resultPath, request)
}

// ProcessError is a bounded child failure. The non-zero exit code is the
// authority; the machine error code and message are diagnostics only.
type ProcessError struct {
	Code    string
	Message string
}

func (e *ProcessError) Error() string {
	switch {
	case e.Code != "" && e.Message != "":
		return "processor failed: " + e.Code + ": " + e.Message
	case e.Message != "":
		return "processor failed: " + e.Message
	default:
		return "processor failed"
	}
}

type machineRequest struct {
	Schema       string `json:"schema"`
	JobID        string `json:"job_id"`
	PackID       string `json:"pack_id"`
	Version      int    `json:"version"`
	SheetPath    string `json:"sheet_path"`
	OutputDir    string `json:"output_dir"`
	Rows         int    `json:"rows"`
	Columns      int    `json:"columns"`
	OutlineWidth int    `json:"outline_width"`
	OutlineColor string `json:"outline_color"`
}

type machineResult struct {
	Schema              string           `json:"schema"`
	JobID               string           `json:"job_id"`
	PackID              string           `json:"pack_id"`
	Version             int              `json:"version"`
	Stickers            []machineSticker `json:"stickers"`
	Diagnostics         json.RawMessage  `json:"diagnostics"`
	SuggestedThemeColor string           `json:"suggested_theme_color"`
	Error               *machineError    `json:"error"`
}

type machineSticker struct {
	Filename string `json:"filename"`
	Thumb    string `json:"thumb"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type machineError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func readSuccess(path string, request JobRequest) (JobResult, error) {
	data, err := readBounded(path)
	if err != nil {
		return JobResult{}, err
	}
	var document machineResult
	if err := json.Unmarshal(data, &document); err != nil {
		return JobResult{}, fmt.Errorf("jobs: decode processor result: %w", err)
	}
	if document.Schema != resultSchema {
		return JobResult{}, fmt.Errorf("jobs: unexpected result schema %q", document.Schema)
	}
	if !sameIdentity(document.JobID, document.PackID, document.Version, request) {
		return JobResult{}, errors.New("jobs: result identity does not match the request")
	}
	if document.Error != nil {
		return JobResult{}, &ProcessError{Code: document.Error.Code, Message: document.Error.Message}
	}
	if len(document.Stickers) == 0 {
		return JobResult{}, errors.New("jobs: processor result has no stickers")
	}
	result := JobResult{
		Stickers:    make([]ResultSticker, len(document.Stickers)),
		Diagnostics: diagnosticsText(document.Diagnostics),
	}
	// A malformed suggestion is dropped; it must never fail the job.
	if color, ok := packs.NormalizeThemeColor(document.SuggestedThemeColor); ok {
		result.SuggestedThemeColor = color
	}
	for i, sticker := range document.Stickers {
		result.Stickers[i] = ResultSticker{
			File:      sticker.Filename,
			ThumbFile: sticker.Thumb,
			Width:     sticker.Width,
			Height:    sticker.Height,
		}
	}
	return result, nil
}

// readFailure reports the machine error code of a failed run when the result
// document is readable and belongs to this request.
func readFailure(path string, request JobRequest) *ProcessError {
	data, err := readBounded(path)
	if err != nil {
		return nil
	}
	var document machineResult
	if err := json.Unmarshal(data, &document); err != nil {
		return nil
	}
	if document.Error == nil || !sameIdentity(document.JobID, document.PackID, document.Version, request) {
		return nil
	}
	return &ProcessError{Code: document.Error.Code, Message: document.Error.Message}
}

func sameIdentity(jobID, packID string, version int, request JobRequest) bool {
	return jobID == request.JobID && packID == request.PackID && version == request.Version
}

// diagnosticsText keeps the child diagnostics as compact JSON text.
func diagnosticsText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return ""
	}
	text := compact.String()
	if len(text) > maxDiagnostics {
		return ""
	}
	return text
}

// readBounded reads at most MaxResultBytes and refuses a larger document.
func readBounded(path string) ([]byte, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, fmt.Errorf("jobs: open processor result: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxResultBytes+1))
	if err != nil {
		return nil, fmt.Errorf("jobs: read processor result: %w", err)
	}
	if len(data) > MaxResultBytes {
		return nil, errors.New("jobs: processor result exceeds 1 MiB")
	}
	return data, nil
}

func writeFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("jobs: create request: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("jobs: write request: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("jobs: fsync request: %w", err)
	}
	return file.Close()
}

func stderrMessage(stderr *boundedBuffer, runErr error) string {
	message := strings.TrimSpace(stderr.String())
	if message == "" {
		message = runErr.Error()
	}
	if len(message) > maxMessageChars {
		message = message[:maxMessageChars]
	}
	return message
}

type boundedBuffer struct {
	buf   bytes.Buffer
	limit int
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if remaining := b.limit - b.buf.Len(); remaining > 0 {
		if len(p) > remaining {
			b.buf.Write(p[:remaining])
		} else {
			b.buf.Write(p)
		}
	}
	return len(p), nil
}

func (b *boundedBuffer) String() string {
	return b.buf.String()
}
