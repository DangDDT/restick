import "dart:async";
import "dart:convert";

import "package:restick_web/core/database/app_database.dart";
import "package:restick_web/core/error/dio_exception_mapper.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/features/packs/data/datasources/pack_api.dart";
import "package:restick_web/features/packs/data/datasources/upload_client.dart";
import "package:restick_web/features/packs/data/models/pack_dto.dart";
import "package:restick_web/features/packs/domain/entities/pack.dart";
import "package:restick_web/features/packs/domain/repositories/pack_repository.dart";

class PackRepositoryImpl implements PackRepository {
  PackRepositoryImpl({
    required PackApi packApi,
    required UploadClient uploadClient,
    required AppDatabase database,
  })  : _packApi = packApi,
        _uploadClient = uploadClient,
        _database = database;

  final PackApi _packApi;
  final UploadClient _uploadClient;
  final AppDatabase _database;

  @override
  Future<({String id, String uploadPrefix})> createPack({
    required String name,
    required String slug,
    required String author,
    required List<String> tags,
    required String sourceKind,
  }) async {
    try {
      final CreatePackResponseDto response = await _packApi.createPack(
        <String, dynamic>{
          "name": name,
          "slug": slug,
          "author": author,
          "tags": tags,
          "source_kind": sourceKind,
        },
      );
      return (id: response.id, uploadPrefix: response.uploadPrefix);
    } on Object catch (error) {
      throw mapDioException(error);
    }
  }

  @override
  Future<List<PackSummary>> listMine({bool forceRefresh = false}) {
    return _listCached(
      cacheKey: "mine",
      forceRefresh: forceRefresh,
      fetch: _packApi.listMine,
    );
  }

  @override
  Future<List<PackSummary>> listQueue({bool forceRefresh = false}) {
    return _listCached(
      cacheKey: "queue",
      forceRefresh: forceRefresh,
      fetch: _packApi.listQueue,
    );
  }

  @override
  Future<List<PackSummary>> listPublished({bool forceRefresh = false}) {
    return _listCached(
      cacheKey: "published",
      forceRefresh: forceRefresh,
      fetch: _packApi.listPublished,
    );
  }

  Future<List<PackSummary>> _listCached({
    required String cacheKey,
    required bool forceRefresh,
    required Future<PackListResponseDto> Function() fetch,
  }) async {
    if (!forceRefresh) {
      final String? cached = await _database.readPackList(cacheKey);
      if (cached != null) {
        final List<PackSummary> stale = PackListResponseDto.fromJson(
          <String, dynamic>{"items": decodeJsonList(cached)},
        ).items.map((PackSummaryDto dto) => dto.toEntity()).toList();
        unawaited(_refreshList(cacheKey, fetch));
        return stale;
      }
    }
    return _refreshList(cacheKey, fetch);
  }

  Future<List<PackSummary>> _refreshList(
    String cacheKey,
    Future<PackListResponseDto> Function() fetch,
  ) async {
    try {
      final PackListResponseDto response = await fetch();
      await _database.writePackList(
        cacheKey,
        jsonEncode(
          response.items.map((PackSummaryDto dto) => dto.toJson()).toList(),
        ),
      );
      return response.items.map((PackSummaryDto dto) => dto.toEntity()).toList();
    } on Object catch (error) {
      final String? cached = await _database.readPackList(cacheKey);
      if (cached != null) {
        return PackListResponseDto.fromJson(
          <String, dynamic>{"items": decodeJsonList(cached)},
        ).items.map((PackSummaryDto dto) => dto.toEntity()).toList();
      }
      throw mapDioException(error);
    }
  }

  @override
  Future<PackDetail> getMine(String id, {bool forceRefresh = false}) {
    return _getDetailCached(
      id: id,
      forceRefresh: forceRefresh,
      fetch: () => _packApi.getMine(id),
    );
  }

  @override
  Future<PackDetail> getAdmin(String id, {bool forceRefresh = false}) {
    return _getDetailCached(
      id: id,
      forceRefresh: forceRefresh,
      fetch: () => _packApi.getAdmin(id),
    );
  }

  Future<PackDetail> _getDetailCached({
    required String id,
    required bool forceRefresh,
    required Future<PackDetailDto> Function() fetch,
  }) async {
    if (!forceRefresh) {
      final String? cached = await _database.readPackDetail(id);
      if (cached != null) {
        final PackDetail stale =
            PackDetailDto.fromJson(decodeJsonMap(cached)).toEntity();
        unawaited(_refreshDetail(id, fetch));
        return stale;
      }
    }
    return _refreshDetail(id, fetch);
  }

  Future<PackDetail> _refreshDetail(
    String id,
    Future<PackDetailDto> Function() fetch,
  ) async {
    try {
      final PackDetailDto dto = await fetch();
      await _database.writePackDetail(id, jsonEncode(dto.toJson()));
      return dto.toEntity();
    } on Object catch (error) {
      final String? cached = await _database.readPackDetail(id);
      if (cached != null) {
        return PackDetailDto.fromJson(decodeJsonMap(cached)).toEntity();
      }
      throw mapDioException(error);
    }
  }

  @override
  Future<void> submit(String id) async {
    try {
      await _packApi.submit(id);
    } on Object catch (error) {
      throw mapDioException(error);
    }
  }

  @override
  Future<void> confirm(String id) async {
    try {
      await _packApi.confirm(id);
    } on Object catch (error) {
      throw mapDioException(error);
    }
  }

  @override
  Future<void> approve(String id) async {
    try {
      await _packApi.approve(id);
    } on Object catch (error) {
      throw mapDioException(error);
    }
  }

  @override
  Future<void> reject({
    required String id,
    required String reasonCode,
    required String reasonText,
  }) async {
    try {
      await _packApi.reject(
        id,
        <String, dynamic>{
          "reason_code": reasonCode,
          "rejection_code": reasonCode,
          "reason_text": reasonText,
        },
      );
    } on Object catch (error) {
      throw mapDioException(error);
    }
  }

  @override
  Future<void> revoke({
    required String id,
    required String reasonCode,
    required String reasonText,
  }) async {
    try {
      await _packApi.revoke(
        id,
        <String, dynamic>{
          "reason_code": reasonCode,
          "revocation_code": reasonCode,
          "reason_text": reasonText,
        },
      );
    } on Object catch (error) {
      throw mapDioException(error);
    }
  }

  @override
  Future<void> deleteRejected(String id) async {
    try {
      await _packApi.deleteRejected(id);
    } on Object catch (error) {
      throw mapDioException(error);
    }
  }

  @override
  Future<void> ownerRevoke(String id) async {
    try {
      await _packApi.ownerRevoke(id);
    } on Object catch (error) {
      throw mapDioException(error);
    }
  }

  @override
  Future<List<RejectionStat>> getRejectionAnalytics({
    bool forceRefresh = false,
  }) async {
    if (!forceRefresh) {
      final String? cached = await _database.readRejectionStats();
      if (cached != null) {
        final List<RejectionStat> stale = RejectionAnalyticsResponseDto.fromJson(
          <String, dynamic>{"analytics": decodeJsonList(cached)},
        ).analytics.map((RejectionStatDto dto) => dto.toEntity()).toList();
        unawaited(_refreshAnalytics());
        return stale;
      }
    }
    return _refreshAnalytics();
  }

  Future<List<RejectionStat>> _refreshAnalytics() async {
    try {
      final RejectionAnalyticsResponseDto response =
          await _packApi.getRejectionAnalytics();
      await _database.writeRejectionStats(
        jsonEncode(
          response.analytics
              .map((RejectionStatDto dto) => dto.toJson())
              .toList(),
        ),
      );
      return response.analytics
          .map((RejectionStatDto dto) => dto.toEntity())
          .toList();
    } on Object catch (error) {
      final String? cached = await _database.readRejectionStats();
      if (cached != null) {
        return RejectionAnalyticsResponseDto.fromJson(
          <String, dynamic>{"analytics": decodeJsonList(cached)},
        ).analytics.map((RejectionStatDto dto) => dto.toEntity()).toList();
      }
      throw mapDioException(error);
    }
  }

  @override
  Future<PackDetail> updatePack({
    required String id,
    required String name,
    required String slug,
    required List<String> tags,
  }) async {
    try {
      final PackDetailDto dto = await _packApi.updatePack(
        id,
        <String, dynamic>{
          "name": name,
          "slug": slug,
          "tags": tags,
        },
      );
      await _database.writePackDetail(id, jsonEncode(dto.toJson()));
      return dto.toEntity();
    } on Object catch (error) {
      throw mapDioException(error);
    }
  }

  @override
  Future<PackDetail> updateStickersMetadata({
    required String id,
    required List<({String file, String emojiHint, List<String> keywords})> items,
  }) async {
    try {
      final PackDetailDto dto = await _packApi.updateStickersMetadata(
        id,
        <String, dynamic>{
          "stickers": items
              .map(
                (item) => <String, dynamic>{
                  "file": item.file,
                  "emoji_hint": item.emojiHint,
                  "keywords": item.keywords,
                },
              )
              .toList(),
        },
      );
      await _database.writePackDetail(id, jsonEncode(dto.toJson()));
      return dto.toEntity();
    } on Object catch (error) {
      throw mapDioException(error);
    }
  }

  @override
  Future<String> createAndUpload({
    required String name,
    required String slug,
    required String author,
    required List<String> tags,
    required String sourceKind,
    required List<UploadFile> files,
  }) async {
    if (files.isEmpty) {
      throw const UnexpectedFailure("Choose a sheet or files first.");
    }
    final ({String id, String uploadPrefix}) created = await createPack(
      name: name,
      slug: slug,
      author: author,
      tags: tags,
      sourceKind: sourceKind,
    );
    for (final UploadFile file in files) {
      await _uploadClient.putObject(
        key: "${created.uploadPrefix}${file.name}",
        bytes: file.bytes,
        contentType: file.contentType,
      );
    }
    await submit(created.id);
    return created.id;
  }
}
