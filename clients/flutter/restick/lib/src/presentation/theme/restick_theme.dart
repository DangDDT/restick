import "package:equatable/equatable.dart";
import "package:material_ui/material_ui.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/presentation/theme/restick_brand.dart";

@immutable
class RestickTheme extends ThemeExtension<RestickTheme> with Equatable {
  const RestickTheme({
    required this.downloaded,
    required this.downloading,
    required this.failed,
    required this.revoked,
    required this.remote,
    this.brand = RestickBrand.purple,
    this.brandSoft = RestickBrand.purpleSoft,
    this.ink = RestickBrand.ink,
    this.canvas = RestickBrand.canvas,
  });

  factory RestickTheme.fromColorScheme(ColorScheme scheme) {
    return RestickTheme(
      downloaded: scheme.primary,
      downloading: scheme.tertiary,
      failed: scheme.error,
      revoked: scheme.onSurfaceVariant,
      remote: scheme.outline,
    );
  }

  static RestickTheme of(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    return theme.extension<RestickTheme>() ?? RestickTheme.fromColorScheme(theme.colorScheme);
  }

  final Color downloaded;
  final Color downloading;
  final Color failed;
  final Color revoked;
  final Color remote;

  /// Marketplace brand tokens; override via [copyWith] to re-skin.
  final Color brand;
  final Color brandSoft;
  final Color ink;
  final Color canvas;

  Color get mutedInk => ink == RestickBrand.ink
      ? const Color(0xFF6B5F78) : ink.withValues(alpha: 0.7);

  Color colorFor(PackDownloadStatus status) {
    switch (status) {
      case PackDownloadStatus.downloaded:
        return downloaded;
      case PackDownloadStatus.downloading:
        return downloading;
      case PackDownloadStatus.failed:
        return failed;
      case PackDownloadStatus.revoked:
        return revoked;
      case PackDownloadStatus.remote:
        return remote;
    }
  }

  @override
  RestickTheme copyWith({
    Color? downloaded,
    Color? downloading,
    Color? failed,
    Color? revoked,
    Color? remote,
    Color? brand,
    Color? brandSoft,
    Color? ink,
    Color? canvas,
  }) {
    return RestickTheme(
      downloaded: downloaded ?? this.downloaded,
      downloading: downloading ?? this.downloading,
      failed: failed ?? this.failed,
      revoked: revoked ?? this.revoked,
      remote: remote ?? this.remote,
      brand: brand ?? this.brand,
      brandSoft: brandSoft ?? this.brandSoft,
      ink: ink ?? this.ink,
      canvas: canvas ?? this.canvas,
    );
  }

  @override
  RestickTheme lerp(ThemeExtension<RestickTheme>? other, double t) {
    if (other is! RestickTheme) {
      return this;
    }
    return RestickTheme(
      downloaded: Color.lerp(downloaded, other.downloaded, t)!,
      downloading: Color.lerp(downloading, other.downloading, t)!,
      failed: Color.lerp(failed, other.failed, t)!,
      revoked: Color.lerp(revoked, other.revoked, t)!,
      remote: Color.lerp(remote, other.remote, t)!,
      brand: Color.lerp(brand, other.brand, t)!,
      brandSoft: Color.lerp(brandSoft, other.brandSoft, t)!,
      ink: Color.lerp(ink, other.ink, t)!,
      canvas: Color.lerp(canvas, other.canvas, t)!,
    );
  }

  @override
  List<Object?> get props => <Object?>[
        downloaded,
        downloading,
        failed,
        revoked,
        remote,
        brand,
        brandSoft,
        ink,
        canvas,
      ];
}

abstract final class RestickBreakpoints {
  static const double compact = 600;
  static const double medium = 900;

  static int gridColumns(double width) {
    if (width >= medium) {
      return 8;
    }
    if (width >= compact) {
      return 6;
    }
    return 4;
  }
}

ThemeData buildRestickThemeData({
  required Brightness brightness,
  ColorScheme? colorScheme,
}) {
  final ColorScheme scheme = colorScheme ??
      ColorScheme.fromSeed(
        seedColor: const Color(0xFF0D9488),
        brightness: brightness,
      );
  return ThemeData(
    useMaterial3: true,
    brightness: brightness,
    colorScheme: scheme,
    extensions: <ThemeExtension<dynamic>>[
      RestickTheme.fromColorScheme(scheme),
    ],
  );
}
