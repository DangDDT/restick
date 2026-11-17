package jobs

import (
	"os"
	"path/filepath"
	"testing"
)

func readSuggested(t *testing.T, color string) (JobResult, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "result.json")
	body := `{"schema":"` + resultSchema + `","job_id":"j1","pack_id":"p1","version":1,
		"stickers":[{"filename":"001.png","thumb":"001.webp","width":8,"height":8}],
		"suggested_theme_color":"` + color + `"}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return readSuccess(path, JobRequest{JobID: "j1", PackID: "p1", Version: 1})
}

func TestReadSuccessParsesSuggestedThemeColor(t *testing.T) {
	result, err := readSuggested(t, "#a98bf0")
	if err != nil || result.SuggestedThemeColor != "#A98BF0" {
		t.Fatalf("got %q, %v; want #A98BF0", result.SuggestedThemeColor, err)
	}
}

func TestReadSuccessIgnoresInvalidSuggestedThemeColor(t *testing.T) {
	result, err := readSuggested(t, "purple")
	if err != nil || result.SuggestedThemeColor != "" {
		t.Fatalf("got %q, %v; want empty and no error", result.SuggestedThemeColor, err)
	}
}
