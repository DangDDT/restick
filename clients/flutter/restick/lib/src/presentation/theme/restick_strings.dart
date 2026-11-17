import "package:material_ui/material_ui.dart";

/// UI copy of the marketplace. Pass a custom instance to translate or reword.
@immutable
class RestickStrings {
  const RestickStrings({
    required this.title,
    required this.tagline,
    required this.download,
    required this.downloading,
    required this.waiting,
    required this.retry,
    required this.owned,
    required this.remove,
    required this.newShelf,
    required this.featured,
    required this.searchHint,
    required this.bearGreeting,
    required this.emptyTitle,
    required this.errorTitle,
    required this.pluralizeStickers,
    required this.downloadAllFormat,
    required this.a11yDownloadingFormat,
    required this.a11yDoneFormat,
    required this.a11yFailedFormat,
    this.readyInCollection = "Sẵn sàng trong Của tôi",
  });

  static const RestickStrings vi = RestickStrings(
    title: "Cửa hàng sticker",
    tagline: "Chọn bộ sticker bạn thích",
    download: "Tải về",
    downloading: "Đang tải",
    waiting: "Đang chờ…",
    retry: "Thử lại",
    owned: "Đã có",
    remove: "Gỡ",
    newShelf: "Vừa lên kệ",
    featured: "Nổi bật",
    searchHint: "Tìm bộ sticker",
    bearGreeting: "Chào bạn, hôm nay chọn bộ nào nhỉ?",
    emptyTitle: "Chưa có bộ sticker nào",
    errorTitle: "Không tải được cửa hàng",
    pluralizeStickers: false,
    downloadAllFormat: "Tải về {count}",
    a11yDownloadingFormat: "Đang tải {name}, {percent} phần trăm",
    a11yDoneFormat: "Đã tải xong {name}",
    a11yFailedFormat: "Tải {name} thất bại, chạm để thử lại",
  );

  static const RestickStrings en = RestickStrings(
    title: "Sticker store",
    tagline: "Pick the sticker packs you love",
    download: "Download",
    downloading: "Downloading",
    waiting: "Waiting…",
    retry: "Retry",
    owned: "Owned",
    remove: "Remove",
    newShelf: "New on the shelf",
    featured: "Featured",
    searchHint: "Search sticker packs",
    bearGreeting: "Hi there, which pack today?",
    emptyTitle: "No sticker packs yet",
    errorTitle: "Could not load the store",
    pluralizeStickers: true,
    downloadAllFormat: "Download {count}",
    a11yDownloadingFormat: "Downloading {name}, {percent} percent",
    a11yDoneFormat: "{name} downloaded",
    a11yFailedFormat: "Downloading {name} failed, tap to retry",
    readyInCollection: "Ready in My collection",
  );

  final String title;
  final String tagline;
  final String download;
  final String downloading;
  final String waiting;
  final String retry;
  final String owned;
  final String remove;
  final String newShelf;
  final String featured;
  final String searchHint;
  final String bearGreeting;
  final String emptyTitle;
  final String errorTitle;
  final String readyInCollection;

  /// English inflects "sticker"/"stickers"; Vietnamese never does.
  final bool pluralizeStickers;
  final String downloadAllFormat;
  final String a11yDownloadingFormat;
  final String a11yDoneFormat;
  final String a11yFailedFormat;

  String stickerCount(int count) {
    return pluralizeStickers && count != 1
        ? "$count stickers"
        : "$count sticker";
  }

  String downloadAllLabel(int count) =>
      downloadAllFormat.replaceFirst("{count}", stickerCount(count));

  String packMeta(int count, String author) =>
      "${stickerCount(count)} · $author";

  String a11yDownloading(String name, int percent) => a11yDownloadingFormat
      .replaceFirst("{name}", name)
      .replaceFirst("{percent}", "$percent");

  String a11yDone(String name) => a11yDoneFormat.replaceFirst("{name}", name);

  String a11yFailed(String name) =>
      a11yFailedFormat.replaceFirst("{name}", name);
}
