import "package:equatable/equatable.dart";
import "package:flutter/material.dart" as sdk;
import "package:google_fonts/google_fonts.dart";
import "package:material_ui/material_ui.dart";

@immutable
class PackStatusColors extends ThemeExtension<PackStatusColors>
    with Equatable {
  const PackStatusColors({
    required this.draft,
    required this.processing,
    required this.confirm,
    required this.pending,
    required this.published,
    required this.rejected,
    required this.revoked,
  });

  final Color draft;
  final Color processing;
  final Color confirm;
  final Color pending;
  final Color published;
  final Color rejected;
  final Color revoked;

  Color forStatus(String status) {
    switch (status) {
      case "draft":
        return draft;
      case "processing":
        return processing;
      case "awaiting_confirm":
        return confirm;
      case "pending_review":
        return pending;
      case "published":
        return published;
      case "rejected":
        return rejected;
      case "revoked":
        return revoked;
      default:
        return draft;
    }
  }

  @override
  PackStatusColors copyWith({
    Color? draft,
    Color? processing,
    Color? confirm,
    Color? pending,
    Color? published,
    Color? rejected,
    Color? revoked,
  }) {
    return PackStatusColors(
      draft: draft ?? this.draft,
      processing: processing ?? this.processing,
      confirm: confirm ?? this.confirm,
      pending: pending ?? this.pending,
      published: published ?? this.published,
      rejected: rejected ?? this.rejected,
      revoked: revoked ?? this.revoked,
    );
  }

  @override
  PackStatusColors lerp(ThemeExtension<PackStatusColors>? other, double t) {
    if (other is! PackStatusColors) {
      return this;
    }
    return PackStatusColors(
      draft: Color.lerp(draft, other.draft, t)!,
      processing: Color.lerp(processing, other.processing, t)!,
      confirm: Color.lerp(confirm, other.confirm, t)!,
      pending: Color.lerp(pending, other.pending, t)!,
      published: Color.lerp(published, other.published, t)!,
      rejected: Color.lerp(rejected, other.rejected, t)!,
      revoked: Color.lerp(revoked, other.revoked, t)!,
    );
  }

  @override
  List<Object?> get props => <Object?>[
        draft,
        processing,
        confirm,
        pending,
        published,
        rejected,
        revoked,
      ];
}

class AppTheme {
  static const PackStatusColors statusColors = PackStatusColors(
    draft: Color(0xFF94A3B8),
    processing: Color(0xFFD97706),
    confirm: Color(0xFF2563EB),
    pending: Color(0xFF7C3AED),
    published: Color(0xFF059669),
    rejected: Color(0xFFDC2626),
    revoked: Color(0xFF475569),
  );

  static ThemeData light() {
    final ColorScheme scheme = ColorScheme.fromSeed(
      seedColor: const Color(0xFF0D9488),
      primary: const Color(0xFF0D9488),
      surface: const Color(0xFFFFFFFF),
    ).copyWith(
      surfaceContainerLowest: const Color(0xFFF8FAFC),
      onSurface: const Color(0xFF0F172A),
      onSurfaceVariant: const Color(0xFF475569),
      outline: const Color(0x14000000),
      tertiary: const Color(0xFF4F46E5),
    );
    return _build(scheme, Brightness.light);
  }

  static ThemeData dark() {
    final ColorScheme scheme = ColorScheme.fromSeed(
      seedColor: const Color(0xFF14B8A6),
      brightness: Brightness.dark,
      primary: const Color(0xFF14B8A6),
      surface: const Color(0xFF111827),
    ).copyWith(
      surfaceContainerLowest: const Color(0xFF0B0F17),
      onSurface: const Color(0xFFF9FAFB),
      onSurfaceVariant: const Color(0xFF9CA3AF),
      outline: const Color(0x14FFFFFF),
      tertiary: const Color(0xFF6366F1),
    );
    return _build(scheme, Brightness.dark);
  }

  static ThemeData _build(ColorScheme scheme, Brightness brightness) {
    final sdk.TextTheme sdkBase = brightness == Brightness.dark
        ? sdk.ThemeData.dark().textTheme
        : sdk.ThemeData.light().textTheme;
    final TextTheme jakarta = _adaptTextTheme(
      GoogleFonts.plusJakartaSansTextTheme(sdkBase),
    );
    final TextTheme outfit = _adaptTextTheme(
      GoogleFonts.outfitTextTheme(sdkBase),
    );
    return ThemeData(
      useMaterial3: true,
      brightness: brightness,
      colorScheme: scheme,
      scaffoldBackgroundColor: scheme.surfaceContainerLowest,
      textTheme: jakarta.copyWith(
        displayLarge: outfit.displayLarge,
        displayMedium: outfit.displayMedium,
        displaySmall: outfit.displaySmall,
        headlineLarge: outfit.headlineLarge,
        headlineMedium: outfit.headlineMedium,
        headlineSmall: outfit.headlineSmall,
        titleLarge: outfit.titleLarge,
      ),
      cardTheme: CardThemeData(
        color: scheme.surface,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(14),
          side: BorderSide(color: scheme.outline),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: scheme.surface,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(color: scheme.outline),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(color: scheme.outline),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(color: scheme.primary),
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: scheme.primary,
          foregroundColor: scheme.onPrimary,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
          ),
        ),
      ),
      extensions: const <ThemeExtension<dynamic>>[statusColors],
    );
  }

  static TextTheme _adaptTextTheme(sdk.TextTheme theme) {
    return TextTheme(
      displayLarge: theme.displayLarge,
      displayMedium: theme.displayMedium,
      displaySmall: theme.displaySmall,
      headlineLarge: theme.headlineLarge,
      headlineMedium: theme.headlineMedium,
      headlineSmall: theme.headlineSmall,
      titleLarge: theme.titleLarge,
      titleMedium: theme.titleMedium,
      titleSmall: theme.titleSmall,
      bodyLarge: theme.bodyLarge,
      bodyMedium: theme.bodyMedium,
      bodySmall: theme.bodySmall,
      labelLarge: theme.labelLarge,
      labelMedium: theme.labelMedium,
      labelSmall: theme.labelSmall,
    );
  }
}
