import "package:material_ui/material_ui.dart";
import "package:restick/restick.dart";

/// Drop-in marketplace widget with the optional server-side tag filter.
class MarketplacePage extends StatefulWidget {
  const MarketplacePage({super.key});

  @override
  State<MarketplacePage> createState() => _MarketplacePageState();
}

class _MarketplacePageState extends State<MarketplacePage> {
  final TextEditingController _tag = TextEditingController();
  String? _activeTag;

  @override
  void dispose() {
    _tag.dispose();
    super.dispose();
  }

  void _apply() {
    setState(
      () => _activeTag = _tag.text.trim().isEmpty ? null : _tag.text.trim(),
    );
  }

  void _clear() {
    _tag.clear();
    setState(() => _activeTag = null);
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: <Widget>[
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 8),
          child: Row(
            spacing: 8,
            children: <Widget>[
              Expanded(
                child: TextField(
                  key: const ValueKey<String>("marketplace.tag"),
                  controller: _tag,
                  textInputAction: TextInputAction.search,
                  onSubmitted: (_) => _apply(),
                  decoration: InputDecoration(
                    isDense: true,
                    labelText: "Filter by tag",
                    hintText: "e.g. con vat",
                    border: const OutlineInputBorder(),
                    prefixIcon: const Icon(Icons.filter_alt_outlined, size: 20),
                    suffixIcon: _activeTag == null
                        ? null
                        : IconButton(
                            tooltip: "Clear tag",
                            onPressed: _clear,
                            icon: const Icon(Icons.close, size: 18),
                          ),
                  ),
                ),
              ),
              FilledButton.tonal(onPressed: _apply, child: const Text("Apply")),
            ],
          ),
        ),
        if (_activeTag != null)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Text(
              "Showing packs tagged \"$_activeTag\" · pull to refresh",
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
        // A new key recreates the cubit when the tag changes. Without a tag
        // param the widget asks the gateway for the whole catalog.
        Expanded(
          child: StickerMarketplace(
            key: ValueKey<String>(_activeTag ?? "__all__"),
            tag: _activeTag,
          ),
        ),
      ],
    );
  }
}
