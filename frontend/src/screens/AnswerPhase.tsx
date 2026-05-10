/**
 * AnswerPhase — the in-character answering screen.
 *
 * Stage 2 design port. Mounted by `screens/Game.tsx` when
 * `room.value.state === "answering"`. Source of truth: `room`/`me` signals
 * from `store.ts`. Submits via `api.submitAnswer`. The 1Hz `now` signal
 * drives the timer countdown — we deliberately do NOT spawn our own
 * setInterval.
 *
 * Design ref: `design/handoff-v1/friendslop-screens.jsx` → `AnsweringScreen`.
 */
import { useEffect, useRef, useState } from "preact/hooks";
import * as api from "../api";
import { CharCard, TimerRing, accentFor } from "../components/atoms";
import { error, loading, me, now, room } from "../store";

interface Props {
  code: string;
}

export function AnswerPhase({ code }: Props) {
  const r = room.value;
  const m = me.value;

  // Timer-total caching: the backend doesn't expose answer_timeout_seconds
  // on the public snapshot, so we approximate "total" by snapshotting the
  // remaining time on the first render of each round. The ring will read
  // 100% on entry and tick down naturally. Reset whenever the round number
  // changes. KNOWN LIMITATION: a player who joins late (or hard-refreshes
  // mid-round) will see the ring start partially filled rather than at the
  // true total; the displayed time is still accurate.
  const totalRef = useRef<{ round: number; total: number } | null>(null);

  const existing = m?.your_answer_for_current_round ?? null;
  const [text, setText] = useState(existing ?? "");
  const [submittedLocal, setSubmittedLocal] = useState(Boolean(existing));

  // When the round flips or the server confirms a different stored answer,
  // re-sync local state. (Same flow as the legacy AnswerForm.)
  useEffect(() => {
    setText(existing ?? "");
    setSubmittedLocal(Boolean(existing));
  }, [existing, r?.current_round?.number]);

  if (!r || !r.current_round || !m) {
    return <div class="placeholder">Loading game…</div>;
  }

  const cr = r.current_round;
  const deadline = cr.answer_deadline;
  const remaining =
    deadline !== null
      ? Math.max(0, Math.floor((deadline - now.value) / 1000))
      : 0;

  // Cache total for this round on first observation.
  if (!totalRef.current || totalRef.current.round !== cr.number) {
    totalRef.current = { round: cr.number, total: Math.max(remaining, 1) };
  }
  const total = totalRef.current.total;
  const urgent = remaining <= 10;

  const activePlayers = r.players.filter((p) => !p.left).length;

  const isLocked = submittedLocal || existing !== null;

  const onSubmit = async (e: Event) => {
    e.preventDefault();
    const trimmed = text.trim();
    if (trimmed.length === 0) {
      error.value = "Write something first.";
      return;
    }
    loading.value = true;
    try {
      await api.submitAnswer(code, cr.number, trimmed);
      setSubmittedLocal(true);
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      loading.value = false;
    }
  };

  const charAccent = m.your_character ? accentFor(m.your_character.id) : undefined;

  return (
    <div
      class="fs"
      style={{
        minHeight: "100vh",
        background: "var(--fs-bg)",
        padding: "16px 20px 24px",
        display: "flex",
        flexDirection: "column",
        gap: 0,
      }}
    >
      {/* top: timer + meta */}
      <div
        class="fs-row fs-between"
        style={{ alignItems: "flex-start", gap: 12 }}
      >
        <TimerRing
          remaining={remaining}
          total={total}
          size={68}
          urgent={urgent}
        />
        <div
          class="fs-col"
          style={{ flex: 1, marginLeft: 4, gap: 4, alignItems: "flex-start" }}
        >
          <span class="fs-tiny">round {cr.number}</span>
          <span class="fs-lbl">
            {activePlayers} player{activePlayers === 1 ? "" : "s"}
          </span>
        </div>
      </div>

      {/* question card */}
      <div
        class="fs-card"
        style={{
          marginTop: 14,
          padding: "16px 18px",
          background: "linear-gradient(180deg, var(--fs-bg-2), var(--fs-bg-1))",
          borderColor: "var(--fs-bg-3)",
        }}
      >
        <div class="fs-tiny" style={{ color: "var(--fs-accent)" }}>
          THE QUESTION
        </div>
        <div
          class="fs-display"
          style={{ fontSize: 26, marginTop: 6, lineHeight: 1.1 }}
        >
          {cr.question_text}
        </div>
      </div>

      {/* you are playing */}
      <div class="fs-tiny fs-tac" style={{ marginTop: 16 }}>
        YOU ARE PLAYING
      </div>
      <div style={{ marginTop: 8 }}>
        {m.your_character ? (
          <CharCard
            id={m.your_character.id}
            name={m.your_character.name}
            desc={m.your_character.blurb}
            accent={charAccent}
            tilt={-1}
          />
        ) : (
          <div class="fs-card fs-muted" style={{ padding: 14 }}>
            (no character assigned)
          </div>
        )}
      </div>

      {/* answer area */}
      <label class="fs-tiny" style={{ marginTop: 16, display: "block" }}>
        answer in their voice
      </label>
      <form
        onSubmit={onSubmit}
        style={{
          marginTop: 6,
          display: "flex",
          flexDirection: "column",
          gap: 10,
        }}
      >
        <div
          class="fs-card"
          style={{
            padding: "12px 14px",
            background: "var(--fs-bg-1)",
            borderColor: "var(--fs-line)",
          }}
        >
          <textarea
            rows={5}
            maxLength={500}
            placeholder="write something they'd say…"
            value={text}
            disabled={isLocked}
            onInput={(e) => setText((e.target as HTMLTextAreaElement).value)}
            style={{
              width: "100%",
              minHeight: 110,
              background: "transparent",
              border: "none",
              outline: "none",
              resize: "vertical",
              color: "var(--fs-fg)",
              fontFamily: "'JetBrains Mono', ui-monospace, monospace",
              fontSize: 14,
              lineHeight: 1.5,
              letterSpacing: "0.02em",
              padding: 0,
            }}
          />
        </div>

        <div class="fs-row fs-between" style={{ alignItems: "center" }}>
          <span class="fs-lbl fs-mono">{text.length}/500</span>
          {isLocked ? (
            <span class="fs-chip fs-chip--on">ANSWER LOCKED</span>
          ) : (
            <span class="fs-tiny fs-muted">
              {urgent ? "ring goes pink — don't choke" : ""}
            </span>
          )}
        </div>

        {isLocked ? (
          <button class="fs-btn fs-btn--disabled" type="button" disabled>
            answer locked, waiting…
          </button>
        ) : (
          <button
            class="fs-btn fs-btn--primary"
            type="submit"
            disabled={loading.value || text.trim().length === 0}
            style={{ width: "100%" }}
          >
            LOCK IT IN →
          </button>
        )}
      </form>
    </div>
  );
}
