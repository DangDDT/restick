// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Vietnamese (`vi`).
class AppLocalizationsVi extends AppLocalizations {
  AppLocalizationsVi([String locale = 'vi']) : super(locale);

  @override
  String get appTitle => 'Restick';

  @override
  String get navContribHeader => 'Contributor';

  @override
  String get navMyPacks => 'Bộ của tôi';

  @override
  String get navUploadPack => 'Tải lên';

  @override
  String get navAdminHeader => 'Quản trị';

  @override
  String get navQueue => 'Chờ duyệt';

  @override
  String get navCatalog => 'Đã xuất bản';

  @override
  String get navAnalytics => 'Thống kê từ chối';

  @override
  String get navApiDocs => 'Tài liệu API';

  @override
  String get titleApiDocs => 'Tài liệu API';

  @override
  String get descApiDocs =>
      'Swagger của toàn bộ HTTP REST api_svc đang hỗ trợ. Dùng Authorize để gắn Bearer JWT hoặc X-Api-Key rồi thử gọi.';

  @override
  String get titleMyPacks => 'Bộ sticker của tôi';

  @override
  String get descMyPacks => 'Theo dõi trạng thái tải lên, xử lý và xuất bản.';

  @override
  String get btnUpload => 'Tải bộ mới';

  @override
  String get titleNewPack => 'Tải bộ sticker mới';

  @override
  String get descNewPack =>
      'Chọn một sheet để tự cắt, hoặc tải từng ảnh PNG/WebP.';

  @override
  String get noticeHeader => 'Lưu ý';

  @override
  String get noticeBody =>
      'Restick không tạo ảnh từ prompt. Hãy tải sticker bạn đã chuẩn bị sẵn.';

  @override
  String get labelTitle => 'Tên bộ';

  @override
  String get labelSlug => 'Mã định danh';

  @override
  String get labelAuthor => 'Tác giả';

  @override
  String get labelTags => 'Thẻ (cách nhau bằng dấu phẩy)';

  @override
  String get labelSource => 'Nguồn ảnh';

  @override
  String get sourceSheet => 'Một sheet (tự cắt sticker)';

  @override
  String get sourceFiles => 'Nhiều file riêng (.png / .webp)';

  @override
  String get labelUpload => 'Ảnh sticker';

  @override
  String get dropzoneText => 'Nhấp hoặc kéo thả ảnh vào đây';

  @override
  String get dropzoneHint => 'PNG, WEBP hoặc JPG, tối đa 15MB';

  @override
  String get btnCancel => 'Huỷ';

  @override
  String get btnSubmit => 'Gửi xử lý';

  @override
  String get sectionStickers => 'Sticker đã chuẩn hoá (WebP 512×512)';

  @override
  String get hintClickSend => 'Nhấp sticker để xem trước trong chat';

  @override
  String get bannerConfirmTitle => 'Xuất bản ngay';

  @override
  String get bannerConfirmBody =>
      'Bộ này do admin tải lên, nên có thể xuất bản thẳng mà không cần xếp hàng duyệt.';

  @override
  String get btnConfirmPublish => 'Xuất bản ngay';

  @override
  String get titleQueue => 'Hàng chờ duyệt';

  @override
  String get descQueue =>
      'Kiểm tra bản quyền và nội dung an toàn trước khi xuất bản.';

  @override
  String get btnApprove => 'Duyệt';

  @override
  String get btnReject => 'Từ chối';

  @override
  String get titleCatalog => 'Đã xuất bản';

  @override
  String get descCatalog =>
      'Các bộ đang được phục vụ cho ứng dụng dùng Flutter SDK.';

  @override
  String get btnRevoke => 'Thu hồi';

  @override
  String get titleAnalytics => 'Thống kê từ chối';

  @override
  String get descAnalytics => 'Tỉ lệ từ chối theo từng nhóm lý do.';

  @override
  String get chartTitle => 'Tỉ lệ từ chối theo nhóm';

  @override
  String get btnRefresh => 'Làm mới';

  @override
  String get deviceTitle => 'Xem trước trên thiết bị';

  @override
  String get deviceSub => 'Mô phỏng sticker trong khung chat';

  @override
  String get msgIntro =>
      'Xin chào! Đây là bản xem trước khi dùng Restick Flutter SDK.';

  @override
  String get msgPrompt => 'Nhấp sticker bên trái để gửi vào chat.';

  @override
  String get signIn => 'Đăng nhập';

  @override
  String get signUp => 'Tạo tài khoản';

  @override
  String get signOut => 'Đăng xuất';

  @override
  String get email => 'Email';

  @override
  String get password => 'Mật khẩu';

  @override
  String get displayName => 'Tên hiển thị';

  @override
  String get loginSubtitle =>
      'Không tạo ảnh từ từ khoá — tải artwork đã làm sẵn.';

  @override
  String get themeDark => 'Giao diện tối';

  @override
  String get themeLight => 'Giao diện sáng';

  @override
  String get language => 'Ngôn ngữ';

  @override
  String get modalRejectTitle => 'Từ chối bộ sticker';

  @override
  String get modalRejectBody =>
      'Cho biết lý do từ chối. Người gửi sẽ nhận được thông báo này.';

  @override
  String get modalRevokeTitle => 'Thu hồi bộ đã xuất bản';

  @override
  String get modalRevokeBody =>
      'Ứng dụng trên thiết bị sẽ gỡ bộ này. Vui lòng ghi rõ lý do.';

  @override
  String get labelQuickPresets => 'Chọn nhanh:';

  @override
  String get presetCopyright => 'Vi phạm bản quyền';

  @override
  String get presetQuality => 'Chất lượng kém / vỡ nét';

  @override
  String get presetDuplicate => 'Trùng nội dung';

  @override
  String get presetMisleading => 'Thông tin sai lệch';

  @override
  String get presetExplicit => 'Không phù hợp';

  @override
  String get presetDmca => 'Gỡ theo DMCA';

  @override
  String get presetMalicious => 'Nội dung độc hại';

  @override
  String get presetAuthorRequest => 'Tác giả rút lại';

  @override
  String get presetTos => 'Vi phạm điều khoản';

  @override
  String get labelReason => 'Lý do';

  @override
  String get btnSubmitReason => 'Gửi';

  @override
  String get publishTitle => 'Xuất bản bộ này?';

  @override
  String get publishBody =>
      'Sau khi xuất bản, sticker sẽ hiện trên Flutter SDK.';

  @override
  String get btnPublish => 'Xuất bản';

  @override
  String get emptyPacks => 'Chưa có bộ nào. Thử tải bộ mới nhé.';

  @override
  String get emptyQueue => 'Hiện không có bộ nào đang chờ duyệt.';

  @override
  String get emptyPublished => 'Chưa có bộ nào được xuất bản.';

  @override
  String get emptyAnalytics => 'Chưa có dữ liệu thống kê.';

  @override
  String get statusDraft => 'Nháp';

  @override
  String get statusProcessing => 'Đang xử lý';

  @override
  String get statusConfirm => 'Chờ xác nhận';

  @override
  String get statusPending => 'Chờ duyệt';

  @override
  String get statusPublished => 'Đã xuất bản';

  @override
  String get statusRejected => 'Bị từ chối';

  @override
  String get statusRevoked => 'Đã thu hồi';

  @override
  String filesSelected(int count) {
    return 'Đã chọn $count file';
  }

  @override
  String get chooseImages => 'Chọn ảnh';

  @override
  String get uploading => 'Đang tải lên…';

  @override
  String get signingIn => 'Đang đăng nhập…';

  @override
  String get creatingAccount => 'Đang tạo tài khoản…';

  @override
  String get roleAdmin => 'Admin';

  @override
  String get roleContributor => 'Contributor';

  @override
  String rejectReasonLabel(String reason) {
    return 'Lý do từ chối: $reason';
  }

  @override
  String packSlugMeta(String slug, int count) {
    return '$slug · $count sticker';
  }

  @override
  String get menuDelete => 'Xoá';

  @override
  String get confirmDeleteRejectedTitle => 'Xoá vĩnh viễn?';

  @override
  String get confirmDeleteRejectedBody =>
      'Bộ bị từ chối này sẽ bị xoá hẳn và không lấy lại được.';

  @override
  String get confirmRevokePublishedTitle => 'Thu hồi bộ đã xuất bản?';

  @override
  String get confirmRevokePublishedBody =>
      'Bộ sẽ biến mất khỏi danh mục công khai. Thiết bị vẫn nhận tín hiệu để gỡ bản cục bộ; lịch sử được giữ lại để đối chiếu.';

  @override
  String get btnConfirmDelete => 'Xoá vĩnh viễn';

  @override
  String get btnConfirmRevoke => 'Thu hồi';

  @override
  String get navApiKeys => 'API Keys';

  @override
  String get titleApiKeys => 'API Keys';

  @override
  String get descApiKeys =>
      'Quản lý API Key cá nhân và máy chủ để xác thực ứng dụng client và SDK.';

  @override
  String get btnCreateApiKey => 'Tạo API Key';

  @override
  String get labelKeyName => 'Tên / Nhãn API Key';

  @override
  String get labelExpiresInDays => 'Hạn sử dụng (số ngày, tùy chọn)';

  @override
  String get labelExpiresNever => 'Không bao giờ hết hạn';

  @override
  String get labelKeyPrefix => 'Tiền tố key';

  @override
  String get labelCreatedAt => 'Ngày tạo';

  @override
  String get labelExpiresAt => 'Hết hạn';

  @override
  String get labelLastUsedAt => 'Lần dùng cuối';

  @override
  String get labelNeverUsed => 'Chưa từng dùng';

  @override
  String get btnRotateKey => 'Xoay vòng key';

  @override
  String get btnDeleteKey => 'Thu hồi & Xóa';

  @override
  String get dialogCreatedKeyTitle => 'Lưu lại API Key của bạn';

  @override
  String get dialogCreatedKeyWarning =>
      'Hãy sao chép key này ngay bây giờ. Vì lý do bảo mật, bạn sẽ không thể xem lại chuỗi này sau khi đóng hộp thoại!';

  @override
  String get btnCopyKey => 'Sao chép vào bộ nhớ tạm';

  @override
  String get copiedToClipboard => 'Đã sao chép vào bộ nhớ tạm!';

  @override
  String get confirmRotateKeyTitle => 'Xoay vòng API Key này?';

  @override
  String get confirmRotateKeyBody =>
      'Key hiện tại sẽ lập tức ngừng hoạt động và một chuỗi key mới sẽ được tạo.';

  @override
  String get confirmDeleteKeyTitle => 'Xóa và thu hồi API Key này?';

  @override
  String get confirmDeleteKeyBody =>
      'Mọi ứng dụng hoặc SDK đang dùng API key này sẽ bị từ chối truy cập ngay lập tức.';

  @override
  String get emptyApiKeys =>
      'Chưa có API key nào. Nhấn \'Tạo API Key\' để cấp mới.';

  @override
  String get tabPersonalKeys => 'Key cá nhân';

  @override
  String get tabSystemKeys => 'Toàn hệ thống (Admin)';
}
