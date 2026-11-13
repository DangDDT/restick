package assets

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func clipBytes(header []byte) []byte {
	return append(append([]byte{}, header...), bytes.Repeat([]byte{0x42}, 64)...)
}

var (
	mp4Header  = []byte{0, 0, 0, 0x20, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}
	movHeader  = []byte{0, 0, 0, 0x14, 'f', 't', 'y', 'p', 'q', 't', ' ', ' '}
	webmHeader = []byte{0x1A, 0x45, 0xDF, 0xA3, 0x9F, 0x42, 0x86, 0x81, 0x01, 0x42, 0xF7, 0x81}
)

func TestSaveClipSniffsContainer(t *testing.T) {
	store := newTestStore(t)
	for _, tc := range []struct {
		name      string
		header    []byte
		filename  string
		ext       string
		mediaType string
	}{
		{"mp4", mp4Header, "clip.mp4", "mp4", "video/mp4"},
		{"mov", movHeader, "clip.mov", "mov", "video/quicktime"},
		{"webm", webmHeader, "clip.webm", "webm", "video/webm"},
		{"name ignored", mp4Header, "../../x.webm", "mp4", "video/mp4"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := clipBytes(tc.header)
			clip, err := store.SaveClip(context.Background(), bytes.NewReader(data), tc.filename)
			if err != nil {
				t.Fatalf("SaveClip: %v", err)
			}
			if want := "clips/" + clip.ID + "." + tc.ext; clip.RelativePath != want {
				t.Fatalf("RelativePath = %q, want %q", clip.RelativePath, want)
			}
			if clip.MediaType != tc.mediaType || clip.ByteSize != int64(len(data)) || len(clip.ChecksumSHA256) != 64 {
				t.Fatalf("clip = %+v", clip)
			}
			p, err := store.ClipPath(clip.ID)
			if err != nil {
				t.Fatalf("ClipPath: %v", err)
			}
			if got, _ := os.ReadFile(p); !bytes.Equal(got, data) {
				t.Fatal("stored bytes differ")
			}
			if err := store.RemoveClip(clip.ID); err != nil {
				t.Fatalf("RemoveClip: %v", err)
			}
			if _, err := store.ClipPath(clip.ID); !errors.Is(err, ErrNotFound) {
				t.Fatalf("ClipPath after remove = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestSaveClipRejectsNonVideoAndOversize(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if _, err := store.SaveClip(ctx, bytes.NewReader(encodePNG(t, 4, 4)), "a.mp4"); !errors.Is(err, ErrInvalidClip) {
		t.Fatalf("png = %v, want ErrInvalidClip", err)
	}
	if _, err := store.SaveClip(ctx, bytes.NewReader([]byte{0, 0, 0, 8, 'f', 't'}), "a.mp4"); !errors.Is(err, ErrInvalidClip) {
		t.Fatalf("short = %v, want ErrInvalidClip", err)
	}
	big := make([]byte, MaxClipBytes+1)
	copy(big, mp4Header)
	if _, err := store.SaveClip(ctx, bytes.NewReader(big), "a.mp4"); !errors.Is(err, ErrClipTooLarge) {
		t.Fatalf("oversize = %v, want ErrClipTooLarge", err)
	}
	entries, _ := os.ReadDir(filepath.Join(store.dataDir, "clips"))
	if len(entries) != 0 {
		t.Fatalf("rejected clips left files: %v", entries)
	}
}

func TestClipPathRejectsBadIDAndSymlink(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.ClipPath("../etc"); !errors.Is(err, ErrInvalidClipID) {
		t.Fatalf("bad id = %v, want ErrInvalidClipID", err)
	}
	id := "11111111-2222-4333-8444-555555555555"
	outside := filepath.Join(t.TempDir(), "secret.mp4")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(store.dataDir, "clips", id+".mp4")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ClipPath(id); err == nil {
		t.Fatal("symlinked clip was accepted")
	}
}

func TestNewStoreCreatesClipAndAnimationDirs(t *testing.T) {
	store := newTestStore(t)
	for _, sub := range []string{"clips", "animations"} {
		if info, err := os.Stat(filepath.Join(store.dataDir, sub)); err != nil || !info.IsDir() {
			t.Fatalf("%s missing: %v", sub, err)
		}
	}
}
