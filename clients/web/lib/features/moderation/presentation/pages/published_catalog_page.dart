import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick_web/core/error/error_cubit.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/core/widgets/app_shell.dart";
import "package:restick_web/core/widgets/reason_dialogs.dart";
import "package:restick_web/features/moderation/presentation/cubit/published_catalog_cubit.dart";
import "package:restick_web/l10n/app_localizations.dart";

class PublishedCatalogPage extends StatelessWidget {
  const PublishedCatalogPage({super.key});

  Future<void> _revoke(BuildContext context, String id) async {
    final ReasonResult? result = await showDialog<ReasonResult>(
      context: context,
      builder: (BuildContext context) => const RevokeReasonDialog(),
    );
    if (result == null || !context.mounted) {
      return;
    }
    try {
      await context.read<PublishedCatalogCubit>().revoke(
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
    return BlocBuilder<PublishedCatalogCubit, PublishedCatalogState>(
      builder: (BuildContext context, PublishedCatalogState state) {
        return Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            spacing: 16,
            children: [
              Text(
                l10n.titleCatalog,
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              Text(
                l10n.descCatalog,
                style: Theme.of(context).textTheme.bodyMedium,
              ),
              if (!state.hasLoaded && state.isRefreshing)
                const LinearProgressIndicator(minHeight: 2)
              else if (state.items.isEmpty)
                Text(l10n.emptyPublished)
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
                        onTap: () {},
                        trailing: IconButton.filled(
                          tooltip: l10n.btnRevoke,
                          style: IconButton.styleFrom(
                            backgroundColor:
                                Theme.of(context).colorScheme.error,
                            foregroundColor:
                                Theme.of(context).colorScheme.onError,
                          ),
                          onPressed: () => _revoke(context, pack.id),
                          icon: const Icon(Icons.block_rounded),
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
