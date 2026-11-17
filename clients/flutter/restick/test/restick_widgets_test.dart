import "package:flutter/scheduler.dart";
import "package:flutter_test/flutter_test.dart";
import "package:material_ui/material_ui.dart";
import "package:restick/src/presentation/widgets/die_cut_sticker.dart";
import "package:restick/src/presentation/widgets/pack_skeleton.dart";
import "package:restick/src/presentation/widgets/restick_mascot.dart";

Widget _host(Widget child, {bool disableAnimations = false}) => MaterialApp(
  home: MediaQuery(
    data: MediaQueryData(disableAnimations: disableAnimations),
    child: Scaffold(body: Center(child: child)),
  ),
);

void main() {
  test("mascotEyeOpenness is one smooth blink", () {
    expect(mascotEyeOpenness(0), 1.0);
    expect(mascotEyeOpenness(0.95), 1.0);
    expect(mascotEyeOpenness(0.96), 1.0);
    expect(mascotEyeOpenness(0.98), 0.1);
    expect(mascotEyeOpenness(0.99), allOf(greaterThan(0.1), lessThan(1.0)));
    expect(mascotEyeOpenness(1), 1.0);
    expect(mascotEyeOpenness(0.97), lessThan(mascotEyeOpenness(0.965)));
    expect(mascotEyeOpenness(0.985), greaterThan(mascotEyeOpenness(0.98)));
    expect(mascotEyeOpenness(0.995), greaterThan(mascotEyeOpenness(0.99)));
  });

  testWidgets("RestickMascot paints without error at size 36 and 96", (
    WidgetTester tester,
  ) async {
    for (final double size in <double>[36, 96]) {
      for (final RestickMood mood in RestickMood.values) {
        await tester.pumpWidget(_host(RestickMascot(size: size, mood: mood)));
        await tester.pump(const Duration(milliseconds: 500));
      }
    }
    expect(tester.takeException(), isNull);
  });

  testWidgets("RestickMascot does not schedule frames when disableAnimations", (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      _host(const RestickMascot(size: 96), disableAnimations: true),
    );
    await tester.pump();
    expect(SchedulerBinding.instance.hasScheduledFrame, isFalse);
    expect(tester.hasRunningAnimations, isFalse);
  });

  testWidgets("DieCutSticker shows placeholder when url empty or load fails", (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(_host(const DieCutSticker(url: "", size: 64)));
    expect(find.byIcon(Icons.emoji_emotions_outlined), findsOneWidget);

    await tester.pumpWidget(
      _host(const DieCutSticker(url: "http://127.0.0.1:1/x.png", size: 64)),
    );
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 200)),
    );
    await tester.pump();
    expect(find.byIcon(Icons.emoji_emotions_outlined), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets("PackSkeleton stops animating when TickerMode disabled", (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(_host(const PackSkeleton(width: 100, height: 60)));
    await tester.pump();
    expect(tester.hasRunningAnimations, isTrue);

    await tester.pumpWidget(
      _host(
        const TickerMode(
          enabled: false,
          child: PackSkeleton(width: 100, height: 60),
        ),
      ),
    );
    await tester.pump();
    expect(tester.hasRunningAnimations, isFalse);
  });
}
