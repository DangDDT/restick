import "dart:async";
import "dart:convert";
import "dart:io";
import "dart:typed_data";
import "dart:ui" as ui;

import "package:dio/dio.dart";
import "package:dio/io.dart";
import "package:drift/native.dart";
import "package:flutter/painting.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:flutter_test/flutter_test.dart";
import "package:material_ui/material_ui.dart";
import "package:restick/src/core/error/restick_error_cubit.dart";
import "package:restick/src/core/restick_client.dart";
import "package:restick/src/core/storage/prefs_storage.dart";
import "package:restick/src/data/datasources/restick_api.dart";
import "package:restick/src/data/dto/pack_dto.dart";
import "package:restick/src/data/local/app_database.dart";
import "package:restick/src/data/local/pack_downloader.dart";
import "package:restick/src/data/local/pack_file_store.dart";
import "package:restick/src/data/repositories/pack_repository_impl.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/domain/entities/sticker_preview.dart";
import "package:restick/src/presentation/cubit/marketplace_cubit.dart";
import "package:restick/src/presentation/scope/restick_scope.dart";
import "package:restick/src/presentation/theme/restick_strings.dart";
import "package:restick/src/presentation/theme/restick_theme.dart";
import "package:restick/src/presentation/widgets/download_button.dart";
import "package:restick/src/presentation/widgets/pack_hero_carousel.dart";
import "package:restick/src/presentation/widgets/restick_mascot.dart";
import "package:restick/src/presentation/widgets/sticker_marketplace.dart";
import "package:shared_preferences/shared_preferences.dart";

import "../example/lib/src/home_shell.dart";
import "marketplace_fakes.dart";

class _RefreshRepository extends FakeMarketplaceRepository {
  _RefreshRepository(List<StickerPack> packs) : super(remote: packs);
  List<StickerPack>? cached;
  Completer<List<StickerPack>>? result;
  @override
  Future<List<StickerPack>?> cachedMarketplace({String? tag}) async => cached;
  @override
  Future<List<StickerPack>> listMarketplace({String? tag}) =>
      result?.future ?? super.listMarketplace(tag: tag);
}

// Capture the real NetworkImage HTTP boundary, without a server or query keys.
class _Headers implements HttpHeaders {
  final Map<String, String> values = <String, String>{};
  @override
  void add(String name, Object value, {bool preserveHeaderCase = false}) {
    values[name.toLowerCase()] = "$value";
  }
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}
class _ImageResponse extends Stream<List<int>> implements HttpClientResponse {
  static final Uint8List bytes = base64Decode(
    "UklGRh4AAABXRUJQVlA4TBEAAAAvAUAAAAdQ5KJUq/+BiOh/AAA=",
  );
  @override
  int get statusCode => 200;
  @override
  int get contentLength => bytes.length;
  @override
  HttpClientResponseCompressionState get compressionState =>
      HttpClientResponseCompressionState.notCompressed;
  @override
  StreamSubscription<List<int>> listen(void Function(List<int>)? onData,
      {Function? onError, void Function()? onDone, bool? cancelOnError}) =>
      Stream<List<int>>.value(bytes).listen(onData, onError: onError,
          onDone: onDone, cancelOnError: cancelOnError);
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}
class _ImageRequest implements HttpClientRequest {
  _ImageRequest(this.url, this.requests);
  final Uri url;
  final List<(Uri, Map<String, String>)> requests;
  @override
  final _Headers headers = _Headers();
  @override
  Future<HttpClientResponse> close() async {
    requests.add((url, Map<String, String>.of(headers.values)));
    return _ImageResponse();
  }
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}
class _ImageClient implements HttpClient {
  final List<(Uri, Map<String, String>)> requests = <(Uri, Map<String, String>)>[];
  @override
  Future<HttpClientRequest> getUrl(Uri url) async => _ImageRequest(url, requests);
  @override
  set autoUncompress(bool value) {}
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

Future<void> _mount(WidgetTester tester, RestickClient client, {
  bool scoped = false,
  bool reduce = true,
  RestickTheme? theme,
}) async {
  tester.view.physicalSize = const Size(400, 1000);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  final Widget marketplace = StickerMarketplace(
    client: scoped ? null : client,
    strings: RestickStrings.en,
  );
  await tester.pumpWidget(MaterialApp(
    theme: ThemeData(extensions: <ThemeExtension<dynamic>>[if (theme != null) theme]),
    builder: (BuildContext context, Widget? child) => MediaQuery(
      data: MediaQuery.of(context).copyWith(disableAnimations: reduce), child: child!),
    home: Scaffold(body: scoped
        ? RestickScope(client: client, showLoadingBar: false, child: marketplace)
        : marketplace),
  ));
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 600));
}
MarketplaceCubit _cubit(WidgetTester tester) =>
    tester.element(find.byType(PackHeroCarousel)).read<MarketplaceCubit>();
Future<void> _emit(WidgetTester tester, FakeMarketplaceRepository repo,
    PackDownloadStatus status, {double progress = 0, String message = ""}) async {
  repo.downloads.add(PackDownloadState(packId: "p0", status: status,
      progress: progress, message: message));
  await tester.pump();
}

Future<void> _filterP0(WidgetTester tester) async {
  await tester.tap(find.byKey(const ValueKey<String>("marketplace.search.open")));
  await tester.pump();
  await tester.enterText(find.byKey(const ValueKey<String>("marketplace.search")), "p0");
  await tester.pump();
}

class _Api extends RestickApi {
  _Api() : super(Dio());
  static const PackSummaryDto pack = PackSummaryDto(
    id: "p0", slug: "p0", name: "Pack p0", author: "Author", tags: <String>[],
    stickerCount: 3, version: 1, thumbnailUrl: "", checksumSha256: "",
    previews: <StickerPreviewDto>[
      StickerPreviewDto(thumbUrl: "", fileUrl: ""),
      StickerPreviewDto(thumbUrl: "", fileUrl: ""),
      StickerPreviewDto(thumbUrl: "", fileUrl: ""),
    ],
  );
  @override
  Future<PackListResponseDto> listPacks({String? tag}) async =>
      const PackListResponseDto(items: <PackSummaryDto>[pack]);
  @override
  Future<PackSummaryDto> getPack(String id) async => pack;
  @override
  Future<BundleMetaDto> getBundle(String id) async => const BundleMetaDto(
    url: "https://objects.test/bundle.zip", checksumSha256: "checksum", version: 1,
  );
}
class _FailingDownload extends DioForNative {
  Completer<void> reached = Completer<void>();
  Completer<void> fail = Completer<void>();
  Completer<void>? allowProgress;
  @override
  Future<Response<T>> get<T>(String path, {
    Map<String, dynamic>? queryParameters, Options? options,
    Object? data,
    CancelToken? cancelToken, ProgressCallback? onReceiveProgress,
  }) async {
    if (allowProgress != null) await allowProgress!.future;
    onReceiveProgress!(67, 100);
    reached.complete();
    await fail.future;
    throw DioException(requestOptions: RequestOptions(path: path),
        type: DioExceptionType.connectionError, message: "offline");
  }
}

void main() {
  for (final bool scoped in <bool>[true, false]) {
    testWidgets("image HTTP key reaches cards and sheet scoped=$scoped", (tester) async {
      final _ImageClient http = _ImageClient();
      debugNetworkImageHttpClientProvider = () => http;
      addTearDown(() { debugNetworkImageHttpClientProvider = null; imageCache.clear(); });
      final StickerPack pack = fakePack("p0").copyWith(previews: const <StickerPreview>[
        StickerPreview(thumbUrl: "https://objects.test/thumb.webp", fileUrl: "https://objects.test/full.webp"),
      ]);
      final RestickClient client = fakeRestickClient(FakeMarketplaceRepository(remote: <StickerPack>[pack]));
      addTearDown(client.dispose);
      await _mount(tester, client, scoped: scoped);
      await tester.runAsync(() => Future<void>.delayed(const Duration(milliseconds: 20)));
      await tester.pump();
      expect(http.requests.where((r) => r.$1.path == "/full.webp"), isNotEmpty);
      await tester.tap(find.text("Pack p0"));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 500));
      await tester.runAsync(() => Future<void>.delayed(const Duration(milliseconds: 20)));
      await tester.pump();
      expect(http.requests.where((r) => r.$1.path == "/thumb.webp"), isNotEmpty);
      for (final request in http.requests) {
        expect(request.$2["x-api-key"], "test");
        expect(request.$1.query, isEmpty);
      }
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets("cached and network refresh keep busy cards and sheet without progress", (tester) async {
    final _RefreshRepository repo = _RefreshRepository(<StickerPack>[fakePack("p0", tags: <String>["tag"])]);
    final RestickClient client = fakeRestickClient(repo);
    addTearDown(client.dispose);
    await _mount(tester, client);
    final MarketplaceCubit cubit = _cubit(tester);
    await _filterP0(tester);
    await _emit(tester, repo, PackDownloadStatus.downloading, progress: 0.67);
    await tester.tap(find.text("Pack p0"));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    repo.cached = <StickerPack>[fakePack("p0")];
    repo.result = Completer<List<StickerPack>>();
    final Future<void> refresh = cubit.load();
    await tester.pump();
    void expectBusy() {
      for (final PackDownloadButton b in tester.widgetList<PackDownloadButton>(find.byType(PackDownloadButton))) {
        expect(b.pack.downloadStatus, PackDownloadStatus.downloading);
        expect(b.pack.downloadProgress, 0.67);
      }
      expect(find.byKey(const ValueKey<String>("marketplace.pack.remove.p0")), findsNothing);
    }
    expectBusy();
    repo.result!.complete(<StickerPack>[fakePack("p0", name: "Fresh p0")]);
    await refresh;
    await tester.pump();
    expectBusy();
    await tester.tap(find.byKey(const ValueKey<String>("marketplace.pack.download.p0")).last);
    expect(repo.downloadCalls, isEmpty);
    await _emit(tester, repo, PackDownloadStatus.failed, progress: 0.67, message: "Connection lost");
    expect(find.text("Connection lost"), findsWidgets);
    await _emit(tester, repo, PackDownloadStatus.downloading);
    expect(find.text("Connection lost"), findsNothing);
    await _emit(tester, repo, PackDownloadStatus.downloaded, progress: 1);
    expect(find.text(RestickStrings.en.readyInCollection), findsWidgets);
    repo.cached = null;
    repo.result = Completer<List<StickerPack>>()..complete(<StickerPack>[]);
    await cubit.load();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    expect(cubit.state.packs, isEmpty);
    expect(find.byKey(const ValueKey<String>("restick.sheet")), findsNothing);
  });

  testWidgets("real receive progress failure keeps coloured slots and reason then resets retry", (tester) async {
    SharedPreferences.setMockInitialValues(<String, Object>{});
    final SharedPreferences prefs = await SharedPreferences.getInstance();
    final RestickDatabase database = await RestickDatabase.open(NativeDatabase.memory());
    final _FailingDownload dio = _FailingDownload();
    final RestickErrorCubit errors = RestickErrorCubit();
    final PackRepositoryImpl repo = PackRepositoryImpl(api: _Api(), downloadDio: dio,
        database: database, fileStore: PackFileStore(), downloader: PackDownloader(),
        prefs: RestickPrefsStorage(preferences: prefs), apiBase: Uri.parse("https://api.test"), errorCubit: errors);
    final RestickClient client = fakeRestickClient(repo);
    addTearDown(client.dispose);
    addTearDown(errors.close);
    await tester.runAsync(() => repo.listMarketplace());
    await _mount(tester, client);
    final MarketplaceCubit cubit = _cubit(tester);
    await _filterP0(tester);
    await tester.tap(find.text("Pack p0"));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    final Future<void> download = cubit.download("p0");
    await tester.runAsync(() => dio.reached.future);
    await tester.pump();
    dio.fail.complete();
    await tester.runAsync(() => download);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 600));
    expect(cubit.state.packs.single.downloadProgress, 0.67);
    expect(cubit.state.packs.single.downloadStatus, PackDownloadStatus.failed);
    expect(find.byKey(const ValueKey<String>("restick.sheet.sticker.grey.0")), findsNothing);
    expect(find.byKey(const ValueKey<String>("restick.sheet.sticker.grey.1")), findsNothing);
    expect(find.byKey(const ValueKey<String>("restick.sheet.sticker.grey.2")), findsOneWidget);
    expect(find.byKey(const ValueKey<String>("restick.card.strip.grey.0")), findsNothing);
    expect(find.byKey(const ValueKey<String>("restick.card.strip.grey.1")), findsNothing);
    expect(find.byKey(const ValueKey<String>("restick.card.strip.grey.2")), findsOneWidget);
    final String reason = cubit.state.downloads["p0"]!.message;
    expect(reason, isNotEmpty);
    expect(find.text(reason), findsWidgets);
    dio.reached = Completer<void>();
    dio.fail = Completer<void>();
    dio.allowProgress = Completer<void>();
    final Future<void> retry = cubit.download("p0");
    await tester.pump();
    expect(cubit.state.downloads["p0"]!.message, isEmpty);
    expect(cubit.state.packs.single.downloadProgress, 0);
    dio.allowProgress!.complete();
    await tester.runAsync(() => dio.reached.future);
    dio.fail.complete();
    await tester.runAsync(() => retry);
  });

  testWidgets("theme tokens and accessible tag targets reach modal consumers", (tester) async {
    final RestickTheme tokens = RestickTheme.fromColorScheme(ThemeData().colorScheme).copyWith(
        brand: const Color(0xFF006600), brandSoft: const Color(0xFF00FFFF),
        ink: const Color(0xFF990000), canvas: const Color(0xFF1122FF));
    final RestickClient client = fakeRestickClient(FakeMarketplaceRepository(remote: <StickerPack>[fakePack("p0", tags: <String>["tag"])]));
    addTearDown(client.dispose);
    await _mount(tester, client, scoped: true, theme: tokens);
    expect(tester.widgetList<ColoredBox>(find.byType(ColoredBox)).any((w) => w.color == tokens.canvas), isTrue);
    final Finder chip = find.byKey(const ValueKey<String>("marketplace.chip.tag"));
    expect(tester.getSize(chip).height, greaterThanOrEqualTo(48));
    expect(tester.getSize(chip).width, greaterThanOrEqualTo(48));
    expect(tester.widget<Text>(find.text("Pack p0")).style!.color, tokens.ink);
    await tester.tap(find.text("Pack p0"));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    final BuildContext sheet = tester.element(find.byKey(const ValueKey<String>("restick.sheet")));
    expect(Theme.of(sheet).extension<RestickTheme>(), tokens);
    expect(tester.widgetList<Text>(find.text("Pack p0")).every((w) => w.style!.color == tokens.ink), isTrue);
    final List<DecoratedBox> artwork = tester.widgetList<DecoratedBox>(find.byType(DecoratedBox)).toList();
    expect(artwork.any((w) => w.decoration is BoxDecoration && (w.decoration as BoxDecoration).color == tokens.brandSoft), isTrue);
    final CustomPaint mascot = tester.widget<CustomPaint>(find.descendant(
      of: find.byType(RestickMascot).last, matching: find.byType(CustomPaint)).first);
    final ui.PictureRecorder recorder = ui.PictureRecorder();
    mascot.painter!.paint(Canvas(recorder), const Size.square(64));
    final ui.Picture picture = recorder.endRecording();
    final ui.Image image = (await tester.runAsync(() => picture.toImage(64, 64)))!;
    final ByteData pixels = (await tester.runAsync(() => image.toByteData()))!;
    Color pixel(int x, int y) {
      final int offset = (y * 64 + x) * 4;
      return Color.fromARGB(pixels.getUint8(offset + 3), pixels.getUint8(offset),
          pixels.getUint8(offset + 1), pixels.getUint8(offset + 2));
    }
    expect(pixel(8, 16), tokens.brand);
    expect(pixel(15, 16), tokens.brandSoft);
    expect(pixel(32, 40), tokens.ink);
    image.dispose();
    picture.dispose();
  });

  for (final bool reduce in <bool>[false, true]) {
    for (final bool compact in <bool>[false, true]) {
    testWidgets("completion stamps and cheers once reduced=$reduce compact=$compact", (tester) async {
      final _RefreshRepository repo = _RefreshRepository(<StickerPack>[fakePack("p0")]);
      final RestickClient client = fakeRestickClient(repo);
      addTearDown(client.dispose);
      await _mount(tester, client, reduce: reduce);
      final MarketplaceCubit cubit = _cubit(tester);
      if (compact) await _filterP0(tester);
      expect(find.byWidgetPredicate((w) => w is RestickMascot && w.mood == RestickMood.cheer), findsNothing);
      await _emit(tester, repo, PackDownloadStatus.downloading, progress: 0.67);
      await _emit(tester, repo, PackDownloadStatus.downloaded, progress: 1);
      await tester.pump(const Duration(milliseconds: 100));
      expect(find.text(RestickStrings.en.readyInCollection), findsWidgets);
      expect(find.byKey(const ValueKey<String>("restick.download.burst")), reduce ? findsNothing : findsOneWidget);
      expect(find.byWidgetPredicate((w) => w is RestickMascot && w.mood == RestickMood.cheer), reduce ? findsNothing : findsOneWidget);
      final Finder stamp = find.byKey(const ValueKey<String>("restick.card.stamp.p0"));
      expect(stamp, findsOneWidget);
      if (!reduce) {
        expect(tester.widget<Transform>(stamp).transform.storage[0], isNot(1));
      }
      if (reduce) expect(tester.widget<Transform>(stamp).transform.storage[0], 1);
      await tester.pump(const Duration(seconds: 2));
      expect(find.byWidgetPredicate((w) => w is RestickMascot && w.mood == RestickMood.cheer), findsNothing);
      repo.remote = <StickerPack>[fakePack("p0", status: PackDownloadStatus.downloaded)];
      await cubit.load();
      await tester.pump();
      expect(tester.widget<Transform>(find.byKey(const ValueKey<String>("restick.card.stamp.p0"))).transform.storage[0], 1);
      expect(find.byWidgetPredicate((w) => w is RestickMascot && w.mood == RestickMood.cheer), findsNothing);
    });
    }
  }

  testWidgets("owned mount stays still and swipe frames retain card content", (tester) async {
    final FakeMarketplaceRepository repo = FakeMarketplaceRepository(remote: <StickerPack>[
      fakePack("p0", status: PackDownloadStatus.downloaded), fakePack("p1"),
    ]);
    final RestickClient client = fakeRestickClient(repo);
    addTearDown(client.dispose);
    await _mount(tester, client, reduce: false);
    final Finder stamp = find.byKey(const ValueKey<String>("restick.card.stamp.p0"));
    expect(tester.widget<Transform>(stamp).transform.storage[0], 1);
    expect(find.byWidgetPredicate((w) => w is RestickMascot && w.mood == RestickMood.cheer), findsNothing);
    final Text title = tester.widget<Text>(find.text("Pack p0"));
    final TestGesture drag = await tester.startGesture(tester.getCenter(find.byType(PageView)));
    await drag.moveBy(const Offset(-20, 0));
    await tester.pump(const Duration(milliseconds: 16));
    await drag.moveBy(const Offset(-20, 0));
    await tester.pump(const Duration(milliseconds: 16));
    expect(tester.widget<Text>(find.text("Pack p0")), same(title));
    await drag.up();
    await tester.pump(const Duration(seconds: 1));
  });

  testWidgets("example hidden tab pauses idle motion and resumes retained state", (tester) async {
    tester.view.physicalSize = const Size(400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final FakeMarketplaceRepository repo = FakeMarketplaceRepository(remote: <StickerPack>[fakePack("p0")]);
    final RestickClient client = fakeRestickClient(repo);
    addTearDown(client.dispose);
    await tester.pumpWidget(MaterialApp(home: RestickScope(client: client, showLoadingBar: false,
      child: HomeShell(client: client, themeMode: ThemeMode.light,
        onThemeModeChanged: (_) {}, onDisconnect: () async {},
        failures: ValueNotifier<List<String>>(<String>[])))));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 600));
    final State carousel = tester.state(find.byType(PackHeroCarousel));
    final MarketplaceCubit cubit = _cubit(tester);
    await _emit(tester, repo, PackDownloadStatus.downloading, progress: 0.42);
    await tester.tap(find.text("Gallery"));
    await tester.pump();
    await tester.pump(const Duration(seconds: 1));
    final Finder floats = find.byKey(const ValueKey<String>("restick.hero.float"), skipOffstage: false);
    final Matrix4 before = tester.widget<Transform>(floats).transform.clone();
    await tester.pump(const Duration(seconds: 1));
    expect(tester.widget<Transform>(floats).transform, before);
    await tester.tap(find.text("Marketplace"));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));
    expect(tester.state(find.byType(PackHeroCarousel)), same(carousel));
    expect(_cubit(tester), same(cubit));
    expect(cubit.state.packs.single.downloadProgress, 0.42);
    expect(tester.widget<Transform>(floats).transform, isNot(before));
  });
}
