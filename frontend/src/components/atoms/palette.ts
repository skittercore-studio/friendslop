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
 */
export function accentFor(id: string): AccentHex {
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
export function accentForPlayer(id: string): AccentHex {
  return accentFor(`p:${id}`);
}
