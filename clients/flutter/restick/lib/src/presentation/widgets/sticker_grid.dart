import "package:flutter/material.dart";
import "package:restick/src/domain/entities/sticker_asset.dart";
import "package:restick/src/presentation/theme/restick_theme.dart";
import "package:restick/src/presentation/widgets/sticker_grid_tile.dart";

class StickerGrid extends StatelessWidget {
  const StickerGrid({
    required this.assets,
    required this.onStickerSelected,
    this.columns,
    super.key,
  });

  final List<StickerAsset> assets;
  final ValueChanged<StickerAsset> onStickerSelected;
  final int? columns;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) {
        final int crossAxisCount =
            columns ?? RestickBreakpoints.gridColumns(constraints.maxWidth);
        return GridView.builder(
          gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount: crossAxisCount,
            mainAxisSpacing: 8,
            crossAxisSpacing: 8,
          ),
          itemCount: assets.length,
          itemBuilder: (BuildContext context, int index) {
            return StickerGridTile(
              asset: assets[index],
              onSelected: onStickerSelected,
            );
          },
        );
      },
    );
  }
}
