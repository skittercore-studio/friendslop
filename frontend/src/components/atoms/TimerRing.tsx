import type { JSX } from "preact";

interface Props {
  /** Seconds remaining. Clamped 0..total inside the component. */
  remaining: number;
  /** Total seconds in this phase (denominator for the ring fill). */
  total: number;
  /** Diameter in px. Default 64. */
  size?: number;
  /** Force urgent styling (pink). Otherwise auto-triggers under 10s. */
  urgent?: boolean;
}

/**
 * Circular countdown ring. Stroke fills counter-clockwise as time runs
 * out, switches to --fs-live (pink) in the urgent state. Pure SVG, no
 * canvas, no animation library — the only motion is `transition` on
 * stroke-dasharray which the ring inherits when `remaining` ticks down.
 *
 * Server-authoritative: parent re-renders this every tick with a fresh
 * `remaining`. Don't try to animate locally between ticks.
 */
export function TimerRing({
  remaining,
  total,
  size = 64,
  urgent,
}: Props): JSX.Element {
  const safeTotal = Math.max(1, total);
  const safeRemaining = Math.max(0, Math.min(remaining, safeTotal));
  const r = (size - 8) / 2;
  const c = 2 * Math.PI * r;
  const pct = safeRemaining / safeTotal;
  const dash = c * pct;
  const m = Math.floor(safeRemaining / 60);
  const s = String(safeRemaining % 60).padStart(2, "0");
  const isUrgent = urgent ?? safeRemaining <= 10;
  const colour = isUrgent ? "var(--fs-live)" : "var(--fs-accent)";

  return (
    <div class="fs-ring" style={{ width: size, height: size }}>
      <svg width={size} height={size}>
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          stroke="var(--fs-line)"
          strokeWidth="3"
          fill="none"
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          stroke={colour}
          strokeWidth="3"
          fill="none"
          strokeDasharray={`${dash} ${c}`}
          strokeLinecap="round"
          style={{
            transition: "stroke-dasharray 0.4s linear, stroke 0.3s",
          }}
        />
      </svg>
      <span
        class="fs-ring__time"
        style={{ color: isUrgent ? "var(--fs-live)" : "var(--fs-fg)" }}
      >
        {m}:{s}
      </span>
    </div>
  );
}
