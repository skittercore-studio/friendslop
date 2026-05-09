// Tiny global store using @preact/signals.
// Mirrors the public room snapshot + private /me view. On every relevant SSE
// event we refetch both — simpler than reasoning about partial mutations.

import { signal, computed } from "@preact/signals";
import type {
  PrivateMeView,
  PublicRoomSnapshot,
  SSECharcreateSubmitted,
  SSEGameAbandoned,
  SSEGameWon,
  SSETrueAssignment,
} from "./types";
import * as api from "./api";

export type Screen =
  | { kind: "landing" }
  | { kind: "lobby"; code: string }
  | { kind: "charcreate"; code: string }
  | { kind: "game"; code: string }
  | { kind: "endgame"; code: string };

export const screen = signal<Screen>({ kind: "landing" });

export const room = signal<PublicRoomSnapshot | null>(null);
export const me = signal<PrivateMeView | null>(null);
export const sseStatus = signal<"open" | "reconnecting" | "closed" | "idle">(
  "idle",
);

export const error = signal<string | null>(null);
export const loading = signal<boolean>(false);

// Last revealed true assignments (only set after game.won)
export const trueAssignments = signal<SSETrueAssignment[] | null>(null);
export const abandonedReason = signal<string | null>(null);

// Live charcreate submitted counter (driven by SSE events directly).
export const charcreateProgress = signal<{
  submitted: number;
  total: number;
} | null>(null);

// Per-room session storage so a refresh keeps you logged in (paired with the
// HttpOnly cookie the backend already sets — this is just for re-hydrating
// the active room code on the client).
const STORAGE_KEY = "friendslop:active";

interface PersistedSession {
  code: string;
}

export function persistSession(code: string) {
  try {
    sessionStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ code } satisfies PersistedSession),
    );
  } catch {
    // ignore — private mode etc.
  }
}

export function clearSession() {
  try {
    sessionStorage.removeItem(STORAGE_KEY);
  } catch {
    // ignore
  }
}

export function readSession(): PersistedSession | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as PersistedSession;
  } catch {
    return null;
  }
}

let subscription: { close: () => void } | null = null;

/** Refetch both public room and private /me. */
export async function refresh(code: string): Promise<void> {
  try {
    const [r, m] = await Promise.all([
      api.getRoom(code),
      api.getMe(code).catch(() => null),
    ]);
    room.value = r;
    me.value = m;
    routeFromState();
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

export function routeFromState() {
  const r = room.value;
  if (!r) return;
  const code = r.code;
  switch (r.state) {
    case "lobby":
      screen.value = { kind: "lobby", code };
      break;
    case "charcreate":
      screen.value = { kind: "charcreate", code };
      break;
    case "answering":
    case "guessing":
    case "scoring":
      screen.value = { kind: "game", code };
      break;
    case "won":
    case "abandoned":
      screen.value = { kind: "endgame", code };
      break;
  }
}

export async function enterRoom(code: string): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    persistSession(code);
    await refresh(code);
    if (subscription) subscription.close();
    subscription = api.subscribeEvents(
      code,
      (name, data) => {
        // Spec §5: refetch on relevant events. Abandon/win deserve a special
        // case to capture payload-only data first.
        if (name === "game.won") {
          const w = data as SSEGameWon;
          trueAssignments.value = w.true_assignments;
        }
        if (name === "game.abandoned") {
          const a = data as SSEGameAbandoned;
          abandonedReason.value = a.reason;
        }
        if (name === "charcreate.submitted") {
          const c = data as SSECharcreateSubmitted;
          charcreateProgress.value = {
            submitted: c.submitted_count,
            total: c.total_players,
          };
        }
        if (name === "heartbeat") return;
        // Everything else: refetch.
        void refresh(code);
      },
      (status) => {
        sseStatus.value = status;
      },
    );
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

export async function leaveAndReset(code: string): Promise<void> {
  try {
    await api.leaveRoom(code);
  } catch {
    // ignore — we're leaving anyway
  }
  resetToLanding();
}

export function resetToLanding() {
  if (subscription) {
    subscription.close();
    subscription = null;
  }
  clearSession();
  room.value = null;
  me.value = null;
  trueAssignments.value = null;
  abandonedReason.value = null;
  sseStatus.value = "idle";
  screen.value = { kind: "landing" };
}

// Convenience selectors

export const otherPlayers = computed(() => {
  const r = room.value;
  const m = me.value;
  if (!r || !m) return [];
  return r.players.filter((p) => p.id !== m.player_id && !p.left);
});

export const isHost = computed(() => me.value?.is_host ?? false);
