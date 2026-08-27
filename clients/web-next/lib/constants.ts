export const REJECTION_CODES = [
  "copyright",
  "quality",
  "duplicate",
  "misleading",
  "explicit",
] as const;

export const REVOCATION_CODES = [
  "dmca",
  "malicious",
  "author_request",
  "tos_breach",
] as const;

export const REJECTION_LABELS: Record<string, string> = {
  copyright: "Copyright / Trademark",
  quality: "Technical Quality",
  duplicate: "Duplicate / Spam",
  misleading: "Misleading Metadata",
  explicit: "Community Safety",
};

export const REVOCATION_LABELS: Record<string, string> = {
  dmca: "DMCA Take-down",
  malicious: "Reported Malicious",
  author_request: "Author Withdrawal",
  tos_breach: "Terms of Service",
};

export const REJECTION_DEFAULT_TEXT: Record<string, string> = {
  copyright:
    "Vi phạm Bản quyền & Sở hữu Trí tuệ: Bộ sticker chứa hình ảnh/nhân vật thuộc bản quyền thương hiệu độc quyền mà chưa được cấp phép.",
  quality:
    "Chất lượng Kỹ thuật Kém: Ảnh bị mờ pixel, nét vẽ bị vỡ hoặc tách viền Alpha bị lẹm mờ không đảm bảo độ rõ nét trên nền chat.",
  duplicate:
    "Nội dung Trùng lặp: Bộ sticker bị trùng 100% với một bộ sticker đã xuất bản trước đó hoặc các ảnh trong bộ gần như giống hệt nhau.",
  misleading:
    "Thông tin Sai lệch / Mạo danh: Tên bộ sticker hoặc từ khóa (tags) cố tình mạo danh thương hiệu/nghệ sĩ khác hoặc chứa từ khóa spam.",
  explicit:
    "Vi phạm Tiêu chuẩn Cộng đồng: Nội dung chứa hình ảnh khiêu dâm, bạo lực, phát ngôn thù hận hoặc không phù hợp với chuẩn mực xã hội.",
};

export const REVOCATION_DEFAULT_TEXT: Record<string, string> = {
  dmca: "Yêu cầu Gỡ bỏ Bản quyền DMCA: Bộ sticker bị chính chủ gửi yêu cầu tháo gỡ khẩn cấp do vi phạm bản quyền.",
  malicious:
    "Nội dung Độc hại bị Báo cáo: Bộ sticker bị nhiều người dùng báo cáo chứa nội dung lừa đảo hoặc vi phạm an toàn.",
  author_request:
    "Tác giả Yêu cầu Rút lại: Contributor chủ động gửi yêu cầu thu hồi bộ sticker của họ khỏi hệ thống.",
  tos_breach:
    "Vi phạm Điều khoản Dịch vụ: Vi phạm các quy định bổ sung về an toàn thông tin và tiêu chuẩn nền tảng.",
};

export function statusTone(status: string): "green" | "amber" | "red" | "gray" | "blue" {
  switch (status) {
    case "published":
      return "green";
    case "processing":
    case "awaiting_confirm":
      return "blue";
    case "rejected":
    case "revoked":
      return "red";
    case "draft":
      return "amber";
    default:
      return "gray";
  }
}
