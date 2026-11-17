import "package:flutter/services.dart";
import "package:flutter_test/flutter_test.dart";
import "package:material_ui/material_ui.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/presentation/theme/restick_strings.dart";
import "package:restick/src/presentation/widgets/download_button.dart";

const RestickStrings _s = RestickStrings.vi;
const ValueKey<String> _downloadKey = ValueKey<String>(
  "marketplace.pack.download.p1",
);
const ValueKey<String> _removeKey = ValueKey<String>(
  "marketplace.pack.remove.p1",
);
const ValueKey<String> _shakeKey = ValueKey<String>("restick.download.shake");
const ValueKey<String> _fillKey = ValueKey<String>("restick.download.fill");
const ValueKey<String> _tickKey = ValueKey<String>("restick.download.tick");
const ValueKey<String> _burstKey = ValueKey<String>("restick.download.burst");
const ValueKey<String> _pillKey = ValueKey<String>("restick.download.pill");

StickerPack _pack({
  PackDownloadStatus status = PackDownloadStatus.remote,
  double progress = 0,
  String id = "p1",
}) {
  return StickerPack(
    id: id,
    slug: id,
    name: "Mèo",
    author: "Restick",
    tags: const <String>[],
    stickerCount: 24,
    version: 1,
    thumbnailUrl: "",
    checksumSha256: "",
    downloadStatus: status,
    downloadProgress: progress,
  );
}

class _Harness {
  int downloads = 0;
  int removes = 0;

  Future<void> show(
    WidgetTester tester,
    StickerPack pack, {
    bool compact = true,
    bool disableAnimations = false,
  }) {
    return tester.pumpWidget(
      MaterialApp(
        home: MediaQuery(
          data: MediaQueryData(disableAnimations: disableAnimations),
          child: Scaffold(
            body: Center(
              child: SizedBox(
                width: 320,
                child: Center(
                  child: PackDownloadButton(
                    pack: pack,
                    compact: compact,
                    onDownload: () => downloads++,
                    onRemove: () => removes++,
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

double _fill(WidgetTester tester) =>
    tester.widget<Transform>(find.byKey(_fillKey)).transform.storage[0];

double _shakeX(WidgetTester tester) =>
    tester
        .widget<Transform>(find.byKey(_shakeKey))
        .transform
        .getTranslation()
        .x;

Color? _pillColor(WidgetTester tester) {
  final AnimatedContainer pill = tester.widget<AnimatedContainer>(
    find.byKey(_pillKey),
  );
  return (pill.decoration! as BoxDecoration).color;
}

double _tickProgress(WidgetTester tester) {
  final CustomPaint paint = tester.widget<CustomPaint>(find.byKey(_tickKey));
  return (paint.painter! as TickPainter).progress;
}

void main() {
  for (final PackDownloadStatus end in <PackDownloadStatus>[
    PackDownloadStatus.downloaded,
    PackDownloadStatus.failed,
  ]) {
    testWidgets("enabling reduced motion cancels active $end effects", (
      WidgetTester tester,
    ) async {
      final _Harness h = _Harness();
      await h.show(tester, _pack(status: PackDownloadStatus.downloading, progress: 0.67));
      await h.show(tester, _pack(status: end, progress: 0.67));
      await tester.pump(const Duration(milliseconds: 100));
      if (end == PackDownloadStatus.downloaded) {
        expect(find.byKey(_burstKey), findsOneWidget);
        expect(_tickProgress(tester), lessThan(1));
      } else {
        expect(_shakeX(tester), isNot(0));
      }
      final Color? colour = _pillColor(tester);
      await h.show(tester, _pack(status: end, progress: 0.67), disableAnimations: true);
      expect(find.byKey(_burstKey), findsNothing);
      expect(_shakeX(tester), 0);
      expect(_pillColor(tester), colour);
      if (end == PackDownloadStatus.downloaded) {
        expect(_tickProgress(tester), 1);
      }
      await tester.pump(const Duration(seconds: 1));
      expect(tester.binding.transientCallbackCount, 0);
    });
  }

  testWidgets("remote shows download label and fires onDownload once", (
    WidgetTester tester,
  ) async {
    final _Harness h = _Harness();
    await h.show(tester, _pack());
    expect(find.text(_s.download), findsOneWidget);

    await tester.tap(find.byKey(_downloadKey));
    await tester.pump();
    expect(h.downloads, 1);

    await tester.pumpWidget(const SizedBox());
    await h.show(tester, _pack(), compact: false);
    expect(find.text(_s.downloadAllLabel(24)), findsOneWidget);
  });

  testWidgets("rapid double tap fires onDownload once until status changes", (
    WidgetTester tester,
  ) async {
    final _Harness h = _Harness();
    await h.show(tester, _pack());
    await tester.tap(find.byKey(_downloadKey));
    await tester.pump(const Duration(milliseconds: 20));
    await tester.tap(find.byKey(_downloadKey));
    await tester.pump(const Duration(milliseconds: 20));
    expect(h.downloads, 1);

    await h.show(
      tester,
      _pack(status: PackDownloadStatus.downloading, progress: 0.2),
    );
    await tester.tap(find.byKey(_downloadKey), warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 50));
    expect(h.downloads, 1, reason: "downloading is not tappable");

    await h.show(tester, _pack(status: PackDownloadStatus.failed));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(_downloadKey));
    await tester.pump();
    expect(h.downloads, 2);
  });

  testWidgets("tapping shows waiting state within 100ms before any progress", (
    WidgetTester tester,
  ) async {
    final _Harness h = _Harness();
    await h.show(tester, _pack());
    await tester.tap(find.byKey(_downloadKey));
    await tester.pump(const Duration(milliseconds: 100));
    expect(find.text(_s.waiting), findsOneWidget);
    expect(_pillColor(tester), Colors.white);
  });

  testWidgets("progress fill follows downloadProgress and shows percent", (
    WidgetTester tester,
  ) async {
    final _Harness h = _Harness();
    await h.show(
      tester,
      _pack(status: PackDownloadStatus.downloading, progress: 0.42),
    );
    await tester.pump(const Duration(milliseconds: 400));
    expect(find.text("42%"), findsOneWidget);
    expect(_fill(tester), closeTo(0.42, 0.001));

    await h.show(
      tester,
      _pack(status: PackDownloadStatus.downloading, progress: 0.8),
    );
    await tester.pump(const Duration(milliseconds: 100));
    expect(_fill(tester), allOf(greaterThan(0.42), lessThan(0.8)));
    await tester.pump(const Duration(milliseconds: 400));
    expect(find.text("80%"), findsOneWidget);
    expect(_fill(tester), closeTo(0.8, 0.001));
  });

  testWidgets("downloading to downloaded plays success then settles on owned", (
    WidgetTester tester,
  ) async {
    final _Harness h = _Harness();
    await h.show(
      tester,
      _pack(status: PackDownloadStatus.downloading, progress: 0.98),
    );
    await tester.pump(const Duration(milliseconds: 50));
    await h.show(tester, _pack(status: PackDownloadStatus.downloaded));
    await tester.pump(const Duration(milliseconds: 16));
    expect(find.byKey(_burstKey), findsOneWidget);
    await tester.pump(const Duration(milliseconds: 150));
    expect(_tickProgress(tester), allOf(greaterThan(0), lessThan(1)));

    await tester.pumpAndSettle();
    expect(find.text(_s.owned), findsOneWidget);
    expect(find.byKey(_tickKey), findsOneWidget);
    expect(_tickProgress(tester), 1);
    expect(find.byKey(_burstKey), findsNothing);
    expect(find.byKey(_downloadKey), findsNothing);

    await tester.tap(find.text(_s.owned));
    await tester.pump();
    expect(h.downloads + h.removes, 0, reason: "owned chip is not tappable");
  });

  testWidgets("non-compact owned shows remove action with remove key", (
    WidgetTester tester,
  ) async {
    final _Harness h = _Harness();
    await h.show(
      tester,
      _pack(status: PackDownloadStatus.downloaded),
      compact: false,
    );
    await tester.pumpAndSettle();
    expect(find.byKey(_burstKey), findsNothing, reason: "no success on mount");
    await tester.tap(find.byKey(_removeKey));
    await tester.pump();
    expect(h.removes, 1);
  });

  testWidgets("failed shows retry label and tap fires onDownload", (
    WidgetTester tester,
  ) async {
    final _Harness h = _Harness();
    await h.show(tester, _pack(status: PackDownloadStatus.failed));
    expect(find.text(_s.retry), findsOneWidget);
    expect(_pillColor(tester), const Color(0xFFFFE3EC));
    await tester.tap(find.byKey(_downloadKey));
    await tester.pump();
    expect(h.downloads, 1);
    expect(find.text(_s.waiting), findsOneWidget);
  });

  testWidgets("failed after downloading shakes once", (
    WidgetTester tester,
  ) async {
    final _Harness h = _Harness();
    await h.show(
      tester,
      _pack(status: PackDownloadStatus.downloading, progress: 0.5),
    );
    await tester.pump(const Duration(milliseconds: 50));
    expect(_shakeX(tester), 0);
    await h.show(tester, _pack(status: PackDownloadStatus.failed));
    await tester.pump(const Duration(milliseconds: 90));
    expect(_shakeX(tester).abs(), greaterThan(1));
    await tester.pump(const Duration(milliseconds: 500));
    expect(_shakeX(tester), 0);
    await tester.pumpAndSettle();
    expect(_shakeX(tester), 0);
    expect(tester.hasRunningAnimations, isFalse);
  });

  testWidgets(
    "progress dropping back to 0 after failure does not show waiting spinner forever",
    (WidgetTester tester) async {
      final _Harness h = _Harness();
      await h.show(tester, _pack());
      await tester.tap(find.byKey(_downloadKey));
      await tester.pump();
      await h.show(
        tester,
        _pack(status: PackDownloadStatus.downloading, progress: 0.3),
      );
      await tester.pump(const Duration(milliseconds: 50));
      await h.show(tester, _pack(status: PackDownloadStatus.failed));
      await tester.pumpAndSettle();
      await h.show(tester, _pack());
      await tester.pumpAndSettle();
      expect(find.text(_s.download), findsOneWidget);
      expect(find.text(_s.waiting), findsNothing);

      await tester.tap(find.byKey(_downloadKey));
      await tester.pump(const Duration(milliseconds: 100));
      expect(h.downloads, 2);
      expect(find.text(_s.waiting), findsOneWidget);

      // Fast failure: downloading+failed coalesce, so the widget only sees a
      // new failed instance (failed -> failed). It must fail right away.
      await h.show(tester, _pack(status: PackDownloadStatus.failed));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(_downloadKey));
      await tester.pump(const Duration(milliseconds: 100));
      expect(find.text(_s.waiting), findsOneWidget);
      await h.show(tester, _pack(status: PackDownloadStatus.failed));
      await tester.pump(const Duration(milliseconds: 90));
      expect(_shakeX(tester).abs(), greaterThan(1));
      await tester.pumpAndSettle();
      expect(find.text(_s.retry), findsOneWidget);
      expect(find.text(_s.waiting), findsNothing);

      // Last resort: no update ever arrives for this pack.
      await tester.tap(find.byKey(_downloadKey));
      await tester.pump(const Duration(milliseconds: 100));
      expect(h.downloads, 4);
      expect(find.text(_s.waiting), findsOneWidget);
      await tester.pump(const Duration(seconds: 31));
      await tester.pumpAndSettle();
      expect(find.text(_s.retry), findsOneWidget);
      await tester.tap(find.byKey(_downloadKey));
      await tester.pump();
      expect(h.downloads, 5);
    },
  );

  testWidgets(
    "disableAnimations skips shake and burst but still shows state colors and tick",
    (WidgetTester tester) async {
      final _Harness h = _Harness();
      await h.show(
        tester,
        _pack(status: PackDownloadStatus.downloading, progress: 0.5),
        disableAnimations: true,
      );
      await tester.pump();
      expect(_fill(tester), closeTo(0.5, 0.001));
      expect(_pillColor(tester), Colors.white);

      await h.show(
        tester,
        _pack(status: PackDownloadStatus.downloaded),
        disableAnimations: true,
      );
      await tester.pump();
      expect(find.byKey(_burstKey), findsNothing);
      expect(_tickProgress(tester), 1);
      expect(find.text(_s.owned), findsOneWidget);
      expect(tester.hasRunningAnimations, isFalse);

      await h.show(
        tester,
        _pack(status: PackDownloadStatus.downloading, progress: 0.5),
        disableAnimations: true,
      );
      await tester.pump();
      await h.show(
        tester,
        _pack(status: PackDownloadStatus.failed),
        disableAnimations: true,
      );
      await tester.pump(const Duration(milliseconds: 90));
      expect(_shakeX(tester), 0);
      expect(_pillColor(tester), const Color(0xFFFFE3EC));
      expect(tester.hasRunningAnimations, isFalse);
    },
  );

  testWidgets("semantics announce downloading percent, done and failed", (
    WidgetTester tester,
  ) async {
    final SemanticsHandle handle = tester.ensureSemantics();
    final _Harness h = _Harness();
    await h.show(
      tester,
      _pack(status: PackDownloadStatus.downloading, progress: 0.42),
    );
    await tester.pump(const Duration(milliseconds: 400));
    expect(
      find.bySemanticsLabel(_s.a11yDownloading("Mèo", 42)),
      findsOneWidget,
    );
    expect(
      tester.getSemantics(find.bySemanticsLabel(_s.a11yDownloading("Mèo", 42))),
      isSemantics(
        label: _s.a11yDownloading("Mèo", 42),
        isLiveRegion: true,
      ),
    );

    await h.show(tester, _pack(status: PackDownloadStatus.downloaded));
    await tester.pumpAndSettle();
    expect(find.bySemanticsLabel(_s.a11yDone("Mèo")), findsOneWidget);

    await h.show(tester, _pack(status: PackDownloadStatus.failed));
    await tester.pumpAndSettle();
    expect(find.bySemanticsLabel(_s.a11yFailed("Mèo")), findsOneWidget);
    handle.dispose();
  });

  testWidgets("haptics called on tap, success, failure", (
    WidgetTester tester,
  ) async {
    final List<Object?> haptics = <Object?>[];
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (MethodCall call) async {
        if (call.method == "HapticFeedback.vibrate") {
          haptics.add(call.arguments);
        }
        return null;
      },
    );
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        null,
      ),
    );
    final _Harness h = _Harness();
    await h.show(tester, _pack());
    await tester.tap(find.byKey(_downloadKey));
    await tester.pump();
    expect(haptics, <Object?>["HapticFeedbackType.selectionClick"]);

    await h.show(
      tester,
      _pack(status: PackDownloadStatus.downloading, progress: 0.5),
    );
    await tester.pump(const Duration(milliseconds: 50));
    await h.show(tester, _pack(status: PackDownloadStatus.downloaded));
    await tester.pumpAndSettle();
    expect(haptics.last, "HapticFeedbackType.lightImpact");

    await h.show(
      tester,
      _pack(status: PackDownloadStatus.downloading, progress: 0.5),
    );
    await tester.pump(const Duration(milliseconds: 50));
    await h.show(tester, _pack(status: PackDownloadStatus.failed));
    await tester.pumpAndSettle();
    expect(haptics, <Object?>[
      "HapticFeedbackType.selectionClick",
      "HapticFeedbackType.lightImpact",
      "HapticFeedbackType.mediumImpact",
    ]);
  });

  testWidgets("button size is stable across states", (
    WidgetTester tester,
  ) async {
    for (final bool compact in <bool>[true, false]) {
      final _Harness h = _Harness();
      final List<Size> sizes = <Size>[];
      for (final StickerPack pack in <StickerPack>[
        _pack(),
        _pack(status: PackDownloadStatus.downloading, progress: 0.42),
        _pack(status: PackDownloadStatus.downloaded),
        _pack(status: PackDownloadStatus.failed),
      ]) {
        await h.show(tester, pack, compact: compact);
        await tester.pump(const Duration(milliseconds: 400));
        sizes.add(tester.getSize(find.byType(PackDownloadButton)));
      }
      expect(sizes.toSet(), hasLength(1), reason: "compact=$compact $sizes");
      await tester.pumpWidget(const SizedBox());
    }
  });

  testWidgets("fast failure (failed -> new failed) buzzes and announces", (
    WidgetTester tester,
  ) async {
    final SemanticsHandle handle = tester.ensureSemantics();
    final List<Object?> haptics = <Object?>[];
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (MethodCall call) async {
        if (call.method == "HapticFeedback.vibrate") {
          haptics.add(call.arguments);
        }
        return null;
      },
    );
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        null,
      ),
    );
    final _Harness h = _Harness();
    final StickerPack failed = _pack(status: PackDownloadStatus.failed);
    await h.show(tester, failed);
    await tester.tap(find.byKey(_downloadKey));
    await tester.pump();
    expect(find.bySemanticsLabel(_s.a11yFailed("Mèo")), findsNothing);

    // Same instance (unrelated parent rebuild) keeps waiting.
    await h.show(tester, failed);
    await tester.pump(const Duration(milliseconds: 50));
    expect(find.text(_s.waiting), findsOneWidget);

    await h.show(tester, _pack(status: PackDownloadStatus.failed));
    await tester.pumpAndSettle();
    expect(haptics.last, "HapticFeedbackType.mediumImpact");
    expect(find.bySemanticsLabel(_s.a11yFailed("Mèo")), findsOneWidget);
    handle.dispose();
  });

  testWidgets("success cross-fades labels instead of remounting the pill", (
    WidgetTester tester,
  ) async {
    final _Harness h = _Harness();
    await h.show(
      tester,
      _pack(status: PackDownloadStatus.downloading, progress: 0.98),
    );
    await tester.pump(const Duration(milliseconds: 50));
    await h.show(tester, _pack(status: PackDownloadStatus.downloaded));
    await tester.pump(const Duration(milliseconds: 50));
    expect(find.text("98%"), findsOneWidget, reason: "outgoing label");
    expect(find.text(_s.owned), findsOneWidget, reason: "incoming label");
    await tester.pumpAndSettle();
    expect(find.text("98%"), findsNothing);

    // Owned -> remote (after a remove): colour morphs white -> pink.
    await h.show(
      tester,
      _pack(status: PackDownloadStatus.downloaded),
      compact: false,
    );
    await tester.pumpAndSettle();
    await h.show(tester, _pack(), compact: false);
    await tester.pump(const Duration(milliseconds: 50));
    final DecoratedBox box = tester.widget<DecoratedBox>(
      find
          .descendant(
            of: find.byKey(_pillKey),
            matching: find.byType(DecoratedBox),
          )
          .first,
    );
    final Color? mid = (box.decoration as BoxDecoration).color;
    expect(mid, isNot(Colors.white));
    expect(mid, isNot(const Color(0xFFE6197F)));
    await tester.pumpAndSettle();
  });

  testWidgets("press scale resets when status changes mid-press", (
    WidgetTester tester,
  ) async {
    final _Harness h = _Harness();
    await h.show(tester, _pack());
    double scale() =>
        tester
            .widget<AnimatedScale>(
              find.byKey(const ValueKey<String>("restick.download.press")),
            )
            .scale;
    final TestGesture gesture = await tester.startGesture(
      tester.getCenter(find.byKey(_downloadKey)),
    );
    await tester.pump(const Duration(milliseconds: 200));
    expect(scale(), 0.96);
    await h.show(tester, _pack(status: PackDownloadStatus.failed));
    await tester.pump();
    expect(scale(), 1);
    await gesture.cancel();
    await tester.pumpAndSettle();
  });

  testWidgets("switching to another pack resets without celebrating", (
    WidgetTester tester,
  ) async {
    final _Harness h = _Harness();
    await h.show(
      tester,
      _pack(status: PackDownloadStatus.downloading, progress: 0.5),
    );
    await tester.pump(const Duration(milliseconds: 50));
    await h.show(
      tester,
      _pack(id: "p2", status: PackDownloadStatus.downloaded),
    );
    await tester.pump(const Duration(milliseconds: 16));
    expect(find.byKey(_burstKey), findsNothing);
    await tester.pumpAndSettle();
    expect(find.text(_s.owned), findsOneWidget);
  });
}
