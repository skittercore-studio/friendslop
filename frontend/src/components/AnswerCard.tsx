import type { PublicCharacter } from "../types";

interface Props {
  character: PublicCharacter | null;
  text: string;
}

/**
 * A single revealed answer. ALWAYS attributed to a CHARACTER, never a player.
 * If the character lookup fails (shouldn't happen — defensive), we render the
 * id verbatim rather than guessing.
 */
export function AnswerCard({ character, text }: Props) {
  return (
    <div class="answer-card">
      <div class="answer-attrib">
        <strong>{character?.name ?? "Unknown character"}</strong>{" "}
        <span class="said">said:</span>
      </div>
      <div class="answer-text">{text}</div>
    </div>
  );
}
