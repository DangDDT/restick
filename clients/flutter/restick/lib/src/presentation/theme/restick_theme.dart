import "package:equatable/equatable.dart";
import "package:flutter/material.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";

@immutable
class RestickTheme extends ThemeExtension<RestickTheme> with Equatable {
  const RestickTheme({
    required this.downloaded,
    required this.downloading,
    required this.failed,
    required this.revoked,
    required this.remote,
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

  final Color downloaded;
  final Color downloading;
  final Color failed;
  final Color revoked;
  final Color remote;

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
  }) {
    return RestickTheme(
      downloaded: downloaded ?? this.downloaded,
      downloading: downloading ?? this.downloading,
      failed: failed ?? this.failed,
      revoked: revoked ?? this.revoked,
      remote: remote ?? this.remote,
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
    );
  }

  @override
  List<Object?> get props => <Object?>[
        downloaded,
        downloading,
        failed,
        revoked,
        remote,
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
