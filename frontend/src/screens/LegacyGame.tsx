import { useEffect, useMemo, useState } from "preact/hooks";
import * as api from "../api";
import { AnswerCard } from "../components/AnswerCard";
import { CharacterPool } from "../components/CharacterPool";
import { GuessGrid } from "../components/GuessGrid";
import { Scoreboard } from "../components/Scoreboard";
import { error, loading, me, room } from "../store";
import type { PublicCharacter } from "../types";

interface Props {
  code: string;
}

/**
 * LegacyGame — the original monolithic game screen.
 *
 * Renamed from `Game` during the Stage 2 design port so the new
 * state-dispatcher in `Game.tsx` can fall through here while AnswerPhase
 * and GuessPhase stubs are still in place. Once Stage 2 lands and both
 * new screens cover all states, delete this file along with the now-unused
 * components/AnswerCard, components/CharacterPool, components/GuessGrid.
 */
export function LegacyGame({ code }: Props) {
  const r = room.value;
  const m = me.value;

  if (!r || !r.current_round) {
    return <div class="placeholder">Loading game…</div>;
  }

  const cr = r.current_round;
  const characters = r.characters ?? [];
  const charById = useMemo(
    () => new Map<string, PublicCharacter>(characters.map((c) => [c.id, c])),
    [characters],
  );

  return (
    <div class="game">
      <section class="game-top">
        <CharacterPool characters={characters} />
      </section>

      <div class="game-mid">
        <section class="game-left">
          <RoundCard />
          {(r.state === "guessing" || r.state === "scoring") && cr.answers && (
            <div class="answers-stack">
              <h3>This round&rsquo;s answers</h3>
              {cr.answers.map((a) => (
                <AnswerCard
                  key={a.character_id}
                  character={charById.get(a.character_id) ?? null}
                  text={a.text}
                />
              ))}
            </div>
          )}
          {r.state === "answering" && <AnswerForm code={code} />}
          {m?.your_character && (
            <aside class="your-char">
              <h4>You are playing</h4>
              <div class="char-preview">
                <strong>{m.your_character.name}</strong>
                <p>{m.your_character.blurb}</p>
              </div>
              <p class="hint">
                Write in this character&rsquo;s voice. The pool above shows
                everyone&rsquo;s costume — but who&rsquo;s wearing which is the
                puzzle.
              </p>
            </aside>
          )}
        </section>

        <section class="game-right">
          <GuessGrid code={code} />
        </section>
      </div>

      <section class="game-bottom">
        <Scoreboard />
      </section>
    </div>
  );
}

function RoundCard() {
  const r = room.value;
  if (!r?.current_round) return null;
  const cr = r.current_round;

  return (
    <div class="round-card">
      <div class="round-head">
        <span class="round-number">Round {cr.number}</span>
        <span class="round-state" data-state={r.state}>
          {r.state}
        </span>
      </div>
      <div class="round-question">{cr.question_text}</div>
      <div class="round-deadlines">
        <Deadline label="Answer by" ts={cr.answer_deadline} />
        <Deadline label="Guess by" ts={cr.guess_deadline} />
      </div>
    </div>
  );
}

function Deadline({ label, ts }: { label: string; ts: number | null }) {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, []);
  if (!ts) return null;
  const remaining = Math.max(0, Math.floor((ts - now) / 1000));
  const m = Math.floor(remaining / 60);
  const s = remaining % 60;
  return (
    <span class="deadline">
      {label}{" "}
      <strong>
        {m}:{String(s).padStart(2, "0")}
      </strong>
    </span>
  );
}

function AnswerForm({ code }: { code: string }) {
  const r = room.value;
  const m = me.value;

  const existing = m?.your_answer_for_current_round ?? "";
  const [text, setText] = useState(existing);
  const [submittedLocal, setSubmittedLocal] = useState(Boolean(existing));

  useEffect(() => {
    setText(existing);
    setSubmittedLocal(Boolean(existing));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [existing, r?.current_round?.number]);

  if (!r?.current_round) return null;

  const onSubmit = async (e: Event) => {
    e.preventDefault();
    const trimmed = text.trim();
    if (trimmed.length === 0) {
      error.value = "Write something first.";
      return;
    }
    loading.value = true;
    try {
      await api.submitAnswer(code, r.current_round!.number, trimmed);
      setSubmittedLocal(true);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  };

  return (
    <form class="answer-form" onSubmit={onSubmit}>
      <h3>Your answer</h3>
      <textarea
        rows={5}
        maxLength={500}
        placeholder="Write in character…"
        value={text}
        disabled={submittedLocal}
        onInput={(e) => setText((e.target as HTMLTextAreaElement).value)}
      />
      <div class="answer-meta">
        <span class="hint">{text.length}/500 chars</span>
        <button
          class="primary"
          type="submit"
          disabled={submittedLocal || loading.value}
        >
          {submittedLocal ? "Submitted" : "Submit answer"}
        </button>
      </div>
    </form>
  );
}
