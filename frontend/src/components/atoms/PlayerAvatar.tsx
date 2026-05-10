import { JSX } from "preact";
import { accentForPlayer, type AccentHex } from "./palette";

interface PlayerAvatarProps {
  /** Player id — drives default accent. */
  id?: string;
  /** Display label — first letter rendered if longer. */
  name: string;
  size?: "sm" | "md" | "lg" | "xl";
  /**
   * If "accent", fills with the resolved accent and uses dark text.
   * If "outline" (default), neutral surface with accent ring.
   * If "empty", dashed placeholder.
   */
  variant?: "outline" | "accent" | "empty";
  accent?: AccentHex | string;
  className?: string;
  title?: string;
  onClick?: JSX.MouseEventHandler<HTMLButtonElement | HTMLDivElement>;
}

const SIZE_CLASS: Record<NonNullable<PlayerAvatarProps["size"]>, string> = {
  sm: "fs-av--sm",
  md: "",
  lg: "fs-av--lg",
  xl: "fs-av--xl",
};

export function PlayerAvatar({
  id,
  name,
  size = "md",
  variant = "outline",
  accent,
  className,
  title,
  onClick,
}: PlayerAvatarProps): JSX.Element {
  const resolved = accent ?? (id ? accentForPlayer(id) : "#ffd60a");
  const initial = (name?.trim()?.[0] ?? "?").toUpperCase();
  const variantClass =
    variant === "accent"
      ? "fs-av--accent"
      : variant === "empty"
        ? "fs-av--empty"
        : "";
  const cls = ["fs-av", SIZE_CLASS[size], variantClass, className]
    .filter(Boolean)
    .join(" ");
  const style: JSX.CSSProperties =
    variant === "outline"
      ? {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          ["--char-accent" as any]: resolved,
          borderColor: resolved,
        }
      : variant === "accent"
        ? {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            ["--char-accent" as any]: resolved,
            background: resolved,
          }
        : {};

  if (onClick) {
    return (
      <button
        type="button"
        className={cls}
        style={style}
        title={title ?? name}
        onClick={onClick as JSX.MouseEventHandler<HTMLButtonElement>}
      >
        {variant === "empty" ? "?" : initial}
      </button>
    );
  }
  return (
    <div className={cls} style={style} title={title ?? name}>
      {variant === "empty" ? "?" : initial}
    </div>
  );
}
