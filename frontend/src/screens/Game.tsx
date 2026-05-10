/**
 * Game — state-machine dispatcher.
 *
 * The router in `store.ts` collapses the three in-game backend states
 * (`answering` / `guessing` / `scoring`) into a single `screen.kind ===
 * "game"`. This component then dispatches to the right phase screen
 * based on the live `room.state`.
 *
 * - answering             → AnswerPhase
 * - guessing | scoring    → GuessPhase
 * - anything else (defensive) → loading placeholder
 */
import { room } from "../store";
import { AnswerPhase } from "./AnswerPhase";
import { GuessPhase } from "./GuessPhase";

interface Props {
  code: string;
}

export function Game({ code }: Props) {
  const r = room.value;
  if (!r) {
    return <div class="placeholder">Loading game…</div>;
  }

  switch (r.state) {
    case "answering":
      return <AnswerPhase code={code} />;
    case "guessing":
    case "scoring":
      return <GuessPhase code={code} />;
    default:
      // Defensive — the parent router should never send us anything else,
      // but if state slipped (e.g. mid-transition SSE), fall back to a
      // gentle loading state rather than rendering nothing.
      return <div class="placeholder">Loading game…</div>;
  }
}
