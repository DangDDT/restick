export interface User {
  id: string;
  email: string;
  display_name: string;
  role: "contributor" | "admin";
}

export interface TokenResponse {
  access_token: string;
  expires_in: number;
  user: User;
}

export interface PackSummary {
  id: string;
  slug: string;
  name: string;
  author: string;
  tags: string[];
  sticker_count: number;
  version: number;
  status: string;
  thumbnail_url: string;
  checksum_sha256: string;
}

export interface Sticker {
  file: string;
  emoji_hint: string;
  keywords: string[];
  width: number;
  height: number;
  file_url: string;
  thumb_url: string;
}

export interface PackDetail extends PackSummary {
  stickers: Sticker[];
  reject_reason: string;
  upload_prefix: string;
}

export interface PackListResponse {
  items: PackSummary[];
}

export interface CreatePackResponse {
  id: string;
  upload_prefix: string;
  status: string;
}

export interface APIKey {
  id: string;
  user_id?: string | null;
  user_email?: string;
  name: string;
  key_prefix: string;
  is_active: boolean;
  expires_at?: string | null;
  last_used_at?: string | null;
  created_at: string;
}

export interface CreatedAPIKey extends APIKey {
  raw_key: string;
}

export interface APIKeyListResponse {
  items: APIKey[];
}

export interface RejectionStat {
  reason_code: string;
  rejection_count: number;
  last_rejected_at?: string;
}

export interface RejectionAnalyticsResponse {
  analytics: RejectionStat[];
}

export interface ApiErrorBody {
  error?: {
    code?: string;
    message?: string;
  };
}
