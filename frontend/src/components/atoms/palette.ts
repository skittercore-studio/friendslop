// Friendslop accent palette — 12 sticky hues.
// Order matches design/handoff-v1/friendslop-tokens.css (--c-yellow … --c-peri).
export const PALETTE = [
  "#ffd60a", // yellow
  "#ff3d8a", // pink
  "#00e8d4", // teal
  "#c4ff3a", // lime
  "#ff7a3a", // orange
  "#b89cff", // lavender
  "#4ad6ff", // sky
  "#e065ff", // magenta
  "#ff6b6b", // coral
  "#6effbe", // mint
  "#ffb13d", // gold
  "#7e9cff", // peri
] as const;

export type AccentHex = (typeof PALETTE)[number];

/**
 * Deterministic character → palette accent.
 * Hashes the character id (FNV-1a 32-bit) and indexes into PALETTE.
 * Stable across rounds and across clients — same character_id always
 * gets the same hue, no server coordination needed.
 *
 * If `id` is undefined or empty, falls back to PALETTE[0] (yellow) so
 * call sites that thread an optional id stay clean without a guard.
 */
export function accentFor(id: string | undefined): AccentHex {
  if (!id) return PALETTE[0];
  let h = 0x811c9dc5;
  for (let i = 0; i < id.length; i++) {
    h ^= id.charCodeAt(i);
    // FNV prime 16777619 — using Math.imul to keep it 32-bit
    h = Math.imul(h, 0x01000193);
  }
  // unsigned modulo
  const idx = (h >>> 0) % PALETTE.length;
  return PALETTE[idx];
}

/**
 * Same as accentFor but for player ids — keeps the lobby ring & avatars
 * tinted consistently per player. Distinct namespace so a character and a
 * player with the same id can still collide; in practice they won't.
 */
export function accentForPlayer(id: string | undefined): AccentHex {
  if (!id) return PALETTE[0];
  return accentFor(`p:${id}`);
}

/**
 * A small repertoire of single-glyph symbols for character cards. The
 * design uses a glyph-per-character (☠ ☕ ✦ ◈ …) but the backend doesn't
 * yet ship a glyph field — we synthesise one deterministically by id so
 * the same character keeps the same glyph across rounds and reloads.
 *
 * When the server adds a glyph field, callers should prefer that and
 * fall back to this only when the field is missing.
 */
const GLYPHS = [
  "◈",
  "✦",
  "☼",
  "♆",
  "☾",
  "☄",
  "✺",
  "✪",
  "❖",
  "⚙",
  "⚜",
  "✧",
] as const;

export function glyphFor(id: string | undefined): string {
  if (!id) return GLYPHS[0];
  // Reuse the same FNV walk as accentFor, but indexed into GLYPHS.
  let h = 0x811c9dc5;
  for (let i = 0; i < id.length; i++) {
    h ^= id.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return GLYPHS[(h >>> 0) % GLYPHS.length];
}
