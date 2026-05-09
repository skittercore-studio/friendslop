import { useState } from "preact/hooks";
import * as api from "../api";
import { CharacterPool } from "../components/CharacterPool";
import { Scoreboard } from "../components/Scoreboard";
import {
  abandonedReason,
  enterRoom,
  error,
  loading,
  me,
  resetToLanding,
  room,
  trueAssignments,
} from "../store";

interface Props {
  code: string;
}

export function Endgame({ code }: Props) {
  const r = room.value;
  const m = me.value;
  const [creating, setCreating] = useState(false);

  if (!r) return <div class="placeholder">Loading…</div>;

  const isAbandoned = r.state === "abandoned";
  const winnerId = r.winner_player_id;
  const winner = winnerId ? r.players.find((p) => p.id === winnerId) : null;

  // Build a player_id → character lookup. Prefer SSE-delivered true_assignments
  // (definitive). If we missed the event, fall back to nothing.
  const assigns = trueAssignments.value;
  const charById = new Map(r.characters?.map((c) => [c.id, c]) ?? []);
  const playerById = new Map(r.players.map((p) => [p.id, p]));

  const onPlayAgain = async () => {
    if (!m) {
      resetToLanding();
      return;
    }
    setCreating(true);
    loading.value = true;
    try {
      const res = await api.createRoom({
        host_name: m.name,
        mode: r.mode,
        pool_source: r.pool_source,
        // Carry over a sensible default; live timers from the previous room
        // aren't exposed in the public snapshot, so we use 120/120/300.
        answer_timeout_seconds: r.mode === "live" ? 120 : null,
        guess_timeout_seconds: r.mode === "live" ? 120 : null,
        charcreate_timeout_seconds:
          r.mode === "live" && r.pool_source === "playerwritten" ? 300 : null,
      });
      await enterRoom(res.room_code);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      setCreating(false);
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
        <button
          class="primary"
          disabled={creating || loading.value}
          onClick={onPlayAgain}
        >
          {creating ? "Creating…" : "Play again"}
        </button>
        <button class="ghost" onClick={() => resetToLanding()}>
          Back to landing
        </button>
      </div>
    </div>
  );
}
