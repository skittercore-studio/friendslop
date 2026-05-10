import { ComponentChildren, JSX } from "preact";

interface IndexCardProps {
  children: ComponentChildren;
  /** Render the red corkboard pin at the top. */
  pinned?: boolean;
  /** Slight rotation to sell the corkboard vibe. */
  tilt?: number;
  /** Locked-in visual state — slight darkening, no pin animation. */
  locked?: boolean;
  className?: string;
  style?: JSX.CSSProperties;
  onClick?: JSX.MouseEventHandler<HTMLDivElement>;
}

/**
 * `.fs-idx` corkboard cell. The guess grid is built out of these.
 */
export function IndexCard({
  children,
  pinned = false,
  tilt = 0,
  locked = false,
  className,
  style,
  onClick,
}: IndexCardProps): JSX.Element {
  const cls = ["fs-idx", locked ? "fs-idx--locked" : "", className]
    .filter(Boolean)
    .join(" ");
  return (
    <div
      className={cls}
      style={{
        transform: tilt ? `rotate(${tilt}deg)` : undefined,
        opacity: locked ? 0.92 : 1,
        cursor: onClick ? "pointer" : undefined,
        ...style,
      }}
      onClick={onClick}
    >
      {pinned && <span className="fs-pin" />}
      {children}
    </div>
  );
}
