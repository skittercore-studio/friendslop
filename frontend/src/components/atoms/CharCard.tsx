import type { JSX } from "preact";
import { accentFor, glyphFor } from "../../lib/accent";

interface Props {
  /** Stable ID — used to derive sticky accent + glyph if not overridden. */
  id?: string;
  name: string;
  desc?: string;
  /** Override the auto-derived glyph (single emoji/symbol). */
  glyph?: string;
  /** Override the auto-derived accent. Falls back to id-derived hue. */
  accent?: string;
  /** Slight rotation in degrees for the trading-card feel. */
  tilt?: number;
  /** Mini variant for inline / list use — tighter padding, smaller name. */
  mini?: boolean;
  /** Click handler — when provided, card becomes a button. */
  onClick?: JSX.MouseEventHandler<HTMLDivElement>;
}

/**
 * Character card with the accent stripe across the top. The accent is
 * sticky-by-id by default — pass an explicit `accent` only when the
 * design needs to override (e.g. dimmed/incoming preview cards).
 */
export function CharCard({
  id,
  name,
  desc,
  glyph,
  accent,
  tilt = 0,
  mini = false,
  onClick,
}: Props): JSX.Element {
  const resolvedAccent = accent ?? accentFor(id);
  const resolvedGlyph = glyph ?? (id ? glyphFor(id) : undefined);
  return (
    <div
      class="fs-charcard"
      onClick={onClick}
      style={{
        // CSS custom properties are typed loosely; cast keeps strict TS happy.
        ["--char-accent" as string]: resolvedAccent,
        transform: tilt ? `rotate(${tilt}deg)` : undefined,
        padding: mini ? 10 : 14,
        cursor: onClick ? "pointer" : undefined,
      }}
    >
      <div
        class="fs-charcard__name"
        style={mini ? { fontSize: 16 } : undefined}
      >
        {resolvedGlyph && (
          <span class="glyph" style={{ marginRight: 8 }}>
            {resolvedGlyph}
          </span>
        )}
        {name}
      </div>
      {desc && <div class="fs-charcard__desc">{desc}</div>}
    </div>
  );
}
