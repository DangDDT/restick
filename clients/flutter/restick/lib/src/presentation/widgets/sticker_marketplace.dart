import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick/src/core/restick_client.dart";
import "package:restick/src/presentation/cubit/marketplace_cubit.dart";
import "package:restick/src/presentation/scope/restick_scope.dart";
import "package:restick/src/presentation/widgets/empty_state.dart";
import "package:restick/src/presentation/widgets/marketplace_pack_tile.dart";

class StickerMarketplace extends StatelessWidget {
  const StickerMarketplace({this.client, this.tag, super.key});

  final RestickClient? client;
  final String? tag;

  @override
  Widget build(BuildContext context) {
    return BlocProvider<MarketplaceCubit>(
      create: (_) => MarketplaceCubit(
        client: client ?? RestickScope.of(context),
      )..load(tag: tag),
      child: const _MarketplaceView(),
    );
  }
}

class _MarketplaceView extends StatelessWidget {
  const _MarketplaceView();

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<MarketplaceCubit, MarketplaceState>(
      builder: (BuildContext context, MarketplaceState state) {
        if (state.packs.isEmpty && state.isLoading) {
          return const SizedBox.shrink();
        }
        if (state.packs.isEmpty) {
          return RestickEmptyState(
            message: state.failure?.message ?? "No sticker packs available.",
            icon: Icons.storefront_outlined,
          );
        }
        return RefreshIndicator(
          onRefresh: () => context.read<MarketplaceCubit>().load(),
          child: ListView.separated(
            itemCount: state.packs.length,
            separatorBuilder: (BuildContext context, int index) {
              return const Divider(height: 1);
            },
            itemBuilder: (BuildContext context, int index) {
              final MarketplaceCubit cubit = context.read<MarketplaceCubit>();
              return MarketplacePackTile(
                pack: state.packs[index],
                onDownload: () => cubit.download(state.packs[index].id),
                onDelete: () => cubit.delete(state.packs[index].id),
              );
            },
          ),
        );
      },
    );
  }
}
