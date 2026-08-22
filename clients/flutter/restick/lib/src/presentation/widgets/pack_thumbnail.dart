import "package:flutter/material.dart";

class PackThumbnail extends StatelessWidget {
  const PackThumbnail({required this.url, this.size = 48, super.key});

  final String url;
  final double size;

  @override
  Widget build(BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    if (url.isEmpty) {
      return _PlaceholderThumb(size: size, color: colors.surfaceContainerHighest);
    }
    return ClipRRect(
      borderRadius: BorderRadius.circular(8),
      child: Image.network(
        url,
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
