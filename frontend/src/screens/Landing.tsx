import { useState } from "preact/hooks";
import * as api from "../api";
import { enterRoom, error, loading } from "../store";
import type { PoolSource, RoomMode } from "../types";

type Tab = "create" | "join";

export function Landing() {
  const [tab, setTab] = useState<Tab>("create");
  return (
    <div class="landing">
      <h1 class="landing-title">friendslop</h1>
      <p class="landing-tag">
        a text-based party game. write in character. guess who wrote which.
      </p>
      <div class="tabs">
        <button
          class={tab === "create" ? "tab active" : "tab"}
          onClick={() => setTab("create")}
        >
          Create room
        </button>
        <button
          class={tab === "join" ? "tab active" : "tab"}
          onClick={() => setTab("join")}
        >
          Join with code
        </button>
      </div>

      {tab === "create" ? <CreateForm /> : <JoinForm />}
    </div>
  );
}

function CreateForm() {
  const [name, setName] = useState("");
  const [mode, setMode] = useState<RoomMode>("live");
  const [poolSource, setPoolSource] = useState<PoolSource>("curated");
  const [answerSecs, setAnswerSecs] = useState(120);
  const [guessSecs, setGuessSecs] = useState(120);
  const [charcreateSecs, setCharcreateSecs] = useState(300);

  const submit = async (e: Event) => {
    e.preventDefault();
    if (!name.trim()) {
      error.value = "Pick a display name first.";
      return;
    }
    loading.value = true;
    try {
      const isLive = mode === "live";
      const isPlayerWritten = poolSource === "playerwritten";
      const res = await api.createRoom({
        host_name: name.trim(),
        mode,
        pool_source: poolSource,
        answer_timeout_seconds: isLive ? answerSecs : null,
        guess_timeout_seconds: isLive ? guessSecs : null,
        charcreate_timeout_seconds:
          isLive && isPlayerWritten ? charcreateSecs : null,
      });
      await enterRoom(res.room_code);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  };

  return (
    <form class="form" onSubmit={submit}>
      <label class="field">
        <span>Your name</span>
        <input
          type="text"
          maxLength={40}
          value={name}
          onInput={(e) => setName((e.target as HTMLInputElement).value)}
          required
        />
      </label>

      <label class="field">
        <span>Mode</span>
        <select
          value={mode}
          onChange={(e) =>
            setMode((e.target as HTMLSelectElement).value as RoomMode)
          }
        >
          <option value="live">Live (timed rounds)</option>
          <option value="async">Async (24h windows)</option>
        </select>
      </label>

      <label class="field">
        <span>Character pool</span>
        <select
          value={poolSource}
          onChange={(e) =>
            setPoolSource(
              (e.target as HTMLSelectElement).value as PoolSource,
            )
          }
        >
          <option value="curated">Curated (built-in pool)</option>
          <option value="playerwritten">Player-written</option>
        </select>
      </label>

      {mode === "live" && (
        <fieldset class="timers">
          <legend>Timers (seconds)</legend>
          <label class="field inline">
            <span>Answer</span>
            <input
              type="number"
              min={15}
              max={3600}
              value={answerSecs}
              onInput={(e) =>
                setAnswerSecs(
                  Number((e.target as HTMLInputElement).value) || 0,
                )
              }
            />
          </label>
          <label class="field inline">
            <span>Guess</span>
            <input
              type="number"
              min={15}
              max={3600}
              value={guessSecs}
              onInput={(e) =>
                setGuessSecs(Number((e.target as HTMLInputElement).value) || 0)
              }
            />
          </label>
          {poolSource === "playerwritten" && (
            <label class="field inline">
              <span>Char create</span>
              <input
                type="number"
                min={30}
                max={3600}
                value={charcreateSecs}
                onInput={(e) =>
                  setCharcreateSecs(
                    Number((e.target as HTMLInputElement).value) || 0,
                  )
                }
              />
            </label>
          )}
        </fieldset>
      )}

      <button class="primary" type="submit" disabled={loading.value}>
        {loading.value ? "Creating…" : "Create room"}
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
      error.value = "Need a code and a name.";
      return;
    }
    loading.value = true;
    try {
      await api.joinRoom(trimmedCode, trimmedName);
      await enterRoom(trimmedCode);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  };

  return (
    <form class="form" onSubmit={submit}>
      <label class="field">
        <span>Room code</span>
        <input
          type="text"
          class="code-input"
          maxLength={8}
          value={code}
          onInput={(e) =>
            setCode((e.target as HTMLInputElement).value.toUpperCase())
          }
          required
        />
      </label>
      <label class="field">
        <span>Your name</span>
        <input
          type="text"
          maxLength={40}
          value={name}
          onInput={(e) => setName((e.target as HTMLInputElement).value)}
          required
        />
      </label>
      <button class="primary" type="submit" disabled={loading.value}>
        {loading.value ? "Joining…" : "Join room"}
      </button>
    </form>
  );
}
