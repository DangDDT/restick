import "dart:async";

import "package:flutter/scheduler.dart";
import "package:flutter_test/flutter_test.dart";
import "package:material_ui/material_ui.dart";
import "package:restick/src/core/error/failure.dart";
import "package:restick/src/core/restick_client.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/presentation/theme/restick_strings.dart";
import "package:restick/src/presentation/widgets/die_cut_sticker.dart";
import "package:restick/src/presentation/widgets/pack_compact_card.dart";
import "package:restick/src/presentation/widgets/pack_hero_carousel.dart";
import "package:restick/src/presentation/widgets/pack_skeleton.dart";
import "package:restick/src/presentation/widgets/restick_mascot.dart";
import "package:restick/src/presentation/widgets/sticker_marketplace.dart";

import "marketplace_fakes.dart";

const RestickStrings _vi = RestickStrings.vi;

Finder _greyStrip(int i) =>
    find.byKey(ValueKey<String>("restick.card.strip.grey.$i"));

Future<FakeMarketplaceRepository> _pump(
  WidgetTester tester, {
  List<StickerPack> packs = const <StickerPack>[],
  Failure? failure,
  bool hang = false,
  RestickStrings strings = _vi,
  double textScale = 1,
  bool disableAnimations = true,
  ValueNotifier<bool>? tickers,
  Size size = const Size(400, 1000),
  Duration settle = const Duration(milliseconds: 600),
}) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  final FakeMarketplaceRepository repository = FakeMarketplaceRepository(
    remote: packs,
    failure: failure,
    hang: hang,
  );
  final RestickClient client = fakeRestickClient(repository);
  addTearDown(client.dispose);
  final ValueNotifier<bool> ticking = tickers ?? ValueNotifier<bool>(true);
  await tester.pumpWidget(
    MaterialApp(
      home: Builder(
        builder:
            (BuildContext context) => MediaQuery(
              data: MediaQuery.of(context).copyWith(
                textScaler: TextScaler.linear(textScale),
                disableAnimations: disableAnimations,
              ),
              child: ValueListenableBuilder<bool>(
                valueListenable: ticking,
                builder:
                    (BuildContext context, bool enabled, Widget? child) =>
                        TickerMode(enabled: enabled, child: child!),
                child: Scaffold(
                  body: StickerMarketplace(client: client, strings: strings),
                ),
              ),
            ),
      ),
    ),
  );
  await tester.pump();
  await tester.pump(settle);
  return repository;
}

List<StickerPack> _packs(int count) =>
    List<StickerPack>.generate(count, (int i) => fakePack("p$i"));

void main() {
  testWidgets("shows skeleton while loading and no spinner", (
    WidgetTester tester,
  ) async {
    await _pump(tester, hang: true);
    expect(find.byType(PackSkeleton), findsWidgets);
    expect(find.byType(CircularProgressIndicator), findsNothing);
    expect(find.text(_vi.emptyTitle), findsNothing);
  });

  testWidgets("empty list shows empty state with mascot", (
    WidgetTester tester,
  ) async {
    await _pump(tester);
    expect(find.text(_vi.emptyTitle), findsOneWidget);
    expect(find.byType(RestickMascot), findsWidgets);
    expect(find.byType(PackSkeleton), findsNothing);
  });

  testWidgets("failure with no packs shows error state and retry", (
    WidgetTester tester,
  ) async {
    final FakeMarketplaceRepository repository = await _pump(
      tester,
      failure: const NetworkFailure(),
    );
    expect(find.text(_vi.errorTitle), findsOneWidget);
    expect(
      find.byWidgetPredicate(
        (Widget w) => w is RestickMascot && w.mood == RestickMood.sad,
      ),
      findsOneWidget,
    );
    expect(repository.listCalls, 1);
    await tester.tap(find.byKey(const ValueKey<String>("marketplace.retry")));
    await tester.pump();
    expect(repository.listCalls, 2);
  });

  testWidgets("renders carousel for first three packs and compact cards for "
      "the rest", (WidgetTester tester) async {
    await _pump(tester, packs: _packs(5));
    final Finder carousel = find.byType(PackHeroCarousel);
    expect(carousel, findsOneWidget);
    expect(
      tester
          .widget<PackHeroCarousel>(carousel)
          .packs
          .map((StickerPack p) => p.id),
      <String>["p0", "p1", "p2"],
    );
    for (int i = 0; i < 3; i++) {
      expect(
        find.byKey(ValueKey<String>("restick.hero.dot.$i")),
        findsOneWidget,
      );
    }
    expect(
      find.descendant(of: carousel, matching: find.text("Pack p0")),
      findsOneWidget,
    );
    expect(find.text(_vi.newShelf), findsOneWidget);
    final Iterable<String> cards = tester
        .widgetList<PackCompactCard>(find.byType(PackCompactCard))
        .map((PackCompactCard c) => c.pack.id);
    expect(cards, <String>["p3", "p4"]);
  });

  testWidgets("search filters by name author and tag with Vietnamese "
      "diacritics", (WidgetTester tester) async {
    await _pump(
      tester,
      packs: <StickerPack>[
        fakePack("cat", name: "Mèo Cam Ngố", tags: <String>["con vật"]),
        fakePack("panda", name: "Gấu Trúc", author: "Đông Nhi"),
        fakePack("frog", name: "Ếch Xanh", tags: <String>["tết"]),
        fakePack("duck", name: "Vịt Vàng"),
      ],
    );
    await tester.tap(
      find.byKey(const ValueKey<String>("marketplace.search.open")),
    );
    await tester.pump();
    final Finder field = find.byKey(
      const ValueKey<String>("marketplace.search"),
    );

    Future<Iterable<String>> shown(String query) async {
      await tester.enterText(field, query);
      await tester.pump();
      return tester
          .widgetList<PackCompactCard>(find.byType(PackCompactCard))
          .map((PackCompactCard c) => c.pack.id);
    }

    expect(await shown("meo"), <String>["cat"]);
    expect(await shown("MÈO"), <String>["cat"]);
    expect(await shown("dong"), <String>["panda"]);
    expect(await shown("TET"), <String>["frog"]);
    expect(await shown("con vat"), <String>["cat"]);
    expect(find.byType(PackHeroCarousel), findsNothing);
    expect(await shown(""), <String>["duck"]);
  });

  testWidgets("tag chips filter the loaded packs and toggle off", (
    WidgetTester tester,
  ) async {
    await _pump(
      tester,
      packs: <StickerPack>[
        fakePack("cat", tags: <String>["con vật"]),
        fakePack("panda", tags: <String>["meme"]),
        fakePack("frog"),
        fakePack("duck", tags: <String>["meme"]),
      ],
    );
    final Finder meme = find.byKey(
      const ValueKey<String>("marketplace.chip.meme"),
    );
    await tester.tap(meme);
    await tester.pump();
    expect(
      tester
          .widgetList<PackCompactCard>(find.byType(PackCompactCard))
          .map((PackCompactCard c) => c.pack.id),
      <String>["panda", "duck"],
    );
    await tester.tap(meme);
    await tester.pump();
    expect(find.byType(PackHeroCarousel), findsOneWidget);
  });

  testWidgets("pack without previews falls back to thumbnail and still "
      "renders", (WidgetTester tester) async {
    await _pump(
      tester,
      packs: <StickerPack>[
        fakePack("p0", thumbnailUrl: "http://x/t0.png", previewCount: 0),
        fakePack("p1", previewCount: 1),
        fakePack("p2", previewCount: 2),
        fakePack("p3", thumbnailUrl: "http://x/t3.png", previewCount: 0),
      ],
    );
    Finder sticker(String url) => find.byWidgetPredicate(
      (Widget w) => w is DieCutSticker && w.url == url,
    );
    expect(sticker("http://x/t0.png"), findsOneWidget);
    expect(sticker("http://x/t3.png"), findsOneWidget);
    expect(find.byType(PackCompactCard), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets("long Vietnamese title with textScaler 2.0 at 320dp has no "
      "overflow", (WidgetTester tester) async {
    const String longName =
        "Những Chú Mèo Cam Siêu Ngố Và Bạn Bè Đáng Yêu Ở Sài Gòn Ồn Ào";
    await _pump(
      tester,
      size: const Size(320, 640),
      textScale: 2,
      packs: List<StickerPack>.generate(
        5,
        (int i) => fakePack(
          "p$i",
          name: longName,
          author: "Nguyễn Thị Ánh Dương Đặng",
          tags: <String>["động vật dễ thương"],
          previewCount: i,
        ),
      ),
    );
    expect(tester.takeException(), isNull);
    await tester.drag(find.byType(CustomScrollView), const Offset(0, -600));
    await tester.pump(const Duration(milliseconds: 600));
    expect(find.byType(PackCompactCard), findsWidgets);
    expect(tester.takeException(), isNull);
  });

  testWidgets("compact card strip greys while downloading and recolors by "
      "progress thresholds", (WidgetTester tester) async {
    final FakeMarketplaceRepository repository = await _pump(
      tester,
      packs: _packs(4),
    );
    Future<void> emit(PackDownloadStatus status, double progress) async {
      repository.downloads.add(
        PackDownloadState(packId: "p3", status: status, progress: progress),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 600));
    }

    List<bool> grey() => <bool>[
      for (int i = 0; i < 3; i++) tester.any(_greyStrip(i)),
    ];

    expect(grey(), <bool>[false, false, false]);
    await emit(PackDownloadStatus.downloading, 0);
    expect(grey(), <bool>[true, true, true]);
    await emit(PackDownloadStatus.downloading, 0.34);
    expect(grey(), <bool>[false, true, true]);
    await emit(PackDownloadStatus.downloading, 0.67);
    expect(grey(), <bool>[false, false, true]);
    await emit(PackDownloadStatus.failed, 0.67);
    expect(grey(), <bool>[false, false, true]);
    await emit(PackDownloadStatus.downloaded, 1);
    expect(grey(), <bool>[false, false, false]);
  });

  testWidgets("idle motion stops when TickerMode disabled", (
    WidgetTester tester,
  ) async {
    final ValueNotifier<bool> tickers = ValueNotifier<bool>(true);
    await _pump(
      tester,
      packs: _packs(4),
      disableAnimations: false,
      tickers: tickers,
    );
    await tester.pump(const Duration(seconds: 1));
    const ValueKey<String> floatKey = ValueKey<String>("restick.hero.float");
    final Matrix4 before =
        tester.widget<Transform>(find.byKey(floatKey)).transform.clone();
    await tester.pump(const Duration(milliseconds: 700));
    expect(
      tester.widget<Transform>(find.byKey(floatKey)).transform,
      isNot(equals(before)),
    );
    expect(SchedulerBinding.instance.hasScheduledFrame, isTrue);

    tickers.value = false;
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));
    expect(SchedulerBinding.instance.hasScheduledFrame, isFalse);
  });

  testWidgets("idle motion does not run with disableAnimations", (
    WidgetTester tester,
  ) async {
    await _pump(tester, packs: _packs(4));
    await tester.pump(const Duration(milliseconds: 100));
    expect(SchedulerBinding.instance.hasScheduledFrame, isFalse);
  });

  testWidgets("pull to refresh calls MarketplaceCubit.load", (
    WidgetTester tester,
  ) async {
    final FakeMarketplaceRepository repository = await _pump(
      tester,
      packs: _packs(4),
    );
    expect(repository.listCalls, 1);
    await tester.fling(
      find.byType(CustomScrollView),
      const Offset(0, 400),
      1000,
    );
    await tester.pump();
    await tester.pump(const Duration(seconds: 1));
    await tester.pump(const Duration(seconds: 1));
    expect(repository.listCalls, 2);
  });

  testWidgets("download key present once and compact owned has none", (
    WidgetTester tester,
  ) async {
    final FakeMarketplaceRepository repository = await _pump(
      tester,
      packs: <StickerPack>[
        ..._packs(3),
        fakePack("p3"),
        fakePack("p4", status: PackDownloadStatus.downloaded),
      ],
    );
    const ValueKey<String> download = ValueKey<String>(
      "marketplace.pack.download.p3",
    );
    expect(find.byKey(download), findsOneWidget);
    expect(
      find.byKey(const ValueKey<String>("marketplace.pack.download.p0")),
      findsOneWidget,
    );
    expect(
      find.byKey(const ValueKey<String>("marketplace.pack.download.p4")),
      findsNothing,
    );
    expect(
      find.byKey(const ValueKey<String>("marketplace.pack.remove.p4")),
      findsNothing,
    );
    await tester.tap(find.byKey(download));
    await tester.pump();
    expect(repository.downloadCalls, <String>["p3"]);
  });

  testWidgets("en strings show 1 sticker and 24 stickers", (
    WidgetTester tester,
  ) async {
    const RestickStrings en = RestickStrings.en;
    await _pump(
      tester,
      strings: en,
      packs: <StickerPack>[
        ..._packs(3),
        fakePack("one", stickerCount: 1),
        fakePack("many", stickerCount: 24),
      ],
    );
    expect(find.text(en.packMeta(1, "Restick")), findsOneWidget);
    expect(find.text(en.packMeta(24, "Restick")), findsOneWidget);
    expect(find.text(en.newShelf), findsOneWidget);
  });

  testWidgets("rebuilds during the card entrance do not throw", (
    WidgetTester tester,
  ) async {
    final FakeMarketplaceRepository repository = await _pump(
      tester,
      packs: _packs(6),
      disableAnimations: false,
      settle: const Duration(milliseconds: 50),
    );
    expect(find.byType(PackCompactCard), findsWidgets);
    repository.downloads.add(
      const PackDownloadState(
        packId: "p4",
        status: PackDownloadStatus.downloading,
        progress: 0.5,
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));
    expect(tester.takeException(), isNull);
    await tester.pump(const Duration(milliseconds: 600));
    expect(tester.takeException(), isNull);
  });

  testWidgets("strip greys and recolours with animations on", (
    WidgetTester tester,
  ) async {
    final FakeMarketplaceRepository repository = await _pump(
      tester,
      packs: _packs(4),
      disableAnimations: false,
    );
    for (final double p in <double>[0, 0.5, 1]) {
      repository.downloads.add(
        PackDownloadState(
          packId: "p3",
          status: PackDownloadStatus.downloading,
          progress: p,
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 200));
      expect(tester.takeException(), isNull);
      await tester.pump(const Duration(milliseconds: 500));
    }
    expect(_greyStrip(0), findsNothing);
    expect(_greyStrip(2), findsNothing);
  });

  testWidgets("idle motion stops while another route covers the page", (
    WidgetTester tester,
  ) async {
    await _pump(tester, packs: _packs(4), disableAnimations: false);
    final NavigatorState navigator = tester.state<NavigatorState>(
      find.byType(Navigator),
    );
    unawaited(
      navigator.push(
        MaterialPageRoute<void>(
          builder: (_) => const Scaffold(body: Text("other")),
        ),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(seconds: 1));
    await tester.pump(const Duration(milliseconds: 100));
    expect(SchedulerBinding.instance.hasScheduledFrame, isFalse);
    expect(
      tester
          .widget<RestickMascot>(
            find.byType(RestickMascot, skipOffstage: false),
          )
          .blink,
      isFalse,
    );

    navigator.pop();
    await tester.pump();
    await tester.pump(const Duration(seconds: 1));
    expect(tester.takeException(), isNull);
    expect(
      find.byKey(const ValueKey<String>("restick.hero.float")),
      findsOneWidget,
    );
  });

  testWidgets("hero float stops when the carousel scrolls out of view", (
    WidgetTester tester,
  ) async {
    await _pump(
      tester,
      packs: _packs(14),
      disableAnimations: false,
      size: const Size(400, 700),
    );
    final Finder float = find.byKey(
      const ValueKey<String>("restick.hero.float"),
      skipOffstage: false,
    );
    expect(float, findsOneWidget);

    await tester.drag(find.byType(CustomScrollView), const Offset(0, -600));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 600));
    expect(float, findsNothing);

    await tester.drag(find.byType(CustomScrollView), const Offset(0, 800));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 600));
    expect(float, findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
