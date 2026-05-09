import { useEffect, useState } from "preact/hooks";
import * as api from "../api";
import { charcreateProgress, error, loading, me, room } from "../store";

interface Props {
  code: string;
}

export function CharCreate({ code }: Props) {
  const r = room.value;
  const m = me.value;

  const [charName, setCharName] = useState("");
  const [blurb, setBlurb] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [submittedChar, setSubmittedChar] = useState<{
    name: string;
    blurb: string;
  } | null>(null);

  // Detect prior submission via /me's authored character id (presence ⇒ submitted).
  useEffect(() => {
    if (m?.your_authored_character_id && !submitted) {
      setSubmitted(true);
    }
  }, [m?.your_authored_character_id]);

  const onSubmit = async (e: Event) => {
    e.preventDefault();
    const n = charName.trim();
    const b = blurb.trim();
    if (n.length < 1 || n.length > 60) {
      error.value = "Character name must be 1-60 characters.";
      return;
    }
    if (b.length < 20 || b.length > 300) {
      error.value = "Blurb must be 20-300 characters.";
      return;
    }
    loading.value = true;
    try {
      await api.submitCharacter(code, n, b);
      setSubmittedChar({ name: n, blurb: b });
      setSubmitted(true);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  };

  if (!r) {
    return <div class="placeholder">Loading…</div>;
  }

  const totalPlayers = r.players.filter((p) => !p.left).length;
  const counter = charcreateProgress.value ?? {
    submitted: submitted ? 1 : 0,
    total: totalPlayers,
  };

  return (
    <div class="charcreate">
      <h2>Write a character</h2>
      <p class="hint">
        Make it distinctive — voice, quirks, vibe. Other players will be
        randomly assigned characters from the combined pool, so don&rsquo;t
        make yours self-portraitey unless you want to be guessed instantly.
      </p>

      {!submitted ? (
        <form class="form" onSubmit={onSubmit}>
          <label class="field">
            <span>Name</span>
            <input
              type="text"
              maxLength={60}
              value={charName}
              onInput={(e) =>
                setCharName((e.target as HTMLInputElement).value)
              }
              required
            />
          </label>
          <label class="field">
            <span>Blurb (20-300 chars)</span>
            <textarea
              rows={5}
              maxLength={300}
              value={blurb}
              onInput={(e) =>
                setBlurb((e.target as HTMLTextAreaElement).value)
              }
              required
            />
            <span class="field-hint">{blurb.length}/300</span>
          </label>
          <button class="primary" type="submit" disabled={loading.value}>
            Submit character
          </button>
        </form>
      ) : (
        <div class="submitted-box">
          <h3>Your submission</h3>
          {submittedChar ? (
            <div class="char-preview">
              <strong>{submittedChar.name}</strong>
              <p>{submittedChar.blurb}</p>
            </div>
          ) : (
            <p class="hint">Locked in. Waiting for others.</p>
          )}
        </div>
      )}

      <div class="charcreate-counter">
        <span class="big-number">
          {counter.submitted} of {counter.total}
        </span>{" "}
        players have submitted
      </div>
    </div>
  );
}
