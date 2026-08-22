import "package:flutter/material.dart";
import "package:restick/src/presentation/scope/restick_scope.dart";

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
      return _PlaceholderThumb(size: size, color: colors.surfaceContainerHighest);
    }
    final RestickScope? scope =
        context.findAncestorWidgetOfExactType<RestickScope>();
    final Map<String, String>? effectiveHeaders = headers ??
        (scope != null
            ? <String, String>{"X-Api-Key": scope.client.config.apiKey}
            : null);

    return ClipRRect(
      borderRadius: BorderRadius.circular(8),
      child: Image.network(
        url,
        headers: effectiveHeaders,
        width: size,
        height: size,
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
