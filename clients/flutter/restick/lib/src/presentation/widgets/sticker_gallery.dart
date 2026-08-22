import "package:flutter/material.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick/src/core/restick_client.dart";
import "package:restick/src/domain/entities/sticker_asset.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/presentation/cubit/gallery_cubit.dart";
import "package:restick/src/presentation/scope/restick_scope.dart";
import "package:restick/src/presentation/widgets/empty_state.dart";
import "package:restick/src/presentation/widgets/sticker_grid.dart";
import "package:restick/src/presentation/widgets/sticker_pack_chip.dart";

class StickerGallery extends StatelessWidget {
  const StickerGallery({
    required this.onStickerSelected,
    this.client,
    this.compact = false,
    super.key,
  });

  final RestickClient? client;
  final ValueChanged<StickerAsset> onStickerSelected;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    return BlocProvider<GalleryCubit>(
      create: (_) => GalleryCubit(
        client: client ?? RestickScope.of(context),
      )..load(),
      child: _GalleryView(
        onStickerSelected: onStickerSelected,
        compact: compact,
      ),
    );
  }
}

class _GalleryView extends StatelessWidget {
  const _GalleryView({
    required this.onStickerSelected,
    required this.compact,
  });

  final ValueChanged<StickerAsset> onStickerSelected;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<GalleryCubit, GalleryState>(
      builder: (BuildContext context, GalleryState state) {
        if (state.packs.isEmpty && !state.isLoading) {
          return const RestickEmptyState(
            message:
                "No downloaded stickers yet. Open the marketplace to download a pack.",
            icon: Icons.photo_library_outlined,
          );
        }
        return Column(
          spacing: 8,
          children: <Widget>[
            _GalleryPackStrip(state: state, compact: compact),
            Expanded(
              child: state.stickers.isEmpty
                  ? const RestickEmptyState(
                      message: "This pack has no local stickers.",
                    )
                  : StickerGrid(
                      assets: state.stickers,
                      onStickerSelected: onStickerSelected,
                      columns: compact ? 5 : null,
                    ),
            ),
          ],
        );
      },
    );
  }
}

class _GalleryPackStrip extends StatelessWidget {
  const _GalleryPackStrip({required this.state, required this.compact});

  final GalleryState state;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: compact ? 44 : 48,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 8),
        itemCount: state.packs.length,
        separatorBuilder: (BuildContext context, int index) {
          return const SizedBox(width: 8);
        },
        itemBuilder: (BuildContext context, int index) {
          final StickerPack pack = state.packs[index];
          return StickerPackChip(
            pack: pack,
            selected: pack.id == state.selectedPackId,
            onSelected: (StickerPack selected) {
              context.read<GalleryCubit>().selectPack(selected.id);
            },
          );
        },
      ),
    );
  }
}
