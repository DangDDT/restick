import "package:material_ui/material_ui.dart";

class SwaggerIframe extends StatelessWidget {
  const SwaggerIframe({required this.src, super.key});

  final String src;

  @override
  Widget build(BuildContext context) {
    return ColoredBox(
      color: Theme.of(context).colorScheme.surfaceContainerHighest,
      child: Center(child: Text(src)),
    );
  }
}
