import "package:material_ui/material_ui.dart";
import "package:restick/restick.dart";

import "src/home_shell.dart";
import "src/setup_screen.dart";

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const RestickExampleApp());
}

/// Hosts the SDK lifecycle: configuration, client ownership and theming.
class RestickExampleApp extends StatefulWidget {
  const RestickExampleApp({super.key});

  @override
  State<RestickExampleApp> createState() => _RestickExampleAppState();
}

class _RestickExampleAppState extends State<RestickExampleApp> {
  RestickClient? _client;
  ThemeMode _themeMode = ThemeMode.system;
  final ValueNotifier<List<String>> _failures = ValueNotifier<List<String>>(
    <String>[],
  );

  @override
  void dispose() {
    _client?.dispose().ignore();
    _failures.dispose();
    super.dispose();
  }

  Future<void> _open(RestickConfig config) async {
    final RestickClient client = await Restick.open(config);
    if (!mounted) {
      await client.dispose();
      return;
    }
    setState(() => _client = client);
  }

  Future<void> _disconnect() async {
    final RestickClient? client = _client;
    setState(() => _client = null);
    await client?.dispose();
  }

  void _reportFailure(Failure failure) {
    final String stamp = DateTime.now().toIso8601String().substring(11, 19);
    _failures.value = <String>[
      ..._failures.value,
      "$stamp ${failure.runtimeType}: ${failure.message}",
    ];
  }

  @override
  Widget build(BuildContext context) {
    final RestickClient? client = _client;
    return MaterialApp(
      title: "Restick example",
      debugShowCheckedModeBanner: false,
      theme: buildRestickThemeData(brightness: Brightness.light),
      darkTheme: buildRestickThemeData(brightness: Brightness.dark),
      themeMode: _themeMode,
      localizationsDelegates: GlobalMaterialLocalizations.delegates,
      builder: (BuildContext context, Widget? child) {
        // Bridge legacy SDK Material widgets until packages migrate.
        // ignore: deprecated_member_use
        return MaterialUiCompatibilityBridge(child: child!);
      },
      home: client == null
          ? SetupScreen(onOpen: _open, onFailure: _reportFailure)
          : RestickScope(
              client: client,
              child: HomeShell(
                client: client,
                themeMode: _themeMode,
                onThemeModeChanged: (ThemeMode mode) {
                  setState(() => _themeMode = mode);
                },
                onDisconnect: _disconnect,
                failures: _failures,
              ),
            ),
    );
  }
}
