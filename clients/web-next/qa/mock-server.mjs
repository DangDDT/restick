// QA mock for the Restick studio. Serves a `next export` directory and an
// in-memory copy of the admin REST API (clients/web-next/lib/types.ts). It is
// test tooling only: it never talks to a real instance.
//
// Usage: node qa/mock-server.mjs [staticDir] [port]
//   env: STATIC_DIR (default ../out), PORT (default 4500)
//   login password: "test"; session cookie: restick_admin
//   control: POST /__mock/scenario {"name":"default|empty|error|slow|loggedout"}
//            (switching resets the in-memory data)

import http from "node:http";
import { createHash, randomBytes } from "node:crypto";
import { existsSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { deflateSync } from "node:zlib";

const here = path.dirname(fileURLToPath(import.meta.url));
const STATIC_DIR = path.resolve(process.argv[2] || process.env.STATIC_DIR || path.join(here, "../out"));
const PORT = Number(process.argv[3] || process.env.PORT || 4500);
const PASSWORD = "test";
const COOKIE = "restick_admin";
const SCENARIOS = ["default", "empty", "error", "slow", "loggedout"];

const MIME = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".mjs": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".txt": "text/plain; charset=utf-8",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".webp": "image/webp",
  ".ico": "image/x-icon",
  ".woff2": "font/woff2",
  ".woff": "font/woff",
  ".yaml": "text/yaml; charset=utf-8",
};

// ---------- deterministic helpers ----------

const sha = (s) => createHash("sha256").update(s).digest("hex");
const BASE_TIME = Date.parse("2026-10-01T08:00:00Z");
const ts = (offsetMin) => new Date(BASE_TIME + offsetMin * 60_000).toISOString();

function rng(str) {
  let s = 0;
  for (const c of str) s = (Math.imul(s, 31) + c.charCodeAt(0)) | 0;
  return () => {
    s = (s + 0x6d2b79f5) | 0;
    let t = Math.imul(s ^ (s >>> 15), 1 | s);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

const PALETTE = ["#FF7A59", "#FFC145", "#4CC9F0", "#7BD389", "#B388EB", "#FF6B9D", "#5B8DEF", "#F4A261"];

// A real PNG for the white-reference route, so the browser clipboard path in
// copy-reference-image.ts sees image/png like the Go service does.
const CRC_TABLE = (() => {
  const table = new Int32Array(256);
  for (let n = 0; n < 256; n++) {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    table[n] = c;
  }
  return table;
})();
function crc32(buf) {
  let c = -1;
  for (const byte of buf) c = CRC_TABLE[(c ^ byte) & 0xff] ^ (c >>> 8);
  return (c ^ -1) >>> 0;
}
function pngChunk(type, data) {
  const name = Buffer.from(type, "ascii");
  const length = Buffer.alloc(4);
  length.writeUInt32BE(data.length);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(Buffer.concat([name, data])));
  return Buffer.concat([length, name, data, crc]);
}
function onWhitePng(seed) {
  const size = 128;
  const color = PALETTE[Math.floor(rng(seed)() * PALETTE.length)];
  const [r, g, b] = [1, 3, 5].map((i) => Number.parseInt(color.slice(i, i + 2), 16));
  const rows = [];
  for (let y = 0; y < size; y++) {
    const row = Buffer.alloc(1 + size * 4);
    for (let x = 0; x < size; x++) {
      const i = 1 + x * 4;
      const inside = (x - 63.5) ** 2 + (y - 63.5) ** 2 < 52 ** 2;
      row[i] = inside ? r : 255;
      row[i + 1] = inside ? g : 255;
      row[i + 2] = inside ? b : 255;
      row[i + 3] = 255;
    }
    rows.push(row);
  }
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(size, 0);
  ihdr.writeUInt32BE(size, 4);
  ihdr[8] = 8;
  ihdr[9] = 6;
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    pngChunk("IHDR", ihdr),
    pngChunk("IDAT", deflateSync(Buffer.concat(rows))),
    pngChunk("IEND", Buffer.alloc(0)),
  ]);
}

// A sticker-like blob: smooth closed curve, thick white border, dark outline,
// simple face. `variant` changes the mouth/eyes so stickers differ.
function stickerSvg(seed, { bg = false, color } = {}) {
  const rand = rng(seed);
  const fill = color || PALETTE[Math.floor(rand() * PALETTE.length)];
  const cx = 256, cy = 256, R = 170, n = 7;
  const pts = [];
  for (let i = 0; i < n; i++) {
    const a = (i / n) * Math.PI * 2;
    const r = R * (0.82 + 0.18 * rand());
    pts.push([cx + Math.cos(a) * r, cy + Math.sin(a) * r]);
  }
  let d = `M${pts[0][0].toFixed(1)} ${pts[0][1].toFixed(1)}`;
  for (let i = 0; i < n; i++) {
    const p0 = pts[(i - 1 + n) % n], p1 = pts[i], p2 = pts[(i + 1) % n], p3 = pts[(i + 2) % n];
    const c1 = [p1[0] + (p2[0] - p0[0]) / 6, p1[1] + (p2[1] - p0[1]) / 6];
    const c2 = [p2[0] - (p3[0] - p1[0]) / 6, p2[1] - (p3[1] - p1[1]) / 6];
    d += ` C${c1[0].toFixed(1)} ${c1[1].toFixed(1)} ${c2[0].toFixed(1)} ${c2[1].toFixed(1)} ${p2[0].toFixed(1)} ${p2[1].toFixed(1)}`;
  }
  d += " Z";
  const mouthKind = Math.floor(rand() * 3);
  const mouth =
    mouthKind === 0
      ? `<path d="M206 300 Q256 350 306 300" fill="none" stroke="#1f2937" stroke-width="14" stroke-linecap="round"/>`
      : mouthKind === 1
        ? `<ellipse cx="256" cy="312" rx="26" ry="18" fill="#1f2937"/>`
        : `<path d="M214 306 Q234 326 256 306 Q278 326 298 306" fill="none" stroke="#1f2937" stroke-width="12" stroke-linecap="round"/>`;
  const bgRect = bg
    ? `<rect width="512" height="512" rx="72" fill="${PALETTE[(Math.floor(rand() * 8) + 2) % 8]}" opacity="0.35"/>`
    : "";
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" width="512" height="512">
${bgRect}
<path d="${d}" fill="#ffffff" stroke="#ffffff" stroke-width="36" stroke-linejoin="round"/>
<path d="${d}" fill="${fill}" stroke="#1f2937" stroke-width="10" stroke-linejoin="round"/>
<ellipse cx="200" cy="215" rx="32" ry="40" fill="#ffffff"/><ellipse cx="312" cy="215" rx="32" ry="40" fill="#ffffff"/>
<circle cx="208" cy="222" r="14" fill="#1f2937"/><circle cx="320" cy="222" r="14" fill="#1f2937"/>
<ellipse cx="160" cy="278" rx="22" ry="13" fill="#FF8FA3" opacity="0.8"/><ellipse cx="352" cy="278" rx="22" ry="13" fill="#FF8FA3" opacity="0.8"/>
${mouth}
<ellipse cx="190" cy="150" rx="34" ry="18" fill="#ffffff" opacity="0.55" transform="rotate(-25 190 150)"/>
</svg>`;
}

// ---------- fixture data ----------

const GROUP_DEFS = [
  { id: "g-face", name: "Khuôn mặt", description: "Hình dáng khuôn mặt và đường nét cơ bản.", scope: "identity" },
  { id: "g-hair", name: "Kiểu tóc", description: "Các kiểu tóc cho nhân vật.", scope: "identity" },
  { id: "g-outfit", name: "Trang phục", description: "Quần áo dùng trong pack.", scope: "pack" },
  { id: "g-expr", name: "Biểu cảm", description: "Cảm xúc cho từng sticker.", scope: "sticker" },
  { id: "g-pose", name: "Tư thế", description: "Dáng đứng, động tác của nhân vật.", scope: "sticker" },
  { id: "g-scene", name: "Bối cảnh", description: "Bối cảnh nền cho cả pack.", scope: "pack" },
  { id: "g-style", name: "Phong cách vẽ", description: "Kỹ thuật vẽ và chất liệu.", scope: "pack" },
  { id: "g-palette", name: "Bảng màu", description: "Bộ bốn màu chủ đạo của pack.", scope: "pack" },
];

const FRAGMENT_NAMES = {
  "g-face": [
    ["Mặt tròn", "Round face"], ["Mặt trái xoan", "Oval face"], ["Má phúng phính", "Chubby cheeks"],
    ["Mắt to tròn", "Big round eyes"], ["Mắt híp cười", "Smiling squint eyes"], ["Mũi chấm nhỏ", "Tiny dot nose"],
    ["Miệng chữ w", "W-shaped mouth"], ["Tàn nhang nhẹ", "Light freckles"],
  ],
  "g-hair": [
    ["Tóc bob ngắn", "Short bob hair"], ["Tóc đuôi ngựa", "Ponytail"], ["Tóc xoăn bồng", "Fluffy curly hair"],
    ["Tóc mái bằng", "Blunt bangs"], ["Tóc búi cao", "High bun"], ["Tóc dài thẳng", "Long straight hair"],
    ["Tóc ngắn cua", "Buzz cut"], ["Tóc hai bím", "Twin braids"],
  ],
  "g-outfit": [
    ["Áo hoodie", "Hoodie"], ["Áo dài", "Ao dai"], ["Váy đầm", "Dress"], ["Áo len cổ lọ", "Turtleneck sweater"],
    ["Áo sơ mi kẻ", "Striped shirt"], ["Yếm bò", "Denim overalls"], ["Áo khoác phao", "Puffer jacket"],
    ["Đồng phục học sinh", "School uniform"],
  ],
  "g-expr": [
    ["Vui vẻ", "Happy"], ["Buồn bã", "Sad"], ["Giận dữ", "Angry"], ["Ngạc nhiên", "Surprised"],
    ["Ngại ngùng", "Shy"], ["Yêu đời", "Joyful"], ["Ngủ gật", "Sleepy"], ["Khóc nức nở", "Crying"],
  ],
  "g-pose": [
    ["Vẫy tay", "Waving"], ["Chắp tay cảm ơn", "Thank you bow"], ["Nhảy múa", "Dancing"], ["Ngồi ôm gối", "Hugging knees"],
    ["Giơ ngón cái", "Thumbs up"], ["Chạy nhanh", "Running"], ["Ôm tim", "Heart hands"], ["Chỉ tay", "Pointing"],
  ],
  "g-scene": [
    ["Phố cổ Hà Nội", "Hanoi old quarter"], ["Bãi biển", "Beach"], ["Cánh đồng lúa", "Rice field"],
    ["Lớp học", "Classroom"], ["Quán cà phê", "Coffee shop"], ["Sân trường mùa thu", "Autumn schoolyard"],
    ["Chợ Tết", "Tet market"], ["Vũ trụ sao", "Starry space"],
  ],
  "g-style": [
    ["Tranh màu nước", "Watercolor"], ["Nét mực tối giản", "Minimal ink line"], ["Pixel art", "Pixel art"],
    ["Sticker 3D bóng", "Glossy 3D sticker"], ["Hoạt hình chibi", "Chibi anime"], ["Vẽ chì phác thảo", "Pencil sketch"],
    ["Flat vector", "Flat vector"], ["Tranh sơn dầu", "Oil painting"],
  ],
};

const PALETTES = [
  ["Bảng màu Xuân", "Spring palette", ["#F9C6D3", "#8BCB94", "#FFF4C2", "#5D8F6B"]],
  ["Bảng màu Hạ", "Summer palette", ["#FF7A59", "#FFC145", "#4CC9F0", "#1B4965"]],
  ["Bảng màu Thu", "Autumn palette", ["#C8553D", "#F4A261", "#E9C46A", "#432818"]],
  ["Bảng màu Đông", "Winter palette", ["#CAF0F8", "#90E0EF", "#0077B6", "#03045E"]],
];

function buildFragments() {
  const out = [];
  let idx = 0;
  for (const g of GROUP_DEFS) {
    if (g.id === "g-palette") {
      PALETTES.forEach(([vi, en, [p, s, a, l]], i) => {
        out.push({
          id: `f-pal-${i + 1}`, group_id: g.id, name: vi, name_en: en,
          prompt: `Color palette: primary ${p}, secondary ${s}, accent ${a}, line ${l}`,
          notes: "Dùng cho toàn bộ pack.", source: "Bộ màu nội bộ",
          scope: g.scope, colors: { primary: p, secondary: s, accent: a, line: l },
          archived: false, created_at: ts(idx), updated_at: ts(idx + 5),
        });
        idx++;
      });
      continue;
    }
    FRAGMENT_NAMES[g.id].forEach(([vi, en], i) => {
      idx++;
      out.push({
        id: `f-${g.id.slice(2)}-${i + 1}`, group_id: g.id, name: vi, name_en: en,
        prompt: `Chibi sticker character, ${en.toLowerCase()}, clean dark outline, flat colors`,
        notes: i % 3 === 0 ? "Ưu tiên cho nhân vật chính." : "", source: "Thư viện Restick v2",
        scope: g.scope, archived: g.id === "g-face" && i === 7 || g.id === "g-hair" && i === 7,
        created_at: ts(idx), updated_at: ts(idx + 20),
      });
    });
  }
  return out;
}

function buildGroups() {
  return GROUP_DEFS.map((g, i) => ({
    id: g.id, name: g.name, description: g.description, sort_order: i + 1,
    created_at: ts(0), updated_at: ts(i),
  }));
}

function buildAssets() {
  return [
    { id: "ref-mi", filename: "mi-reference.svg", svg: stickerSvg("mi-ref", { color: "#FFC145" }), width: 512, height: 512 },
    { id: "ref-bong", filename: "gau-bong-ref.svg", svg: stickerSvg("bong-ref", { color: "#B388EB" }), width: 512, height: 512 },
    { id: "ref-bo", filename: "cun-bo-ref.svg", svg: stickerSvg("bo-ref", { color: "#4CC9F0" }), width: 512, height: 512 },
  ].map((a, i) => ({
    id: a.id, filename: a.filename, content_type: "image/svg+xml",
    byte_size: Buffer.byteLength(a.svg), width: a.width, height: a.height,
    checksum_sha256: sha(a.svg), created_at: ts(i), data: Buffer.from(a.svg),
  }));
}

function buildCharacters() {
  return [
    { id: "char-mi", name: "Mèo Mi", description: "Mèo vàng tinh nghịch, hay cười híp mắt.", identity_notes: "Lông vàng, đuôi dài, luôn đeo nơ đỏ.", rights_notes: "Nội bộ Restick. Không dùng cho sản phẩm bên ngoài.", reference_asset_ids: ["ref-mi"], archived: false, created_at: ts(0), updated_at: ts(10) },
    { id: "char-bong", name: "Gấu Bông", description: "Gấu tím hiền lành, thích ôm.", identity_notes: "Thân tròn, tai nhỏ, màu tím pastel.", rights_notes: "Đã có giấy phép sử dụng đến 2027.", reference_asset_ids: ["ref-bong"], archived: false, created_at: ts(1), updated_at: ts(11) },
    { id: "char-bo", name: "Cún Bơ", description: "Cún xanh năng động, hay đi học.", identity_notes: "Tai cụp, áo đồng phục xanh.", rights_notes: "Chờ xác nhận quyền từ đối tác.", reference_asset_ids: ["ref-bo"], archived: true, created_at: ts(2), updated_at: ts(12) },
    { id: "char-tho", name: "Thỏ Tí", description: "Thỏ nhỏ nhút nhát, chưa có ảnh tham chiếu.", identity_notes: "Tai dài, má hồng.", rights_notes: "", reference_asset_ids: [], archived: false, created_at: ts(3), updated_at: ts(13) },
    { id: "char-old", name: "Hồ sơ cũ không ảnh", description: "Nhân vật lịch sử đã lưu trữ.", identity_notes: "Giữ prompt cũ.", rights_notes: "", reference_asset_ids: [], archived: true, created_at: ts(4), updated_at: ts(14) },
  ];
}

function snapshotSelection(fragment, groupsById) {
  return {
    group_id: fragment.group_id,
    group_name: groupsById.get(fragment.group_id)?.name ?? "",
    fragment_id: fragment.id,
    name: fragment.name,
    prompt: fragment.prompt,
    scope: fragment.scope,
  };
}

function buildDesigns(fragments, groups, characters) {
  const byId = new Map(fragments.map((f) => [f.id, f]));
  const gById = new Map(groups.map((g) => [g.id, g]));
  const mk = (id, name, charId, draftPart, minute) => {
    const draft = {
      id, name, character_id: charId, rows: draftPart.rows, columns: draftPart.columns,
      reference_asset_id: draftPart.reference_asset_id ?? characters.find((c) => c.id === charId)?.reference_asset_ids[0],
      group_ids: draftPart.group_ids, fragment_ids: draftPart.fragment_ids,
      pinned_fragment_ids: draftPart.pinned_fragment_ids ?? [], palette_id: draftPart.palette_id,
      free_text: draftPart.free_text, outline_width: 12, outline_color: "#1F2937",
      cells: draftPart.cells.map((cell) => ({ ...cell, label: cell.label ?? "" })),
    };
    return { id, name, character_id: charId, rows: draft.rows, columns: draft.columns, outline_width: 12, outline_color: "#1F2937", draft, ...compileDesign(draft, byId, gById, characters), created_at: ts(minute), updated_at: ts(minute + 30) };
  };
  return [
    mk("design-1", "Mèo Mi – chào buổi sáng", "char-mi", {
      rows: 2, columns: 2, group_ids: ["g-face", "g-hair", "g-expr"],
      fragment_ids: ["f-face-1", "f-face-4", "f-hair-2", "f-expr-1"], pinned_fragment_ids: ["f-face-4"],
      palette_id: "f-pal-1", free_text: "Nét vẽ dày, ánh sáng buổi sáng.",
      cells: [
        { action: "Vẫy tay", expression: "Vui vẻ", fragment_ids: ["f-expr-1"], free_text: "" },
        { action: "Ngồi ôm gối", expression: "Ngại ngùng", fragment_ids: ["f-expr-5"], free_text: "" },
        { action: "Nhảy múa", expression: "Yêu đời", fragment_ids: ["f-expr-6"], free_text: "" },
        { action: "Ôm tim", expression: "Vui vẻ", fragment_ids: [], free_text: "Thêm ánh sao nhỏ" },
      ],
    }, 0),
    mk("design-2", "Gấu Bông ngày Tết", "char-bong", {
      rows: 1, columns: 3, group_ids: ["g-face", "g-scene"], fragment_ids: ["f-face-3", "f-scene-7"],
      palette_id: "f-pal-3", free_text: "Màu đỏ vàng ngày Tết.",
      cells: [
        { action: "Chắp tay cảm ơn", expression: "Vui vẻ", fragment_ids: ["f-pose-2"], free_text: "" },
        { action: "Giơ ngón cái", expression: "Yêu đời", fragment_ids: [], free_text: "" },
        { action: "Vẫy tay", expression: "Ngạc nhiên", fragment_ids: [], free_text: "" },
      ],
    }, 60),
    mk("design-3", "Cún Bơ đi học", "char-bo", {
      rows: 2, columns: 2, group_ids: ["g-outfit", "g-scene"], fragment_ids: ["f-outfit-8", "f-scene-4"],
      palette_id: "", free_text: "",
      cells: [
        { action: "Chạy nhanh", expression: "Vui vẻ", fragment_ids: [], free_text: "" },
        { action: "Chỉ tay", expression: "Giận dữ", fragment_ids: [], free_text: "" },
        { action: "Ngủ gật", expression: "Ngủ gật", fragment_ids: [], free_text: "" },
        { action: "", expression: "", fragment_ids: [], free_text: "" },
      ],
    }, 120),
    mk("design-food", "Ba món Việt có chữ", "", {
      rows: 1, columns: 3, group_ids: [], fragment_ids: [], palette_id: "f-pal-1", free_text: "Phong cách sticker đồng bộ, giữ màu tự nhiên của món ăn.",
      cells: [
        { action: "", expression: "", fragment_ids: [], free_text: "Bowl of spicy beef noodle soup", label: "Bún bò Huế" },
        { action: "", expression: "", fragment_ids: [], free_text: "Bowl of flat rice noodles and herbs", label: "Phở Hà Nội" },
        { action: "", expression: "", fragment_ids: [], free_text: "Crisp golden rice pancake with vegetables", label: "Bánh xèo miền Tây" },
      ],
    }, 150),
    mk("design-legacy", "Thiết kế cũ chưa chọn ảnh", "char-mi", {
      rows: 1, columns: 1, group_ids: [], fragment_ids: [], palette_id: "", reference_asset_id: "", free_text: "",
      cells: [{ action: "Vẫy tay", expression: "", fragment_ids: [], free_text: "" }],
    }, 180),
    mk("design-archived-empty", "Hồ sơ lưu trữ không ảnh", "char-old", {
      rows: 1, columns: 1, group_ids: [], fragment_ids: [], palette_id: "", reference_asset_id: "", free_text: "",
      cells: [{ action: "Vẫy tay", expression: "", fragment_ids: [], free_text: "" }],
    }, 210),
  ];
}

// Pure function of the draft, so suggest/compile/save agree with each other.
function compileDesign(draft, byId, gById, characters) {
  const selections = (draft.fragment_ids ?? []).map((id) => byId.get(id)).filter(Boolean).map((f) => snapshotSelection(f, gById));
  const character = characters.find((c) => c.id === draft.character_id);
  const palette = byId.get(draft.palette_id);
  const cells = (draft.cells ?? []).map((cell, i) => ({
    row: Math.floor(i / draft.columns) + 1,
    column: (i % draft.columns) + 1,
    action: cell.action,
    expression: cell.expression,
    free_text: cell.free_text,
    label: String(cell.label ?? "").trim(),
    fragments: (cell.fragment_ids ?? []).map((id) => byId.get(id)).filter(Boolean).map((f) => snapshotSelection(f, gById)),
  }));
  const snapshot = {
    schema_version: 1,
    character: character ? { id: character.id, name: character.name, description: character.description, identity_notes: character.identity_notes, reference_asset_id: draft.reference_asset_id || undefined } : undefined,
    selections,
    palette: palette?.colors ? { fragment_id: palette.id, name: palette.name, prompt: palette.prompt, ...palette.colors } : undefined,
    free_text: draft.free_text ?? "",
    rows: draft.rows,
    columns: draft.columns,
    outline_width: draft.outline_width ?? 12,
    outline_color: draft.outline_color ?? "#1F2937",
    cells,
    compiled_at: ts(999),
  };
  const lines = [
    character ? `Use the same character: ${character.name}. ${character.identity_notes}. Attach the selected reference image manually.` : "Draw the subjects described separately in each cell, with one consistent style.",
    ...selections.map((s) => `- [${s.group_name}] ${s.prompt}`),
    palette ? `Bảng màu: ${palette.name}` : "",
    draft.free_text ? `Ghi chú: ${draft.free_text}` : "",
    `Lưới ${draft.rows}x${draft.columns}, viền ${snapshot.outline_width}px ${snapshot.outline_color}.`,
    ...cells.map((cell) => `Cell ${cell.row}.${cell.column}: drawing description ${JSON.stringify([cell.action, cell.expression, cell.free_text, ...cell.fragments.map((f) => f.prompt)].filter(Boolean).join("; "))}. ${cell.label ? `Exact visible label: ${JSON.stringify(cell.label)}. Preserve spelling and accents; join the lettering to the artwork on a connected ribbon.` : "No lettering in this cell."}`),
    "Flat white #FFFFFF background, exact grid and count, clear margins, no grid lines, watermark or frames. Render only the requested labels, never drawing instructions as lettering.",
  ].filter(Boolean);
  return { compiled_prompt: lines.join("\n"), snapshot };
}

function stickerList(slug, count, names) {
  return Array.from({ length: count }, (_, i) => {
    const n = i + 1;
    return {
      sort_order: i,
      filename: `${String(n).padStart(2, "0")}-${names[i % names.length]}.png`,
      checksum_sha256: sha(`${slug}-${n}`),
      width: 512, height: 512,
      emoji_hint: ["😀", "😢", "😡", "😮", "😳", "🥰", "😴", "😭", "👋", "🙏", "💃", "🤗"][i % 12],
      keywords: i % 4 === 3 ? null : [names[i % names.length], "sticker"],
      object_url: `/mock-assets/packs/${slug}/stickers/${n}.svg`,
      thumb_url: `/mock-assets/packs/${slug}/stickers/${n}.svg`,
    };
  });
}

function buildSheets(designs) {
  const mk = (designId, provider, note, filename, seed, minute) => {
    const design = designs.find((d) => d.id === designId);
    const svg = stickerSvg(`${designId}-${seed}`, { bg: true });
    return {
      id: `sheet-${seed}`, design_id: designId, provider, note,
      prompt_snapshot: design?.compiled_prompt ?? "", filename, media_type: "image/svg+xml",
      byte_size: Buffer.byteLength(svg), width: 512, height: 512,
      checksum_sha256: sha(svg), created_at: ts(minute), data: Buffer.from(svg),
    };
  };
  return [
    mk("design-1", "chatgpt", "Bản đầu", "meo-mi-sheet-1.svg", "mi-1", 5),
    mk("design-1", "gemini", "Nét dày hơn", "meo-mi-sheet-2.svg", "mi-2", 10),
    mk("design-2", "ideogram", "", "gau-bong-sheet-1.svg", "bong-1", 65),
    mk("design-2", "gemini", "Đỏ đậm hơn", "gau-bong-sheet-2.svg", "bong-2", 70),
  ];
}

function sheetView(sheet) {
  const { data, ...view } = sheet;
  return view;
}

function packSheetView(sheets, id) {
  const sheet = sheets.find((s) => s.id === id);
  if (!sheet) return null;
  return {
    id: sheet.id, provider: sheet.provider, filename: sheet.filename,
    width: sheet.width, height: sheet.height, byte_size: sheet.byte_size,
    checksum_sha256: sheet.checksum_sha256,
    image_url: `/api/v1/admin/designs/${sheet.design_id}/sheets/${sheet.id}/image`,
  };
}

const animationKey = (packId, index) => `${packId}:${index}`;

function animationView(item) {
  if (!item.static) {
    const age = Date.now() - Date.parse(item.created_at);
    if (item.status === "queued" && age > 500) item.status = "running";
    if (item.status === "running" && age > 1800) {
      item.status = "ready";
      item.frames = 24;
      item.fps = 12;
      item.duration_ms = 1800;
      item.quality = 75;
      item.width = 512;
      item.height = 512;
      item.animated_byte_size = item.clip_byte_size;
      item.animated_checksum_sha256 = sha(`${item.pack_id}:${item.sticker_index}:${item.clip_checksum}`);
    }
  }
  return {
    id: item.id, pack_id: item.pack_id, sticker_index: item.sticker_index,
    motion_text: item.motion_text, static_checksum_sha256: item.static_checksum_sha256,
    clip_byte_size: item.clip_byte_size, status: item.status, use_animated: item.use_animated,
    stale: item.stale, animated_checksum_sha256: item.animated_checksum_sha256,
    animated_byte_size: item.animated_byte_size, frames: item.frames, fps: item.fps,
    duration_ms: item.duration_ms, quality: item.quality, width: item.width, height: item.height,
    error_code: item.error_code, error_message: item.error_message,
    preview_url: `/api/v1/admin/packs/${item.pack_id}/stickers/${item.sticker_index}/animation/preview`,
  };
}

function animationSummaryView(item) {
  const view = animationView(item);
  return {
    status: view.status, use_animated: view.use_animated, stale: view.stale,
    preview_url: view.preview_url, animated_checksum_sha256: view.animated_checksum_sha256,
    animated_byte_size: view.animated_byte_size, frames: view.frames, fps: view.fps,
    duration_ms: view.duration_ms, quality: view.quality,
    error_code: view.error_code, error_message: view.error_message,
  };
}

// Mirrors the web's unpublishedChanges: the effective draft is ready, enabled
// and not stale; a disabled draft reverts an already committed animation.
function animationChanges(pack, animations) {
  return pack.stickers.flatMap((sticker, index) => {
    const draft = animations.get(animationKey(pack.id, index));
    if (!draft) return [];
    const active = draft.status === "ready" && draft.use_animated && !draft.stale;
    if (active) {
      return draft.animated_checksum_sha256 !== sticker.animated?.checksum_sha256
        ? [{ index, kind: "animated" }]
        : [];
    }
    return !draft.use_animated && sticker.animated ? [{ index, kind: "static" }] : [];
  });
}

function applyAnimationPublish(job, pack) {
  for (const change of job.animationChanges ?? []) {
    const sticker = pack.stickers[change.index];
    if (!sticker) continue;
    if (change.kind === "static") {
      delete sticker.animated;
      continue;
    }
    const item = store.animations.get(animationKey(pack.id, change.index));
    if (!item) continue;
    sticker.animated = {
      object_url: `/mock-assets/packs/${pack.slug}/animated/${change.index}.svg`,
      checksum_sha256: item.animated_checksum_sha256,
      byte_size: item.animated_byte_size ?? 4096,
      frames: item.frames ?? 24,
      duration_ms: item.duration_ms ?? 1800,
    };
  }
  job.result = { stickers: pack.stickers.length };
}

// pk-ready-1 carries the interesting draft matrix: an unchanged committed
// variant, an unpublished add, a disabled revert, a stale draft and a failure.
function seedAnimations(packs) {
  const animations = new Map();
  const ready = packs.find((pack) => pack.id === "pk-ready-1");
  if (!ready) return animations;
  const mk = (index, overrides = {}) => {
    const sticker = ready.stickers[index];
    animations.set(animationKey(ready.id, index), {
      id: `anim-${index}`, pack_id: ready.id, sticker_index: index,
      motion_text: ["vẫy tay chào", "nhảy nhẹ theo nhịp", "lắc đầu cười", "nháy mắt", "cười mở miệng"][index] ?? "chuyển động nhẹ",
      static_checksum_sha256: sticker.checksum_sha256, clip_byte_size: 4096,
      status: "ready", use_animated: true, stale: false,
      animated_checksum_sha256: sha(`anim-${ready.slug}-${index}`), animated_byte_size: 4096,
      frames: 24, fps: 12, duration_ms: 1800, quality: 75, width: 512, height: 512,
      created_at: ts(420 + index), static: true, ...overrides,
    });
  };
  const commit = (index) => {
    const item = animations.get(animationKey(ready.id, index));
    ready.stickers[index].animated = {
      object_url: `/mock-assets/packs/${ready.slug}/animated/${index}.svg`,
      checksum_sha256: item.animated_checksum_sha256, byte_size: item.animated_byte_size,
      frames: item.frames, duration_ms: item.duration_ms,
    };
  };
  mk(0);
  commit(0);
  mk(1);
  mk(2, { use_animated: false });
  commit(2);
  mk(3, { stale: true });
  mk(4, {
    status: "failed", animated_checksum_sha256: undefined, animated_byte_size: undefined,
    error_code: "invalid_clip", error_message: "Clip không giải mã được.",
  });
  return animations;
}

function buildStore() {
  const groups = buildGroups();
  const fragments = buildFragments();
  const characters = buildCharacters();
  const assets = buildAssets();
  const designs = buildDesigns(fragments, groups, characters);
  const sheets = buildSheets(designs);

  const jobs = new Map();
  const mkJob = (o) => {
    jobs.set(o.id, { request: { source: "sheet.png", rows: 3 }, result: null, error_code: undefined, error_message: undefined, diagnostics: undefined, started_at: null, finished_at: null, static: true, ...o });
  };
  mkJob({ id: "job-fail-1", pack_id: "pk-draft-1", version: 0, status: "failed", created_at: ts(200), started_at: ts(200), finished_at: ts(203), error_code: "unprocessable", error_message: "Ô sticker số 4 bị cắt lệch.", diagnostics: "grid 3x4 detected, 11 of 12 cells passed bounds check" });
  mkJob({ id: "job-run-1", pack_id: "pk-draft-1", version: 1, status: "running", created_at: ts(900), started_at: ts(900), finished_at: null });
  mkJob({ id: "job-ok-1", pack_id: "pk-ready-1", version: 2, status: "succeeded", created_at: ts(400), started_at: ts(400), finished_at: ts(406), result: { stickers: 12 } });
  mkJob({ id: "job-ok-0", pack_id: "pk-revoked-1", version: 1, status: "succeeded", created_at: ts(300), started_at: ts(300), finished_at: ts(305), result: { stickers: 9 } });
  mkJob({ id: "job-fail-2", pack_id: "pk-revoked-1", version: 2, status: "failed", created_at: ts(700), started_at: ts(700), finished_at: ts(701), error_code: "internal", error_message: "Máy chủ gặp lỗi. Thử lại sau.", diagnostics: "decoder: unexpected EOF" });

  const packs = [
    {
      id: "pk-draft-1", slug: "meo-mi-vui", name: "Mèo Mi vui nhộn", author: "Restick Studio",
      description: "Bộ sticker Mèo Mi cho các tình huống hằng ngày.", tags: ["mèo", "hằng ngày"],
      design_id: "design-1", status: "draft", current_version: 0, sticker_count: 10,
      checksum_sha256: sha("pk-draft-1"), thumbnail_url: "/mock-assets/packs/meo-mi-vui/thumb.svg",
      sheet_id: "sheet-mi-1", sheet: packSheetView(sheets, "sheet-mi-1"), created_at: ts(0), updated_at: ts(900),
      stickers: stickerList("meo-mi-vui", 10, ["vui", "buồn", "giận", "ngạc-nhiên", "ngại", "yêu", "ngủ", "khóc", "vẫy", "cảm-ơn"]),
    },
    {
      id: "pk-ready-1", slug: "gau-bong-tet", name: "Gấu Bông ngày Tết", author: "Restick Studio",
      description: "Lời chúc Tết dễ thương với Gấu Bông.", tags: ["tết", "lễ hội", "gấu"],
      design_id: "design-2", status: "ready", current_version: 2, sticker_count: 12,
      checksum_sha256: sha("pk-ready-1"), thumbnail_url: "/mock-assets/packs/gau-bong-tet/thumb.svg",
      sheet_id: "sheet-bong-1", sheet: packSheetView(sheets, "sheet-bong-1"), created_at: ts(100), updated_at: ts(406),
      stickers: stickerList("gau-bong-tet", 12, ["chúc-tết", "lì-xì", "cảm-ơn", "vui", "nhảy", "ôm", "ngủ", "yêu", "bánh-chưng", "pháo", "hoa-mai", "ngạc-nhiên"]),
    },
    {
      id: "pk-revoked-1", slug: "cun-bo-hoc-duong", name: "Cún Bơ đi học", author: "Restick Studio",
      description: "Bộ đã thu hồi vì chờ xác nhận quyền.", tags: null,
      design_id: "design-3", status: "revoked", current_version: 1, sticker_count: 9,
      checksum_sha256: sha("pk-revoked-1"), thumbnail_url: "/mock-assets/packs/cun-bo-hoc-duong/thumb.svg",
      sheet_id: "", sheet: null, created_at: ts(50), updated_at: ts(705),
      stickers: stickerList("cun-bo-hoc-duong", 9, ["đi-học", "chạy", "chỉ", "ngủ", "buồn", "giận", "vui", "bút-chì", "cặp-sách"]),
    },
  ];

  const animations = seedAnimations(packs);

  const keys = [
    { id: "key-1", name: "Bot Telegram", prefix: "rsk_7f3a9c", active: true, created_at: ts(10), expires_at: null, disabled_at: null },
    { id: "key-2", name: "Thử nghiệm cũ", prefix: "rsk_2b81de", active: false, created_at: ts(-4000), expires_at: ts(-1000), disabled_at: null },
    { id: "key-3", name: "Đối tác Zalo", prefix: "rsk_c04e77", active: false, created_at: ts(-2000), expires_at: null, disabled_at: ts(-100) },
  ];

  return {
    groups, fragments, characters, assets, designs, sheets, packs, jobs, keys, sessions: new Set(),
    animations,
    keyPacks: new Map([["key-1", new Set(["pk-ready-1"])]]),
  };
}

// ---------- state ----------

let store = buildStore();
let scenario = "default";
let seq = 100;
const nextId = (p) => `${p}-${(seq++).toString(36)}`;
const now = () => new Date().toISOString();

// ---------- HTTP plumbing ----------

function send(res, status, body, headers = {}) {
  const isBuf = Buffer.isBuffer(body);
  const payload = body === undefined || body === null ? null : isBuf || typeof body === "string" ? body : JSON.stringify(body);
  res.writeHead(status, {
    "Cache-Control": "no-store",
    ...(payload !== null && !isBuf && typeof body !== "string" ? { "Content-Type": "application/json; charset=utf-8" } : {}),
    ...headers,
  });
  res.end(payload ?? undefined);
}

const fail = (res, status, code, message) => send(res, status, { error: { code, message } });

function readBody(req) {
  return new Promise((resolve, reject) => {
    const chunks = [];
    req.on("data", (c) => chunks.push(c));
    req.on("end", () => resolve(Buffer.concat(chunks)));
    req.on("error", reject);
  });
}

function parseJson(buf) {
  if (!buf.length) return {};
  try {
    return JSON.parse(buf.toString("utf8"));
  } catch {
    return null;
  }
}

function parseMultipart(buf, contentType) {
  const m = /boundary=(?:"([^"]+)"|([^;]+))/i.exec(contentType || "");
  if (!m) return null;
  const delim = Buffer.from(`--${m[1] || m[2]}`);
  const parts = {};
  let pos = buf.indexOf(delim);
  while (pos !== -1) {
    const start = pos + delim.length;
    if (buf.slice(start, start + 2).toString() === "--") break;
    const headEnd = buf.indexOf("\r\n\r\n", start + 2);
    if (headEnd === -1) break;
    const headers = buf.slice(start + 2, headEnd).toString("utf8");
    const bodyStart = headEnd + 4;
    const next = buf.indexOf(delim, bodyStart);
    if (next === -1) break;
    const name = /name="([^"]*)"/.exec(headers)?.[1];
    const filename = /filename="([^"]*)"/.exec(headers)?.[1];
    const type = /content-type:\s*([^\r\n]+)/i.exec(headers)?.[1]?.trim();
    if (name) parts[name] = { filename, type, data: buf.slice(bodyStart, next - 2) };
    pos = next;
  }
  return parts;
}

function pngSize(data) {
  if (data.length >= 24 && data.readUInt32BE(0) === 0x89504e47) return [data.readUInt32BE(16), data.readUInt32BE(20)];
  return [1024, 1024];
}

function cookieToken(req) {
  const raw = req.headers.cookie || "";
  const m = new RegExp(`(?:^|;\\s*)${COOKIE}=([^;]+)`).exec(raw);
  return m ? m[1] : null;
}

const isAuthed = (req) => {
  const t = cookieToken(req);
  return Boolean(t && store.sessions.has(t));
};

const pageOf = (items, url) => {
  const page = Math.max(1, Number(url.searchParams.get("page")) || 1);
  const size = Math.min(200, Math.max(1, Number(url.searchParams.get("page_size")) || 20));
  const visible = scenario === "empty" ? [] : items;
  return { items: visible.slice((page - 1) * size, page * size), page, page_size: size, total: visible.length };
};

const hidden = () => scenario === "empty";

function nowIso() {
  return now();
}

// Job states advance lazily: seeded jobs marked static keep their state.
function jobView(job) {
  const nowMs = Date.now();
  if (!job.static && job.status === "queued" && nowMs - Date.parse(job.created_at) > (job.fast ? 400 : 1500)) {
    job.status = "running";
    job.started_at = nowIso();
  }
  if (!job.static && job.status === "running" && nowMs - Date.parse(job.created_at) > (job.fast ? 1200 : 6000)) {
    job.status = "succeeded";
    job.finished_at = nowIso();
    const pack = store.packs.find((item) => item.id === job.pack_id);
    if (pack && job.animationChanges) applyAnimationPublish(job, pack);
    else job.result = { stickers: 10 };
  }
  return {
    id: job.id, pack_id: job.pack_id, version: job.version, status: job.status,
    request: job.request, result: job.result, error_code: job.error_code, error_message: job.error_message,
    diagnostics: job.diagnostics, created_at: job.created_at, started_at: job.started_at, finished_at: job.finished_at,
  };
}

function packView(p) {
  const { stickers, ...rest } = p;
  return rest;
}

function packDetail(p) {
  const jobs = [...store.jobs.values()].filter((j) => j.pack_id === p.id).sort((a, b) => b.created_at.localeCompare(a.created_at));
  const latest = jobs[0];
  const design = store.designs.find((d) => d.id === p.design_id);
  const animations = new Map(
    [...store.animations.values()].filter((item) => item.pack_id === p.id).map((item) => [item.sticker_index, item]),
  );
  return {
    ...packView(p),
    ...(design
      ? {
          design: {
            id: design.id, name: design.name, rows: design.rows, columns: design.columns,
            outline_width: design.outline_width, outline_color: design.outline_color,
            character_id: design.character_id, compiled_prompt: design.compiled_prompt,
          },
        }
      : {}),
    stickers: p.stickers.map((sticker) => {
      const draft = animations.get(sticker.sort_order);
      return draft ? { ...sticker, animation: animationSummaryView(draft) } : sticker;
    }),
    bundle_url: p.status === "ready" ? `/mock-assets/packs/${p.slug}/bundle.zip` : "",
    ...(latest ? { latest_job: jobView(latest) } : {}),
  };
}

// Full grant set for a key, sorted like auth.KeyPacks.
const grantIds = (keyId) => [...(store.keyPacks.get(keyId) ?? new Set())].sort();

// ---------- API ----------

function fragmentPayload(body, existing) {
  const f = {
    group_id: String(body.group_id ?? ""), name: String(body.name ?? "").trim(), name_en: String(body.name_en ?? ""),
    prompt: String(body.prompt ?? ""), notes: String(body.notes ?? ""), source: String(body.source ?? ""),
    scope: body.scope,
  };
  if (body.colors) f.colors = body.colors;
  return { ...existing, ...f, colors: body.colors ?? undefined };
}

function characterReferenceIds(body) {
  if (body.reference_asset_ids === undefined) return [];
  if (!Array.isArray(body.reference_asset_ids) || body.reference_asset_ids.some((id) => typeof id !== "string")) return null;
  const ids = [...new Set(body.reference_asset_ids.map((id) => id.trim()).filter(Boolean))];
  return ids.every((id) => store.assets.some((asset) => asset.id === id)) ? ids : null;
}

function validateDraft(draft, compile = true, requireName = false) {
  if (!draft) return "Thiếu thiết kế.";
  if (requireName && !draft.name?.trim()) return "Thiếu tên thiết kế.";
  const character = store.characters.find((item) => item.id === draft.character_id);
  if (draft.character_id && !character) return "Nhân vật không tồn tại.";
  const selected = [...(draft.fragment_ids ?? []), ...(draft.pinned_fragment_ids ?? []), draft.palette_id, ...(draft.cells ?? []).flatMap((cell) => cell.fragment_ids ?? [])];
  if (character && selected.some((id) => store.fragments.find((fragment) => fragment.id === id)?.scope === "identity")) return "Hồ sơ nhân vật không dùng chung mảnh Nhận diện.";
  if (!compile) return null;
  if (!character && draft.reference_asset_id) return "Ảnh tham chiếu cần một nhân vật.";
  if (character && (!draft.reference_asset_id || !character.reference_asset_ids.includes(draft.reference_asset_id) || !store.assets.some((asset) => asset.id === draft.reference_asset_id))) return "Chọn một ảnh thuộc hồ sơ nhân vật trước khi biên dịch hoặc lưu.";
  if (!Number.isInteger(draft.rows) || !Number.isInteger(draft.columns) || draft.rows < 1 || draft.rows > 8 || draft.columns < 1 || draft.columns > 8 || !Array.isArray(draft.cells) || draft.cells.length !== draft.rows * draft.columns) return "Lưới cần đúng số ô, từ 1–8 hàng/cột.";
  if (draft.cells.some((cell) => ![cell.action, cell.expression, cell.free_text].some((text) => text?.trim()) && !(cell.fragment_ids ?? []).length)) return "Mỗi ô cần mô tả hình, hành động, biểu cảm hoặc mảnh ghép; chỉ có chữ là chưa đủ.";
  return null;
}

function normalizeDraft(draft) {
  return { ...draft, reference_asset_id: draft.reference_asset_id || undefined, cells: (draft.cells ?? []).map((cell) => ({ ...cell, label: String(cell.label ?? "").trim() })) };
}

async function handleApi(req, res, url) {
  const p = url.pathname;
  const method = req.method;
  const raw = await readBody(req);
  const ct = req.headers["content-type"] || "";
  const json = ct.includes("application/json") || !ct.includes("multipart") ? parseJson(raw) : null;

  // ----- session -----
  if (p === "/api/v1/admin/session") {
    if (method === "POST") {
      if (json === null) return fail(res, 400, "bad_request", "JSON không hợp lệ.");
      if (json.password !== PASSWORD) return fail(res, 401, "invalid_credentials", "Mật khẩu không đúng.");
      const token = randomBytes(24).toString("hex");
      store.sessions.add(token);
      return send(res, 204, undefined, { "Set-Cookie": `${COOKIE}=${token}; Path=/; HttpOnly; SameSite=Strict` });
    }
    if (method === "DELETE") {
      const t = cookieToken(req);
      if (t) store.sessions.delete(t);
      return send(res, 204, undefined, { "Set-Cookie": `${COOKIE}=; Path=/; Max-Age=0; HttpOnly; SameSite=Strict` });
    }
    if (method === "GET") {
      if (!isAuthed(req)) return fail(res, 401, "unauthorized", "Cần đăng nhập.");
      return send(res, 200, { username: "owner" });
    }
    return fail(res, 405, "bad_request", "Method not allowed.");
  }

  if (!isAuthed(req)) return fail(res, 401, "unauthorized", "Cần đăng nhập.");
  if (json === null && method !== "POST") return fail(res, 400, "bad_request", "JSON không hợp lệ.");

  const notFound = () => fail(res, 404, "not_found", "Không tìm thấy dữ liệu.");

  // ----- groups -----
  if (p === "/api/v1/admin/fragment-groups") {
    if (method === "GET") return send(res, 200, { items: hidden() ? [] : store.groups });
    if (method === "POST") {
      if (!json?.name?.trim()) return fail(res, 422, "unprocessable", "Thiếu tên nhóm.");
      const g = { id: nextId("g"), name: json.name.trim(), description: json.description ?? "", sort_order: json.sort_order ?? store.groups.length + 1, created_at: nowIso(), updated_at: nowIso() };
      store.groups.push(g);
      return send(res, 201, g);
    }
  }
  let m = /^\/api\/v1\/admin\/fragment-groups\/([^/]+)$/.exec(p);
  if (m) {
    const g = store.groups.find((x) => x.id === m[1]);
    if (!g || hidden()) return notFound();
    if (method === "PUT") {
      if (!json?.name?.trim()) return fail(res, 422, "unprocessable", "Thiếu tên nhóm.");
      Object.assign(g, { name: json.name.trim(), description: json.description ?? "", sort_order: json.sort_order ?? g.sort_order, updated_at: nowIso() });
      return send(res, 200, g);
    }
    if (method === "DELETE") {
      if (store.fragments.some((f) => f.group_id === g.id && !f.archived)) return fail(res, 409, "group_not_empty", "Nhóm còn mảnh ghép nên không xóa được.");
      store.groups = store.groups.filter((x) => x.id !== g.id);
      return send(res, 204);
    }
  }

  // ----- fragments -----
  if (p === "/api/v1/admin/fragments") {
    if (method === "GET") {
      let items = store.fragments.filter((f) => url.searchParams.get("include_archived") === "true" || !f.archived);
      const q = (url.searchParams.get("q") || "").toLowerCase();
      if (q) items = items.filter((f) => [f.name, f.name_en, f.prompt, f.notes].join(" ").toLowerCase().includes(q));
      if (url.searchParams.get("group_id")) items = items.filter((f) => f.group_id === url.searchParams.get("group_id"));
      if (url.searchParams.get("scope")) items = items.filter((f) => f.scope === url.searchParams.get("scope"));
      return send(res, 200, pageOf(items, url));
    }
    if (method === "POST") {
      if (!json?.name?.trim() || !store.groups.some((g) => g.id === json.group_id)) return fail(res, 422, "unprocessable", "Thiếu tên hoặc nhóm không tồn tại.");
      const f = { ...fragmentPayload(json, {}), id: nextId("f"), archived: false, created_at: nowIso(), updated_at: nowIso() };
      store.fragments.push(f);
      return send(res, 201, f);
    }
  }
  m = /^\/api\/v1\/admin\/fragments\/([^/]+)$/.exec(p);
  if (m) {
    const f = store.fragments.find((x) => x.id === m[1]);
    if (!f || hidden()) return notFound();
    if (method === "GET") return send(res, 200, f);
    if (method === "PUT") {
      if (!json?.name?.trim()) return fail(res, 422, "unprocessable", "Thiếu tên.");
      Object.assign(f, fragmentPayload(json, f), { updated_at: nowIso() });
      return send(res, 200, f);
    }
    if (method === "DELETE") {
      f.archived = true;
      f.updated_at = nowIso();
      return send(res, 204);
    }
  }

  // ----- characters -----
  if (p === "/api/v1/admin/characters") {
    if (method === "GET") return send(res, 200, { items: hidden() ? [] : store.characters.filter((character) => !character.archived) });
    if (method === "POST") {
      if (!json?.name?.trim()) return fail(res, 422, "unprocessable", "Thiếu tên nhân vật.");
      const references = characterReferenceIds(json);
      if (!references) return fail(res, 422, "unprocessable", "Ảnh tham chiếu không tồn tại.");
      const c = { id: nextId("char"), name: json.name.trim(), description: json.description ?? "", identity_notes: json.identity_notes ?? "", rights_notes: json.rights_notes ?? "", reference_asset_ids: references, archived: false, created_at: nowIso(), updated_at: nowIso() };
      store.characters.push(c);
      return send(res, 201, c);
    }
  }
  m = /^\/api\/v1\/admin\/characters\/([^/]+)$/.exec(p);
  if (m) {
    const c = store.characters.find((x) => x.id === m[1]);
    if (!c || hidden()) return notFound();
    if (method === "GET") return send(res, 200, c);
    if (method === "PUT") {
      const references = characterReferenceIds(json);
      if (!references) return fail(res, 422, "unprocessable", "Ảnh tham chiếu không tồn tại.");
      Object.assign(c, { name: json.name?.trim() ?? c.name, description: json.description ?? "", identity_notes: json.identity_notes ?? "", rights_notes: json.rights_notes ?? "", reference_asset_ids: references, updated_at: nowIso() });
      return send(res, 200, c);
    }
    if (method === "DELETE") {
      c.archived = true;
      c.updated_at = nowIso();
      return send(res, 204);
    }
  }

  // ----- reference assets -----
  if (p === "/api/v1/admin/reference-assets" && method === "POST") {
    const parts = parseMultipart(raw, ct);
    const file = parts?.file;
    if (!file) return fail(res, 400, "bad_request", "Thiếu tệp.");
    const type = file.type || "application/octet-stream";
    if (!["image/png", "image/jpeg", "image/webp"].includes(type)) return fail(res, 422, "invalid_image", "Ảnh không hợp lệ: chỉ nhận PNG/JPEG/WebP, không animation, tối đa 8 MiB và 4096 px mỗi cạnh.");
    const [width, height] = pngSize(file.data);
    const asset = { id: nextId("ref"), filename: file.filename || "upload", content_type: type, byte_size: file.data.length, width, height, checksum_sha256: sha(file.data), created_at: nowIso(), data: file.data };
    store.assets.push(asset);
    const { data, ...view } = asset;
    return send(res, 201, view);
  }
  m = /^\/api\/v1\/admin\/reference-assets\/([^/]+)$/.exec(p);
  if (m && method === "GET") {
    const a = store.assets.find((x) => x.id === m[1]);
    if (!a) return notFound();
    return send(res, 200, a.data, { "Content-Type": a.content_type });
  }

  // ----- designs -----
  if (p === "/api/v1/admin/designs") {
    if (method === "GET") return send(res, 200, pageOf(store.designs, url));
    if (method === "POST") {
      const validation = validateDraft(json, true, true);
      if (validation) return fail(res, 422, "unprocessable", validation);
      const d = buildDesignFromDraft(json, `design-${nextId("d")}`);
      if (!d) return fail(res, 422, "unprocessable", "Thiếu tên thiết kế.");
      store.designs.unshift(d);
      return send(res, 201, d);
    }
  }
  if (p === "/api/v1/admin/designs/suggest" && method === "POST") {
    const validation = validateDraft(json, false);
    if (validation) return fail(res, 422, "unprocessable", validation);
    const groupIds = json.group_ids ?? [];
    const pinned = json.pinned_fragment_ids ?? [];
    const fragment_ids = groupIds
      .map((gid) => store.fragments.find((fragment) => fragment.group_id === gid && pinned.includes(fragment.id)) ?? store.fragments.find((fragment) => fragment.group_id === gid && !fragment.archived && (!json.character_id || fragment.scope !== "identity")))
      .filter(Boolean)
      .map((fragment) => fragment.id);
    return send(res, 200, { ...json, fragment_ids, pinned_fragment_ids: pinned });
  }
  if (p === "/api/v1/admin/designs/compile" && method === "POST") {
    const validation = validateDraft(json);
    if (validation) return fail(res, 422, "unprocessable", validation);
    const byId = new Map(store.fragments.map((f) => [f.id, f]));
    const gById = new Map(store.groups.map((g) => [g.id, g]));
    return send(res, 200, compileDesign(normalizeDraft(json), byId, gById, store.characters));
  }
  m = /^\/api\/v1\/admin\/designs\/([^/]+)$/.exec(p);
  if (m) {
    const d = store.designs.find((x) => x.id === m[1]);
    if (!d || hidden()) return notFound();
    if (method === "GET") return send(res, 200, d);
    if (method === "PUT") {
      const validation = validateDraft(json, true, true);
      if (validation) return fail(res, 422, "unprocessable", validation);
      const updated = buildDesignFromDraft(json, d.id, d.created_at);
      if (!updated) return fail(res, 422, "unprocessable", "Thiếu tên thiết kế.");
      Object.assign(d, updated);
      return send(res, 200, d);
    }
    if (method === "DELETE") {
      if (store.sheets.some((s) => s.design_id === d.id)) {
        return fail(res, 409, "design_has_sheets", "Thiết kế còn sheet nên không xóa được.");
      }
      store.designs = store.designs.filter((x) => x.id !== d.id);
      return send(res, 204);
    }
  }

  // ----- design sheets -----
  m = /^\/api\/v1\/admin\/designs\/([^/]+)\/sheets$/.exec(p);
  if (m) {
    const design = store.designs.find((x) => x.id === m[1]);
    if (!design || hidden()) return notFound();
    if (method === "GET") {
      const items = store.sheets
        .filter((s) => s.design_id === design.id)
        .sort((a, b) => b.created_at.localeCompare(a.created_at));
      return send(res, 200, { items: items.map(sheetView) });
    }
    if (method === "POST") {
      const parts = parseMultipart(raw, ct);
      const file = parts?.file;
      if (!file) return fail(res, 400, "bad_request", "Thiếu tệp.");
      const provider = parts.provider?.data.toString("utf8") ?? "";
      if (!/^[a-z0-9_-]{1,32}$/.test(provider)) return fail(res, 422, "unprocessable", "Provider không hợp lệ.");
      const type = file.type || "application/octet-stream";
      if (!["image/png", "image/jpeg", "image/webp"].includes(type)) {
        return fail(res, 422, "invalid_image", "Ảnh không hợp lệ: chỉ nhận PNG/JPEG/WebP, không animation, tối đa 8 MiB và 4096 px mỗi cạnh.");
      }
      const checksum = sha(file.data);
      const duplicate = store.sheets.find((s) => s.design_id === design.id && s.checksum_sha256 === checksum);
      if (duplicate) return send(res, 200, sheetView(duplicate));
      if (store.sheets.filter((s) => s.design_id === design.id).length >= 20) {
        return fail(res, 409, "sheet_limit", "Thiết kế đã đủ 20 sheet.");
      }
      const [width, height] = pngSize(file.data);
      const sheet = {
        id: nextId("sheet"), design_id: design.id, provider,
        note: (parts.note?.data.toString("utf8") ?? "").slice(0, 500),
        prompt_snapshot: design.compiled_prompt, filename: file.filename || "sheet",
        media_type: type, byte_size: file.data.length, width, height,
        checksum_sha256: checksum, created_at: nowIso(), data: file.data,
      };
      store.sheets.push(sheet);
      return send(res, 201, sheetView(sheet));
    }
  }
  m = /^\/api\/v1\/admin\/designs\/([^/]+)\/sheets\/([^/]+)\/image$/.exec(p);
  if (m && method === "GET") {
    const sheet = store.sheets.find((s) => s.id === m[2] && s.design_id === m[1]);
    if (!sheet || hidden()) return notFound();
    return send(res, 200, sheet.data, { "Content-Type": sheet.media_type });
  }
  m = /^\/api\/v1\/admin\/designs\/([^/]+)\/sheets\/([^/]+)$/.exec(p);
  if (m) {
    const sheet = store.sheets.find((s) => s.id === m[2] && s.design_id === m[1]);
    if (!sheet || hidden()) return notFound();
    if (method === "DELETE") {
      if (store.packs.some((pk) => pk.sheet_id === sheet.id)) {
        return fail(res, 409, "sheet_in_use", "Sheet đang được một pack dùng nên không xóa được.");
      }
      store.sheets = store.sheets.filter((x) => x.id !== sheet.id);
      return send(res, 204);
    }
  }

  // ----- packs -----
  if (p === "/api/v1/admin/packs") {
    if (method === "GET") {
      let items = store.packs;
      const q = (url.searchParams.get("q") || "").toLowerCase();
      if (q) items = items.filter((x) => `${x.name} ${x.slug} ${x.author}`.toLowerCase().includes(q));
      if (url.searchParams.get("status")) items = items.filter((x) => x.status === url.searchParams.get("status"));
      if (url.searchParams.get("tag")) items = items.filter((x) => (x.tags ?? []).includes(url.searchParams.get("tag")));
      const pg = pageOf(items, url);
      return send(res, 200, { ...pg, items: pg.items.map(packView) });
    }
    if (method === "POST") {
      if (!json?.slug || !json?.name?.trim() || !json?.author?.trim()) {
        return fail(res, 422, "unprocessable", "Name and author are required.");
      }
      const sheet = store.sheets.find((s) => s.id === json.sheet_id && s.design_id === json.design_id);
      if (!json.design_id || !sheet) return fail(res, 422, "unprocessable", "Sheet phải thuộc thiết kế đã chọn.");
      if (store.packs.some((x) => x.slug === json.slug)) return fail(res, 409, "slug_taken", "Slug này đã được dùng. Chọn slug khác.");
      const pk = { id: nextId("pk"), slug: json.slug, name: json.name, author: json.author ?? "", description: json.description ?? "", tags: json.tags?.length ? json.tags : null, design_id: json.design_id, status: "draft", current_version: 0, sticker_count: 0, checksum_sha256: sha(json.slug), thumbnail_url: `/mock-assets/packs/${json.slug}/thumb.svg`, sheet_id: sheet.id, sheet: packSheetView(store.sheets, sheet.id), created_at: nowIso(), updated_at: nowIso(), stickers: [] };
      store.packs.unshift(pk);
      return send(res, 201, packView(pk));
    }
  }
  m = /^\/api\/v1\/admin\/packs\/([^/]+)(\/[a-z]+)?$/.exec(p);
  if (m) {
    const pk = store.packs.find((x) => x.id === m[1]);
    const action = m[2];
    if (!pk || hidden()) return notFound();
    if (!action) {
      if (method === "GET") return send(res, 200, packDetail(pk));
      if (method === "PUT") {
        if (!String(json?.name ?? "").trim() || !String(json?.author ?? "").trim()) {
          return fail(res, 422, "unprocessable", "Name and author are required.");
        }
        if (pk.status === "revoked") return fail(res, 410, "pack_revoked", "Pack đã bị thu hồi; thao tác này không còn hiệu lực.");
        const designId = json.design_id ?? pk.design_id;
        const sheet = store.sheets.find((s) => s.id === json.sheet_id && s.design_id === designId);
        if (!designId || !sheet) return fail(res, 422, "unprocessable", "Sheet phải thuộc thiết kế đã chọn.");
        const active = [...store.jobs.values()].some((j) => j.pack_id === pk.id && (j.status === "queued" || j.status === "running"));
        if (sheet.id !== pk.sheet_id && active) return fail(res, 409, "job_active", "Pack đang có job chạy. Chờ job kết thúc rồi thử lại.");
        Object.assign(pk, { name: json.name ?? pk.name, author: json.author ?? pk.author, description: json.description ?? "", tags: json.tags?.length ? json.tags : null, design_id: designId, sheet_id: sheet.id, sheet: packSheetView(store.sheets, sheet.id), updated_at: nowIso() });
        return send(res, 200, packView(pk));
      }
      if (method === "DELETE") {
        if (pk.status !== "draft") return fail(res, 409, "not_draft", "Chỉ pack nháp mới xóa được. Pack đã sẵn sàng phải thu hồi.");
        store.packs = store.packs.filter((x) => x.id !== pk.id);
        return send(res, 204);
      }
    }
    if (action === "/process" && method === "POST") {
      if (pk.status === "revoked") return fail(res, 409, "pack_revoked", "Pack đã bị thu hồi; thao tác này không còn hiệu lực.");
      if (!pk.sheet) return fail(res, 422, "source_missing", "Pack chưa có sheet. Chọn sheet cho pack trước khi xử lý.");
      const active = [...store.jobs.values()].some((j) => j.pack_id === pk.id && (j.status === "queued" || j.status === "running"));
      if (active) return fail(res, 409, "job_active", "Pack đang có job chạy. Chờ job kết thúc rồi thử lại.");
      const job = { id: nextId("job"), pack_id: pk.id, version: pk.current_version + 1, status: "queued", request: { source: pk.sheet.filename }, result: null, created_at: nowIso(), started_at: null, finished_at: null, static: false };
      store.jobs.set(job.id, job);
      return send(res, 202, jobView(job));
    }
    if (action === "/revoke" && method === "POST") {
      pk.status = "revoked";
      pk.updated_at = nowIso();
      return send(res, 200, packView(pk));
    }
  }

  // ----- sticker animations -----
  if (p === "/api/v1/admin/animations/compile" && method === "POST") {
    const pk = store.packs.find((x) => x.id === json?.pack_id);
    const sticker = pk?.stickers.find((s) => s.sort_order === json?.sticker_index);
    if (!pk || hidden()) return notFound();
    if (!sticker) return fail(res, 404, "not_found", "Sticker không tồn tại.");
    if (typeof json.motion_text !== "string" || !json.motion_text.trim()) {
      return fail(res, 422, "unprocessable", "Thiếu mô tả chuyển động.");
    }
    return send(res, 200, {
      prompt: `Animate the sticker "${sticker.filename}": ${json.motion_text.trim()}. Keep the character identity, centered framing, 512x512, seamless loop.`,
      emoji_hint: sticker.emoji_hint,
    });
  }
  m = /^\/api\/v1\/admin\/packs\/([^/]+)\/animations$/.exec(p);
  if (m && method === "GET") {
    const pk = store.packs.find((x) => x.id === m[1]);
    if (!pk || hidden()) return notFound();
    const items = [...store.animations.values()]
      .filter((item) => item.pack_id === pk.id)
      .sort((a, b) => a.sticker_index - b.sticker_index)
      .map(animationView);
    return send(res, 200, { items });
  }
  m = /^\/api\/v1\/admin\/packs\/([^/]+)\/publish-animations$/.exec(p);
  if (m && method === "POST") {
    const pk = store.packs.find((x) => x.id === m[1]);
    if (!pk || hidden()) return notFound();
    if (pk.status === "revoked") return fail(res, 410, "pack_revoked", "Pack đã bị thu hồi; thao tác này không còn hiệu lực.");
    const active = [...store.jobs.values()].some((j) => j.pack_id === pk.id && (j.status === "queued" || j.status === "running"));
    if (active) return fail(res, 409, "job_active", "Pack đang có job chạy. Chờ job kết thúc rồi thử lại.");
    const changes = animationChanges(pk, store.animations);
    if (!changes.length) return fail(res, 409, "no_changes", "Không có thay đổi animation nào để phát hành.");
    const job = {
      id: nextId("job"), pack_id: pk.id, version: pk.current_version + 1, status: "queued",
      request: { animations: changes.length }, result: null, created_at: nowIso(), started_at: null,
      finished_at: null, static: false, fast: true, animationChanges: changes,
    };
    store.jobs.set(job.id, job);
    return send(res, 202, jobView(job));
  }
  m = /^\/api\/v1\/admin\/packs\/([^/]+)\/stickers\/(\d+)\/animation$/.exec(p);
  if (m) {
    const pk = store.packs.find((x) => x.id === m[1]);
    const index = Number(m[2]);
    const sticker = pk?.stickers.find((s) => s.sort_order === index);
    if (!pk || hidden()) return notFound();
    if (!sticker) return notFound();
    const key = animationKey(pk.id, index);
    const existing = store.animations.get(key);
    if (method === "POST") {
      if (existing && (existing.status === "queued" || existing.status === "running")) {
        return fail(res, 409, "animation_busy", "Sticker này đang có animation chạy.");
      }
      const parts = parseMultipart(raw, ct);
      const clip = parts?.clip;
      if (!clip || !clip.data.length) return fail(res, 400, "bad_request", "Thiếu clip.");
      const motionText = (parts.motion_text?.data.toString("utf8") ?? "").trim();
      if (!motionText) return fail(res, 422, "unprocessable", "Thiếu mô tả chuyển động.");
      const item = {
        id: existing?.id ?? nextId("anim"), pack_id: pk.id, sticker_index: index,
        motion_text: motionText, static_checksum_sha256: sticker.checksum_sha256,
        clip_byte_size: clip.data.length, clip_checksum: sha(clip.data), status: "queued",
        use_animated: true, stale: false, animated_checksum_sha256: undefined,
        animated_byte_size: undefined, frames: undefined, fps: undefined, duration_ms: undefined,
        quality: undefined, width: undefined, height: undefined, error_code: undefined,
        error_message: undefined, created_at: nowIso(), static: false,
      };
      store.animations.set(key, item);
      return send(res, 202, animationView(item));
    }
    if (!existing) return notFound();
    if (method === "PATCH") {
      if (typeof json?.use_animated !== "boolean") {
        return fail(res, 422, "unprocessable", "use_animated là bắt buộc.");
      }
      if (existing.status !== "ready") return fail(res, 409, "not_ready", "Animation chưa xử lý xong.");
      existing.use_animated = json.use_animated;
      return send(res, 200, animationView(existing));
    }
    if (method === "DELETE") {
      store.animations.delete(key);
      return send(res, 204);
    }
  }
  m = /^\/api\/v1\/admin\/packs\/([^/]+)\/stickers\/(\d+)\/animation\/preview$/.exec(p);
  if (m && method === "GET") {
    const pk = store.packs.find((x) => x.id === m[1]);
    if (!pk || hidden()) return notFound();
    const item = store.animations.get(animationKey(pk.id, Number(m[2])));
    if (!item) return notFound();
    const view = animationView(item);
    if (view.status !== "ready") return fail(res, 409, "not_ready", "Animation chưa xử lý xong.");
    // The real route serves WebP; the mock serves SVG so the preview needs no
    // animated-image fixture, and the QA scenario only asserts rendering.
    return send(res, 200, stickerSvg(`${pk.slug}-anim-${m[2]}`, { bg: false }), { "Content-Type": "image/svg+xml" });
  }
  m = /^\/api\/v1\/admin\/packs\/([^/]+)\/stickers\/(\d+)\/on-white\.png$/.exec(p);
  if (m && method === "GET") {
    const pk = store.packs.find((x) => x.id === m[1]);
    if (!pk || hidden()) return notFound();
    if (!pk.stickers.some((s) => s.sort_order === Number(m[2]))) return notFound();
    return send(res, 200, onWhitePng(`${pk.slug}-${m[2]}`), { "Content-Type": "image/png" });
  }

  // ----- jobs -----
  m = /^\/api\/v1\/admin\/jobs\/([^/]+)$/.exec(p);
  if (m && method === "GET") {
    const j = store.jobs.get(m[1]);
    if (!j || hidden()) return notFound();
    return send(res, 200, jobView(j));
  }

  // ----- api keys -----
  if (p === "/api/v1/admin/api-keys") {
    if (method === "GET") return send(res, 200, { items: hidden() ? [] : store.keys.map(keyView) });
    if (method === "POST") {
      if (!json?.name?.trim()) return fail(res, 422, "unprocessable", "Thiếu tên khóa.");
      const created = newKey(json.name.trim(), json.expires_at ?? null);
      store.keys.unshift(created.key);
      store.keyPacks.set(created.key.id, new Set());
      return send(res, 201, { id: created.key.id, name: created.key.name, prefix: created.key.prefix, raw: created.raw });
    }
  }
  m = /^\/api\/v1\/admin\/api-keys\/([^/]+)\/packs$/.exec(p);
  if (m) {
    const k = store.keys.find((x) => x.id === m[1]);
    if (!k || hidden()) return notFound();
    if (method === "GET") return send(res, 200, { pack_ids: grantIds(k.id) });
    if (method === "PUT") {
      if (!json || json.pack_ids === undefined || json.pack_ids === null) {
        return fail(res, 400, "bad_request", "pack_ids is required.");
      }
      if (!Array.isArray(json.pack_ids)) return fail(res, 400, "bad_request", "pack_ids must be an array.");
      const ids = [...new Set(json.pack_ids.map(String))];
      if (ids.some((id) => !store.packs.some((pk) => pk.id === id))) return fail(res, 422, "unprocessable", "Pack không tồn tại.");
      store.keyPacks.set(k.id, new Set(ids));
      return send(res, 200, { pack_ids: grantIds(k.id) });
    }
  }
  m = /^\/api\/v1\/admin\/api-keys\/([^/]+)(\/rotate)?$/.exec(p);
  if (m) {
    const k = store.keys.find((x) => x.id === m[1]);
    if (!k || hidden()) return notFound();
    if (m[2] && method === "POST") {
      // Rotation disables the old row and creates a replacement carrying the grants.
      const fresh = newKey(k.name, k.expires_at);
      const grants = new Set(store.keyPacks.get(k.id) ?? []);
      k.active = false;
      k.disabled_at = nowIso();
      store.keyPacks.set(fresh.key.id, grants);
      store.keys.unshift(fresh.key);
      return send(res, 201, { id: fresh.key.id, name: fresh.key.name, prefix: fresh.key.prefix, raw: fresh.raw });
    }
    if (!m[2] && method === "DELETE") {
      k.active = false;
      k.disabled_at = nowIso();
      return send(res, 204);
    }
  }

  return fail(res, 404, "not_found", "Không tìm thấy đường dẫn.");
}

function keyView(k) {
  return { id: k.id, name: k.name, prefix: k.prefix, active: k.active, created_at: k.created_at, expires_at: k.expires_at, disabled_at: k.disabled_at, pack_count: (store.keyPacks.get(k.id) ?? new Set()).size };
}

function newKey(name, expiresAt) {
  const raw = `rsk_${randomBytes(24).toString("hex")}`;
  return { raw, key: { id: nextId("key"), name, prefix: raw.slice(0, 10), active: true, created_at: nowIso(), expires_at: expiresAt, disabled_at: null } };
}

function buildDesignFromDraft(draft, id, createdAt = nowIso()) {
  if (!draft?.name?.trim()) return null;
  const byId = new Map(store.fragments.map((f) => [f.id, f]));
  const gById = new Map(store.groups.map((g) => [g.id, g]));
  const d = { ...normalizeDraft(draft), id, name: draft.name.trim(), outline_width: draft.outline_width ?? 4, outline_color: draft.outline_color ?? "#FFFFFF" };
  const compiled = compileDesign(d, byId, gById, store.characters);
  return { id, name: d.name, character_id: d.character_id ?? "", rows: d.rows, columns: d.columns, outline_width: d.outline_width, outline_color: d.outline_color, draft: d, snapshot: compiled.snapshot, compiled_prompt: compiled.compiled_prompt, created_at: createdAt, updated_at: nowIso() };
}

// ---------- mock assets (generated, stateless) ----------

function mockAsset(p, res) {
  let m = /^\/mock-assets\/packs\/([^/]+)\/thumb\.svg$/.exec(p);
  if (m) return send(res, 200, stickerSvg(`${m[1]}-thumb`, { bg: true }), { "Content-Type": MIME[".svg"] });
  m = /^\/mock-assets\/packs\/([^/]+)\/stickers\/(\d+)\.svg$/.exec(p);
  if (m) return send(res, 200, stickerSvg(`${m[1]}-${m[2]}`), { "Content-Type": MIME[".svg"] });
  m = /^\/mock-assets\/packs\/([^/]+)\/animated\/(\d+)\.svg$/.exec(p);
  if (m) return send(res, 200, stickerSvg(`${m[1]}-animated-${m[2]}`, { bg: false }), { "Content-Type": MIME[".svg"] });
  m = /^\/mock-assets\/packs\/([^/]+)\/bundle\.zip$/.exec(p);
  if (m) return send(res, 200, Buffer.from(`mock bundle for ${m[1]} (not a real zip)\n`), { "Content-Type": "application/zip", "Content-Disposition": `attachment; filename="${m[1]}.zip"` });
  return false;
}

// ---------- static export ----------

function resolveStatic(pathname) {
  const decoded = decodeURIComponent(pathname);
  const rel = decoded === "/" ? "index.html" : decoded.replace(/^\/+/, "");
  const safe = path.normalize(rel).replace(/^(\.\.[/\\])+/, "");
  const base = path.join(STATIC_DIR, safe);
  const candidates = [base, path.join(base, "index.html"), `${base}.html`];
  for (const c of candidates) {
    if (c.startsWith(STATIC_DIR) && existsSync(c) && statSync(c).isFile()) return c;
  }
  return null;
}

function serveStatic(req, res, url) {
  const file = resolveStatic(url.pathname);
  if (!file) {
    const notFoundPage = path.join(STATIC_DIR, "404.html");
    if (existsSync(notFoundPage)) return send(res, 404, readFileSync(notFoundPage), { "Content-Type": MIME[".html"] });
    return send(res, 404, "Not found", { "Content-Type": "text/plain" });
  }
  const body = readFileSync(file);
  send(res, 200, req.method === "HEAD" ? "" : body, { "Content-Type": MIME[path.extname(file)] || "application/octet-stream" });
}

// ---------- server ----------

const DOCS_HTML = `<!doctype html><html lang="vi"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Restick API docs (mock)</title></head>
<body style="font-family:system-ui;padding:24px"><h1>Tài liệu API (mock)</h1><p>Trang tạm thay cho Swagger UI của máy chủ Go.</p><p><a href="/openapi.yaml">/openapi.yaml</a></p></body></html>`;
const OPENAPI_YAML = `openapi: 3.0.3
info:
  title: Restick admin API (mock stub)
  version: 0.0.0-mock
paths: {}
`;

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function handler(req, res) {
  const url = new URL(req.url, "http://localhost");
  try {
    if (url.pathname === "/__mock/scenario") {
      if (req.method === "GET") return send(res, 200, { scenario });
      if (req.method === "POST") {
        const body = parseJson(await readBody(req));
        if (!body || !SCENARIOS.includes(body.name)) return fail(res, 400, "bad_request", `name phải là một trong: ${SCENARIOS.join(", ")}`);
        scenario = body.name;
        const keepSessions = scenario === "loggedout" ? new Set() : store.sessions;
        store = buildStore();
        store.sessions = keepSessions;
        return send(res, 200, { scenario });
      }
      return fail(res, 405, "bad_request", "Method not allowed.");
    }
    if (url.pathname.startsWith("/api/") || url.pathname.startsWith("/v1/")) {
      if (scenario === "slow") await sleep(1500);
      if (scenario === "error" && url.pathname.startsWith("/api/v1/admin/") && url.pathname !== "/api/v1/admin/session") {
        await readBody(req);
        return fail(res, 500, "internal", "Máy chủ gặp lỗi. Thử lại sau.");
      }
      return await handleApi(req, res, url);
    }
    if (url.pathname.startsWith("/mock-assets/")) {
      if (mockAsset(url.pathname, res) !== false) return;
      return send(res, 404, "Not found", { "Content-Type": "text/plain" });
    }
    if (url.pathname === "/docs") return send(res, 200, DOCS_HTML, { "Content-Type": MIME[".html"] });
    if (url.pathname === "/openapi.yaml") return send(res, 200, OPENAPI_YAML, { "Content-Type": MIME[".yaml"] });
    return serveStatic(req, res, url);
  } catch (err) {
    console.error("[mock] handler error", err);
    if (!res.headersSent) fail(res, 500, "internal", "Máy chủ gặp lỗi. Thử lại sau.");
  }
}

if (!existsSync(STATIC_DIR)) {
  console.error(`[mock] static dir not found: ${STATIC_DIR}\n  run "npx next build" in clients/web-next first, or set STATIC_DIR`);
  process.exit(1);
}

http.createServer(handler).listen(PORT, "127.0.0.1", () => {
  console.log(`[mock] http://localhost:${PORT}  static=${STATIC_DIR}  scenario=${scenario}`);
  console.log(`[mock] login password "test"; control: POST /__mock/scenario {"name":"${SCENARIOS.join('|')}"}`);
});
