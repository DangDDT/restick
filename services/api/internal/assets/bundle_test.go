package assets

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/uuid"
)

func testManifest(packID string, count int) PackManifest {
	manifest := PackManifest{
		PackID:      packID,
		Slug:        "demo-pack",
		Name:        "Demo Pack",
		Author:      "Tester",
		Description: "immutable snapshot",
		Tags:        []string{"animals", "cute"},
		Version:     1,
	}
	for i := range count {
		manifest.Stickers = append(manifest.Stickers, ManifestSticker{
			File:      fixedObjectName(i),
			ThumbFile: fixedThumbName(i),
			Width:     2,
			Height:    2,
			EmojiHint: "🙂",
			Keywords:  []string{"demo"},
		})
	}
	return manifest
}

func fixedObjectName(index int) string {
	return fmt.Sprintf("%03d.webp", index)
}

func fixedThumbName(index int) string {
	return "thumb_" + fixedObjectName(index)
}

func writeOutput(t *testing.T, store *Store, jobID string, files map[string][]byte) string {
	t.Helper()
	workDir, err := store.WorkDir(jobID)
	if err != nil {
		t.Fatalf("WorkDir: %v", err)
	}
	output := filepath.Join(workDir, "output")
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(output, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return output
}

func TestBundleLayoutAndStickerChecksum(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	packID := uuid.NewString()
	jobID := "job_" + uuid.NewString()
	object0 := mustWebP(t, tinyWebP2x2)
	object1 := mustWebP(t, tinyWebP3x3)
	writeOutput(t, store, jobID, map[string][]byte{
		"000.webp":       object0,
		"thumb_000.webp": object0,
		"001.webp":       object1,
		"thumb_001.webp": object1,
	})
	manifest := testManifest(packID, 2)
	manifest.Stickers[1].Width = 3
	manifest.Stickers[1].Height = 3

	staged, err := store.StageBundle(ctx, jobID, manifest)
	if err != nil {
		t.Fatalf("StageBundle: %v", err)
	}
	if staged.StickerCount != 2 {
		t.Errorf("sticker count = %d", staged.StickerCount)
	}
	wantSum := sha256.New()
	wantSum.Write(object0)
	wantSum.Write(object1)
	wantChecksum := hex.EncodeToString(wantSum.Sum(nil))
	if staged.ChecksumSHA256 != wantChecksum {
		t.Errorf("checksum = %s, want concat of sticker bytes %s", staged.ChecksumSHA256, wantChecksum)
	}
	if _, err := os.Stat(filepath.Join(store.dataDir, "packs", packID)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("staged version is already visible under packs/: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.dataDir, "bundles", packID)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("staged bundle is already visible under bundles/: %v", err)
	}

	committed, err := store.CommitFiles(ctx, staged)
	if err != nil {
		t.Fatalf("CommitFiles: %v", err)
	}
	wantPrefix := "packs/" + packID + "/v1"
	if committed.OutputDir != wantPrefix {
		t.Errorf("output dir = %q", committed.OutputDir)
	}
	if committed.BundlePath != "bundles/"+packID+"/v1.zip" {
		t.Errorf("bundle path = %q", committed.BundlePath)
	}
	if committed.ThumbnailPath != wantPrefix+"/thumb.webp" {
		t.Errorf("thumbnail path = %q", committed.ThumbnailPath)
	}
	if committed.ChecksumSHA256 != wantChecksum || committed.StickerCount != 2 {
		t.Errorf("committed metadata = %+v", committed)
	}
	if committed.BundleByteSize != staged.BundleByteSize || staged.BundleByteSize <= 0 {
		t.Errorf("bundle size = %d", committed.BundleByteSize)
	}
	if len(committed.Stickers) != 2 {
		t.Fatalf("sticker rows = %d", len(committed.Stickers))
	}
	if committed.Stickers[0].ObjectPath != wantPrefix+"/000.webp" ||
		committed.Stickers[0].ThumbPath != wantPrefix+"/thumbs/thumb_000.webp" {
		t.Errorf("sticker 0 paths = %+v", committed.Stickers[0])
	}
	if committed.Stickers[0].ChecksumSHA256 != hex.EncodeToString(sha256Sum(object0)) {
		t.Errorf("sticker 0 checksum mismatch")
	}
	if committed.Stickers[1].ChecksumSHA256 != hex.EncodeToString(sha256Sum(object1)) {
		t.Errorf("sticker 1 checksum mismatch")
	}

	bundleAbs := filepath.Join(store.dataDir, filepath.FromSlash(committed.BundlePath))
	zipReader, err := zip.OpenReader(bundleAbs)
	if err != nil {
		t.Fatalf("open bundle: %v", err)
	}
	defer zipReader.Close()
	var names []string
	for _, member := range zipReader.File {
		names = append(names, member.Name)
	}
	sort.Strings(names)
	wantNames := []string{"000.webp", "001.webp", "pack.json", "thumb.webp", "thumbs/thumb_000.webp", "thumbs/thumb_001.webp"}
	if len(names) != len(wantNames) {
		t.Fatalf("zip members = %v", names)
	}
	for i := range wantNames {
		if names[i] != wantNames[i] {
			t.Fatalf("zip members = %v, want %v", names, wantNames)
		}
	}

	packJSON := readZipMember(t, zipReader, "pack.json")
	var document struct {
		Schema         string `json:"schema"`
		ID             string `json:"id"`
		Version        int    `json:"version"`
		ChecksumSHA256 string `json:"checksum_sha256"`
		IPPolicy       string `json:"ip_policy"`
		Stickers       []struct {
			File  string `json:"file"`
			Thumb string `json:"thumb"`
		} `json:"stickers"`
	}
	if err := json.Unmarshal(packJSON, &document); err != nil {
		t.Fatalf("decode pack.json: %v", err)
	}
	if document.Schema != PackSchema || document.ID != packID || document.Version != 1 ||
		document.ChecksumSHA256 != wantChecksum || document.IPPolicy != "original_only" {
		t.Errorf("pack.json = %+v", document)
	}
	if len(document.Stickers) != 2 || document.Stickers[0].File != "000.webp" ||
		document.Stickers[0].Thumb != "thumbs/thumb_000.webp" {
		t.Errorf("pack.json stickers = %+v", document.Stickers)
	}
	if !bytes.Equal(packJSON, committed.ManifestJSON) {
		t.Errorf("committed manifest bytes differ from pack.json")
	}

	if !bytes.Equal(readZipMember(t, zipReader, "000.webp"), object0) ||
		!bytes.Equal(readZipMember(t, zipReader, "thumbs/thumb_001.webp"), object1) {
		t.Errorf("zip sticker bytes differ")
	}
	if !bytes.Equal(readZipMember(t, zipReader, "thumb.webp"), object0) {
		t.Errorf("pack thumbnail is not the first sticker thumbnail")
	}
	zipSum := sha256.Sum256(mustReadFile(t, bundleAbs))
	if hex.EncodeToString(zipSum[:]) == wantChecksum {
		t.Errorf("checksum must hash sticker bytes, not the ZIP")
	}

	for _, rel := range []string{
		committed.ThumbnailPath,
		committed.Stickers[0].ObjectPath,
		committed.Stickers[1].ThumbPath,
		committed.BundlePath,
	} {
		f, err := store.OpenCommitted(rel)
		if err != nil {
			t.Errorf("OpenCommitted(%q): %v", rel, err)
			continue
		}
		f.Close()
	}
	entries, err := os.ReadDir(filepath.Join(store.dataDir, "work", jobID))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "output" {
		t.Errorf("stage was left behind: %v", entries)
	}
}

func TestStageBundleRejectsUnsafeMembers(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	packID := uuid.NewString()
	jobID := "job_" + uuid.NewString()
	object := mustWebP(t, tinyWebP2x2)
	writeOutput(t, store, jobID, map[string][]byte{
		"000.webp":       object,
		"thumb_000.webp": object,
	})
	valid := testManifest(packID, 1)

	cases := []struct {
		name     string
		manifest PackManifest
		expected error
	}{
		{
			name: "traversal member",
			manifest: func() PackManifest {
				m := valid
				m.Stickers = []ManifestSticker{{File: "../000.webp", ThumbFile: "thumb_000.webp", Width: 2, Height: 2}}
				return m
			}(),
			expected: ErrInvalidManifest,
		},
		{
			name: "renamed member",
			manifest: func() PackManifest {
				m := valid
				m.Stickers = []ManifestSticker{{File: "001.webp", ThumbFile: "thumb_000.webp", Width: 2, Height: 2}}
				return m
			}(),
			expected: ErrInvalidManifest,
		},
		{
			name: "missing file",
			manifest: func() PackManifest {
				m := valid
				m.Stickers = []ManifestSticker{{File: "002.webp", ThumbFile: "thumb_002.webp", Width: 2, Height: 2}}
				return m
			}(),
			expected: ErrInvalidManifest,
		},
		{
			name: "dimension mismatch",
			manifest: func() PackManifest {
				m := valid
				m.Stickers[0].Width = 64
				return m
			}(),
			expected: ErrInvalidManifest,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := store.StageBundle(ctx, jobID, tc.manifest); !errors.Is(err, tc.expected) {
				t.Fatalf("error = %v, want %v", err, tc.expected)
			}
		})
	}

	nonWebP := filepath.Join(store.dataDir, "work", jobID, "output", "003.webp")
	if err := os.WriteFile(nonWebP, encodePNG(t, 2, 2), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := valid
	manifest.Stickers = []ManifestSticker{{File: "003.webp", ThumbFile: "thumb_000.webp", Width: 2, Height: 2}}
	if _, err := store.StageBundle(ctx, jobID, manifest); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("non-webp member error = %v", err)
	}

	secret := filepath.Join(t.TempDir(), "secret.webp")
	if err := os.WriteFile(secret, object, 0o644); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(store.dataDir, "work", jobID, "output", "004.webp")
	if err := os.Symlink(secret, linked); err != nil {
		t.Fatal(err)
	}
	manifest.Stickers = []ManifestSticker{{File: "004.webp", ThumbFile: "thumb_000.webp", Width: 2, Height: 2}}
	if _, err := store.StageBundle(ctx, jobID, manifest); !errors.Is(err, ErrInvalidManifest) && !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("symlinked member error = %v", err)
	}
}

func TestOversizedBundleLeavesNoCommittedArtifact(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	packID := uuid.NewString()
	jobID := "job_" + uuid.NewString()
	workDir, err := store.WorkDir(jobID)
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(workDir, "output")
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatal(err)
	}
	object := mustWebP(t, tinyWebP2x2)
	manifest := testManifest(packID, 5)
	for i := range 5 {
		path := filepath.Join(output, fixedObjectName(i))
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(object); err != nil {
			t.Fatal(err)
		}
		if err := f.Truncate(7 << 20); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(output, fixedThumbName(i)), object, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := store.StageBundle(ctx, jobID, manifest); !errors.Is(err, ErrBundleTooLarge) {
		t.Fatalf("error = %v, want ErrBundleTooLarge", err)
	}
	if _, err := os.Stat(filepath.Join(store.dataDir, "packs", packID)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("pack artifacts were published: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.dataDir, "bundles", packID)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("bundle artifacts were published: %v", err)
	}
	entries, err := os.ReadDir(workDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "output" {
		t.Errorf("stage directory was not cleaned: %v", entries)
	}
}

func sha256Sum(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

func readZipMember(t *testing.T, reader *zip.ReadCloser, name string) []byte {
	t.Helper()
	for _, member := range reader.File {
		if member.Name != name {
			continue
		}
		file, err := member.Open()
		if err != nil {
			t.Fatalf("open zip member %s: %v", name, err)
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read zip member %s: %v", name, err)
		}
		return data
	}
	t.Fatalf("zip member %s missing", name)
	return nil
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
