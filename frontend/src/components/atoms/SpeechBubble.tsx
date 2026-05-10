import { ComponentChildren, JSX } from "preact";

interface SpeechBubbleProps {
  children: ComponentChildren;
  /** Bottom tail position. "left" matches the design default. */
  tail?: "left" | "right" | "none";
  className?: string;
  style?: JSX.CSSProperties;
}

/**
 * `.fs-bubble` from the tokens. Used for character-attributed answer reveals.
 * The CSS pseudo-element draws the tail on the bottom-left by default; the
 * "right" variant flips it via inline style override.
 */
export function SpeechBubble({
  children,
  tail = "left",
  className,
  style,
}: SpeechBubbleProps): JSX.Element {
  const cls = ["fs-bubble", className].filter(Boolean).join(" ");
  const tailStyle: JSX.CSSProperties =
    tail === "right"
      ? { ...style }
      : tail === "none"
        ? { ...style }
        : style ?? {};
  return (
    <div
      className={cls}
      style={tailStyle}
      data-tail={tail}
    >
      {children}
    </div>
  );
}
