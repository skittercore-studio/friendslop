/**
 * GuessPhase — Stage 2 design port.
 *
 * Combines the RevealScreen (top half: stagger-fade reveal of each answer
 * with character + speech bubble) and the GridScreen (bottom half:
 * corkboard 1:1 mapping grid) into a single mounted screen. The dispatcher
 * in screens/Game.tsx mounts this for both `guessing` and `scoring` —
 * during scoring the editable column collapses into the historical strip.
 *
 * Reads from the global `room` / `me` / `now` signals; submits via
 * api.submitGuess. No local timers (relies on the 1Hz tick from
 * store.now). 1:1 enforcement validation is inline (a character can
 * be assigned to at most one player in the current round's draft).
 */
import { useMemo, useRef, useState } from "preact/hooks";
import * as api from "../api";
import {
  CharCard,
  IndexCard,
  SpeechBubble,
  TimerRing,
  accentFor,
} from "../components/atoms";
import { error, loading, me, now, room } from "../store";
import type { PublicCharacter, PublicPlayer } from "../types";

interface Props {
  code: string;
}

export function GuessPhase({ code }: Props) {
  const r = room.value;
  const m = me.value;
  const tick = now.value;

  // Cache total seconds on first render of this round so the ring's
  // denominator is stable as time ticks down. Same pattern as AnswerPhase.
  const roundNumber = r?.current_round?.number ?? null;
  const totalRef = useRef<{ round: number; total: number } | null>(null);
  const guessDeadline = r?.current_round?.guess_deadline ?? null;
  const remainingSec = guessDeadline
    ? Math.max(0, Math.floor((guessDeadline - tick) / 1000))
    : 0;
  if (
    roundNumber !== null &&
    (!totalRef.current || totalRef.current.round !== roundNumber)
  ) {
    totalRef.current = {
      round: roundNumber,
      total: Math.max(1, remainingSec || 1),
    };
  }
  const total = totalRef.current?.total ?? Math.max(1, remainingSec || 1);

  if (!r || !m || !r.current_round) {
    return (
      <div class="fs" style={{ padding: 20 }}>
        <span class="fs-faint">Loading…</span>
      </div>
    );
  }

  const cr = r.current_round;
  const characters = r.characters ?? [];
  const charById = new Map<string, PublicCharacter>(
    characters.map((c) => [c.id, c]),
  );

  const isScoring = r.state === "scoring";
  const isGuessing = r.state === "guessing";

  // Reveal section data: answers ordered as the server returned them.
  const answers = cr.answers ?? [];

  // Grid section data ----------------------------------------------------
  const others = r.players.filter(
    (p) => p.id !== m.player_id && !p.left,
  );
  const otherCount = others.length;

  const pastByRound = new Map(
    m.your_past_guesses.map((g) => [g.round_number, g]),
  );
  const pastRounds = m.your_past_guesses
    .map((g) => g.round_number)
    .sort((a, b) => a - b);

  const includeCurrent =
    isGuessing && !pastRounds.includes(cr.number);

  const alreadySubmitted = m.your_guess_for_current_round !== null;
  const editable = isGuessing && !alreadySubmitted;

  // Local draft mapping: target_player_id → character_id (or "").
  const draftSeed = useMemo(() => {
    const draft: Record<string, string> = {};
    if (m.your_guess_for_current_round) {
      Object.assign(draft, m.your_guess_for_current_round);
    }
    for (const p of others) {
      if (!(p.id in draft)) draft[p.id] = "";
    }
    return draft;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [m.player_id, cr.number, alreadySubmitted]);

  const [draft, setDraft] = useState<Record<string, string>>(draftSeed);

  // Reset draft on round transition / submit lock change.
  const draftKey = `${cr.number}:${alreadySubmitted}`;
  const [lastDraftKey, setLastDraftKey] = useState(draftKey);
  if (lastDraftKey !== draftKey) {
    setDraft(draftSeed);
    setLastDraftKey(draftKey);
  }

  // Picker state — which row is currently choosing a character.
  const [pickerFor, setPickerFor] = useState<string | null>(null);

  // 1:1 enforcement: a character can only be assigned to one row.
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
  const allFilled = filledIds.length === otherCount && otherCount > 0;
  const valid = allFilled && !dup;

  const setCell = (playerId: string, charId: string) => {
    setDraft((d) => ({ ...d, [playerId]: charId }));
  };

  const onSubmit = async () => {
    if (!valid || !editable) return;
    loading.value = true;
    try {
      await api.submitGuess(code, cr.number, draft);
      setPickerFor(null);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  };

  // Columns to render: all past rounds + (maybe) the current one.
  const columnRounds = [...pastRounds];
  if (includeCurrent) columnRounds.push(cr.number);

  return (
    <div
      class="fs"
      style={{
        minHeight: "100vh",
        background: "var(--fs-bg)",
        padding: "16px 18px 28px",
        display: "flex",
        flexDirection: "column",
        gap: 16,
      }}
    >
      {/* ── Top bar: title + timer ─────────────────────────────── */}
      <div
        class="fs-row fs-between"
        style={{ alignItems: "flex-start" }}
      >
        <div class="fs-col" style={{ gap: 2 }}>
          <span class="fs-display" style={{ fontSize: 26 }}>
            {isScoring ? "the verdict" : "the answers"}
          </span>
          <span class="fs-lbl">
            round {cr.number} · {answers.length}/{otherCount + 1} read
          </span>
        </div>
        {guessDeadline !== null && !isScoring && (
          <TimerRing
            remaining={remainingSec}
            total={total}
            size={56}
            urgent={remainingSec <= 10}
          />
        )}
      </div>

      {/* ── Question label ────────────────────────────────────── */}
      <div
        class="fs-lbl"
        style={{
          fontStyle: "italic",
          padding: "0 2px",
          color: "var(--fs-fg-mute)",
        }}
      >
        “{cr.question_text}”
      </div>

      {/* ── Reveal section ────────────────────────────────────── */}
      <div class="fs-col" style={{ gap: 10 }}>
        {answers.length === 0 ? (
          <div class="fs-faint fs-tiny">no answers revealed yet…</div>
        ) : (
          answers.map((a, i) => {
            const isFresh = i === answers.length - 1;
            const ch = charById.get(a.character_id);
            const accent = accentFor(a.character_id);
            return (
              <div
                key={a.character_id}
                class="fs-anim-slide"
                style={{
                  animationDelay: `${0.15 * i}s`,
                  opacity: isFresh ? 1 : 0.85,
                }}
              >
                <div
                  class="fs-row"
                  style={{ gap: 10, alignItems: "flex-start" }}
                >
                  <div style={{ flex: "0 0 100px" }}>
                    <CharCard
                      id={a.character_id}
                      name={ch?.name ?? "?"}
                      mini
                    />
                  </div>
                  <div class="fs-grow" style={{ minWidth: 0 }}>
                    <div
                      class="fs-tiny"
                      style={{ marginBottom: 4 }}
                    >
                      <span style={{ color: accent }}>
                        {(ch?.name ?? "unknown").toUpperCase()}
                      </span>{" "}
                      says
                    </div>
                    <SpeechBubble accent={isFresh ? accent : undefined}>
                      “{a.text}”
                    </SpeechBubble>
                  </div>
                </div>
              </div>
            );
          })
        )}
      </div>

      {/* ── Grid section ──────────────────────────────────────── */}
      <div class="fs-col" style={{ gap: 10, marginTop: 6 }}>
        <div class="fs-row fs-between">
          <span class="fs-display" style={{ fontSize: 20 }}>
            the board
          </span>
          <span class="fs-tiny">
            {editable
              ? "tap empty cell to assign"
              : alreadySubmitted
                ? "locked in"
                : "scoring…"}
          </span>
        </div>

        {columnRounds.length === 0 ? (
          <div class="fs-faint fs-tiny">
            the grid will fill in as rounds play out.
          </div>
        ) : (
          <div class="fs-col" style={{ gap: 8 }}>
            {/* column header */}
            <div
              class="fs-row"
              style={{ gap: 6, paddingLeft: 56 }}
            >
              {columnRounds.map((rn) => {
                const past = pastByRound.get(rn);
                const isCurrent = rn === cr.number && includeCurrent;
                return (
                  <span
                    key={`h-${rn}`}
                    class="fs-tiny fs-tac"
                    style={{ flex: 1 }}
                  >
                    R{rn}{" "}
                    {past ? (
                      <span
                        class="fs-chip"
                        style={{
                          padding: "1px 6px",
                          fontSize: 10,
                          marginLeft: 4,
                        }}
                      >
                        {past.correct_count}/{otherCount}
                      </span>
                    ) : isCurrent && isGuessing ? (
                      <span
                        class="fs-chip fs-chip--live"
                        style={{
                          padding: "1px 6px",
                          fontSize: 10,
                          marginLeft: 4,
                        }}
                      >
                        LIVE
                      </span>
                    ) : null}
                  </span>
                );
              })}
            </div>

            {/* rows */}
            {others.map((p, ri) => (
              <GridRow
                key={p.id}
                rowIndex={ri}
                player={p}
                columnRounds={columnRounds}
                currentRound={cr.number}
                pastByRound={pastByRound}
                charById={charById}
                editable={editable}
                draftCharId={draft[p.id] ?? ""}
                onCellTap={(rn) => {
                  if (rn !== cr.number || !editable) return;
                  // Tap an assigned cell to clear; tap empty to open picker.
                  if (draft[p.id]) {
                    setCell(p.id, "");
                  } else {
                    setPickerFor(p.id);
                  }
                }}
              />
            ))}
          </div>
        )}

        {/* Picker — appears when the user taps an empty editable cell. */}
        {pickerFor && editable && (
          <CharPicker
            characters={characters}
            takenIds={
              new Set(
                Object.entries(draft)
                  .filter(([k, v]) => k !== pickerFor && v)
                  .map(([, v]) => v),
              )
            }
            onPick={(cid) => {
              setCell(pickerFor, cid);
              setPickerFor(null);
            }}
            onCancel={() => setPickerFor(null)}
            forName={
              others.find((p) => p.id === pickerFor)?.name ?? "player"
            }
          />
        )}
      </div>

      <div class="fs-grow" />

      {/* ── Submit / status ───────────────────────────────────── */}
      {includeCurrent && (
        <div class="fs-col" style={{ gap: 8 }}>
          {alreadySubmitted ? (
            <div
              class="fs-chip"
              style={{
                alignSelf: "center",
                background: "var(--fs-bg-2)",
                color: "var(--fs-fg)",
                padding: "8px 14px",
                fontSize: 13,
              }}
            >
              guess locked in for R{cr.number}
            </div>
          ) : (
            <>
              {dup && (
                <span
                  class="fs-tiny"
                  style={{ color: "var(--fs-live)" }}
                >
                  each character can only be pinned to one player.
                </span>
              )}
              {!allFilled && !dup && (
                <span class="fs-tiny fs-faint">
                  pin a character to every row to lock in.
                </span>
              )}
              <button
                class={
                  valid
                    ? "fs-btn fs-btn--primary"
                    : "fs-btn fs-btn--disabled"
                }
                style={{ width: "100%" }}
                disabled={!valid || loading.value}
                onClick={onSubmit}
              >
                {loading.value ? "locking…" : "LOCK GUESSES →"}
              </button>
            </>
          )}
        </div>
      )}
      {isScoring && (
        <div
          class="fs-chip"
          style={{
            alignSelf: "center",
            background: "var(--fs-bg-2)",
            color: "var(--fs-fg)",
            padding: "8px 14px",
            fontSize: 13,
          }}
        >
          scoring round {cr.number}…
        </div>
      )}
    </div>
  );
}

// ── Grid row ───────────────────────────────────────────────────────────
interface GridRowProps {
  rowIndex: number;
  player: PublicPlayer;
  columnRounds: number[];
  currentRound: number;
  pastByRound: Map<number, { mapping: Record<string, string> }>;
  charById: Map<string, PublicCharacter>;
  editable: boolean;
  draftCharId: string;
  onCellTap: (roundNumber: number) => void;
}

function GridRow({
  rowIndex,
  player,
  columnRounds,
  currentRound,
  pastByRound,
  charById,
  editable,
  draftCharId,
  onCellTap,
}: GridRowProps) {
  return (
    <div class="fs-row" style={{ gap: 6 }}>
      <span
        class="fs-lbl"
        style={{
          width: 50,
          fontWeight: 600,
          color: "var(--fs-fg)",
          fontSize: 13,
          textAlign: "right",
          paddingRight: 4,
          overflow: "hidden",
          textOverflow: "ellipsis",
          whiteSpace: "nowrap",
        }}
      >
        {player.name}
      </span>
      {columnRounds.map((rn, ci) => {
        const isCurrent = rn === currentRound;
        const past = pastByRound.get(rn);
        const tilt = (rowIndex + ci) % 2 === 0 ? -1.4 : 1.2;

        if (isCurrent) {
          // Editable / staged current column.
          const cid = draftCharId;
          const ch = cid ? charById.get(cid) : null;
          if (!cid) {
            return (
              <div key={`c-${rn}`} style={{ flex: 1 }}>
                <div
                  onClick={() => editable && onCellTap(rn)}
                  style={{
                    borderRadius: 6,
                    border: "1.5px dashed var(--fs-line)",
                    height: 56,
                    display: "grid",
                    placeItems: "center",
                    background: "rgba(255,214,10,0.04)",
                    cursor: editable ? "pointer" : "default",
                  }}
                >
                  <span
                    class="fs-tiny fs-faint"
                    style={{ fontSize: 10 }}
                  >
                    {editable ? "tap to pin" : "—"}
                  </span>
                </div>
              </div>
            );
          }
          const accent = accentFor(cid);
          return (
            <div key={`c-${rn}`} style={{ flex: 1 }}>
              <IndexCard
                pinned
                tilt={tilt}
                onClick={editable ? () => onCellTap(rn) : undefined}
                style={{
                  height: 56,
                  boxShadow: editable
                    ? `0 0 0 2px ${accent}, 0 8px 22px -4px rgba(0,0,0,0.5)`
                    : undefined,
                }}
              >
                <div
                  style={{
                    fontSize: 10,
                    fontWeight: 700,
                    letterSpacing: "0.06em",
                    color: accent,
                    textTransform: "uppercase",
                  }}
                >
                  R{rn}
                </div>
                <div
                  style={{
                    fontFamily: "'Lilita One', sans-serif",
                    fontSize: 13,
                    marginTop: 14,
                    color: "#2a2410",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                >
                  {ch?.name ?? "?"}
                </div>
              </IndexCard>
            </div>
          );
        }

        // Past column.
        const cid = past?.mapping[player.id];
        const ch = cid ? charById.get(cid) : null;
        if (!cid) {
          return (
            <div key={`c-${rn}`} style={{ flex: 1 }}>
              <div
                style={{
                  borderRadius: 6,
                  border: "1.5px dashed var(--fs-line-soft)",
                  height: 56,
                  display: "grid",
                  placeItems: "center",
                  background: "transparent",
                  opacity: 0.5,
                }}
              >
                <span
                  class="fs-tiny fs-faint"
                  style={{ fontSize: 10 }}
                >
                  —
                </span>
              </div>
            </div>
          );
        }
        const accent = accentFor(cid);
        return (
          <div key={`c-${rn}`} style={{ flex: 1 }}>
            <IndexCard
              tilt={tilt}
              locked
              style={{ height: 56 }}
            >
              {/* Neutral pin: in mid-game we don't know which past
                  cells were right; the column header pill carries the
                  per-round score. */}
              <div
                class="fs-pin"
                style={{ background: "var(--fs-fg-faint)" }}
              />
              <div
                style={{
                  fontSize: 10,
                  fontWeight: 700,
                  letterSpacing: "0.06em",
                  color: accent,
                  textTransform: "uppercase",
                }}
              >
                R{rn}
              </div>
              <div
                style={{
                  fontFamily: "'Lilita One', sans-serif",
                  fontSize: 13,
                  marginTop: 14,
                  color: "#2a2410",
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap",
                }}
              >
                {ch?.name ?? "?"}
              </div>
            </IndexCard>
          </div>
        );
      })}
    </div>
  );
}

// ── Picker overlay ─────────────────────────────────────────────────────
interface CharPickerProps {
  characters: PublicCharacter[];
  takenIds: Set<string>;
  forName: string;
  onPick: (charId: string) => void;
  onCancel: () => void;
}

function CharPicker({
  characters,
  takenIds,
  forName,
  onPick,
  onCancel,
}: CharPickerProps) {
  const available = characters.filter((c) => !takenIds.has(c.id));
  return (
    <div
      onClick={onCancel}
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.55)",
        display: "flex",
        alignItems: "flex-end",
        justifyContent: "center",
        zIndex: 50,
        padding: 16,
      }}
    >
      <div
        class="fs-card"
        onClick={(e) => e.stopPropagation()}
        style={{
          padding: 14,
          width: "100%",
          maxWidth: 460,
          background: "var(--fs-bg-1)",
        }}
      >
        <div
          class="fs-row fs-between"
          style={{ marginBottom: 10 }}
        >
          <span class="fs-display" style={{ fontSize: 18 }}>
            pin a character
          </span>
          <span class="fs-tiny fs-faint">for {forName}</span>
        </div>
        {available.length === 0 ? (
          <div class="fs-tiny fs-faint">
            all characters are already pinned. tap an existing pin to
            unassign.
          </div>
        ) : (
          <div
            class="fs-col"
            style={{ gap: 8, maxHeight: "60vh", overflowY: "auto" }}
          >
            {available.map((c) => (
              <CharCard
                key={c.id}
                id={c.id}
                name={c.name}
                desc={c.blurb}
                mini
                onClick={() => onPick(c.id)}
              />
            ))}
          </div>
        )}
        <button
          class="fs-btn fs-btn--ghost"
          style={{ width: "100%", marginTop: 10 }}
          onClick={onCancel}
        >
          cancel
        </button>
      </div>
    </div>
  );
}
