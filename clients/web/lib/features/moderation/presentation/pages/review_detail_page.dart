import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:go_router/go_router.dart";
import "package:restick_web/core/error/error_cubit.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/core/widgets/authenticated_image.dart";
import "package:restick_web/core/widgets/reason_dialogs.dart";
import "package:restick_web/core/widgets/status_pill.dart";
import "package:restick_web/features/moderation/presentation/cubit/review_detail_cubit.dart";
import "package:restick_web/features/packs/domain/entities/pack.dart";
import "package:restick_web/l10n/app_localizations.dart";

class ReviewDetailPage extends StatelessWidget {
  const ReviewDetailPage({super.key});

  Future<void> _reject(BuildContext context) async {
    final ReasonResult? result = await showDialog<ReasonResult>(
      context: context,
      builder: (BuildContext context) => const RejectReasonDialog(),
    );
    if (result == null || !context.mounted) {
      return;
    }
    try {
      await context.read<ReviewDetailCubit>().reject(
            reasonCode: result.code,
            reasonText: result.text,
          );
      if (context.mounted) {
        context.go("/admin/queue");
      }
    } on Failure catch (failure) {
      if (context.mounted) {
        context.read<ErrorCubit>().showFailure(failure.message);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return BlocBuilder<ReviewDetailCubit, ReviewDetailState>(
      builder: (BuildContext context, ReviewDetailState state) {
        final PackDetail? detail = state.detail;
        if (detail == null) {
          return const Padding(
            padding: EdgeInsets.all(24),
            child: LinearProgressIndicator(minHeight: 2),
          );
        }
        return Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            spacing: 16,
            children: [
              Row(
                spacing: 12,
                children: [
                  Expanded(
                    child: Text(
                      detail.summary.name,
                      style: Theme.of(context).textTheme.headlineSmall,
                    ),
                  ),
                  StatusPill(status: detail.summary.status),
                ],
              ),
              Row(
                spacing: 12,
                children: [
                  FilledButton.icon(
                    onPressed: () async {
                      try {
                        await context.read<ReviewDetailCubit>().approve();
                        if (context.mounted) {
                          context.go("/admin/queue");
                        }
                      } on Failure catch (failure) {
                        if (context.mounted) {
                          context
                              .read<ErrorCubit>()
                              .showFailure(failure.message);
                        }
                      }
                    },
                    icon: const Icon(Icons.check_rounded),
                    label: Text(l10n.btnApprove),
                  ),
                  FilledButton.icon(
                    style: FilledButton.styleFrom(
                      backgroundColor: Theme.of(context).colorScheme.error,
                    ),
                    onPressed: () => _reject(context),
                    icon: const Icon(Icons.close_rounded),
                    label: Text(l10n.btnReject),
                  ),
                ],
              ),
              Expanded(
                child: GridView.builder(
                  gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
                    maxCrossAxisExtent: 140,
                    mainAxisSpacing: 10,
                    crossAxisSpacing: 10,
                  ),
                  itemCount: detail.stickers.length,
                  itemBuilder: (BuildContext context, int index) {
                    final StickerPreview sticker = detail.stickers[index];
                    final String url = sticker.thumbUrl.isNotEmpty
                        ? sticker.thumbUrl
                        : sticker.fileUrl;
                    return Card(
                      child: Padding(
                        padding: const EdgeInsets.all(10),
                        child: AuthenticatedImage(url: url),
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
