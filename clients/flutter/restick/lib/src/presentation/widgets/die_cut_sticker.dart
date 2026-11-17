import "package:material_ui/material_ui.dart";
import "package:restick/src/presentation/theme/restick_theme.dart";
import "package:restick/src/presentation/widgets/restick_image_request.dart";

/// Sticker image with a white die-cut border (four white offset copies).
class DieCutSticker extends StatelessWidget {
  const DieCutSticker({
    required this.url,
    required this.size,
    this.rotation = 0,
    super.key,
  });

  final String url;
  final double size;

  /// Rotation in radians.
  final double rotation;

  static const double _border = 4;

  @override
  Widget build(BuildContext context) {
    if (url.isEmpty) {
      return _placeholder(context);
    }
    final Map<String, String>? headers = restickImageHeaders(context);
    final int decode = restickDecodeSize(context, size);

    Widget image({Color? tint, Offset offset = Offset.zero}) {
      Widget child = Image.network(
        url,
        headers: headers,
        width: size,
        height: size,
        cacheWidth: decode,
        cacheHeight: decode,
        fit: BoxFit.contain,
        errorBuilder:
            (_, _, _) =>
                tint == null ? _placeholder(context) : const SizedBox.shrink(),
      );
      if (tint != null) {
        child = ColorFiltered(
          colorFilter: ColorFilter.mode(tint, BlendMode.srcIn),
          child: child,
        );
      }
      return Transform.translate(offset: offset, child: child);
    }

    return RepaintBoundary(
      child: Transform.rotate(
        angle: rotation,
        child: SizedBox.square(
          dimension: size,
          child: Stack(
            clipBehavior: Clip.none,
            children: <Widget>[
              for (final Offset o in const <Offset>[
                Offset(_border, 0),
                Offset(-_border, 0),
                Offset(0, _border),
                Offset(0, -_border),
              ])
                image(tint: const Color(0xFFFFFFFF), offset: o),
              image(),
            ],
          ),
        ),
      ),
    );
  }

  Widget _placeholder(BuildContext context) {
    return SizedBox.square(
      dimension: size,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: RestickTheme.of(context).brandSoft,
          shape: BoxShape.circle,
        ),
        child: Icon(
          Icons.emoji_emotions_outlined,
          color: RestickTheme.of(context).ink,
        ),
      ),
    );
  }
}
