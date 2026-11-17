import "package:flutter_test/flutter_test.dart";
import "package:restick/src/core/text/fold_vietnamese.dart";

void main() {
  test("foldVietnamese strips Vietnamese diacritics and case", () {
    expect(foldVietnamese("Mèo"), "meo");
    expect(foldVietnamese("MÈO CAM NGỐ"), "meo cam ngo");
    expect(foldVietnamese("Đông đặc"), "dong dac");
    expect(foldVietnamese("Ếch ỡ ự ẳ ổ Ệ Ữ"), "ech o u a o e u");
    expect(foldVietnamese("Tết Ỳ ý"), "tet y y");
    // Decomposed input (base letter + combining marks).
    expect(foldVietnamese("Mèo ế"), "meo e");
    expect(foldVietnamese("plain 123"), "plain 123");
  });
}
