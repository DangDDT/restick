import "package:material_ui/material_ui.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/presentation/theme/restick_brand.dart";

final RegExp _hex6 = RegExp(r"^#[0-9a-fA-F]{6}$");

/// FNV-1a 32-bit over UTF-16 code units; stable across runs (unlike hashCode).
int stableHash(String value) {
  int hash = 0x811c9dc5;
  for (final int unit in value.codeUnits) {
    hash = ((hash ^ unit) * 0x01000193) & 0xffffffff;
  }
  return hash;
}

/// Card background plus a stronger accent derived from it.
@immutable
class PackTones {
  const PackTones({required this.background, required this.accent});

  factory PackTones.of(StickerPack pack) {
    final Color background =
        _hex6.hasMatch(pack.themeColor)
            ? Color(
              0xFF000000 | int.parse(pack.themeColor.substring(1), radix: 16),
            )
            : RestickBrand.packPastels[stableHash(pack.id) %
                RestickBrand.packPastels.length];
    final HSLColor hsl = HSLColor.fromColor(background);
    final Color accent =
        hsl
            .withLightness((hsl.lightness * 0.5).clamp(0.2, 0.5))
            .withSaturation((hsl.saturation + 0.3).clamp(0.0, 1.0))
            .toColor();
    return PackTones(background: background, accent: accent);
  }

  final Color background;
  final Color accent;
}
