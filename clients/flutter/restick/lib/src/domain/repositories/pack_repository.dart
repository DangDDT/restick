import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_asset.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/domain/entities/tombstone.dart";

abstract class PackRepository {
  Future<List<StickerPack>> listMarketplace({String? tag});

  /// Last marketplace page fetched in this process, if any.
  Future<List<StickerPack>?> cachedMarketplace({String? tag});
  Future<StickerPack> getPack(String packId);
  Future<List<StickerPack>> listGallery();
  Future<void> downloadPack(String packId);
  Future<void> deletePack(String packId);
  Future<List<StickerAsset>> getStickers(String packId);
  Stream<PackDownloadState> watchPack(String packId);
  Stream<PackDownloadState> watchDownloads();
  Future<TombstoneSyncResult> syncRevocations();
  Future<void> saveSelectedPackId(String packId);
  Future<String?> readSelectedPackId();
  Future<void> dispose();
}
