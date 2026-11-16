import "package:material_ui/material_ui.dart";
import "package:restick/src/core/restick_client.dart";
import "package:restick/src/domain/entities/sticker_asset.dart";
import "package:restick/src/presentation/widgets/sticker_gallery.dart";

class StickerPicker extends StatelessWidget {
  const StickerPicker({
    required this.onStickerSelected,
    this.client,
    super.key,
  });

  final RestickClient? client;
  final ValueChanged<StickerAsset> onStickerSelected;

  @override
  Widget build(BuildContext context) {
    return StickerGallery(
      client: client,
      onStickerSelected: onStickerSelected,
      compact: true,
    );
  }
}
