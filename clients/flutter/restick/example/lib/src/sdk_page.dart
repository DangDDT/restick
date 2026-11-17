import "dart:async";

import "package:flutter/foundation.dart";
import "package:material_ui/material_ui.dart";
import "package:restick/restick.dart";

/// Exercises the `RestickClient` API directly, next to the drop-in widgets.
class SdkPage extends StatefulWidget {
  const SdkPage({
    required this.client,
    required this.failures,
    required this.onStickerSelected,
    super.key,
  });

  final RestickClient client;
  final ValueListenable<List<String>> failures;
  final ValueChanged<StickerAsset> onStickerSelected;

  @override
  State<SdkPage> createState() => _SdkPageState();
}

class _SdkPageState extends State<SdkPage> {
  final List<String> _events = <String>[];
  List<StickerPack> _packs = const <StickerPack>[];
  String? _selectedPackId;
  PackDownloadState? _lastDownload;
  StreamSubscription<PackDownloadState>? _downloads;
  StreamSubscription<PackDownloadState>? _packWatch;
  bool _autoSync = true;
  bool _refreshing = false;

  @override
  void initState() {
    super.initState();
    _downloads = widget.client.watchDownloads().listen((
      PackDownloadState state,
    ) {
      if (!mounted) {
        return;
      }
      setState(() => _lastDownload = state);
      _log(
        "watchDownloads · ${_shortId(state.packId)} "
        "${state.status.name} ${(state.progress * 100).round()}%"
        "${state.message.isEmpty ? "" : " · ${state.message}"}",
      );
    });
    unawaited(_refreshCatalog());
  }

  @override
  void dispose() {
    _downloads?.cancel();
    _packWatch?.cancel();
    super.dispose();
  }

  StickerPack? get _selectedPack {
    for (final StickerPack pack in _packs) {
      if (pack.id == _selectedPackId) {
        return pack;
      }
    }
    return null;
  }

  Future<void> _refreshCatalog() async {
    setState(() => _refreshing = true);
    try {
      final List<StickerPack> packs = await widget.client.listMarketplace();
      if (!mounted) {
        return;
      }
      setState(() {
        _packs = packs;
        _selectedPackId =
            packs.any((StickerPack pack) => pack.id == _selectedPackId)
            ? _selectedPackId
            : (packs.isEmpty ? null : packs.first.id);
        _refreshing = false;
      });
      _log("listMarketplace · ${_describe(packs)}");
    } on Object catch (error) {
      if (!mounted) {
        return;
      }
      setState(() => _refreshing = false);
      _log("listMarketplace failed · $error");
    }
  }

  Future<void> _run(String label, Future<Object?> Function() action) async {
    _log("→ $label");
    try {
      final Object? value = await action();
      _log("✓ $label · ${_describe(value)}");
    } on Failure catch (failure) {
      _log("✗ $label · ${failure.runtimeType}: ${failure.message}");
    } on Object catch (error) {
      _log("✗ $label · $error");
    }
  }

  void _watchSelectedPack() {
    final StickerPack? pack = _selectedPack;
    if (pack == null) {
      return;
    }
    _packWatch?.cancel();
    _log("→ watchPack(${_shortId(pack.id)}) subscribed");
    _packWatch = widget.client
        .watchPack(pack.id)
        .listen(
          (PackDownloadState state) => _log(
            "watchPack · ${state.status.name} "
            "${(state.progress * 100).round()}%",
          ),
          onError: (Object error) => _log("watchPack · error: $error"),
        );
  }

  Future<void> _previewStickers() async {
    final StickerPack? pack = _selectedPack;
    if (pack == null) {
      return;
    }
    final List<StickerAsset> assets;
    try {
      assets = await widget.client.getStickers(pack.id);
    } on Object catch (error) {
      _log("✗ getStickers · $error");
      return;
    }
    _log("✓ getStickers · ${_describe(assets)}");
    if (!mounted || assets.isEmpty) {
      return;
    }
    await showDialog<void>(
      context: context,
      builder: (BuildContext dialogContext) => AlertDialog(
        title: Text("${pack.name} · ${assets.length} stickers"),
        content: SizedBox(
          width: 460,
          height: 380,
          child: StickerGrid(
            assets: assets,
            onStickerSelected: (StickerAsset asset) {
              Navigator.of(dialogContext).pop();
              widget.onStickerSelected(asset);
            },
          ),
        ),
        actions: <Widget>[
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(),
            child: const Text("Close"),
          ),
        ],
      ),
    );
  }

  void _log(String message) {
    if (!mounted) {
      return;
    }
    final String stamp = DateTime.now().toIso8601String().substring(11, 19);
    setState(() {
      _events.insert(0, "$stamp  $message");
      if (_events.length > 120) {
        _events.removeLast();
      }
    });
  }

  String _describe(Object? value) {
    if (value == null) {
      return "null (nothing cached in this process)";
    }
    if (value is List<StickerPack>) {
      if (value.isEmpty) {
        return "no packs";
      }
      return "${value.length} pack(s): "
          "${value.map((StickerPack pack) => pack.name).join(", ")}";
    }
    if (value is List<StickerAsset>) {
      return "${value.length} local sticker(s)";
    }
    if (value is StickerPack) {
      return "${value.name} · v${value.version} · "
          "${value.downloadStatus.name} · ${value.stickerCount} stickers";
    }
    if (value is TombstoneSyncResult) {
      return "purged ${value.purgedPackIds.length}, "
          "tombstones ${value.tombstones.length}, "
          "notifiable ${value.notifiableTombstones.length}";
    }
    if (value is String) {
      return value.isEmpty ? '""' : value;
    }
    return value.toString();
  }

  String _shortId(String id) => id.length <= 8 ? id : id.substring(0, 8);

  String _maskKey(String key) {
    if (key.length <= 12) {
      return "****";
    }
    return "${key.substring(0, 8)}…${key.substring(key.length - 4)}";
  }

  @override
  Widget build(BuildContext context) {
    final StickerPack? pack = _selectedPack;
    return ListView(
      padding: const EdgeInsets.all(12),
      children: <Widget>[
        _ConnectionCard(
          client: widget.client,
          autoSync: _autoSync,
          maskedKey: _maskKey(widget.client.config.apiKey),
          onToggleAutoSync: (bool start) {
            if (start) {
              widget.client.startPeriodicAutoSync();
            } else {
              widget.client.stopPeriodicAutoSync();
            }
            setState(() => _autoSync = start);
            _log(start ? "startPeriodicAutoSync" : "stopPeriodicAutoSync");
          },
        ),
        const SizedBox(height: 12),
        _CatalogCard(
          packs: _packs,
          selectedPackId: _selectedPackId,
          refreshing: _refreshing,
          onRefresh: _refreshCatalog,
          onSelect: (String? id) => setState(() => _selectedPackId = id),
          onListMarketplace: () =>
              _run("listMarketplace", widget.client.listMarketplace),
          onCachedMarketplace: () =>
              _run("cachedMarketplace", widget.client.cachedMarketplace),
          onGetPack: pack == null
              ? null
              : () => _run(
                  "getPack(${_shortId(pack.id)})",
                  () => widget.client.getPack(pack.id),
                ),
          onDownload: pack == null
              ? null
              : () => _run(
                  "downloadPack(${_shortId(pack.id)})",
                  () => widget.client.downloadPack(pack.id),
                ),
          onDelete: pack == null
              ? null
              : () => _run(
                  "deletePack(${_shortId(pack.id)})",
                  () => widget.client.deletePack(pack.id),
                ),
          onWatch: pack == null ? null : _watchSelectedPack,
          onGetStickers: pack == null
              ? null
              : () => _run(
                  "getStickers(${_shortId(pack.id)})",
                  () => widget.client.getStickers(pack.id),
                ),
          onPreviewStickers: pack == null ? null : _previewStickers,
          onSelectPack: pack == null
              ? null
              : () => _run(
                  "selectPack(${_shortId(pack.id)})",
                  () => widget.client.selectPack(pack.id),
                ),
          onLastSelectedPack: () =>
              _run("lastSelectedPackId", widget.client.lastSelectedPackId),
          onListGallery: () => _run("listGallery", widget.client.listGallery),
          onSyncRevocations: () =>
              _run("syncRevocations", widget.client.syncRevocations),
        ),
        const SizedBox(height: 12),
        _StateCard(
          client: widget.client,
          selectedPack: pack,
          lastDownload: _lastDownload,
        ),
        const SizedBox(height: 12),
        _FailuresCard(client: widget.client, failures: widget.failures),
        const SizedBox(height: 12),
        _EventLogCard(events: _events, onClear: () => setState(_events.clear)),
      ],
    );
  }
}

class _ConnectionCard extends StatelessWidget {
  const _ConnectionCard({
    required this.client,
    required this.autoSync,
    required this.maskedKey,
    required this.onToggleAutoSync,
  });

  final RestickClient client;
  final bool autoSync;
  final String maskedKey;
  final ValueChanged<bool> onToggleAutoSync;

  @override
  Widget build(BuildContext context) {
    final RestickConfig config = client.config;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          spacing: 6,
          children: <Widget>[
            Text(
              "Configuration",
              style: Theme.of(context).textTheme.titleMedium,
            ),
            _KeyValue(label: "baseUrl", value: config.baseUrl),
            _KeyValue(label: "apiKey", value: maskedKey),
            _KeyValue(
              label: "autoSyncInterval",
              value: "${config.autoSyncInterval.inSeconds}s",
            ),
            _KeyValue(
              label: "persistApiKey",
              value: config.persistApiKey.toString(),
            ),
            _KeyValue(label: "sdkVersion", value: Restick.sdkVersion),
            StreamBuilder<RestickLoadingState>(
              stream: client.loadingCubit.stream,
              initialData: client.loadingCubit.state,
              builder:
                  (
                    BuildContext context,
                    AsyncSnapshot<RestickLoadingState> snapshot,
                  ) => _KeyValue(
                    label: "loadingCubit",
                    value: snapshot.data?.isLoading ?? false
                        ? "active"
                        : "idle",
                  ),
            ),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              visualDensity: VisualDensity.compact,
              title: const Text("Periodic tombstone sync"),
              subtitle: Text(
                autoSync
                    ? "Running every ${config.autoSyncInterval.inSeconds}s"
                    : "Stopped manually",
              ),
              value: autoSync,
              onChanged: onToggleAutoSync,
            ),
          ],
        ),
      ),
    );
  }
}

class _CatalogCard extends StatelessWidget {
  const _CatalogCard({
    required this.packs,
    required this.selectedPackId,
    required this.refreshing,
    required this.onRefresh,
    required this.onSelect,
    required this.onListMarketplace,
    required this.onCachedMarketplace,
    required this.onGetPack,
    required this.onDownload,
    required this.onDelete,
    required this.onWatch,
    required this.onGetStickers,
    required this.onPreviewStickers,
    required this.onSelectPack,
    required this.onLastSelectedPack,
    required this.onListGallery,
    required this.onSyncRevocations,
  });

  final List<StickerPack> packs;
  final String? selectedPackId;
  final bool refreshing;
  final Future<void> Function() onRefresh;
  final ValueChanged<String?> onSelect;
  final VoidCallback onListMarketplace;
  final VoidCallback onCachedMarketplace;
  final VoidCallback? onGetPack;
  final VoidCallback? onDownload;
  final VoidCallback? onDelete;
  final VoidCallback? onWatch;
  final VoidCallback? onGetStickers;
  final VoidCallback? onPreviewStickers;
  final VoidCallback? onSelectPack;
  final VoidCallback onLastSelectedPack;
  final VoidCallback onListGallery;
  final VoidCallback onSyncRevocations;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          spacing: 10,
          children: <Widget>[
            Row(
              children: <Widget>[
                Expanded(
                  child: Text(
                    "Programmatic API",
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                ),
                TextButton.icon(
                  onPressed: refreshing ? null : () => onRefresh().ignore(),
                  icon: refreshing
                      ? const SizedBox.square(
                          dimension: 16,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Icon(Icons.refresh),
                  label: const Text("Refresh catalog"),
                ),
              ],
            ),
            if (packs.isEmpty)
              const Text("Catalog not loaded yet.")
            else
              DropdownButtonFormField<String>(
                key: ValueKey<String>(selectedPackId ?? "__none__"),
                initialValue: selectedPackId,
                isExpanded: true,
                decoration: const InputDecoration(
                  labelText: "Pack",
                  border: OutlineInputBorder(),
                ),
                items: packs
                    .map(
                      (StickerPack pack) => DropdownMenuItem<String>(
                        value: pack.id,
                        child: Text(
                          "${pack.name} · ${pack.stickerCount} stickers",
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                    )
                    .toList(),
                onChanged: onSelect,
              ),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: <Widget>[
                OutlinedButton(
                  onPressed: onListMarketplace,
                  child: const Text("listMarketplace"),
                ),
                OutlinedButton(
                  onPressed: onCachedMarketplace,
                  child: const Text("cachedMarketplace"),
                ),
                OutlinedButton(
                  onPressed: onListGallery,
                  child: const Text("listGallery"),
                ),
                OutlinedButton(
                  onPressed: onGetPack,
                  child: const Text("getPack"),
                ),
                OutlinedButton(
                  onPressed: onDownload,
                  child: const Text("downloadPack"),
                ),
                OutlinedButton(
                  onPressed: onDelete,
                  child: const Text("deletePack"),
                ),
                OutlinedButton(
                  onPressed: onWatch,
                  child: const Text("watchPack"),
                ),
                OutlinedButton(
                  onPressed: onGetStickers,
                  child: const Text("getStickers"),
                ),
                OutlinedButton(
                  onPressed: onPreviewStickers,
                  child: const Text("StickerGrid preview"),
                ),
                OutlinedButton(
                  onPressed: onSelectPack,
                  child: const Text("selectPack"),
                ),
                OutlinedButton(
                  onPressed: onLastSelectedPack,
                  child: const Text("lastSelectedPackId"),
                ),
                OutlinedButton(
                  onPressed: onSyncRevocations,
                  child: const Text("syncRevocations"),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _StateCard extends StatelessWidget {
  const _StateCard({
    required this.client,
    required this.selectedPack,
    required this.lastDownload,
  });

  final RestickClient client;
  final StickerPack? selectedPack;
  final PackDownloadState? lastDownload;

  @override
  Widget build(BuildContext context) {
    final RestickTheme theme =
        Theme.of(context).extension<RestickTheme>() ??
        RestickTheme.fromColorScheme(Theme.of(context).colorScheme);
    final PackDownloadState? download = lastDownload;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          spacing: 8,
          children: <Widget>[
            Text(
              "Widgets & theme",
              style: Theme.of(context).textTheme.titleMedium,
            ),
            Row(
              spacing: 12,
              children: <Widget>[
                if (selectedPack != null)
                  PackThumbnail(
                    url: selectedPack!.thumbnailUrl,
                    size: 56,
                    headers: <String, String>{
                      "X-Api-Key": client.config.apiKey,
                    },
                  ),
                Expanded(
                  child: download == null
                      ? const Text(
                          "Download events appear here (PackDownloadBanner).",
                        )
                      : Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          spacing: 4,
                          children: <Widget>[
                            Text(
                              "Latest download · ${download.packId}",
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: Theme.of(context).textTheme.bodySmall,
                            ),
                            PackDownloadBanner(state: download),
                            if (download.status ==
                                PackDownloadStatus.downloading)
                              LinearProgressIndicator(
                                value: download.progress,
                                minHeight: 3,
                              ),
                          ],
                        ),
                ),
              ],
            ),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: PackDownloadStatus.values
                  .map(
                    (PackDownloadStatus status) => Chip(
                      avatar: CircleAvatar(
                        backgroundColor: theme.colorFor(status),
                      ),
                      label: Text(status.name),
                    ),
                  )
                  .toList(),
            ),
            const Text(
              "RestickScope renders RestickLoadingBar above this screen; "
              "the cubits above feed it.",
              style: TextStyle(fontSize: 12),
            ),
          ],
        ),
      ),
    );
  }
}

class _FailuresCard extends StatelessWidget {
  const _FailuresCard({required this.client, required this.failures});

  final RestickClient client;
  final ValueListenable<List<String>> failures;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          spacing: 6,
          children: <Widget>[
            Text(
              "Failure reporting",
              style: Theme.of(context).textTheme.titleMedium,
            ),
            StreamBuilder<RestickErrorState>(
              stream: client.errorCubit.stream,
              initialData: client.errorCubit.state,
              builder:
                  (
                    BuildContext context,
                    AsyncSnapshot<RestickErrorState> snapshot,
                  ) {
                    final Failure? failure = snapshot.data?.failure;
                    return Row(
                      children: <Widget>[
                        Expanded(
                          child: Text(
                            failure == null
                                ? "RestickErrorCubit · no current failure."
                                : "RestickErrorCubit · "
                                      "${failure.runtimeType}: "
                                      "${failure.message}",
                            style: TextStyle(
                              fontSize: 12,
                              color: Theme.of(context).colorScheme.error,
                            ),
                          ),
                        ),
                        TextButton(
                          onPressed: failure == null
                              ? null
                              : client.errorCubit.clear,
                          child: const Text("Clear"),
                        ),
                      ],
                    );
                  },
            ),
            Text(
              "config.onError history",
              style: Theme.of(context).textTheme.labelLarge,
            ),
            ValueListenableBuilder<List<String>>(
              valueListenable: failures,
              builder:
                  (BuildContext context, List<String> lines, Widget? child) =>
                      lines.isEmpty
                      ? const Text(
                          "No failure reported yet. SDK failures (auth, "
                          "network, revoked packs) land here.",
                        )
                      : Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: lines
                              .map(
                                (String line) => Text(
                                  line,
                                  style: TextStyle(
                                    fontSize: 12,
                                    color: Theme.of(context).colorScheme.error,
                                  ),
                                ),
                              )
                              .toList(),
                        ),
            ),
          ],
        ),
      ),
    );
  }
}

class _EventLogCard extends StatelessWidget {
  const _EventLogCard({required this.events, required this.onClear});

  final List<String> events;
  final VoidCallback onClear;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          spacing: 8,
          children: <Widget>[
            Row(
              children: <Widget>[
                Expanded(
                  child: Text(
                    "Event log (${events.length})",
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                ),
                TextButton(
                  onPressed: events.isEmpty ? null : onClear,
                  child: const Text("Clear"),
                ),
              ],
            ),
            Container(
              height: 220,
              width: double.infinity,
              decoration: BoxDecoration(
                color: Theme.of(context).colorScheme.surfaceContainerHighest,
                borderRadius: BorderRadius.circular(8),
              ),
              padding: const EdgeInsets.all(8),
              child: events.isEmpty
                  ? const Center(
                      child: Text("Run an action to see its result."),
                    )
                  : ListView.builder(
                      itemCount: events.length,
                      itemBuilder: (BuildContext context, int index) => Text(
                        events[index],
                        style: const TextStyle(
                          fontFamily: "monospace",
                          fontSize: 12,
                        ),
                      ),
                    ),
            ),
          ],
        ),
      ),
    );
  }
}

class _KeyValue extends StatelessWidget {
  const _KeyValue({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      spacing: 8,
      children: <Widget>[
        SizedBox(
          width: 116,
          child: Text(
            label,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
              color: Theme.of(context).colorScheme.onSurfaceVariant,
            ),
          ),
        ),
        Expanded(
          child: Text(value, style: Theme.of(context).textTheme.bodySmall),
        ),
      ],
    );
  }
}
