import "dart:async";

import "package:flutter/scheduler.dart";
import "package:flutter/rendering.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:flutter_test/flutter_test.dart";
import "package:material_ui/material_ui.dart";
import "package:restick/src/core/restick_client.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/presentation/cubit/marketplace_cubit.dart";
import "package:restick/src/presentation/theme/restick_strings.dart";
import "package:restick/src/presentation/widgets/die_cut_sticker.dart";
import "package:restick/src/presentation/widgets/download_button.dart";
import "package:restick/src/presentation/widgets/pack_compact_card.dart";
import "package:restick/src/presentation/widgets/pack_detail_sheet.dart";
import "package:restick/src/presentation/widgets/pack_hero_carousel.dart";
import "package:restick/src/presentation/widgets/sticker_marketplace.dart";

import "marketplace_fakes.dart";

const RestickStrings _vi = RestickStrings.vi;
const ValueKey<String> _sheetKey = ValueKey<String>("restick.sheet");

/// Records deletes and, like the real repository, reports the pack as
/// remote again afterwards.
class _DeletingRepository extends FakeMarketplaceRepository {
  _DeletingRepository({super.remote});

  final List<String> deleteCalls = <String>[];

  @override
  Future<void> deletePack(String packId) async {
    deleteCalls.add(packId);
    downloads.add(
      PackDownloadState(packId: packId, status: PackDownloadStatus.remote),
    );
  }
}

Future<_DeletingRepository> _pump(
  WidgetTester tester, {
  required List<StickerPack> packs,
  RestickStrings strings = _vi,
  double textScale = 1,
  bool disableAnimations = false,
  Size size = const Size(400, 1000),
}) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  final _DeletingRepository repository = _DeletingRepository(remote: packs);
  final RestickClient client = fakeRestickClient(repository);
  addTearDown(client.dispose);
  await tester.pumpWidget(
    MaterialApp(
      builder:
          (BuildContext context, Widget? child) => MediaQuery(
            data: MediaQuery.of(context).copyWith(
              textScaler: TextScaler.linear(textScale),
              disableAnimations: disableAnimations,
            ),
            child: child!,
          ),
      home: Scaffold(
        body: StickerMarketplace(client: client, strings: strings),
      ),
    ),
  );
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 600));
  return repository;
}

/// First three packs fill the carousel; the rest are compact cards.
List<StickerPack> _withCard(StickerPack card) => <StickerPack>[
  fakePack("p0"),
  fakePack("p1"),
  fakePack("p2"),
  card,
];

Finder _inSheet(Finder finder) =>
    find.descendant(of: find.byKey(_sheetKey), matching: finder);

Finder _card(String id) => find.byKey(ValueKey<String>(id));

Future<void> _openCard(WidgetTester tester, String id) async {
  await tester.tap(
    find.descendant(of: _card(id), matching: find.text("Pack $id")),
  );
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 700));
}

Future<void> _emit(
  WidgetTester tester,
  FakeMarketplaceRepository repository,
  String id,
  PackDownloadStatus status,
  double progress,
) async {
  repository.downloads.add(
    PackDownloadState(packId: id, status: status, progress: progress),
  );
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 400));
}

void main() {
  testWidgets("opens from a card tap and shows name count and tags", (
    WidgetTester tester,
  ) async {
    await _pump(
      tester,
      packs: _withCard(
        fakePack("p3", tags: <String>["cute", "bear"], stickerCount: 12),
      ),
    );
    expect(find.byKey(_sheetKey), findsNothing);
    await _openCard(tester, "p3");
    expect(find.byKey(_sheetKey), findsOneWidget);
    expect(_inSheet(find.text("Pack p3")), findsOneWidget);
    expect(_inSheet(find.text(_vi.packMeta(12, "Restick"))), findsOneWidget);
    expect(_inSheet(find.text("cute")), findsOneWidget);
    expect(_inSheet(find.text("bear")), findsOneWidget);
    expect(_inSheet(find.text(_vi.downloadAllLabel(12))), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets("opens from the current hero card", (WidgetTester tester) async {
    await _pump(tester, packs: _withCard(fakePack("p3")));
    await tester.tap(
      find.descendant(
        of: find.byType(PackHeroCarousel),
        matching: find.text("Pack p0"),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 700));
    expect(_inSheet(find.text("Pack p0")), findsOneWidget);
  });

  testWidgets("grid shows up to six previews and one big thumbnail when none", (
    WidgetTester tester,
  ) async {
    await _pump(
      tester,
      packs: <StickerPack>[
        ..._withCard(fakePack("p3", previewCount: 9)),
        fakePack("p4", previewCount: 0),
      ],
    );
    await _openCard(tester, "p3");
    expect(_inSheet(find.byType(DieCutSticker)), findsNWidgets(6));
    // Small, alternating tilts so the grid is not a rigid square.
    final List<double> angles =
        tester
            .widgetList<DieCutSticker>(_inSheet(find.byType(DieCutSticker)))
            .map((DieCutSticker s) => s.rotation)
            .toList();
    expect(angles.every((double a) => a != 0 && a.abs() < 0.2), isTrue);
    expect(angles[0].sign, isNot(angles[1].sign));

    Navigator.of(tester.element(find.byKey(_sheetKey))).pop();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 600));
    expect(find.byKey(_sheetKey), findsNothing);

    await _openCard(tester, "p4");
    final Iterable<DieCutSticker> one = tester.widgetList<DieCutSticker>(
      _inSheet(find.byType(DieCutSticker)),
    );
    expect(one, hasLength(1));
    expect(one.single.size, greaterThan(100));
    expect(tester.takeException(), isNull);
  });

  testWidgets("download from sheet updates progress live then shows owned "
      "with remove button", (WidgetTester tester) async {
    final _DeletingRepository repository = await _pump(
      tester,
      packs: _withCard(fakePack("p3")),
    );
    await _openCard(tester, "p3");
    await tester.tap(
      _inSheet(
        find.byKey(const ValueKey<String>("marketplace.pack.download.p3")),
      ),
    );
    await tester.pump();
    expect(repository.downloadCalls, <String>["p3"]);

    await _emit(tester, repository, "p3", PackDownloadStatus.downloading, 0.4);
    expect(_inSheet(find.text("${_vi.downloading} · 40%")), findsOneWidget);
    await _emit(tester, repository, "p3", PackDownloadStatus.downloading, 0.7);
    expect(_inSheet(find.text("${_vi.downloading} · 70%")), findsOneWidget);
    await _emit(tester, repository, "p3", PackDownloadStatus.downloaded, 1);
    await tester.pump(const Duration(seconds: 1));
    expect(
      _inSheet(
        find.byKey(const ValueKey<String>("marketplace.pack.remove.p3")),
      ),
      findsOneWidget,
    );
    expect(find.byKey(_sheetKey), findsOneWidget);
    // Nothing between the button and the screen clips its burst sideways or
    // downward: the route's own ClipRect is full-screen, and the sheet's
    // scroll clip cuts only its top edge (so scrolled content cannot paint
    // above the white sheet) while reaching far past the other three sides.
    final Finder clips = find.ancestor(
      of: _inSheet(find.byType(PackDownloadButton)),
      matching: find.byWidgetPredicate(
        (Widget w) =>
            (w is ClipRect && w.clipBehavior != Clip.none) ||
            w is ClipRRect ||
            w is ClipPath ||
            (w is Material && w.clipBehavior != Clip.none) ||
            (w is SingleChildScrollView && w.clipBehavior != Clip.none),
      ),
    );
    for (final Element clip in clips.evaluate()) {
      final ClipRect rect = clip.widget as ClipRect;
      final CustomClipper<Rect>? clipper = rect.clipper;
      if (clipper == null) {
        expect(clip.size, tester.view.physicalSize);
        continue;
      }
      final Size size = clip.size!;
      final Rect area = clipper.getClip(size);
      expect(area.top, 0);
      expect(area.left, lessThan(-1000));
      expect(area.right, greaterThan(size.width + 1000));
      expect(area.bottom, greaterThan(size.height + 1000));
    }
    expect(tester.takeException(), isNull);
  });

  testWidgets("remove button asks cubit.delete and returns to download state", (
    WidgetTester tester,
  ) async {
    final _DeletingRepository repository = await _pump(
      tester,
      packs: _withCard(fakePack("p3", status: PackDownloadStatus.downloaded)),
    );
    await _openCard(tester, "p3");
    await tester.tap(
      _inSheet(
        find.byKey(const ValueKey<String>("marketplace.pack.remove.p3")),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
    expect(repository.deleteCalls, <String>["p3"]);
    expect(find.byKey(_sheetKey), findsOneWidget);
    expect(
      _inSheet(
        find.byKey(const ValueKey<String>("marketplace.pack.download.p3")),
      ),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets("sheet closes cleanly while a download continues and card "
      "still shows progress", (WidgetTester tester) async {
    final _DeletingRepository repository = await _pump(
      tester,
      packs: _withCard(fakePack("p3")),
    );
    await _openCard(tester, "p3");
    await tester.tap(
      _inSheet(
        find.byKey(const ValueKey<String>("marketplace.pack.download.p3")),
      ),
    );
    await tester.pump();
    await _emit(tester, repository, "p3", PackDownloadStatus.downloading, 0.3);

    // Tap the dimmed barrier above the sheet.
    await tester.tapAt(const Offset(200, 20));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));
    await _emit(tester, repository, "p3", PackDownloadStatus.downloading, 0.6);
    await tester.pump(const Duration(milliseconds: 600));
    expect(find.byKey(_sheetKey), findsNothing);
    expect(
      find.descendant(of: _card("p3"), matching: find.text("60%")),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets("closes when the pack leaves the marketplace list", (
    WidgetTester tester,
  ) async {
    final _DeletingRepository repository = await _pump(
      tester,
      packs: _withCard(fakePack("p3")),
    );
    await _openCard(tester, "p3");
    repository.remote = <StickerPack>[fakePack("p0"), fakePack("p1")];
    final MarketplaceCubit cubit = BlocProvider.of<MarketplaceCubit>(
      tester.element(find.byType(PackHeroCarousel)),
    );
    await cubit.load();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 600));
    expect(find.byKey(_sheetKey), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets("closes when the pack leaves the list while covered and cards "
      "open again", (WidgetTester tester) async {
    final _DeletingRepository repository = await _pump(
      tester,
      packs: <StickerPack>[..._withCard(fakePack("p3")), fakePack("p4")],
    );
    await _openCard(tester, "p3");
    final NavigatorState navigator = Navigator.of(
      tester.element(find.byKey(_sheetKey)),
    );
    unawaited(
      navigator.push(
        MaterialPageRoute<void>(
          builder: (_) => const Scaffold(body: Text("other")),
        ),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 600));
    repository.remote = <StickerPack>[..._withCard(fakePack("p4"))];
    final MarketplaceCubit cubit = BlocProvider.of<MarketplaceCubit>(
      tester.element(find.byType(PackHeroCarousel, skipOffstage: false)),
    );
    await cubit.load();
    await tester.pump();
    navigator.pop();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 600));
    expect(find.byKey(_sheetKey), findsNothing);

    await _openCard(tester, "p4");
    expect(find.byKey(_sheetKey), findsOneWidget);
    expect(_inSheet(find.text("Pack p4")), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets("two previews render centred without errors at textScaler 2", (
    WidgetTester tester,
  ) async {
    await _pump(
      tester,
      textScale: 2,
      packs: _withCard(fakePack("p3", previewCount: 2)),
    );
    await _openCard(tester, "p3");
    expect(_inSheet(find.byType(DieCutSticker)), findsNWidgets(2));
    final Rect grid = tester.getRect(_inSheet(find.byType(Wrap)).first);
    final Rect first = tester.getRect(
      find.byKey(const ValueKey<String>("restick.sheet.sticker.0")),
    );
    final Rect second = tester.getRect(
      find.byKey(const ValueKey<String>("restick.sheet.sticker.1")),
    );
    expect(
      (first.left - grid.left - (grid.right - second.right)).abs(),
      lessThan(1),
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets("idle motion on the home pauses while the sheet is open", (
    WidgetTester tester,
  ) async {
    await _pump(tester, packs: _withCard(fakePack("p3")));
    const ValueKey<String> floatKey = ValueKey<String>("restick.hero.float");
    expect(find.byKey(floatKey), findsOneWidget);

    await _openCard(tester, "p3");
    await tester.pump(const Duration(seconds: 1));
    expect(find.byKey(floatKey), findsNothing);
    expect(SchedulerBinding.instance.hasScheduledFrame, isFalse);

    Navigator.of(tester.element(find.byKey(_sheetKey))).pop();
    await tester.pump();
    await tester.pump(const Duration(seconds: 1));
    expect(find.byKey(_sheetKey), findsNothing);
    expect(find.byKey(floatKey), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets("tapping a sticker wiggles it briefly", (
    WidgetTester tester,
  ) async {
    await _pump(tester, packs: _withCard(fakePack("p3")));
    await _openCard(tester, "p3");
    const ValueKey<String> wiggle = ValueKey<String>("restick.sheet.wiggle");
    expect(find.byKey(wiggle), findsNothing);
    await tester.tap(
      find.byKey(const ValueKey<String>("restick.sheet.sticker.0")),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));
    expect(find.byKey(wiggle), findsOneWidget);
    await tester.pump(const Duration(milliseconds: 600));
    expect(find.byKey(wiggle), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets("scrolled content never paints above the sheet's top edge", (
    WidgetTester tester,
  ) async {
    await _pump(
      tester,
      textScale: 2,
      size: const Size(320, 640),
      packs: _withCard(
        fakePack(
          "p3",
          tags: <String>["dễ thương", "ngộ nghĩnh", "chúc mừng", "gấu"],
          previewCount: 6,
        ),
      ),
    );
    await tester.drag(find.byType(CustomScrollView), const Offset(0, -300));
    await tester.pump(const Duration(milliseconds: 600));
    await tester.tap(
      find.descendant(of: _card("p3"), matching: find.text("Pack p3")),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 700));
    final Finder name = _inSheet(find.text("Pack p3"));
    await tester.drag(name, const Offset(0, -400));
    await tester.pump(const Duration(milliseconds: 600));

    final double sheetTop =
        tester.getRect(find.byKey(_sheetKey)).top + 34; // white bg top
    expect(tester.getRect(name).top, lessThan(sheetTop)); // really scrolled
    // A clip between the name and the sheet cuts it at the white edge.
    final Iterable<RenderClipRect> clips = tester
        .renderObjectList<RenderObject>(
          find.ancestor(of: name, matching: find.byType(ClipRect)),
        )
        .whereType<RenderClipRect>()
        .where((RenderClipRect r) => r.clipBehavior != Clip.none);
    final double visibleTop = clips
        .map(
          (RenderClipRect r) =>
              r
                  .localToGlobal(
                    r.clipper?.getClip(r.size).topLeft ?? Offset.zero,
                  )
                  .dy,
        )
        .fold(tester.getRect(name).top, (double a, double b) => a > b ? a : b);
    expect(visibleTop, greaterThanOrEqualTo(sheetTop - 0.5));
    expect(tester.takeException(), isNull);
  });

  testWidgets("no overflow at textScaler 2.0 and 320dp", (
    WidgetTester tester,
  ) async {
    await _pump(
      tester,
      textScale: 2,
      size: const Size(320, 640),
      packs: _withCard(
        fakePack(
          "p3",
          name: "Pack p3",
          author: "Nguyễn Thị Phương Thảo Ỡ Ự Ẳ",
          tags: <String>["dễ thương", "ngộ nghĩnh", "chúc mừng", "gấu"],
          previewCount: 6,
          stickerCount: 24,
        ),
      ),
    );
    await tester.drag(find.byType(CustomScrollView), const Offset(0, -300));
    await tester.pump(const Duration(milliseconds: 600));
    await tester.tap(
      find.descendant(of: _card("p3"), matching: find.text("Pack p3")),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 700));
    expect(find.byKey(_sheetKey), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets("disableAnimations shows content immediately", (
    WidgetTester tester,
  ) async {
    await _pump(
      tester,
      disableAnimations: true,
      packs: _withCard(fakePack("p3", previewCount: 6)),
    );
    await tester.tap(
      find.descendant(of: _card("p3"), matching: find.text("Pack p3")),
    );
    await tester.pump();
    await tester.pump();
    expect(_inSheet(find.text("Pack p3")), findsOneWidget);
    expect(_inSheet(find.byType(DieCutSticker)), findsNWidgets(6));
    expect(
      find.byKey(const ValueKey<String>("restick.sheet.pop")),
      findsNothing,
    );
    // Nothing in the sheet keeps ticking (the card's ink ripple fades first).
    await tester.pump(const Duration(seconds: 2));
    expect(SchedulerBinding.instance.hasScheduledFrame, isFalse);
  });

  testWidgets("en strings show Download · 1 sticker and Download · 24 "
      "stickers", (WidgetTester tester) async {
    const RestickStrings en = RestickStrings.en;
    await _pump(
      tester,
      strings: en,
      packs: <StickerPack>[
        ..._withCard(fakePack("p3", stickerCount: 1)),
        fakePack("p4", stickerCount: 24),
      ],
    );
    await _openCard(tester, "p3");
    expect(_inSheet(find.text(en.downloadAllLabel(1))), findsOneWidget);
    expect(en.downloadAllLabel(1), contains("1 sticker"));
    expect(en.downloadAllLabel(1), isNot(contains("stickers")));
    Navigator.of(tester.element(find.byKey(_sheetKey))).pop();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 600));

    await _openCard(tester, "p4");
    expect(_inSheet(find.text(en.downloadAllLabel(24))), findsOneWidget);
    expect(en.downloadAllLabel(24), contains("24 stickers"));
  });

  testWidgets("double tap on a card opens only one sheet", (
    WidgetTester tester,
  ) async {
    await _pump(tester, packs: _withCard(fakePack("p3")));
    // Two taps landing before the sheet route builds a frame.
    final VoidCallback open =
        tester.widget<PackCompactCard>(find.byType(PackCompactCard)).onOpen;
    open();
    open();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 700));
    expect(find.byKey(_sheetKey), findsOneWidget);

    Navigator.of(tester.element(find.byKey(_sheetKey))).pop();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 600));
    expect(find.byKey(_sheetKey), findsNothing);
    // The guard is released once the sheet closes.
    await _openCard(tester, "p3");
    expect(find.byKey(_sheetKey), findsOneWidget);
  });

  testWidgets("showPackDetailSheet with an unknown id does nothing", (
    WidgetTester tester,
  ) async {
    await _pump(tester, packs: _withCard(fakePack("p3")));
    final BuildContext context = tester.element(find.byType(PackCompactCard));
    await showPackDetailSheet(
      context,
      cubit: BlocProvider.of<MarketplaceCubit>(context),
      packId: "missing",
    );
    await tester.pump();
    expect(find.byKey(_sheetKey), findsNothing);
  });
}
