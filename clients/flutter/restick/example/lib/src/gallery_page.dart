import "package:material_ui/material_ui.dart";
import "package:restick/restick.dart";

/// Drop-in gallery of locally installed packs, with the optional compact mode.
class GalleryPage extends StatefulWidget {
  const GalleryPage({required this.onStickerSelected, super.key});

  final ValueChanged<StickerAsset> onStickerSelected;

  @override
  State<GalleryPage> createState() => _GalleryPageState();
}

class _GalleryPageState extends State<GalleryPage> {
  bool _compact = false;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: <Widget>[
        SwitchListTile(
          key: const ValueKey<String>("gallery.compact"),
          value: _compact,
          title: const Text("Compact layout"),
          subtitle: const Text(
            "Same cubit: 5 grid columns and 44px pack chips, like the picker.",
          ),
          onChanged: (bool value) => setState(() => _compact = value),
        ),
        Expanded(
          child: StickerGallery(
            onStickerSelected: widget.onStickerSelected,
            compact: _compact,
          ),
        ),
      ],
    );
  }
}
