import "package:material_ui/material_ui.dart";

/// Fixed brand constants of the Restick marketplace (see the redesign spec).
abstract final class RestickBrand {
  static const String fontFamily = "Baloo2";

  /// Package that bundles [fontFamily]; required for package-owned fonts.
  static const String fontPackage = "restick";

  static const Color purple = Color(0xFFA98BF0);
  static const Color purpleSoft = Color(0xFFD2C0FA);
  static const Color cheekPink = Color(0xFFF58BC0);
  static const Color deepPink = Color(0xFFE6197F);
  static const Color ink = Color(0xFF2B2433);
  static const Color canvas = Color(0xFFFFF3FA);

  static const List<Color> packPastels = <Color>[
    Color(0xFFEBE3FF),
    Color(0xFFFFDDEE),
    Color(0xFFFFE6CF),
    Color(0xFFD9F4E7),
    Color(0xFFFFF2C2),
  ];

  static TextStyle text({
    double size = 14,
    FontWeight weight = FontWeight.w600,
    Color? color,
    double? height,
  }) {
    return TextStyle(
      fontFamily: fontFamily,
      package: fontPackage,
      fontSize: size,
      fontWeight: weight,
      color: color ?? ink,
      height: height,
    );
  }
}

/// Shared motion language: expressive ease-out, never blocking input.
abstract final class RestickMotion {
  static const Curve ease = Cubic(0.16, 1, 0.3, 1);
  static const Duration fast = Duration(milliseconds: 150);
  static const Duration routine = Duration(milliseconds: 300);
  static const Duration enter = Duration(milliseconds: 450);
}
