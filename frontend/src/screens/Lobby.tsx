import * as api from "../api";
import {
  error,
  isHost,
  leaveAndReset,
  loading,
  me,
  room,
} from "../store";
import {
  RoomCode,
  StatusPill,
  accentForPlayer,
} from "../components/atoms";
import type { PublicPlayer } from "../types";

interface Props {
  code: string;
}

const SLOTS = 8;
// PLAYTEST: lowered from 4 to 2 for early playtesting. RESTORE TO 4 BEFORE PUBLIC LAUNCH.
const MIN_PLAYERS = 2;
const RING_RADIUS = 110;

export function Lobby({ code }: Props) {
  const r = room.value;
  const m = me.value;

  if (!r) {
    return (
      <div class="fs fs-tac" style={{ padding: 24, color: "var(--fs-fg-mute)" }}>
        Loading lobby…
      </div>
    );
  }

  const activePlayers = r.players.filter((p) => !p.left);
  const canStart = activePlayers.length >= MIN_PLAYERS;
  const needed = Math.max(0, MIN_PLAYERS - activePlayers.length);

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
    <div
      class="fs fs-lobby"
      style={{
        display: "flex",
        flexDirection: "column",
        minHeight: "100vh",
        padding: "16px 20px 24px",
        gap: 16,
        boxSizing: "border-box",
      }}
    >
      {/* Top: room label + share hint */}
      <div class="fs-row fs-between" style={{ marginTop: 4 }}>
        <span class="fs-tiny">room</span>
        <StatusPill kind="live" label={r.mode === "live" ? "LIVE" : "ASYNC"} />
      </div>

      {/* Hero: room code */}
      <div class="fs-tac" style={{ marginTop: 4 }}>
        <RoomCode code={r.code} size={78} />
        <div class="fs-lbl" style={{ marginTop: 2 }}>
          4-letter code · share with friends
        </div>
      </div>

      {/* Player ring with centre counter */}
      <div
        class="fs-grow"
        style={{
          position: "relative",
          minHeight: RING_RADIUS * 2 + 80,
          marginTop: 8,
        }}
      >
        {/* dashed ring */}
        <div
          aria-hidden="true"
          style={{
            position: "absolute",
            left: "50%",
            top: "50%",
            transform: "translate(-50%, -50%)",
            width: RING_RADIUS * 2,
            height: RING_RADIUS * 2,
            border: "1.5px dashed var(--fs-line)",
            borderRadius: "50%",
          }}
        />

        {/* centre n / 8 counter */}
        <div
          style={{
            position: "absolute",
            left: "50%",
            top: "50%",
            transform: "translate(-50%, -50%)",
            textAlign: "center",
            width: 140,
          }}
        >
          <div class="fs-display" style={{ fontSize: 56 }}>
            {activePlayers.length}
            <span style={{ color: "var(--fs-fg-faint)" }}>/{SLOTS}</span>
          </div>
          <div class="fs-tiny" style={{ marginTop: 2 }}>
            in the room
          </div>
        </div>

        {/* slot avatars */}
        {Array.from({ length: SLOTS }).map((_, i) => {
          const angle = (i / SLOTS) * 2 * Math.PI - Math.PI / 2;
          const x = Math.cos(angle) * RING_RADIUS;
          const y = Math.sin(angle) * RING_RADIUS;
          const p: PublicPlayer | undefined = activePlayers[i];
          const slotStyle = {
            position: "absolute" as const,
            left: `calc(50% + ${x}px)`,
            top: `calc(50% + ${y}px)`,
            transform: "translate(-50%, -50%)",
          };
          if (!p) {
            return (
              <div key={`empty-${i}`} style={slotStyle}>
                <div class="fs-av fs-av--lg fs-av--empty">?</div>
              </div>
            );
          }
          const accent = accentForPlayer(p.id);
          const initial = (p.name?.trim()?.[0] ?? "?").toUpperCase();
          const isSelf = m && p.id === m.player_id;
          return (
            <div key={p.id} style={slotStyle}>
              <div
                class="fs-col fs-center"
                style={{ gap: 2, alignItems: "center" }}
              >
                <div
                  class="fs-av fs-av--lg fs-av--accent fs-anim-pop"
                  style={{
                    background: accent,
                    color: "#1a1300",
                    borderColor: "transparent",
                    position: "relative",
                  }}
                  title={p.name}
                >
                  {initial}
                  {p.is_host && (
                    <span
                      aria-label="host"
                      style={{
                        position: "absolute",
                        top: -6,
                        right: -6,
                        width: 20,
                        height: 20,
                        borderRadius: "50%",
                        background: "var(--fs-bg)",
                        border: "1px solid var(--fs-line)",
                        display: "inline-flex",
                        alignItems: "center",
                        justifyContent: "center",
                        fontSize: 12,
                        color: "var(--fs-accent)",
                        lineHeight: 1,
                      }}
                    >
                      ★
                    </span>
                  )}
                </div>
                <div
                  class="fs-tiny"
                  style={{
                    fontSize: 10,
                    marginTop: 2,
                    maxWidth: 80,
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                    color: isSelf ? "var(--fs-fg)" : "var(--fs-fg-mute)",
                  }}
                >
                  {p.name}
                  {isSelf ? " (you)" : ""}
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {/* Pool source + mode pills (informational) */}
      <div class="fs-col" style={{ gap: 8 }}>
        <div class="fs-row fs-between">
          <span class="fs-lbl">pool</span>
          <div class="fs-row" style={{ gap: 6 }}>
            <StatusPill
              kind={r.pool_source === "curated" ? "done" : "idle"}
              label="CURATED"
            />
            <StatusPill
              kind={r.pool_source === "playerwritten" ? "done" : "idle"}
              label="PLAYER-WRITTEN"
            />
          </div>
        </div>
        <div class="fs-row fs-between">
          <span class="fs-lbl">mode</span>
          <div class="fs-row" style={{ gap: 6 }}>
            <StatusPill
              kind={r.mode === "live" ? "done" : "idle"}
              label="LIVE"
            />
            <StatusPill
              kind={r.mode === "async" ? "done" : "idle"}
              label="ASYNC"
            />
          </div>
        </div>
      </div>

      {/* CTA */}
      <div class="fs-col" style={{ gap: 8, marginTop: 4 }}>
        {isHost.value ? (
          <>
            <button
              class={`fs-btn fs-btn--primary${canStart && !loading.value ? "" : " fs-btn--disabled"}`}
              disabled={!canStart || loading.value}
              onClick={onStart}
              style={{ width: "100%" }}
            >
              START THE SLOP →
            </button>
            <div class="fs-lbl fs-tac">
              {canStart
                ? "host only · tap to begin"
                : `host only · need ${needed} more player${needed === 1 ? "" : "s"}`}
            </div>
          </>
        ) : (
          <div
            class="fs-card fs-tac"
            style={{ padding: "14px 16px" }}
          >
            <div class="fs-display" style={{ fontSize: 20 }}>
              waiting for host
            </div>
            <div class="fs-lbl" style={{ marginTop: 4 }}>
              they'll start the slop when ready
            </div>
          </div>
        )}
      </div>

      {/* Footer: leave / abandon */}
      <div
        class="fs-row fs-center"
        style={{ gap: 12, marginTop: "auto", paddingTop: 12 }}
      >
        <button
          class="fs-btn fs-btn--ghost"
          onClick={() => leaveAndReset(code)}
          style={{ fontSize: 14, padding: "10px 14px" }}
        >
          leave room
        </button>
        {isHost.value && (
          <button
            class="fs-btn fs-btn--ghost"
            onClick={onAbandon}
            style={{
              fontSize: 14,
              padding: "10px 14px",
              color: "var(--fs-live)",
              borderColor: "var(--fs-line)",
            }}
          >
            abandon
          </button>
        )}
      </div>

      {error.value && (
        <div
          class="fs-card fs-tac"
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
    </div>
  );
}
