import "package:material_ui/material_ui.dart";
import "package:restick/src/presentation/widgets/restick_image_request.dart";

class PackThumbnail extends StatelessWidget {
  const PackThumbnail({
    required this.url,
    this.size = 48,
    this.headers,
    super.key,
  });

  final String url;
  final double size;
  final Map<String, String>? headers;

  @override
  Widget build(BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    if (url.isEmpty) {
      return _PlaceholderThumb(
        size: size,
        color: colors.surfaceContainerHighest,
      );
    }
    final Map<String, String>? effectiveHeaders =
        restickImageHeaders(context, headers);
    final int decodeSize = restickDecodeSize(context, size);

    return ClipRRect(
      borderRadius: BorderRadius.circular(8),
      child: Image.network(
        url,
        headers: effectiveHeaders,
        width: size,
        height: size,
        cacheWidth: decodeSize,
        cacheHeight: decodeSize,
        fit: BoxFit.cover,
        errorBuilder: (
          BuildContext context,
          Object error,
          StackTrace? stackTrace,
        ) {
          return _PlaceholderThumb(
            size: size,
            color: colors.surfaceContainerHighest,
          );
        },
      ),
    );
  }
}

class _PlaceholderThumb extends StatelessWidget {
  const _PlaceholderThumb({required this.size, required this.color});

  final double size;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return SizedBox.square(
      dimension: size,
      child: ColoredBox(
        color: color,
        child: Icon(
          Icons.emoji_emotions_outlined,
          color: Theme.of(context).colorScheme.onSurfaceVariant,
        ),
      ),
    );
  }
}
