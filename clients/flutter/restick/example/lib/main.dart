import "dart:io";

import "package:flutter/material.dart";
import "package:restick/restick.dart";

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final RestickClient client = await Restick.open(
    RestickConfig.fromEnvironment(),
  );
  runApp(ExampleApp(client: client));
}

class ExampleApp extends StatelessWidget {
  const ExampleApp({required this.client, super.key});

  final RestickClient client;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: "Restick SDK ${Restick.sdkVersion}",
      theme: ThemeData(useMaterial3: true, brightness: Brightness.light),
      darkTheme: ThemeData(useMaterial3: true, brightness: Brightness.dark),
      home: RestickScope(
        client: client,
        child: ExampleHome(client: client),
      ),
    );
  }
}

class ExampleHome extends StatefulWidget {
  const ExampleHome({required this.client, super.key});

  final RestickClient client;

  @override
  State<ExampleHome> createState() => _ExampleHomeState();
}

class _ExampleHomeState extends State<ExampleHome> with WidgetsBindingObserver {
  File? _selected;
  int _tab = 0;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    widget.client.dispose();
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.paused) {
      widget.client.stopPeriodicAutoSync();
    } else if (state == AppLifecycleState.resumed) {
      widget.client.startPeriodicAutoSync();
      widget.client.syncRevocations();
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text("Restick SDK ${Restick.sdkVersion}"),
      ),
      body: Column(
        spacing: 12,
        children: <Widget>[
          if (_selected != null)
            SizedBox(
              height: 160,
              child: Image.file(_selected!, fit: BoxFit.contain),
            ),
          Expanded(
            child: ExampleTabBody(
              tab: _tab,
              onStickerSelected: _onStickerSelected,
            ),
          ),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _tab,
        destinations: const <NavigationDestination>[
          NavigationDestination(
            icon: Icon(Icons.storefront_outlined),
            label: "Marketplace",
          ),
          NavigationDestination(
            icon: Icon(Icons.photo_library_outlined),
            label: "Gallery",
          ),
          NavigationDestination(
            icon: Icon(Icons.emoji_emotions_outlined),
            label: "Picker",
          ),
        ],
        onDestinationSelected: (int index) {
          setState(() {
            _tab = index;
          });
        },
      ),
    );
  }

  void _onStickerSelected(StickerAsset asset) {
    setState(() {
      _selected = asset.file;
    });
  }
}

class ExampleTabBody extends StatelessWidget {
  const ExampleTabBody({
    required this.tab,
    required this.onStickerSelected,
    super.key,
  });

  final int tab;
  final ValueChanged<StickerAsset> onStickerSelected;

  @override
  Widget build(BuildContext context) {
    switch (tab) {
      case 0:
        return const StickerMarketplace();
      case 1:
        return StickerGallery(onStickerSelected: onStickerSelected);
      default:
        return StickerPicker(onStickerSelected: onStickerSelected);
    }
  }
}
