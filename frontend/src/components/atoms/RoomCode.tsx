import { JSX } from "preact";

interface RoomCodeProps {
  code: string;
  /** Pixel size for the type. Default 64. */
  size?: number;
  className?: string;
  /** Render a copy-on-tap button affordance. */
  copyable?: boolean;
}

/**
 * Big gradient room code. Optional copy-to-clipboard tap affordance.
 */
export function RoomCode({
  code,
  size = 64,
  className,
  copyable = false,
}: RoomCodeProps): JSX.Element {
  const cls = ["fs-roomcode", className].filter(Boolean).join(" ");
  const inner = (
    <span
      className={cls}
      style={{ fontSize: size }}
      aria-label={`Room code ${code}`}
    >
      {code}
    </span>
  );
  if (!copyable) return inner;
  return (
    <button
      type="button"
      className="fs-btn fs-btn--ghost"
      style={{
        background: "transparent",
        border: "none",
        padding: 0,
        cursor: "pointer",
      }}
      onClick={() => {
        if (typeof navigator !== "undefined" && navigator.clipboard) {
          navigator.clipboard.writeText(code).catch(() => {
            // best-effort; older Safari throws on insecure origins
          });
        }
      }}
      title="Copy room code"
    >
      {inner}
    </button>
  );
}
