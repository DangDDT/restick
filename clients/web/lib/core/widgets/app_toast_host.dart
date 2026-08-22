import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick_web/core/error/error_cubit.dart";

class AppToastHost extends StatelessWidget {
  const AppToastHost({super.key});

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<ErrorCubit, List<ToastMessage>>(
      builder: (BuildContext context, List<ToastMessage> toasts) {
        if (toasts.isEmpty) {
          return const SizedBox.shrink();
        }
        return Positioned(
          right: 24,
          bottom: 24,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            spacing: 10,
            children: toasts
                .map(
                  (ToastMessage toast) => _ToastCard(
                    message: toast,
                    onDismiss: () =>
                        context.read<ErrorCubit>().dismiss(toast.id),
                  ),
                )
                .toList(),
          ),
        );
      },
    );
  }
}

class _ToastCard extends StatelessWidget {
  const _ToastCard({required this.message, required this.onDismiss});

  final ToastMessage message;
  final VoidCallback onDismiss;

  Color _accent(ColorScheme colors) {
    switch (message.kind) {
      case ToastKind.success:
        return const Color(0xFF10B981);
      case ToastKind.danger:
        return const Color(0xFFEF4444);
      case ToastKind.warning:
        return const Color(0xFFF59E0B);
      case ToastKind.info:
        return const Color(0xFF3B82F6);
    }
  }

  @override
  Widget build(BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    return Material(
      color: colors.surface,
      elevation: 8,
      borderRadius: BorderRadius.circular(14),
      child: Container(
        constraints: const BoxConstraints(minWidth: 320, maxWidth: 420),
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(14),
          border: Border(
            left: BorderSide(color: _accent(colors), width: 4),
            top: BorderSide(color: colors.outline),
            right: BorderSide(color: colors.outline),
            bottom: BorderSide(color: colors.outline),
          ),
        ),
        padding: const EdgeInsets.all(14),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          spacing: 12,
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                spacing: 4,
                children: [
                  Text(
                    message.title,
                    style: Theme.of(context).textTheme.titleSmall,
                  ),
                  Text(
                    message.body,
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                ],
              ),
            ),
            IconButton(
              onPressed: onDismiss,
              icon: const Icon(Icons.close),
              visualDensity: VisualDensity.compact,
            ),
          ],
        ),
      ),
    );
  }
}
