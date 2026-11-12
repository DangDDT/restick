// DTOs mirroring contracts/openapi/v1.yaml (T8). No client-side renames: the
// wire names are used as-is so a contract change fails here first.

export interface ApiErrorBody {
  error?: { code?: string; message?: string };
}

export interface SessionInfo {
  username: string;
}

export interface Group {
  id: string;
  name: string;
  description: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface GroupDraft {
  name: string;
  description: string;
  sort_order: number;
}

export interface GroupList {
  items: Group[];
}

export type Scope = "identity" | "pack" | "sticker";

export interface Colors {
  primary: string;
  secondary: string;
  accent: string;
  line: string;
}

export interface Fragment {
  id: string;
  group_id: string;
  name: string;
  name_en: string;
  prompt: string;
  notes: string;
  source: string;
  scope: Scope;
  colors?: Colors;
  archived: boolean;
  created_at: string;
  updated_at: string;
}

export interface FragmentDraft {
  group_id: string;
  name: string;
  name_en: string;
  prompt: string;
  notes: string;
  source: string;
  scope: Scope;
  colors?: Colors;
}

export interface FragmentPage {
  items: Fragment[];
  page: number;
  page_size: number;
  total: number;
}

export interface Character {
  id: string;
  name: string;
  description: string;
  identity_notes: string;
  rights_notes: string;
  reference_asset_ids: string[];
  archived: boolean;
  created_at: string;
  updated_at: string;
}

export interface CharacterDraft {
  name: string;
  description: string;
  identity_notes: string;
  rights_notes: string;
  reference_asset_ids: string[];
}

export interface CharacterList {
  items: Character[];
}

export interface ReferenceAsset {
  id: string;
  filename: string;
  content_type: string;
  byte_size: number;
  width: number;
  height: number;
  checksum_sha256: string;
  created_at: string;
}

export interface Cell {
  action: string;
  expression: string;
  fragment_ids?: string[];
  free_text: string;
  /** Exact text the image agent writes on this sticker; omitted or empty means no text. */
  label?: string;
}

// Server fields with `omitempty` are optional here: an omitted field is a real
// wire shape, not a missing DTO.
export interface Draft {
  id?: string;
  name: string;
  character_id?: string;
  /** One reference image of the selected character; required whenever character_id is set. */
  reference_asset_id?: string;
  group_ids?: string[];
  fragment_ids?: string[];
  pinned_fragment_ids?: string[];
  palette_id?: string;
  free_text?: string;
  rows: number;
  columns: number;
  cells: Cell[];
  outline_width?: number;
  outline_color?: string;
}

export interface Selection {
  group_id: string;
  group_name: string;
  fragment_id: string;
  name: string;
  prompt: string;
  scope: Scope;
}

export interface SnapshotCell {
  row: number;
  column: number;
  action: string;
  expression: string;
  free_text: string;
  label?: string;
  fragments: Selection[];
}

export interface Snapshot {
  schema_version: number;
  character?: {
    id: string;
    name: string;
    description: string;
    identity_notes: string;
    reference_asset_id?: string;
  };
  selections: Selection[];
  palette?: {
    fragment_id: string;
    name: string;
    prompt: string;
    primary: string;
    secondary: string;
    accent: string;
    line: string;
  };
  free_text: string;
  rows: number;
  columns: number;
  outline_width: number;
  outline_color: string;
  cells: SnapshotCell[];
  compiled_at: string;
}

export interface Design {
  id: string;
  name: string;
  character_id: string;
  rows: number;
  columns: number;
  outline_width: number;
  outline_color: string;
  draft: Draft;
  snapshot: Snapshot;
  compiled_prompt: string;
  created_at: string;
  updated_at: string;
}

export interface DesignPage {
  items: Design[];
  page: number;
  page_size: number;
  total: number;
}

export interface CompiledDesign {
  compiled_prompt: string;
  snapshot: Snapshot;
}

// A design sheet is an image generated outside Restick for one design. Immutable:
// add or delete only. `image_url` is intentionally absent: the list route does
// not return it and the web builds the admin image route from the two ids.
export interface Sheet {
  id: string;
  design_id: string;
  provider: string;
  note: string;
  prompt_snapshot: string;
  filename: string;
  media_type: string;
  byte_size: number;
  width: number;
  height: number;
  checksum_sha256: string;
  created_at: string;
}

export interface SheetList {
  items: Sheet[];
}

export interface AdminPackSheet {
  id: string;
  provider: string;
  filename: string;
  width: number;
  height: number;
  byte_size: number;
  checksum_sha256: string;
  image_url: string;
}

// The admin pack detail's design summary: enough to show what the pack is of.
export interface DesignSummary {
  id: string;
  name: string;
  rows: number;
  columns: number;
  outline_width: number;
  outline_color: string;
  character_id?: string;
  compiled_prompt: string;
}

export type PackStatus = "draft" | "ready" | "revoked";

export interface AdminPack {
  id: string;
  slug: string;
  name: string;
  author: string;
  description: string;
  // Go marshals a pack with no tags as null.
  tags: string[] | null;
  design_id?: string;
  status: PackStatus;
  current_version: number;
  sticker_count: number;
  checksum_sha256: string;
  thumbnail_url: string;
  sheet_id: string;
  sheet: AdminPackSheet | null;
  created_at: string;
  updated_at: string;
}

export interface AdminPackSticker {
  sort_order: number;
  filename: string;
  checksum_sha256: string;
  width: number;
  height: number;
  emoji_hint: string;
  keywords: string[] | null;
  object_url: string;
  thumb_url: string;
}

export interface AdminPackDetail extends AdminPack {
  design?: DesignSummary;
  stickers: AdminPackSticker[];
  bundle_url: string;
  latest_job?: Job;
}

export interface AdminPackPage {
  items: AdminPack[];
  page: number;
  page_size: number;
  total: number;
}

export interface PackCreate {
  slug: string;
  name: string;
  author: string;
  description: string;
  tags: string[];
  design_id?: string;
  sheet_id: string;
}

export interface PackUpdate {
  name: string;
  author: string;
  description: string;
  tags: string[];
  design_id?: string;
  sheet_id: string;
}

export type JobStatus = "queued" | "running" | "succeeded" | "failed" | "interrupted";

export interface Job {
  id: string;
  pack_id: string;
  version: number;
  status: JobStatus;
  request: unknown;
  result: unknown;
  error_code?: string;
  error_message?: string;
  diagnostics?: string;
  created_at: string;
  started_at: string | null;
  finished_at: string | null;
}

export interface Key {
  id: string;
  name: string;
  prefix: string;
  active: boolean;
  created_at: string;
  expires_at: string | null;
  disabled_at: string | null;
  pack_count: number;
}

export interface KeyList {
  items: Key[];
}

export interface KeyPacks {
  pack_ids: string[];
}

export interface CreatedKey {
  id: string;
  name: string;
  prefix: string;
  raw: string;
}
