import "package:material_ui/material_ui.dart";
import "package:restick/src/domain/entities/sticker_asset.dart";

class StickerGridTile extends StatelessWidget {
  const StickerGridTile({
    required this.asset,
    required this.onSelected,
    super.key,
  });

  final StickerAsset asset;
  final ValueChanged<StickerAsset> onSelected;

  @override
  Widget build(BuildContext context) {
    final bool playAnimated =
        asset.isAnimated && !MediaQuery.disableAnimationsOf(context);
    return InkWell(
      key: ValueKey('sticker.grid.tile.${asset.file.path}'),
      onTap: () => onSelected(asset),
      child: Image.file(
        playAnimated ? asset.animatedFile! : asset.preview,
        cacheWidth: 128,
        cacheHeight: 128,
        fit: BoxFit.contain,
      ),
    );
  }
}
