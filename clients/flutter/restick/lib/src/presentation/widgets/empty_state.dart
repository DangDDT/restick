import "package:material_ui/material_ui.dart";

class RestickEmptyState extends StatelessWidget {
  const RestickEmptyState({
    required this.message,
    this.icon = Icons.emoji_emotions_outlined,
    super.key,
  });

  final String message;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        spacing: 12,
        children: <Widget>[
          Icon(icon, color: colors.onSurfaceVariant),
          Text(
            message,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyMedium,
          ),
        ],
      ),
    );
  }
}
