/**
 * GuessPhase — STUB.
 *
 * Stage 2 / Agent D will replace this with the full design-port (combined
 * reveal + corkboard grid). See `design/STAGE_2_PLAN.md` and
 * `design/handoff-v1/friendslop-screens.jsx` (`RevealScreen` + `GridScreen`).
 *
 * Until then, fall through to the legacy game rendering so playtest doesn't
 * regress while Stage 2 work is in flight.
 */
import { LegacyGame } from "./LegacyGame";

interface Props {
  code: string;
}

export function GuessPhase({ code }: Props) {
  return <LegacyGame code={code} />;
}
