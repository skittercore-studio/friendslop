import { me, room } from "../store";

/**
 * Round-over-round leaderboard. Counts only — never the contents of others'
 * guesses (spec §1: "Per Toy's rule").
 */
export function Scoreboard() {
  const r = room.value;
  const m = me.value;

  if (!r) return null;

  const rounds = r.scoreboard?.rounds ?? [];
  const players = r.players.filter((p) => !p.left);
  const otherCount = players.length - 1;

  // Totals per player.
  const totals: Record<string, number> = {};
  for (const p of players) totals[p.id] = 0;
  for (const round of rounds) {
    for (const [pid, count] of Object.entries(round.scores)) {
      totals[pid] = (totals[pid] ?? 0) + count;
    }
  }

  // Sort players by total desc, stable on name.
  const sorted = [...players].sort(
    (a, b) =>
      (totals[b.id] ?? 0) - (totals[a.id] ?? 0) || a.name.localeCompare(b.name),
  );

  return (
    <div class="scoreboard">
      <h3>Scoreboard</h3>
      {rounds.length === 0 ? (
        <p class="hint">No rounds scored yet.</p>
      ) : (
        <div class="scoreboard-scroll">
          <table>
            <thead>
              <tr>
                <th class="sb-name">Player</th>
                {rounds.map((round) => (
                  <th key={`r${round.round_number}`}>R{round.round_number}</th>
                ))}
                <th class="sb-total">Total</th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((p) => (
                <tr
                  key={p.id}
                  class={m && p.id === m.player_id ? "sb-self" : ""}
                >
                  <td class="sb-name">
                    {p.name}
                    {p.is_host && <span class="badge host">host</span>}
                  </td>
                  {rounds.map((round) => (
                    <td key={`r${round.round_number}`}>
                      {round.scores[p.id] ?? 0}
                      <span class="sb-denom">/{otherCount}</span>
                    </td>
                  ))}
                  <td class="sb-total">{totals[p.id] ?? 0}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
