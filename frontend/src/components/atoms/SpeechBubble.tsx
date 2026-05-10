import { ComponentChildren, JSX } from "preact";

interface SpeechBubbleProps {
  children: ComponentChildren;
  /** Bottom tail position. "left" matches the design default. */
  tail?: "left" | "right" | "none";
  /**
   * Highlight the bubble with an accent-coloured ring + glow. Used on
   * the most recently revealed answer in the reveal screen — gives the
   * bubble the "this one just arrived" pop.
   */
  accent?: string;
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
  accent,
  className,
  style,
}: SpeechBubbleProps): JSX.Element {
  const cls = ["fs-bubble", className].filter(Boolean).join(" ");
  // Tail position is reflected via data-tail so the parent can style
  // alternative tail positions in CSS — the default in tokens.css already
  // handles "left". "none" is honoured by hiding ::before via that CSS.
  const accentStyle: JSX.CSSProperties = accent
    ? {
        boxShadow: `0 0 0 1.5px ${accent}, 0 14px 40px -8px ${accent}66`,
        borderColor: accent,
      }
    : {};
  return (
    <div
      class={cls}
      style={{ ...accentStyle, ...style }}
      data-tail={tail}
    >
      {children}
    </div>
  );
}
