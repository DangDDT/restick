import "package:flutter_test/flutter_test.dart";
import "package:restick/src/presentation/theme/restick_strings.dart";

void main() {
  test("en pluralizes sticker count", () {
    expect(RestickStrings.en.stickerCount(0), "0 stickers");
    expect(RestickStrings.en.stickerCount(1), "1 sticker");
    expect(RestickStrings.en.stickerCount(2), "2 stickers");
    expect(RestickStrings.en.stickerCount(24), "24 stickers");
  });

  test("vi never inflects", () {
    expect(RestickStrings.vi.stickerCount(1), "1 sticker");
    expect(RestickStrings.vi.stickerCount(24), "24 sticker");
  });

  test("packMeta and download label use the same plural rule", () {
    expect(RestickStrings.en.packMeta(1, "Mia"), "1 sticker · Mia");
    expect(RestickStrings.en.packMeta(24, "Mia"), "24 stickers · Mia");
    expect(RestickStrings.en.downloadAllLabel(24), "Download 24 stickers");
    expect(RestickStrings.en.downloadAllLabel(1), "Download 1 sticker");
    expect(RestickStrings.vi.packMeta(24, "Mia"), "24 sticker · Mia");
    expect(RestickStrings.vi.downloadAllLabel(24), "Tải về 24 sticker");
    expect(RestickStrings.vi.a11yDownloading("Mèo", 40), contains("40"));
  });
}
