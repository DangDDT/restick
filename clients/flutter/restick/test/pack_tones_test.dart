import "package:flutter_test/flutter_test.dart";
import "package:material_ui/material_ui.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/presentation/theme/pack_tones.dart";
import "package:restick/src/presentation/theme/restick_brand.dart";

StickerPack _pack(String id, [String themeColor = ""]) => StickerPack(
  id: id,
  slug: id,
  name: id,
  author: "a",
  tags: const <String>[],
  stickerCount: 1,
  version: 1,
  thumbnailUrl: "",
  checksumSha256: "",
  themeColor: themeColor,
);

void main() {
  test("uses server theme color when valid", () {
    expect(
      PackTones.of(_pack("x", "#FFDDEE")).background,
      const Color(0xFFFFDDEE),
    );
  });

  test("falls back to stable pastel by id", () {
    expect(stableHash("a"), 0xe40c292c); // FNV-1a reference vector
    final Color first = PackTones.of(_pack("pack-1")).background;
    expect(PackTones.of(_pack("pack-1")).background, first);
    for (final String id in <String>["a", "b", "pack-2", "mèo", ""]) {
      expect(
        RestickBrand.packPastels,
        contains(PackTones.of(_pack(id)).background),
      );
    }
  });

  test("rejects malformed theme color and falls back", () {
    for (final String bad in <String>[
      "#FFDDE",
      "FFDDEE",
      "#GGDDEE",
      "#FFDDEEFF",
    ]) {
      expect(
        PackTones.of(_pack("p", bad)).background,
        PackTones.of(_pack("p")).background,
      );
    }
  });

  test("lowercase hex parses", () {
    expect(
      PackTones.of(_pack("x", "#ffddee")).background,
      const Color(0xFFFFDDEE),
    );
  });
}
