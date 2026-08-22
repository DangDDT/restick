import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:go_router/go_router.dart";
import "package:restick_web/core/error/error_cubit.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/core/widgets/app_shell.dart";
import "package:restick_web/features/packs/domain/entities/pack.dart";
import "package:restick_web/features/packs/presentation/cubit/pack_list_cubit.dart";
import "package:restick_web/l10n/app_localizations.dart";

class MyPacksPage extends StatelessWidget {
  const MyPacksPage({super.key});

  bool _canDelete(PackSummary pack) {
    return pack.status == "rejected" || pack.status == "published";
  }

  Future<void> _onDelete(BuildContext context, PackSummary pack) async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final bool isRejected = pack.status == "rejected";
    final bool? confirmed = await showDialog<bool>(
      context: context,
      builder: (BuildContext context) {
        return AlertDialog(
          title: Text(
            isRejected
                ? l10n.confirmDeleteRejectedTitle
                : l10n.confirmRevokePublishedTitle,
          ),
          content: Text(
            isRejected
                ? l10n.confirmDeleteRejectedBody
                : l10n.confirmRevokePublishedBody,
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(context).pop(false),
              child: Text(l10n.btnCancel),
            ),
            FilledButton(
              onPressed: () => Navigator.of(context).pop(true),
              child: Text(
                isRejected ? l10n.btnConfirmDelete : l10n.btnConfirmRevoke,
              ),
            ),
          ],
        );
      },
    );
    if (confirmed != true || !context.mounted) {
      return;
    }
    try {
      final PackListCubit cubit = context.read<PackListCubit>();
      if (isRejected) {
        await cubit.deleteRejected(pack.id);
      } else {
        await cubit.ownerRevoke(pack.id);
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
    return BlocBuilder<PackListCubit, PackListState>(
      builder: (BuildContext context, PackListState state) {
        return Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            spacing: 16,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      spacing: 4,
                      children: [
                        Text(
                          l10n.titleMyPacks,
                          style: Theme.of(context).textTheme.headlineSmall,
                        ),
                        Text(
                          l10n.descMyPacks,
                          style: Theme.of(context).textTheme.bodyMedium,
                        ),
                      ],
                    ),
                  ),
                  FilledButton.icon(
                    onPressed: () => context.go("/packs/new"),
                    icon: const Icon(Icons.add_rounded),
                    label: Text(l10n.btnUpload),
                  ),
                ],
              ),
              if (!state.hasLoaded && state.isRefreshing)
                const LinearProgressIndicator(minHeight: 2)
              else if (state.items.isEmpty)
                Text(l10n.emptyPacks)
              else
                Expanded(
                  child: GridView.builder(
                    gridDelegate:
                        const SliverGridDelegateWithMaxCrossAxisExtent(
                      maxCrossAxisExtent: 280,
                      mainAxisSpacing: 16,
                      crossAxisSpacing: 16,
                      childAspectRatio: 0.78,
                    ),
                    itemCount: state.items.length,
                    itemBuilder: (BuildContext context, int index) {
                      final PackSummary pack = state.items[index];
                      return PackCard(
                        pack: pack,
                        onTap: () => context.go("/packs/${pack.id}"),
                        onDelete: _canDelete(pack)
                            ? () => _onDelete(context, pack)
                            : null,
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
