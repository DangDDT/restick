import "package:equatable/equatable.dart";

class PackSummary extends Equatable {
  const PackSummary({
    required this.id,
    required this.slug,
    required this.name,
    required this.author,
    required this.tags,
    required this.stickerCount,
    required this.version,
    required this.status,
    required this.thumbnailUrl,
    required this.checksumSha256,
  });

  final String id;
  final String slug;
  final String name;
  final String author;
  final List<String> tags;
  final int stickerCount;
  final int version;
  final String status;
  final String thumbnailUrl;
  final String checksumSha256;

  bool get isProcessing => status == "processing";
  bool get canConfirm => status == "awaiting_confirm";

  @override
  List<Object?> get props => <Object?>[
        id,
        slug,
        name,
        author,
        tags,
        stickerCount,
        version,
        status,
        thumbnailUrl,
        checksumSha256,
      ];
}

class StickerPreview extends Equatable {
  const StickerPreview({
    required this.file,
    required this.emojiHint,
    required this.width,
    required this.height,
    required this.fileUrl,
    required this.thumbUrl,
    this.keywords = const <String>[],
  });

  final String file;
  final String emojiHint;
  final List<String> keywords;
  final int width;
  final int height;
  final String fileUrl;
  final String thumbUrl;

  @override
  List<Object?> get props =>
      <Object?>[file, emojiHint, keywords, width, height, fileUrl, thumbUrl];
}

class PackDetail extends Equatable {
  const PackDetail({
    required this.summary,
    required this.stickers,
    required this.rejectReason,
    required this.uploadPrefix,
  });

  final PackSummary summary;
  final List<StickerPreview> stickers;
  final String rejectReason;
  final String uploadPrefix;

  @override
  List<Object?> get props =>
      <Object?>[summary, stickers, rejectReason, uploadPrefix];
}

enum RejectionCode {
  copyright("copyright"),
  quality("quality"),
  duplicate("duplicate"),
  misleading("misleading"),
  explicit("explicit");

  const RejectionCode(this.code);
  final String code;

  String defaultText({bool isVietnamese = true}) {
    switch (this) {
      case RejectionCode.copyright:
        return isVietnamese
            ? "Vi phạm Bản quyền & Sở hữu Trí tuệ: Bộ sticker chứa hình ảnh/nhân vật thuộc bản quyền thương hiệu độc quyền mà chưa được cấp phép."
            : "Copyright & Trademark Violation: Contains copyrighted character IP or trademarked artwork without authorization.";
      case RejectionCode.quality:
        return isVietnamese
            ? "Chất lượng Kỹ thuật Kém: Ảnh bị mờ pixel, nét vẽ bị vỡ hoặc tách viền Alpha bị lẹm mờ không đảm bảo độ rõ nét trên nền chat."
            : "Technical & Rendering Quality Issues: Image is pixelated, low resolution, or has poor alpha edge cutout rendering.";
      case RejectionCode.duplicate:
        return isVietnamese
            ? "Nội dung Trùng lặp: Bộ sticker bị trùng 100% với một bộ sticker đã xuất bản trước đó hoặc các ảnh trong bộ gần như giống hệt nhau."
            : "Duplicate / Spam Content: Uploaded sticker pack duplicates existing published artwork or contains identical stickers.";
      case RejectionCode.misleading:
        return isVietnamese
            ? "Thông tin Sai lệch / Mạo danh: Tên bộ sticker hoặc từ khóa (tags) cố tình mạo danh thương hiệu/nghệ sĩ khác hoặc chứa từ khóa spam."
            : "Misleading Metadata / Impersonation: Title, slug, or tags falsely impersonate official brands or contain keyword spamming.";
      case RejectionCode.explicit:
        return isVietnamese
            ? "Vi phạm Tiêu chuẩn Cộng đồng: Nội dung chứa hình ảnh khiêu dâm, bạo lực, phát ngôn thù hận hoặc không phù hợp với chuẩn mực xã hội."
            : "Community Safety Violation: Contains explicit, hateful, violent, or inappropriate content violating community standards.";
    }
  }

  static RejectionCode? fromCode(String code) {
    for (final RejectionCode item in RejectionCode.values) {
      if (item.code == code) {
        return item;
      }
    }
    return null;
  }
}

class RejectionStat extends Equatable {
  const RejectionStat({
    required this.reasonCode,
    required this.rejectionCount,
    required this.lastRejectedAt,
  });

  final String reasonCode;
  final int rejectionCount;
  final String lastRejectedAt;

  @override
  List<Object?> get props =>
      <Object?>[reasonCode, rejectionCount, lastRejectedAt];
}

enum RevocationCode {
  dmca("dmca"),
  malicious("malicious"),
  authorRequest("author_request"),
  tosBreach("tos_breach");

  const RevocationCode(this.code);
  final String code;

  String defaultText({bool isVietnamese = true}) {
    switch (this) {
      case RevocationCode.dmca:
        return isVietnamese
            ? "Yêu cầu Gỡ bỏ Bản quyền DMCA: Bộ sticker bị chính chủ gửi yêu cầu tháo gỡ khẩn cấp do vi phạm bản quyền."
            : "DMCA Copyright Take-down Notice: Emergency take-down request received from copyright holder.";
      case RevocationCode.malicious:
        return isVietnamese
            ? "Nội dung Độc hại bị Báo cáo: Bộ sticker bị nhiều người dùng báo cáo chứa nội dung lừa đảo hoặc vi phạm an toàn."
            : "Reported Malicious / Abuse Content: Multiple user reports of misleading or malicious content.";
      case RevocationCode.authorRequest:
        return isVietnamese
            ? "Tác giả Yêu cầu Rút lại: Contributor chủ động gửi yêu cầu thu hồi bộ sticker của họ khỏi hệ thống."
            : "Author Self-Withdrawal Request: Contributor requested voluntary removal of their pack.";
      case RevocationCode.tosBreach:
        return isVietnamese
            ? "Vi phạm Điều khoản Dịch vụ: Vi phạm các quy định bổ sung về an toàn thông tin và tiêu chuẩn nền tảng."
            : "Terms of Service Violation: Breach of platform security, privacy, or content policies.";
    }
  }

  static RevocationCode? fromCode(String code) {
    for (final RevocationCode item in RevocationCode.values) {
      if (item.code == code) {
        return item;
      }
    }
    return null;
  }
}

class UploadFile extends Equatable {
  const UploadFile({
    required this.name,
    required this.bytes,
    required this.contentType,
  });

  final String name;
  final List<int> bytes;
  final String contentType;

  @override
  List<Object?> get props => <Object?>[name, bytes, contentType];
}
