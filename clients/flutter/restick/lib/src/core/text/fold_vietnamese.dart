const Map<String, String> _groups = <String, String>{
  "a": "àáạảãâầấậẩẫăằắặẳẵ",
  "e": "èéẹẻẽêềếệểễ",
  "i": "ìíịỉĩ",
  "o": "òóọỏõôồốộổỗơờớợởỡ",
  "u": "ùúụủũưừứựửữ",
  "y": "ỳýỵỷỹ",
  "d": "đ",
};

final Map<int, String> _base = <int, String>{
  for (final MapEntry<String, String> group in _groups.entries)
    for (final int rune in group.value.runes) rune: group.key,
};

/// Lowercases [value] and strips Vietnamese diacritics ("Mèo" -> "meo",
/// "Đ" -> "d"), for precomposed and decomposed (combining mark) input.
String foldVietnamese(String value) {
  final StringBuffer out = StringBuffer();
  for (final int rune in value.toLowerCase().runes) {
    if (rune >= 0x300 && rune <= 0x36F) {
      continue; // Combining diacritical marks.
    }
    out.write(_base[rune] ?? String.fromCharCode(rune));
  }
  return out.toString();
}
