import "dart:io";

import "package:material_ui/material_ui.dart";
import "package:restick/restick.dart";

/// Pre-fills the form from `--dart-define` so a run can be fully scripted.
const String _envBaseUrl = String.fromEnvironment(
  "RESTICK_BASE_URL",
  defaultValue: String.fromEnvironment(
    "RESTICK_API_BASE",
    defaultValue: "https://restick.dangddt.com",
  ),
);
const String _envApiKey = String.fromEnvironment("RESTICK_API_KEY");

const Map<int, String> _syncIntervals = <int, String>{
  15: "15 seconds",
  60: "1 minute",
  300: "5 minutes",
};

/// Collects [RestickConfig] values before the SDK is opened.
class SetupScreen extends StatefulWidget {
  const SetupScreen({required this.onOpen, required this.onFailure, super.key});

  final Future<void> Function(RestickConfig config) onOpen;
  final void Function(Failure failure) onFailure;

  @override
  State<SetupScreen> createState() => _SetupScreenState();
}

class _SetupScreenState extends State<SetupScreen> {
  final GlobalKey<FormState> _formKey = GlobalKey<FormState>();
  late final TextEditingController _baseUrl = TextEditingController(
    text: _envBaseUrl,
  );
  late final TextEditingController _apiKey = TextEditingController(
    text: _envApiKey,
  );
  int _intervalSeconds = 300;
  // Ad-hoc signed macOS builds cannot use the Keychain (-34018).
  bool _persistApiKey = !Platform.isMacOS;
  bool _obscureKey = true;
  bool _opening = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    // Scripted runs can skip the tap; the form still reports failures.
    if (const bool.fromEnvironment("RESTICK_AUTO_OPEN")) {
      WidgetsBinding.instance.addPostFrameCallback((Duration _) {
        if (mounted) {
          _submit().ignore();
        }
      });
    }
  }

  @override
  void dispose() {
    _baseUrl.dispose();
    _apiKey.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }
    setState(() {
      _opening = true;
      _error = null;
    });
    final RestickConfig config = RestickConfig(
      baseUrl: _baseUrl.text.trim(),
      apiKey: _apiKey.text.trim(),
      autoSyncInterval: Duration(seconds: _intervalSeconds),
      persistApiKey: _persistApiKey,
      onError: widget.onFailure,
    );
    try {
      await widget.onOpen(config);
    } on Object catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _opening = false;
        _error = error.toString();
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    return Scaffold(
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 520),
            child: Form(
              key: _formKey,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                spacing: 16,
                children: <Widget>[
                  Icon(
                    Icons.emoji_emotions_outlined,
                    size: 56,
                    color: colors.primary,
                  ),
                  Text(
                    "Restick SDK ${Restick.sdkVersion}",
                    textAlign: TextAlign.center,
                    style: Theme.of(context).textTheme.headlineSmall,
                  ),
                  Text(
                    "Point the SDK at a Restick gateway and paste a read-only "
                    "API key. Both values can also come from "
                    "--dart-define=RESTICK_BASE_URL / RESTICK_API_KEY.",
                    textAlign: TextAlign.center,
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                  TextFormField(
                    key: const ValueKey<String>("setup.baseUrl"),
                    controller: _baseUrl,
                    keyboardType: TextInputType.url,
                    autocorrect: false,
                    decoration: const InputDecoration(
                      labelText: "Base URL",
                      hintText: "https://restick.dangddt.com",
                      border: OutlineInputBorder(),
                    ),
                    validator: (String? value) {
                      final Uri? uri = Uri.tryParse(value?.trim() ?? "");
                      if (uri == null ||
                          !uri.hasScheme ||
                          (uri.scheme != "http" && uri.scheme != "https")) {
                        return "Enter an http(s) origin.";
                      }
                      return null;
                    },
                  ),
                  TextFormField(
                    key: const ValueKey<String>("setup.apiKey"),
                    controller: _apiKey,
                    autocorrect: false,
                    enableSuggestions: false,
                    obscureText: _obscureKey,
                    decoration: InputDecoration(
                      labelText: "API key",
                      hintText: "rsk_...",
                      border: const OutlineInputBorder(),
                      suffixIcon: IconButton(
                        tooltip: _obscureKey ? "Show key" : "Hide key",
                        icon: Icon(
                          _obscureKey
                              ? Icons.visibility_outlined
                              : Icons.visibility_off_outlined,
                        ),
                        onPressed: () {
                          setState(() => _obscureKey = !_obscureKey);
                        },
                      ),
                    ),
                    validator: (String? value) {
                      if ((value?.trim() ?? "").isEmpty) {
                        return "The gateway requires an API key.";
                      }
                      return null;
                    },
                  ),
                  DropdownButtonFormField<int>(
                    key: const ValueKey<String>("setup.autoSync"),
                    initialValue: _intervalSeconds,
                    decoration: const InputDecoration(
                      labelText: "Tombstone auto-sync",
                      border: OutlineInputBorder(),
                    ),
                    items: _syncIntervals.entries
                        .map(
                          (MapEntry<int, String> entry) =>
                              DropdownMenuItem<int>(
                                value: entry.key,
                                child: Text(entry.value),
                              ),
                        )
                        .toList(),
                    onChanged: (int? value) {
                      if (value != null) {
                        setState(() => _intervalSeconds = value);
                      }
                    },
                  ),
                  SwitchListTile(
                    key: const ValueKey<String>("setup.persistKey"),
                    contentPadding: EdgeInsets.zero,
                    title: const Text("Persist API key in secure storage"),
                    subtitle: Text(
                      Platform.isMacOS
                          ? "RestickConfig.persistApiKey. Ad-hoc macOS builds "
                                "need a signed runner for Keychain access."
                          : "RestickConfig.persistApiKey (Keychain / Keystore).",
                    ),
                    value: _persistApiKey,
                    onChanged: (bool value) {
                      setState(() => _persistApiKey = value);
                    },
                  ),
                  if (_error != null)
                    Text(
                      _error!,
                      key: const ValueKey<String>("setup.error"),
                      style: TextStyle(color: colors.error),
                    ),
                  FilledButton.icon(
                    key: const ValueKey<String>("setup.open"),
                    onPressed: _opening ? null : _submit,
                    icon: _opening
                        ? const SizedBox.square(
                            dimension: 18,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.login),
                    label: Text(_opening ? "Opening..." : "Open SDK"),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
