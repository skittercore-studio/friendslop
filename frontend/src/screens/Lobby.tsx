import * as api from "../api";
import {
  error,
  isHost,
  leaveAndReset,
  loading,
  me,
  room,
} from "../store";

interface Props {
  code: string;
}

export function Lobby({ code }: Props) {
  const r = room.value;
  const m = me.value;

  if (!r) {
    return <div class="placeholder">Loading lobby…</div>;
  }

  const activePlayers = r.players.filter((p) => !p.left);
  const canStart = activePlayers.length >= 4;

  const onStart = async () => {
    loading.value = true;
    try {
      await api.startGame(code);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  };

  const onAbandon = async () => {
    if (!confirm("Abandon room? This ends the game for everyone.")) return;
    loading.value = true;
    try {
      await api.abandonRoom(code);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  };

  return (
    <div class="lobby">
      <section class="room-code-card">
        <div class="room-code-label">Share this code</div>
        <div class="room-code">{r.code}</div>
        <div class="room-pool-label">
          Pool: <strong>{r.pool_source}</strong> · Mode:{" "}
          <strong>{r.mode}</strong>
        </div>
      </section>

      <section class="players-list">
        <h2>Players ({activePlayers.length})</h2>
        <ul>
          {activePlayers.map((p) => (
            <li key={p.id}>
              <span class="player-name">{p.name}</span>
              {p.is_host && <span class="badge host">host</span>}
              {m && p.id === m.player_id && (
                <span class="badge self">you</span>
              )}
            </li>
          ))}
        </ul>
        {activePlayers.length < 4 && (
          <p class="hint">
            Need at least 4 players to start ({4 - activePlayers.length} more
            needed).
          </p>
        )}
      </section>

      <section class="lobby-actions">
        {isHost.value && (
          <button
            class="primary"
            disabled={!canStart || loading.value}
            onClick={onStart}
          >
            Start game
          </button>
        )}
        {isHost.value && (
          <button class="danger ghost" onClick={onAbandon}>
            Abandon room
          </button>
        )}
        <button class="ghost" onClick={() => leaveAndReset(code)}>
          Leave
        </button>
      </section>
    </div>
  );
}
