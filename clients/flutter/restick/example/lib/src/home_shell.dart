import "dart:io";

import "package:flutter/foundation.dart";
import "package:material_ui/material_ui.dart";
import "package:restick/restick.dart";

import "gallery_page.dart";
import "marketplace_page.dart";
import "picker_page.dart";
import "sdk_page.dart";

/// Four entry points: the three drop-in widgets plus the raw client API.
class HomeShell extends StatefulWidget {
  const HomeShell({
    required this.client,
    required this.themeMode,
    required this.onThemeModeChanged,
    required this.onDisconnect,
    required this.failures,
    super.key,
  });

  final RestickClient client;
  final ThemeMode themeMode;
  final ValueChanged<ThemeMode> onThemeModeChanged;
  final Future<void> Function() onDisconnect;
  final ValueListenable<List<String>> failures;

  @override
  State<HomeShell> createState() => _HomeShellState();
}

class _HomeShellState extends State<HomeShell> with WidgetsBindingObserver {
  static const List<String> _tabs = <String>[
    "Marketplace",
    "Gallery",
    "Picker",
    "SDK API",
  ];

  int _index = 0;
  final Set<int> _visited = <int>{0};
  StickerAsset? _selected;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    // Background tombstone sync is cheap when the app is not visible.
    if (state == AppLifecycleState.paused) {
      widget.client.stopPeriodicAutoSync();
    } else if (state == AppLifecycleState.resumed) {
      widget.client.startPeriodicAutoSync();
      widget.client.syncRevocations().ignore();
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Text("Restick SDK ${Restick.sdkVersion}"),
            Text(
              widget.client.config.baseUrl,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
        ),
        actions: <Widget>[
          IconButton(
            tooltip: "Clear selected sticker",
            onPressed: _selected == null
                ? null
                : () => setState(() => _selected = null),
            icon: const Icon(Icons.clear_all),
          ),
          PopupMenuButton<ThemeMode>(
            tooltip: "Theme",
            icon: const Icon(Icons.brightness_6_outlined),
            initialValue: widget.themeMode,
            onSelected: widget.onThemeModeChanged,
            itemBuilder: (BuildContext context) =>
                const <PopupMenuEntry<ThemeMode>>[
                  PopupMenuItem<ThemeMode>(
                    value: ThemeMode.system,
                    child: Text("System theme"),
                  ),
                  PopupMenuItem<ThemeMode>(
                    value: ThemeMode.light,
                    child: Text("Light"),
                  ),
                  PopupMenuItem<ThemeMode>(
                    value: ThemeMode.dark,
                    child: Text("Dark"),
                  ),
                ],
          ),
          IconButton(
            tooltip: "Disconnect",
            onPressed: () => widget.onDisconnect().ignore(),
            icon: const Icon(Icons.logout),
          ),
        ],
      ),
      body: Column(
        children: <Widget>[
          if (_selected != null)
            StickerPreview(
              asset: _selected!,
              onCleared: () => setState(() => _selected = null),
            ),
          Expanded(
            child: IndexedStack(
              index: _index,
              children: List<Widget>.generate(
                _tabs.length,
                (int index) => TickerMode(
                  enabled: index == _index,
                  child: _visited.contains(index) ? _buildTab(index) : const SizedBox.shrink(),
                ),
              ),
            ),
          ),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _index,
        destinations: const <NavigationDestination>[
          NavigationDestination(
            icon: Icon(Icons.storefront_outlined),
            selectedIcon: Icon(Icons.storefront),
            label: "Marketplace",
          ),
          NavigationDestination(
            icon: Icon(Icons.photo_library_outlined),
            selectedIcon: Icon(Icons.photo_library),
            label: "Gallery",
          ),
          NavigationDestination(
            icon: Icon(Icons.emoji_emotions_outlined),
            selectedIcon: Icon(Icons.emoji_emotions),
            label: "Picker",
          ),
          NavigationDestination(
            icon: Icon(Icons.terminal_outlined),
            selectedIcon: Icon(Icons.terminal),
            label: "SDK API",
          ),
        ],
        onDestinationSelected: (int index) {
          setState(() {
            _index = index;
            _visited.add(index);
          });
        },
      ),
    );
  }

  Widget _buildTab(int index) {
    switch (index) {
      case 0:
        return const MarketplacePage();
      case 1:
        return GalleryPage(onStickerSelected: _selectSticker);
      case 2:
        return PickerPage(
          client: widget.client,
          onStickerSelected: _selectSticker,
        );
      default:
        return SdkPage(
          client: widget.client,
          failures: widget.failures,
          onStickerSelected: _selectSticker,
        );
    }
  }

  void _selectSticker(StickerAsset asset) {
    setState(() => _selected = asset);
  }
}

/// Shows the file a host app would send plus its metadata.
class StickerPreview extends StatelessWidget {
  const StickerPreview({
    required this.asset,
    required this.onCleared,
    super.key,
  });

  final StickerAsset asset;
  final VoidCallback onCleared;

  @override
  Widget build(BuildContext context) {
    final bool reducedMotion = MediaQuery.disableAnimationsOf(context);
    final File file = reducedMotion
        ? asset.file
        : (asset.animatedFile ?? asset.file);
    return Card(
      margin: const EdgeInsets.fromLTRB(12, 12, 12, 0),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          spacing: 12,
          children: <Widget>[
            Image.file(
              file,
              key: ValueKey<String>(file.path),
              height: 84,
              fit: BoxFit.contain,
            ),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                spacing: 4,
                children: <Widget>[
                  Wrap(
                    spacing: 8,
                    runSpacing: 4,
                    crossAxisAlignment: WrapCrossAlignment.center,
                    children: <Widget>[
                      Text(
                        asset.emojiHint.isEmpty
                            ? "Sticker selected"
                            : asset.emojiHint,
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      if (asset.isAnimated)
                        const Chip(
                          visualDensity: VisualDensity.compact,
                          label: Text("animated"),
                        ),
                      if (reducedMotion && asset.isAnimated)
                        const Chip(
                          visualDensity: VisualDensity.compact,
                          label: Text("reduced motion: static"),
                        ),
                    ],
                  ),
                  Text(
                    "${asset.width}×${asset.height} · ${file.path.split("/").last}",
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                  Text(
                    file.path,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: Theme.of(context).colorScheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ),
            IconButton(
              tooltip: "Clear",
              onPressed: onCleared,
              icon: const Icon(Icons.close),
            ),
          ],
        ),
      ),
    );
  }
}
