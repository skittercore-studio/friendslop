import { useEffect } from "preact/hooks";
import {
  enterRoom,
  error,
  readSession,
  resetToLanding,
  screen,
  sseStatus,
} from "./store";
import { Landing } from "./screens/Landing";
import { Lobby } from "./screens/Lobby";
import { CharCreate } from "./screens/CharCreate";
import { Game } from "./screens/Game";
import { Endgame } from "./screens/Endgame";

export function App() {
  // On first mount, see if we left a code in sessionStorage.
  useEffect(() => {
    const persisted = readSession();
    if (persisted?.code) {
      void enterRoom(persisted.code);
    }
  }, []);

  const s = screen.value;

  return (
    <div class="app-shell">
      <header class="app-header">
        <span class="brand" onClick={() => resetToLanding()}>
          friendslop
        </span>
        <span class="sse-pill" data-status={sseStatus.value}>
          {sseStatus.value}
        </span>
      </header>

      {error.value && (
        <div class="error-banner" onClick={() => (error.value = null)}>
          {error.value} <span class="dismiss">(dismiss)</span>
        </div>
      )}

      <main class="app-main">
        {s.kind === "landing" && <Landing />}
        {s.kind === "lobby" && <Lobby code={s.code} />}
        {s.kind === "charcreate" && <CharCreate code={s.code} />}
        {s.kind === "game" && <Game code={s.code} />}
        {s.kind === "endgame" && <Endgame code={s.code} />}
      </main>
    </div>
  );
}
