import "package:restick_web/features/packs/domain/entities/pack.dart";

abstract class PackRepository {
  Future<({String id, String uploadPrefix})> createPack({
    required String name,
    required String slug,
    required String author,
    required List<String> tags,
    required String sourceKind,
  });

  Future<List<PackSummary>> listMine({bool forceRefresh = false});

  Future<List<PackSummary>> listQueue({bool forceRefresh = false});

  Future<List<PackSummary>> listPublished({bool forceRefresh = false});

  Future<PackDetail> getMine(String id, {bool forceRefresh = false});

  Future<PackDetail> getAdmin(String id, {bool forceRefresh = false});

  Future<void> submit(String id);

  Future<void> confirm(String id);

  Future<void> approve(String id);

  Future<void> reject({
    required String id,
    required String reasonCode,
    required String reasonText,
  });

  Future<void> revoke({
    required String id,
    required String reasonCode,
    required String reasonText,
  });

  Future<void> deleteRejected(String id);

  Future<void> ownerRevoke(String id);

  Future<List<RejectionStat>> getRejectionAnalytics({
    bool forceRefresh = false,
  });

  Future<PackDetail> updatePack({
    required String id,
    required String name,
    required String slug,
    required List<String> tags,
  });

  Future<PackDetail> updateStickersMetadata({
    required String id,
    required List<({String file, String emojiHint, List<String> keywords})> items,
  });

  Future<String> createAndUpload({
    required String name,
    required String slug,
    required String author,
    required List<String> tags,
    required String sourceKind,
    required List<UploadFile> files,
  });
}
