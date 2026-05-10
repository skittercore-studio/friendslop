import { useEffect, useMemo, useState } from "preact/hooks";
import * as api from "../api";
import { CharCard, StatusPill, accentFor } from "../components/atoms";
import { charcreateProgress, error, loading, me, room } from "../store";

interface Props {
  code: string;
}

const NAME_MIN = 1;
const NAME_MAX = 60;
const BLURB_MIN = 20;
const BLURB_MAX = 300;

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
    if (n.length < NAME_MIN || n.length > NAME_MAX) {
      error.value = `Character name must be ${NAME_MIN}-${NAME_MAX} characters.`;
      return;
    }
    if (b.length < BLURB_MIN || b.length > BLURB_MAX) {
      error.value = `Blurb must be ${BLURB_MIN}-${BLURB_MAX} characters.`;
      return;
    }
    loading.value = true;
    try {
      await api.submitCharacter(code, n, b);
      setSubmittedChar({ name: n, blurb: b });
      setSubmitted(true);
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      loading.value = false;
    }
  };

  // Stable preview accent — keyed off the player's id so it doesn't churn
  // on every keystroke. Falls back to a literal "preview" key pre-/me.
  const previewAccent = useMemo(
    () => accentFor(m?.player_id ?? "preview"),
    [m?.player_id],
  );

  if (!r) {
    return <div class="placeholder">Loading…</div>;
  }

  const totalPlayers = r.players.filter((p) => !p.left).length;
  const progress = charcreateProgress.value;
  const counter = progress ?? {
    submitted: submitted ? 1 : 0,
    total: totalPlayers,
  };

  const trimmedName = charName.trim();
  const trimmedBlurb = blurb.trim();
  const nameValid =
    trimmedName.length >= NAME_MIN && trimmedName.length <= NAME_MAX;
  const blurbValid =
    trimmedBlurb.length >= BLURB_MIN && trimmedBlurb.length <= BLURB_MAX;
  const canSubmit = nameValid && blurbValid && !loading.value;

  const previewName = trimmedName || "Your character";
  const previewBlurb =
    trimmedBlurb || "Their voice, their quirks, their vibe…";

  const counterLabel = `${counter.submitted} / ${counter.total}`;
  const counterPillKind = counter.submitted >= counter.total ? "done" : "live";

  return (
    <div class="fs charcreate" style={{ padding: "10px 22px 24px" }}>
      {/* Header: title + submitted/total chip */}
      <div
        class="fs-row fs-between"
        style={{ paddingTop: 4, alignItems: "center" }}
      >
        <span class="fs-display" style={{ fontSize: 26 }}>
          write a character
        </span>
        <StatusPill kind={counterPillKind} label={counterLabel} />
      </div>
      <div class="fs-lbl" style={{ marginTop: 4 }}>
        Everyone writes one. The others get assigned each other's, in secret.
        Make it distinctive — voice, quirks, vibe.
      </div>

      {!submitted ? (
        <form
          class="fs-col"
          onSubmit={onSubmit}
          style={{ gap: 12, marginTop: 16 }}
        >
          {/* Name field */}
          <div class="fs-col" style={{ gap: 4 }}>
            <label class="fs-tiny" for="cc-name">
              name
            </label>
            <div
              class="fs-card"
              style={{ padding: "12px 14px", position: "relative" }}
            >
              <input
                id="cc-name"
                type="text"
                maxLength={NAME_MAX}
                value={charName}
                placeholder="Sir Reginald the Unwell"
                onInput={(e) =>
                  setCharName((e.target as HTMLInputElement).value)
                }
                required
                style={{
                  width: "100%",
                  background: "transparent",
                  border: "none",
                  outline: "none",
                  color: "var(--fs-fg)",
                  fontFamily: "'Lilita One', 'Space Grotesk', sans-serif",
                  fontSize: 22,
                  letterSpacing: "0.02em",
                  padding: 0,
                }}
              />
            </div>
          </div>

          {/* Blurb field */}
          <div class="fs-col" style={{ gap: 4 }}>
            <label class="fs-tiny" for="cc-blurb">
              description
            </label>
            <div
              class="fs-card"
              style={{ padding: "12px 14px", position: "relative" }}
            >
              <textarea
                id="cc-blurb"
                rows={5}
                maxLength={BLURB_MAX}
                value={blurb}
                placeholder="A Victorian hypochondriac who blames everyone for his many bizarre ailments. Refuses to remove his cravat."
                onInput={(e) =>
                  setBlurb((e.target as HTMLTextAreaElement).value)
                }
                required
                style={{
                  width: "100%",
                  minHeight: 96,
                  background: "transparent",
                  border: "none",
                  outline: "none",
                  resize: "vertical",
                  color: "var(--fs-fg)",
                  fontFamily: "'Space Grotesk', sans-serif",
                  fontSize: 14,
                  lineHeight: 1.45,
                  padding: 0,
                }}
              />
              {/* Live char-count chip in the blurb corner */}
              <span
                class={
                  "fs-chip" +
                  (blurbValid
                    ? " fs-chip--on"
                    : trimmedBlurb.length > BLURB_MAX
                      ? " fs-chip--live"
                      : "")
                }
                style={{
                  position: "absolute",
                  right: 8,
                  bottom: 8,
                  fontSize: 10,
                  padding: "2px 8px",
                }}
              >
                {trimmedBlurb.length} / {BLURB_MAX}
              </span>
            </div>
            <div class="fs-row fs-between" style={{ marginTop: 2 }}>
              <span class="fs-tiny">
                min {BLURB_MIN} chars
                {!blurbValid && trimmedBlurb.length > 0 ? (
                  <span class="fs-faint">
                    {" "}
                    · {Math.max(0, BLURB_MIN - trimmedBlurb.length)} to go
                  </span>
                ) : null}
              </span>
              <span class="fs-tiny fs-faint">
                {trimmedName.length} / {NAME_MAX} name
              </span>
            </div>
          </div>

          {/* Live preview */}
          <div class="fs-row fs-between" style={{ marginTop: 10 }}>
            <span class="fs-tiny">
              ↓ live preview · this is what others will see
            </span>
          </div>
          <div style={{ marginTop: 4 }}>
            <CharCard
              id="preview"
              name={previewName}
              desc={previewBlurb}
              accent={previewAccent}
              tilt={-0.6}
            />
          </div>

          <button
            class={"fs-btn fs-btn--primary"}
            type="submit"
            disabled={!canSubmit}
            style={{ width: "100%", marginTop: 8 }}
          >
            SUBMIT TO POOL →
          </button>
        </form>
      ) : (
        <div class="fs-col" style={{ gap: 12, marginTop: 16 }}>
          <div
            class="fs-card"
            style={{
              padding: "16px 18px",
              textAlign: "center",
            }}
          >
            <div class="fs-tiny" style={{ color: "var(--fs-positive)" }}>
              ✓ LOCKED IN
            </div>
            <div
              class="fs-display"
              style={{ fontSize: 22, marginTop: 6, lineHeight: 1.1 }}
            >
              waiting on the others
            </div>
            <div class="fs-lbl" style={{ marginTop: 6 }}>
              {counter.submitted} / {counter.total} locked in
            </div>
          </div>

          {submittedChar ? (
            <div class="fs-col" style={{ gap: 6 }}>
              <span class="fs-tiny">your submission</span>
              <CharCard
                id="preview"
                name={submittedChar.name}
                desc={submittedChar.blurb}
                accent={previewAccent}
                tilt={-0.6}
              />
            </div>
          ) : null}
        </div>
      )}
    </div>
  );
}
