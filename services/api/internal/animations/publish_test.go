package animations

import (
	"context"
	"testing"
)

func TestPublishableReturnsReadyEnabledNonStale(t *testing.T) {
	h := newHarness(t)
	packID := h.newPack(sum('a'), sum('b'), sum('c'), sum('d'), sum('e'))
	for i := range 5 {
		h.upload(packID, i)
	}
	ready := func(index int, use bool) {
		h.exec(`UPDATE sticker_animations SET status = 'ready', use_animated = ?, animated_path = ?,
			animated_checksum_sha256 = ?, frames = 4, duration_ms = 333, width = 512, height = 512
			WHERE pack_id = ? AND sticker_index = ?`,
			use, "animations/x.webp", sum(byte('0'+index)), packID, index)
	}
	ready(0, true)
	ready(1, false) // toggled off
	// index 2 stays queued
	ready(3, true)
	h.exec(`UPDATE sticker_animations SET static_checksum_sha256 = ? WHERE pack_id = ? AND sticker_index = 3`, sum('z'), packID) // stale
	h.exec(`UPDATE sticker_animations SET status = 'failed' WHERE pack_id = ? AND sticker_index = 4`, packID)

	got, err := h.service.Publishable(context.Background(), packID)
	if err != nil {
		t.Fatalf("Publishable: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("publishable = %+v, want only sticker 0", got)
	}
	a := got[0]
	if a.StickerIndex != 0 || a.Checksum != sum('0') || a.Path != "animations/x.webp" ||
		a.Frames != 4 || a.DurationMS != 333 || a.Width != 512 || a.Height != 512 {
		t.Fatalf("publishable[0] = %+v", a)
	}
}
