import "package:material_ui/material_ui.dart";
import "package:restick/restick.dart";

/// Embedded picker plus the modal-sheet variant used by chat composer bars.
class PickerPage extends StatelessWidget {
  const PickerPage({
    required this.client,
    required this.onStickerSelected,
    super.key,
  });

  final RestickClient client;
  final ValueChanged<StickerAsset> onStickerSelected;

  Future<void> _openSheet(BuildContext context) async {
    final StickerAsset? picked = await showModalBottomSheet<StickerAsset>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (BuildContext sheetContext) => SizedBox(
        height: 380,
        // A modal route sits above RestickScope, so pass the client
        // explicitly instead of relying on scope lookup.
        child: StickerPicker(
          client: client,
          onStickerSelected: (StickerAsset asset) =>
              Navigator.of(sheetContext).pop(asset),
        ),
      ),
    );
    if (picked != null) {
      onStickerSelected(picked);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(12),
      child: Column(
        spacing: 12,
        children: <Widget>[
          Card(
            child: ListTile(
              leading: const CircleAvatar(child: Icon(Icons.forum_outlined)),
              title: const Text("Chat composer preview"),
              subtitle: const Text(
                "Embed StickerPicker under the input, or open it as a sheet.",
              ),
            ),
          ),
          Expanded(
            child: Card(
              clipBehavior: Clip.antiAlias,
              child: StickerPicker(onStickerSelected: onStickerSelected),
            ),
          ),
          FilledButton.icon(
            key: const ValueKey<String>("picker.openSheet"),
            onPressed: () => _openSheet(context),
            icon: const Icon(Icons.keyboard_arrow_up),
            label: const Text("Open picker as bottom sheet"),
          ),
        ],
      ),
    );
  }
}
