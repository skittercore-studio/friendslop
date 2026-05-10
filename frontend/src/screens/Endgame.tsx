import { useState } from "preact/hooks";
import * as api from "../api";
import { CharacterPool } from "../components/CharacterPool";
import { Scoreboard } from "../components/Scoreboard";
import {
  abandonedReason,
  error,
  isHost,
  loading,
  resetToLanding,
  room,
  trueAssignments,
} from "../store";

interface Props {
  code: string;
}

export function Endgame({ code }: Props) {
  const r = room.value;
  const [restarting, setRestarting] = useState(false);

  if (!r) return <div class="placeholder">Loading…</div>;

  const isAbandoned = r.state === "abandoned";
  const winnerId = r.winner_player_id;
  const winner = winnerId ? r.players.find((p) => p.id === winnerId) : null;

  // Build a player_id → character lookup. Prefer SSE-delivered true_assignments
  // (definitive). If we missed the event, fall back to nothing.
  const assigns = trueAssignments.value;
  const charById = new Map(r.characters?.map((c) => [c.id, c]) ?? []);
  const playerById = new Map(r.players.map((p) => [p.id, p]));

  // Host-only "Play again": the same room is reset to lobby with the same
  // players, code, and sessions. Non-hosts see a passive waiting message
  // — when the host hits restart, the SSE state.changed event drives
  // every client back into Lobby via routeFromState.
  const onPlayAgain = async () => {
    setRestarting(true);
    loading.value = true;
    try {
      await api.restartRoom(code);
      // No manual screen swap — the state.changed SSE event will trigger
      // refresh() and routeFromState() will land on Lobby for everyone.
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      setRestarting(false);
      loading.value = false;
    }
  };

  if (isAbandoned) {
    return (
      <div class="endgame">
        <div class="winner-banner abandoned">
          <h2>Room abandoned</h2>
          <p>Reason: {abandonedReason.value ?? "unknown"}</p>
        </div>
        <button class="primary" onClick={() => resetToLanding()}>
          Back to landing
        </button>
      </div>
    );
  }

  return (
    <div class="endgame">
      <div class="winner-banner">
        <h2>
          {winner ? (
            <>
              <span class="trophy">Winner</span> {winner.name}
            </>
          ) : (
            <>Game over</>
          )}
        </h2>
      </div>

      {assigns && assigns.length > 0 ? (
        <section class="reveal-table">
          <h3>True assignments</h3>
          <p class="hint">
            Authorship of player-written characters stays private — what you
            see below is who was playing whom.
          </p>
          <table>
            <thead>
              <tr>
                <th>Player</th>
                <th>Was playing</th>
              </tr>
            </thead>
            <tbody>
              {assigns.map((a) => {
                const p = playerById.get(a.player_id);
                const c = charById.get(a.character_id);
                return (
                  <tr key={a.player_id}>
                    <td>{p?.name ?? a.player_id}</td>
                    <td>{c?.name ?? a.character_id}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </section>
      ) : (
        <section class="reveal-table">
          <p class="hint">
            Final assignments weren&rsquo;t captured (you may have joined the
            screen after the reveal). Try refreshing.
          </p>
        </section>
      )}

      <CharacterPool characters={r.characters} />
      <Scoreboard />

      <div class="endgame-actions">
        {isHost.value ? (
          <button
            class="primary"
            disabled={restarting || loading.value}
            onClick={onPlayAgain}
          >
            {restarting ? "Restarting…" : "Play again"}
          </button>
        ) : (
          <div class="hint waiting-for-host">
            Waiting for the host to start a new game…
          </div>
        )}
        <button class="ghost" onClick={() => resetToLanding()}>
          Back to landing
        </button>
      </div>
    </div>
  );
}
