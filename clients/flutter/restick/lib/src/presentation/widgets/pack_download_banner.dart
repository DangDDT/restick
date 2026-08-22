import "package:flutter/material.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/presentation/theme/restick_theme.dart";

class PackDownloadBanner extends StatelessWidget {
  const PackDownloadBanner({required this.state, super.key});

  final PackDownloadState state;

  @override
  Widget build(BuildContext context) {
    final RestickTheme theme =
        Theme.of(context).extension<RestickTheme>() ??
        RestickTheme.fromColorScheme(Theme.of(context).colorScheme);
    final String label = switch (state.status) {
      PackDownloadStatus.downloading =>
        "Downloading ${(state.progress * 100).round()}%",
      PackDownloadStatus.downloaded => "Downloaded",
      PackDownloadStatus.failed => "Download failed: ${state.message}",
      PackDownloadStatus.revoked => "Removed by server",
      PackDownloadStatus.remote => "Not downloaded",
    };
    return Text(
      label,
      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
            color: theme.colorFor(state.status),
          ),
    );
  }
}
