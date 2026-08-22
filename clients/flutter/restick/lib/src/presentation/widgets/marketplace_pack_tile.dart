import "package:auto_size_text/auto_size_text.dart";
import "package:flutter/material.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/presentation/widgets/pack_download_banner.dart";
import "package:restick/src/presentation/widgets/pack_thumbnail.dart";

class MarketplacePackTile extends StatelessWidget {
  const MarketplacePackTile({
    required this.pack,
    required this.onDownload,
    required this.onDelete,
    super.key,
  });

  final StickerPack pack;
  final VoidCallback onDownload;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final PackDownloadState download = PackDownloadState(
      packId: pack.id,
      status: pack.downloadStatus,
      progress: pack.downloadProgress,
    );
    return ListTile(
      leading: PackThumbnail(url: pack.thumbnailUrl),
      title: AutoSizeText(pack.name, maxLines: 1),
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        spacing: 4,
        children: <Widget>[
          Text(pack.author, style: Theme.of(context).textTheme.bodySmall),
          PackDownloadBanner(state: download),
          if (pack.downloadStatus == PackDownloadStatus.downloading)
            LinearProgressIndicator(value: pack.downloadProgress, minHeight: 2),
        ],
      ),
      isThreeLine: true,
      trailing: _MarketplacePackActions(
        pack: pack,
        onDownload: onDownload,
        onDelete: onDelete,
      ),
    );
  }
}

class _MarketplacePackActions extends StatelessWidget {
  const _MarketplacePackActions({
    required this.pack,
    required this.onDownload,
    required this.onDelete,
  });

  final StickerPack pack;
  final VoidCallback onDownload;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    if (pack.downloadStatus == PackDownloadStatus.downloading) {
      return const SizedBox.square(
        dimension: 24,
        child: CircularProgressIndicator(strokeWidth: 2),
      );
    }
    if (pack.isDownloaded) {
      return TextButton(onPressed: onDelete, child: const Text("Remove"));
    }
    return TextButton(onPressed: onDownload, child: const Text("Download"));
  }
}
