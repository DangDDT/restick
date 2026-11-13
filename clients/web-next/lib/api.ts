// Same-origin API client. Authentication is the owner session cookie
// (`__Host-restick_admin`, HttpOnly): no bearer token, no key in a URL, no
// localStorage. `credentials` stays same-origin so the cookie travels to /api
// and /v1 and nowhere else.

import type { ApiErrorBody } from "./types";

export class ApiError extends Error {
  status: number;
  code: string;

  constructor(status: number, message: string, code: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

interface RequestOptions {
  method?: string;
  json?: unknown;
  form?: FormData;
}

async function toApiError(res: Response): Promise<ApiError> {
  let message = `Yêu cầu thất bại (${res.status}).`;
  let code = "";
  try {
    const body = (await res.json()) as ApiErrorBody;
    if (body.error?.message) message = body.error.message;
    if (body.error?.code) code = body.error.code;
  } catch {
    // Non-JSON response body: keep the status-only message.
  }
  return new ApiError(res.status, message, code);
}

export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers();
  let body: BodyInit | undefined;
  if (options.form) {
    body = options.form;
  } else if (options.json !== undefined) {
    headers.set("Content-Type", "application/json");
    body = JSON.stringify(options.json);
  }
  const res = await fetch(path, {
    method: options.method ?? (body ? "POST" : "GET"),
    headers,
    body,
    credentials: "same-origin",
  });
  if (!res.ok) throw await toApiError(res);
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

// Error codes are the meaningful contract; the server message is a fallback.
const ERROR_TEXT: Record<string, string> = {
  bad_request: "Yêu cầu không hợp lệ.",
  unauthorized: "Phiên đăng nhập đã hết hạn. Hãy đăng nhập lại.",
  invalid_credentials: "Mật khẩu không đúng.",
  origin_not_allowed: "Trình duyệt không gửi đúng origin của dịch vụ. Kiểm tra địa chỉ đang mở.",
  not_found: "Không tìm thấy dữ liệu.",
  rate_limited: "Sai mật khẩu quá nhiều lần. Thử lại sau ít phút.",
  owner_missing: "Hệ thống chưa khởi tạo mật khẩu owner.",
  payload_too_large: "Dữ liệu hoặc tệp gửi lên vượt giới hạn cho phép.",
  invalid_content_type: "Định dạng nội dung gửi lên không hợp lệ.",
  invalid_image: "Ảnh không hợp lệ: chỉ nhận PNG/JPEG/WebP, không animation, tối đa 8 MiB và 4096 px mỗi cạnh.",
  source_missing: "Pack chưa có sheet. Chọn sheet cho pack trước khi xử lý.",
  sheet_limit: "Thiết kế đã đủ 20 sheet. Xóa bớt sheet cũ trước khi tải thêm.",
  sheet_in_use: "Sheet đang được một pack dùng nên không xóa được.",
  design_has_sheets: "Thiết kế còn sheet nên không xóa được. Xóa hết sheet trước.",
  slug_taken: "Slug này đã được dùng. Chọn slug khác.",
  job_active: "Pack đang có job chạy. Chờ job kết thúc rồi thử lại.",
  animation_busy: "Sticker này đang có animation chạy. Chờ xử lý xong rồi thử lại.",
  no_changes: "Không có thay đổi animation nào để phát hành.",
  not_ready: "Animation chưa xử lý xong nên chưa dùng được.",
  not_draft: "Chỉ pack nháp mới xóa được. Pack đã sẵn sàng phải thu hồi.",
  group_not_empty: "Nhóm còn mảnh ghép nên không xóa được.",
  pack_revoked: "Pack đã bị thu hồi; thao tác này không còn hiệu lực.",
  unprocessable: "Dữ liệu chưa hợp lệ. Kiểm tra lại các trường bắt buộc.",
  internal: "Máy chủ gặp lỗi. Thử lại sau.",
};

export function errorText(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.code && ERROR_TEXT[err.code]) return ERROR_TEXT[err.code];
    if (err.status === 401) return ERROR_TEXT.unauthorized;
    return err.message;
  }
  if (err instanceof Error) return err.message;
  return "Đã xảy ra lỗi không xác định.";
}
