import "package:flutter/services.dart";
import "package:flutter_test/flutter_test.dart";
import "package:material_ui/material_ui.dart";
import "package:restick/src/presentation/theme/restick_brand.dart";

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test("Baloo2 weights 500 600 700 800 load from the package bundle", () async {
    for (final int w in <int>[500, 600, 700, 800]) {
      final ByteData data = await rootBundle.load(
        "packages/restick/assets/fonts/Baloo2-$w.ttf",
      );
      expect(data.lengthInBytes, lessThan(100 * 1024));
      expect(data.lengthInBytes, greaterThan(10 * 1024));
    }
  });

  test("RestickBrand.text uses the SDK font family and package", () {
    final TextStyle style = RestickBrand.text(size: 20);
    // TextStyle folds `package:` into the family name.
    expect(style.fontFamily, "packages/restick/Baloo2");
    expect(style.fontWeight, FontWeight.w600);
  });

  testWidgets("Vietnamese sample renders with the bundled font", (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      Directionality(
        textDirection: TextDirection.ltr,
        child: Column(
          children: <Widget>[
            for (final FontWeight w in <FontWeight>[
              FontWeight.w500,
              FontWeight.w600,
              FontWeight.w700,
              FontWeight.w800,
            ])
              Text(
                "Mèo Cam Ngố Thỏ Bông ặ ỡ ự ẳ Đ đ",
                style: RestickBrand.text(weight: w),
              ),
          ],
        ),
      ),
    );
    expect(tester.takeException(), isNull);
  });
}
