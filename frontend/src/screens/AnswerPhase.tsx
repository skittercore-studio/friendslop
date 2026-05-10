/**
 * AnswerPhase — STUB.
 *
 * Stage 2 / Agent C will replace this with the full design-port. See
 * `design/STAGE_2_PLAN.md` and `design/handoff-v1/friendslop-screens.jsx`
 * (`AnsweringScreen`) for the spec.
 *
 * Until then, this stub keeps the dispatcher in `screens/Game.tsx` valid
 * by exporting a placeholder that simply forwards to the legacy game
 * rendering. We intentionally re-import the legacy markup verbatim so
 * playtest doesn't regress while Stage 2 work is in flight.
 */
import { LegacyGame } from "./LegacyGame";

interface Props {
  code: string;
}

export function AnswerPhase({ code }: Props) {
  return <LegacyGame code={code} />;
}
