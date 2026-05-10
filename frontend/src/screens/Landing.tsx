import { useState } from "preact/hooks";
import * as api from "../api";
import { enterRoom, error, loading } from "../store";
import type { PoolSource } from "../types";

type Tab = "create" | "join";

const inputStyle: preact.JSX.CSSProperties = {
  background: "var(--fs-bg-1)",
  border: "1px solid var(--fs-line)",
  borderRadius: 12,
  padding: "12px 14px",
  color: "var(--fs-fg)",
  fontFamily: "inherit",
  fontSize: 16,
  width: "100%",
  boxSizing: "border-box",
  outline: "none",
  transition: "border-color 0.12s ease, box-shadow 0.12s ease",
};

const numberInputStyle: preact.JSX.CSSProperties = {
  ...inputStyle,
  textAlign: "center",
  width: 88,
  padding: "8px 10px",
  fontSize: 15,
};

export function Landing() {
  const [tab, setTab] = useState<Tab>("create");

  return (
    <div
      class="fs fs-landing"
      style={{
        display: "flex",
        flexDirection: "column",
        minHeight: "100vh",
        padding: "32px 20px 20px",
        gap: 24,
        boxSizing: "border-box",
      }}
    >
      {/* Hero */}
      <div class="fs-tac fs-anim-pop" style={{ marginTop: 8 }}>
        <h1
          class="fs-roomcode"
          style={{
            fontSize: 64,
            margin: 0,
            letterSpacing: "0.04em",
            lineHeight: 0.9,
          }}
        >
          friendslop
        </h1>
        <p
          class="fs-muted"
          style={{
            margin: "12px auto 0",
            maxWidth: 360,
            fontSize: 15,
            lineHeight: 1.4,
          }}
        >
          a text-based party game. write in character. guess who wrote which.
        </p>
      </div>

      {/* Tab toggle */}
      <div class="fs-row fs-center" style={{ gap: 8 }}>
        <button
          class={`fs-chip${tab === "create" ? " fs-chip--on" : ""}`}
          style={{
            padding: "10px 18px",
            fontSize: 13,
            cursor: "pointer",
            border: tab === "create" ? "1px solid transparent" : undefined,
          }}
          onClick={() => setTab("create")}
        >
          CREATE ROOM
        </button>
        <button
          class={`fs-chip${tab === "join" ? " fs-chip--on" : ""}`}
          style={{
            padding: "10px 18px",
            fontSize: 13,
            cursor: "pointer",
            border: tab === "join" ? "1px solid transparent" : undefined,
          }}
          onClick={() => setTab("join")}
        >
          JOIN WITH CODE
        </button>
      </div>

      {tab === "create" ? <CreateForm /> : <JoinForm />}

      {error.value && (
        <div
          class="fs-card fs-tac fs-anim-pop"
          style={{
            padding: "10px 14px",
            color: "var(--fs-live)",
            borderColor: "var(--fs-live)",
            fontSize: 13,
          }}
        >
          {error.value}
        </div>
      )}

      {/* Footer */}
      <div
        class="fs-tac fs-tiny"
        style={{ marginTop: "auto", paddingTop: 24, color: "var(--fs-fg-faint)" }}
      >
        by{" "}
        <a
          href="https://skittercore.studio"
          target="_blank"
          rel="noopener"
          style={{ color: "var(--fs-fg-mute)", textDecoration: "none" }}
        >
          skittercore studio
        </a>
      </div>
    </div>
  );
}

function CreateForm() {
  const [name, setName] = useState("");
  const [poolSource, setPoolSource] = useState<PoolSource>("curated");
  const [answerSecs, setAnswerSecs] = useState(120);
  const [guessSecs, setGuessSecs] = useState(120);
  const [charcreateSecs, setCharcreateSecs] = useState(300);
  const [timersOpen, setTimersOpen] = useState(false);

  const submit = async (e: Event) => {
    e.preventDefault();
    if (!name.trim()) {
      error.value = "Pick a display name first.";
      return;
    }
    loading.value = true;
    error.value = null;
    try {
      const isPlayerWritten = poolSource === "playerwritten";
      const res = await api.createRoom({
        host_name: name.trim(),
        pool_source: poolSource,
        answer_timeout_seconds: answerSecs,
        guess_timeout_seconds: guessSecs,
        charcreate_timeout_seconds: isPlayerWritten ? charcreateSecs : null,
      });
      await enterRoom(res.room_code);
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      loading.value = false;
    }
  };

  return (
    <form
      class="fs-col fs-anim-slide"
      style={{ gap: 16 }}
      onSubmit={submit}
    >
      {/* Name */}
      <div class="fs-col" style={{ gap: 6 }}>
        <label class="fs-lbl" for="landing-host-name">
          your name
        </label>
        <input
          id="landing-host-name"
          type="text"
          maxLength={40}
          value={name}
          placeholder="how friends know you"
          onInput={(e) => setName((e.target as HTMLInputElement).value)}
          style={inputStyle}
          required
        />
      </div>

      {/* Pool source */}
      <div class="fs-col" style={{ gap: 6 }}>
        <label class="fs-lbl">character pool</label>
        <div class="fs-row" style={{ gap: 8 }}>
          <ToggleChip
            on={poolSource === "curated"}
            label="CURATED"
            sub="built-in pool"
            onClick={() => setPoolSource("curated")}
          />
          <ToggleChip
            on={poolSource === "playerwritten"}
            label="PLAYER-WRITTEN"
            sub="bring your own"
            onClick={() => setPoolSource("playerwritten")}
          />
        </div>
      </div>

      {/* Timers (collapsible) */}
      <div class="fs-col" style={{ gap: 8 }}>
        <button
          type="button"
          class="fs-lbl"
          onClick={() => setTimersOpen((v) => !v)}
          style={{
            background: "transparent",
            border: "none",
            cursor: "pointer",
            padding: 0,
            textAlign: "left",
            color: "var(--fs-fg-mute)",
          }}
        >
          {timersOpen ? "▾" : "▸"} customise timers
        </button>
        {timersOpen && (
          <div
            class="fs-card fs-col fs-anim-slide"
            style={{ padding: 14, gap: 10 }}
          >
            <TimerField
              label="answer (s)"
              value={answerSecs}
              onChange={setAnswerSecs}
            />
            <TimerField
              label="guess (s)"
              value={guessSecs}
              onChange={setGuessSecs}
            />
            {poolSource === "playerwritten" && (
              <TimerField
                label="charcreate (s)"
                value={charcreateSecs}
                onChange={setCharcreateSecs}
              />
            )}
          </div>
        )}
      </div>

      {/* CTA */}
      <button
        class={`fs-btn fs-btn--primary${loading.value ? " fs-btn--disabled" : ""}`}
        type="submit"
        disabled={loading.value}
        style={{ width: "100%", marginTop: 4 }}
      >
        {loading.value ? "creating…" : "CREATE ROOM →"}
      </button>
    </form>
  );
}

function JoinForm() {
  const [code, setCode] = useState("");
  const [name, setName] = useState("");

  const submit = async (e: Event) => {
    e.preventDefault();
    const trimmedCode = code.trim().toUpperCase();
    const trimmedName = name.trim();
    if (!trimmedCode || !trimmedName) {
      error.value = "need a code and a name.";
      return;
    }
    loading.value = true;
    error.value = null;
    try {
      await api.joinRoom(trimmedCode, trimmedName);
      await enterRoom(trimmedCode);
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      loading.value = false;
    }
  };

  return (
    <form
      class="fs-col fs-anim-slide"
      style={{ gap: 18 }}
      onSubmit={submit}
    >
      {/* Hero room code input */}
      <div class="fs-col fs-tac" style={{ gap: 8 }}>
        <label class="fs-lbl" for="landing-join-code">
          room code
        </label>
        <input
          id="landing-join-code"
          type="text"
          maxLength={8}
          value={code}
          placeholder="ABCD"
          onInput={(e) =>
            setCode((e.target as HTMLInputElement).value.toUpperCase())
          }
          style={{
            ...inputStyle,
            fontFamily:
              '"JetBrains Mono", ui-monospace, SFMono-Regular, monospace',
            fontSize: 36,
            letterSpacing: "0.32em",
            textAlign: "center",
            textTransform: "uppercase",
            padding: "20px 16px",
          }}
          required
        />
      </div>

      <div class="fs-col" style={{ gap: 6 }}>
        <label class="fs-lbl" for="landing-join-name">
          your name
        </label>
        <input
          id="landing-join-name"
          type="text"
          maxLength={40}
          value={name}
          placeholder="how friends know you"
          onInput={(e) => setName((e.target as HTMLInputElement).value)}
          style={inputStyle}
          required
        />
      </div>

      <button
        class={`fs-btn fs-btn--primary${loading.value ? " fs-btn--disabled" : ""}`}
        type="submit"
        disabled={loading.value}
        style={{ width: "100%", marginTop: 4 }}
      >
        {loading.value ? "joining…" : "JOIN ROOM →"}
      </button>
    </form>
  );
}

function ToggleChip({
  on,
  label,
  sub,
  onClick,
}: {
  on: boolean;
  label: string;
  sub: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      class={`fs-chip${on ? " fs-chip--on" : ""}`}
      style={{
        flex: 1,
        flexDirection: "column",
        alignItems: "stretch",
        textAlign: "center",
        padding: "10px 8px",
        gap: 2,
        border: on ? "1px solid transparent" : undefined,
        cursor: "pointer",
      }}
    >
      <span style={{ fontWeight: 600, fontSize: 12 }}>{label}</span>
      <span
        style={{
          fontSize: 10,
          fontWeight: 400,
          textTransform: "none",
          letterSpacing: 0,
          opacity: 0.85,
        }}
      >
        {sub}
      </span>
    </button>
  );
}

function TimerField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: number;
  onChange: (n: number) => void;
}) {
  return (
    <label
      class="fs-row fs-between"
      style={{ alignItems: "center", gap: 12 }}
    >
      <span class="fs-lbl" style={{ flex: 1 }}>
        {label}
      </span>
      <input
        type="number"
        min={15}
        max={3600}
        value={value}
        onInput={(e) =>
          onChange(Number((e.target as HTMLInputElement).value) || 0)
        }
        style={numberInputStyle}
      />
    </label>
  );
}
