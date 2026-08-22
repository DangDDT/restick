import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:go_router/go_router.dart";
import "package:restick_web/core/error/error_cubit.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/core/widgets/app_shell.dart";
import "package:restick_web/core/widgets/reason_dialogs.dart";
import "package:restick_web/features/moderation/presentation/cubit/review_queue_cubit.dart";
import "package:restick_web/l10n/app_localizations.dart";

class ReviewQueuePage extends StatelessWidget {
  const ReviewQueuePage({super.key});

  Future<void> _reject(BuildContext context, String id) async {
    final ReasonResult? result = await showDialog<ReasonResult>(
      context: context,
      builder: (BuildContext context) => const RejectReasonDialog(),
    );
    if (result == null || !context.mounted) {
      return;
    }
    try {
      await context.read<ReviewQueueCubit>().reject(
            id: id,
            reasonCode: result.code,
            reasonText: result.text,
          );
    } on Failure catch (failure) {
      if (context.mounted) {
        context.read<ErrorCubit>().showFailure(failure.message);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return BlocBuilder<ReviewQueueCubit, ReviewQueueState>(
      builder: (BuildContext context, ReviewQueueState state) {
        return Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            spacing: 16,
            children: [
              Text(
                l10n.titleQueue,
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              Text(
                l10n.descQueue,
                style: Theme.of(context).textTheme.bodyMedium,
              ),
              if (!state.hasLoaded && state.isRefreshing)
                const LinearProgressIndicator(minHeight: 2)
              else if (state.items.isEmpty)
                Text(l10n.emptyQueue)
              else
                Expanded(
                  child: GridView.builder(
                    gridDelegate:
                        const SliverGridDelegateWithMaxCrossAxisExtent(
                      maxCrossAxisExtent: 300,
                      mainAxisSpacing: 16,
                      crossAxisSpacing: 16,
                      childAspectRatio: 0.72,
                    ),
                    itemCount: state.items.length,
                    itemBuilder: (BuildContext context, int index) {
                      final pack = state.items[index];
                      return PackCard(
                        pack: pack,
                        onTap: () => context.go("/admin/queue/${pack.id}"),
                        trailing: Row(
                          spacing: 8,
                          children: [
                            Expanded(
                              child: IconButton.filled(
                                tooltip: l10n.btnApprove,
                                onPressed: () async {
                                  try {
                                    await context
                                        .read<ReviewQueueCubit>()
                                        .approve(pack.id);
                                  } on Failure catch (failure) {
                                    if (context.mounted) {
                                      context
                                          .read<ErrorCubit>()
                                          .showFailure(failure.message);
                                    }
                                  }
                                },
                                icon: const Icon(Icons.check_rounded),
                              ),
                            ),
                            Expanded(
                              child: IconButton.filled(
                                tooltip: l10n.btnReject,
                                style: IconButton.styleFrom(
                                  backgroundColor:
                                      Theme.of(context).colorScheme.error,
                                  foregroundColor:
                                      Theme.of(context).colorScheme.onError,
                                ),
                                onPressed: () => _reject(context, pack.id),
                                icon: const Icon(Icons.close_rounded),
                              ),
                            ),
                          ],
                        ),
                      );
                    },
                  ),
                ),
            ],
          ),
        );
      },
    );
  }
}
