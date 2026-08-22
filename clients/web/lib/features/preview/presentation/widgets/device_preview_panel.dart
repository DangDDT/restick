import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick_web/features/preview/presentation/cubit/preview_cubit.dart";
import "package:restick_web/l10n/app_localizations.dart";

class DevicePreviewPanel extends StatelessWidget {
  const DevicePreviewPanel({super.key});

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final ColorScheme colors = Theme.of(context).colorScheme;
    return Container(
      width: 340,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        border: Border(left: BorderSide(color: colors.outline)),
        color: colors.surface.withValues(alpha: 0.5),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        spacing: 8,
        children: [
          Row(
            spacing: 8,
            children: [
              Icon(Icons.phone_iphone, color: colors.primary),
              Expanded(
                child: Text(
                  l10n.deviceTitle,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
              ),
            ],
          ),
          Text(
            l10n.deviceSub,
            style: Theme.of(context).textTheme.bodySmall,
          ),
          const SizedBox(height: 8),
          Expanded(
            child: Center(
              child: AspectRatio(
                aspectRatio: 9 / 18,
                child: DecoratedBox(
                  decoration: BoxDecoration(
                    color: colors.surfaceContainerLowest,
                    borderRadius: BorderRadius.circular(28),
                    border: Border.all(color: colors.outline, width: 2),
                  ),
                  child: Column(
                    children: [
                      Container(
                        height: 28,
                        alignment: Alignment.center,
                        child: Container(
                          width: 80,
                          height: 8,
                          decoration: BoxDecoration(
                            color: colors.outline,
                            borderRadius: BorderRadius.circular(8),
                          ),
                        ),
                      ),
                      Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 12,
                          vertical: 8,
                        ),
                        decoration: BoxDecoration(
                          border: Border(
                            bottom: BorderSide(color: colors.outline),
                          ),
                        ),
                        child: Row(
                          spacing: 8,
                          children: [
                            CircleAvatar(
                              radius: 14,
                              backgroundColor: colors.primary,
                              foregroundColor: colors.onPrimary,
                              child: const Text("RS"),
                            ),
                            Expanded(
                              child: Text(
                                "Restick Chat Demo",
                                style: Theme.of(context).textTheme.labelLarge,
                              ),
                            ),
                          ],
                        ),
                      ),
                      Expanded(
                        child: BlocBuilder<PreviewCubit, PreviewState>(
                          builder:
                              (BuildContext context, PreviewState state) {
                            return ListView.builder(
                              padding: const EdgeInsets.all(12),
                              itemCount: state.messages.length,
                              itemBuilder: (BuildContext context, int index) {
                                final PreviewMessage message =
                                    state.messages[index];
                                return _ChatBubble(
                                  message: message,
                                  l10n: l10n,
                                );
                              },
                            );
                          },
                        ),
                      ),
                      Padding(
                        padding: const EdgeInsets.all(10),
                        child: Row(
                          spacing: 8,
                          children: [
                            const Expanded(
                              child: TextField(
                                readOnly: true,
                                decoration: InputDecoration(
                                  hintText: "Type a message...",
                                  isDense: true,
                                  contentPadding: EdgeInsets.symmetric(
                                    horizontal: 12,
                                    vertical: 10,
                                  ),
                                ),
                              ),
                            ),
                            IconButton(
                              onPressed: () =>
                                  context.read<PreviewCubit>().clearStickers(),
                              icon: const Icon(Icons.refresh),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _ChatBubble extends StatelessWidget {
  const _ChatBubble({required this.message, required this.l10n});

  final PreviewMessage message;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    final String? text = message.text == "intro"
        ? l10n.msgIntro
        : message.text == "prompt"
            ? l10n.msgPrompt
            : message.text;
    final Alignment align =
        message.isSent ? Alignment.centerRight : Alignment.centerLeft;
    final Color bg = message.isSent
        ? colors.primary.withValues(alpha: 0.25)
        : colors.surfaceContainerHighest;
    return Align(
      alignment: align,
      child: Container(
        margin: const EdgeInsets.only(bottom: 8),
        padding: const EdgeInsets.all(10),
        constraints: const BoxConstraints(maxWidth: 220),
        decoration: BoxDecoration(
          color: bg,
          borderRadius: BorderRadius.circular(14),
        ),
        child: message.stickerUrl != null
            ? Image.network(
                message.stickerUrl!,
                width: 96,
                height: 96,
                errorBuilder: (
                  BuildContext context,
                  Object error,
                  StackTrace? stackTrace,
                ) {
                  return const Icon(Icons.broken_image_outlined);
                },
              )
            : Text(text ?? "", style: Theme.of(context).textTheme.bodySmall),
      ),
    );
  }
}
