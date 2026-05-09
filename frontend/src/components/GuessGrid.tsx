import { useMemo, useState } from "preact/hooks";
import * as api from "../api";
import { error, loading, me, room } from "../store";
import type { PublicCharacter, PublicPlayer } from "../types";

/**
 * GuessGrid — the killer feature.
 *
 * Rows: each OTHER player (excluding self).
 * Columns: each round (read-only past rounds + one editable current-round
 *          column when state == "guessing" and you haven't submitted yet).
 * Right margin: per-round correct count shown in the column header for past
 *               rounds (your X / N total others).
 *
 * 1:1 enforcement: a character may only be assigned to one player in a single
 * round. Client validates before enabling submit; server is the source of truth.
 */

interface Props {
  code: string;
}

export function GuessGrid({ code }: Props) {
  const r = room.value;
  const m = me.value;

  if (!r || !m) {
    return <div class="placeholder">Loading…</div>;
  }

  const others = r.players.filter((p) => p.id !== m.player_id && !p.left);
  const characters = r.characters ?? [];
  const charById = new Map<string, PublicCharacter>(
    characters.map((c) => [c.id, c]),
  );
  const otherCount = others.length;

  // Past guesses indexed by round number.
  const pastByRound = new Map(
    m.your_past_guesses.map((g) => [g.round_number, g]),
  );

  const pastRounds = m.your_past_guesses
    .map((g) => g.round_number)
    .sort((a, b) => a - b);
  const currentRoundNumber =
    r.current_round && r.state === "guessing" ? r.current_round.number : null;
  const includeCurrent =
    currentRoundNumber !== null && !pastRounds.includes(currentRoundNumber);

  // Editable mode: state must be guessing AND no submitted guess yet for this round.
  const alreadySubmitted = m.your_guess_for_current_round !== null;
  const editable = r.state === "guessing" && !alreadySubmitted;

  // Local draft mapping: target_player_id → character_id (or empty).
  const initialDraft = useMemo(() => {
    const draft: Record<string, string> = {};
    if (m.your_guess_for_current_round) {
      Object.assign(draft, m.your_guess_for_current_round);
    }
    for (const p of others) {
      if (!(p.id in draft)) draft[p.id] = "";
    }
    return draft;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [m.player_id, r.current_round?.number, alreadySubmitted]);

  const [draft, setDraft] = useState<Record<string, string>>(initialDraft);

  // If round changes (e.g. SSE refresh moved us to a new round), reset draft.
  const draftKey = `${r.current_round?.number ?? "n/a"}:${alreadySubmitted}`;
  const [lastDraftKey, setLastDraftKey] = useState(draftKey);
  if (lastDraftKey !== draftKey) {
    setDraft(initialDraft);
    setLastDraftKey(draftKey);
  }

  const updateDraft = (targetId: string, charId: string) => {
    setDraft((d) => ({ ...d, [targetId]: charId }));
  };

  // Validation: every other player must have a non-empty character, and no
  // character may appear twice. (Self-character is allowed — server doesn't
  // know yours, and it's a valid guess.)
  const filledIds = others.map((p) => draft[p.id]).filter(Boolean);
  const usedSet = new Set<string>();
  let dup = false;
  for (const cid of filledIds) {
    if (usedSet.has(cid)) {
      dup = true;
      break;
    }
    usedSet.add(cid);
  }
  const allFilled = filledIds.length === others.length;
  const valid = allFilled && !dup;

  const onSubmit = async () => {
    if (!valid || currentRoundNumber === null) return;
    loading.value = true;
    try {
      await api.submitGuess(code, currentRoundNumber, draft);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  };

  return (
    <div class="guess-grid-wrap">
      <div class="guess-grid-header">
        <h3>Guess grid</h3>
        <p class="hint">
          Who is who? Each row is a player. Each column is a round. Drop a
          character for each player to record your guess.
        </p>
      </div>

      <div class="guess-grid-scroll">
        <table class="guess-grid">
          <thead>
            <tr>
              <th class="col-player">Player</th>
              {pastRounds.map((rn) => {
                const g = pastByRound.get(rn);
                return (
                  <th key={`r${rn}`} class="col-round past">
                    <div class="rh-label">R{rn}</div>
                    <div class="rh-score">
                      {g ? `${g.correct_count}/${otherCount}` : "—"}
                    </div>
                  </th>
                );
              })}
              {includeCurrent && currentRoundNumber !== null && (
                <th class="col-round current">
                  <div class="rh-label">R{currentRoundNumber}</div>
                  <div class="rh-score">
                    {alreadySubmitted ? "submitted" : editable ? "editing" : "—"}
                  </div>
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            {others.map((p) => (
              <GuessRow
                key={p.id}
                player={p}
                pastRounds={pastRounds}
                pastByRound={pastByRound}
                charById={charById}
                characters={characters}
                includeCurrent={includeCurrent}
                editable={editable}
                draftCharId={draft[p.id] ?? ""}
                takenIds={
                  new Set(
                    Object.entries(draft)
                      .filter(([k, v]) => k !== p.id && v)
                      .map(([, v]) => v),
                  )
                }
                onChange={(cid) => updateDraft(p.id, cid)}
              />
            ))}
          </tbody>
        </table>
      </div>

      {includeCurrent && (
        <div class="guess-grid-actions">
          {alreadySubmitted ? (
            <span class="submitted-pill">
              Guess locked in for R{currentRoundNumber}
            </span>
          ) : (
            <>
              {dup && (
                <span class="hint warn">
                  Each character can only be assigned once per round.
                </span>
              )}
              {!allFilled && (
                <span class="hint">
                  Pick a character for every other player to submit.
                </span>
              )}
              <button
                class="primary"
                disabled={!valid || loading.value}
                onClick={onSubmit}
              >
                Submit guess for R{currentRoundNumber}
              </button>
            </>
          )}
        </div>
      )}
    </div>
  );
}

interface GuessRowProps {
  player: PublicPlayer;
  pastRounds: number[];
  pastByRound: Map<number, { mapping: Record<string, string> }>;
  charById: Map<string, PublicCharacter>;
  characters: PublicCharacter[];
  includeCurrent: boolean;
  editable: boolean;
  draftCharId: string;
  takenIds: Set<string>;
  onChange: (charId: string) => void;
}

function GuessRow({
  player,
  pastRounds,
  pastByRound,
  charById,
  characters,
  includeCurrent,
  editable,
  draftCharId,
  takenIds,
  onChange,
}: GuessRowProps) {
  return (
    <tr>
      <td class="col-player">
        <strong>{player.name}</strong>
      </td>
      {pastRounds.map((rn) => {
        const g = pastByRound.get(rn);
        const cid = g?.mapping[player.id];
        const ch = cid ? charById.get(cid) : null;
        return (
          <td key={`r${rn}`} class="col-round past">
            <span class="cell-char">{ch?.name ?? (cid ? "?" : "—")}</span>
          </td>
        );
      })}
      {includeCurrent && (
        <td class="col-round current">
          {editable ? (
            <select
              value={draftCharId}
              onChange={(e) =>
                onChange((e.target as HTMLSelectElement).value)
              }
            >
              <option value="">— pick —</option>
              {characters.map((c) => (
                <option
                  key={c.id}
                  value={c.id}
                  disabled={takenIds.has(c.id) && c.id !== draftCharId}
                >
                  {c.name}
                </option>
              ))}
            </select>
          ) : (
            <span class="cell-char">
              {draftCharId
                ? (charById.get(draftCharId)?.name ?? "?")
                : "—"}
            </span>
          )}
        </td>
      )}
    </tr>
  );
}
