import "package:restick/src/core/text/fold_vietnamese.dart";
import "package:restick/src/domain/entities/sticker_asset.dart";

final RegExp _nonWord = RegExp(r"[^\p{L}\p{N}]+", unicode: true);

String _words(String value) =>
    " ${foldVietnamese(value).replaceAll(_nonWord, " ").trim()} ";

/// Stickers whose emoji hint appears in [text] (score 2) or whose keyword
/// matches whole words of it, ignoring case and Vietnamese diacritics
/// (score 1). Best matches first, at most [limit].
List<StickerAsset> suggestStickersFor(
  Iterable<StickerAsset> stickers,
  String text, {
  int limit = 12,
}) {
  if (text.trim().isEmpty) return const <StickerAsset>[];
  final String words = _words(text);
  final List<(StickerAsset, int)> scored = <(StickerAsset, int)>[];
  for (final StickerAsset sticker in stickers) {
    final int score =
        sticker.emojiHint.isNotEmpty && text.contains(sticker.emojiHint)
        ? 2
        : sticker.keywords.any((String k) {
            final String keyword = _words(k);
            return keyword.trim().isNotEmpty && words.contains(keyword);
          })
        ? 1
        : 0;
    if (score > 0) scored.add((sticker, score));
  }
  // List.sort is not stable; the index keeps pack order within a score.
  final List<int> order = List<int>.generate(scored.length, (int i) => i)
    ..sort((int a, int b) {
      final int byScore = scored[b].$2.compareTo(scored[a].$2);
      return byScore != 0 ? byScore : a.compareTo(b);
    });
  return <StickerAsset>[for (final int i in order.take(limit)) scored[i].$1];
}
