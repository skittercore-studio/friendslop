// Typed API client + SSE subscription helper.
// All endpoints from spec §4. Cookies (slop_session) are managed by the browser
// because the backend sets them with HttpOnly + SameSite=Lax — we just use
// `credentials: "include"` to make sure they get sent.

import type {
  CreateRoomRequest,
  CreateRoomResponse,
  JoinRoomRequest,
  JoinRoomResponse,
  PrivateMeView,
  PublicRoomSnapshot,
  SSEEventMap,
  SSEEventName,
} from "./types";

const API_BASE = "/api/v1";

class APIError extends Error {
  status: number;
  body: unknown;
  constructor(status: number, message: string, body: unknown) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

async function request<T>(
  path: string,
  init: RequestInit & { json?: unknown } = {},
): Promise<T> {
  const headers = new Headers(init.headers ?? {});
  let body: BodyInit | undefined = init.body ?? undefined;
  if (init.json !== undefined) {
    headers.set("Content-Type", "application/json");
    body = JSON.stringify(init.json);
  }
  const res = await fetch(API_BASE + path, {
    ...init,
    headers,
    body,
    credentials: "include",
  });
  if (!res.ok) {
    let parsed: unknown = null;
    try {
      parsed = await res.json();
    } catch {
      // ignore — body might be empty / non-JSON
    }
    const msg =
      (parsed && typeof parsed === "object" && "error" in parsed
        ? String((parsed as { error: unknown }).error)
        : null) ?? `HTTP ${res.status} ${res.statusText}`;
    throw new APIError(res.status, msg, parsed);
  }
  if (res.status === 204) return undefined as T;
  // Some endpoints return JSON, others 204; treat empty bodies as undefined.
  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

// ------- Public endpoints -------

export function createRoom(opts: CreateRoomRequest): Promise<CreateRoomResponse> {
  return request<CreateRoomResponse>("/rooms", { method: "POST", json: opts });
}

export function joinRoom(code: string, name: string): Promise<JoinRoomResponse> {
  const body: JoinRoomRequest = { name };
  return request<JoinRoomResponse>(`/rooms/${encodeURIComponent(code)}/join`, {
    method: "POST",
    json: body,
  });
}

export function getRoom(code: string): Promise<PublicRoomSnapshot> {
  return request<PublicRoomSnapshot>(`/rooms/${encodeURIComponent(code)}`);
}

// ------- Authenticated (cookie required) -------

export function getMe(code: string): Promise<PrivateMeView> {
  return request<PrivateMeView>(`/rooms/${encodeURIComponent(code)}/me`);
}

export function leaveRoom(code: string): Promise<void> {
  return request<void>(`/rooms/${encodeURIComponent(code)}/leave`, {
    method: "POST",
  });
}

export function startGame(code: string): Promise<void> {
  return request<void>(`/rooms/${encodeURIComponent(code)}/start`, {
    method: "POST",
  });
}

export function abandonRoom(code: string): Promise<void> {
  return request<void>(`/rooms/${encodeURIComponent(code)}/abandon`, {
    method: "POST",
  });
}

export function submitCharacter(
  code: string,
  name: string,
  blurb: string,
): Promise<void> {
  return request<void>(`/rooms/${encodeURIComponent(code)}/character`, {
    method: "POST",
    json: { name, blurb },
  });
}

export function submitAnswer(
  code: string,
  roundNumber: number,
  text: string,
): Promise<void> {
  return request<void>(`/rooms/${encodeURIComponent(code)}/answer`, {
    method: "POST",
    json: { round_number: roundNumber, text },
  });
}

export function submitGuess(
  code: string,
  roundNumber: number,
  mapping: Record<string, string>,
): Promise<void> {
  return request<void>(`/rooms/${encodeURIComponent(code)}/guess`, {
    method: "POST",
    json: { round_number: roundNumber, mapping },
  });
}

// ------- SSE -------

export type SSEHandler = <K extends SSEEventName>(
  name: K,
  data: SSEEventMap[K],
) => void;

export interface SSESubscription {
  close: () => void;
}

const SSE_EVENT_NAMES: SSEEventName[] = [
  "state.changed",
  "player.joined",
  "player.left",
  "charcreate.started",
  "charcreate.submitted",
  "charcreate.completed",
  "round.started",
  "answer.submitted",
  "round.answers_revealed",
  "guess.submitted",
  "round.scored",
  "game.won",
  "game.abandoned",
  "heartbeat",
];

/**
 * Subscribe to the room's SSE stream. Reconnects with exponential backoff
 * on error (1s → 30s cap). Returns a handle with `close()`.
 */
export function subscribeEvents(
  code: string,
  onEvent: SSEHandler,
  onStatusChange?: (status: "open" | "reconnecting" | "closed") => void,
): SSESubscription {
  let closed = false;
  let es: EventSource | null = null;
  let backoff = 1000;
  let timer: ReturnType<typeof setTimeout> | null = null;

  const connect = () => {
    if (closed) return;
    const url = `${API_BASE}/rooms/${encodeURIComponent(code)}/events`;
    es = new EventSource(url, { withCredentials: true });

    es.onopen = () => {
      backoff = 1000;
      onStatusChange?.("open");
    };

    es.onerror = () => {
      // EventSource will auto-reconnect on its own, but it doesn't honour
      // backoff. Force-close and schedule.
      if (es) {
        es.close();
        es = null;
      }
      if (closed) return;
      onStatusChange?.("reconnecting");
      timer = setTimeout(connect, backoff);
      backoff = Math.min(backoff * 2, 30_000);
    };

    for (const name of SSE_EVENT_NAMES) {
      es.addEventListener(name, (ev: MessageEvent) => {
        let data: unknown = null;
        try {
          data = ev.data ? JSON.parse(ev.data) : null;
        } catch {
          data = ev.data;
        }
        // The cast is safe-ish: server is the source of truth for shape.
        onEvent(name, data as SSEEventMap[typeof name]);
      });
    }
  };

  connect();

  return {
    close() {
      closed = true;
      onStatusChange?.("closed");
      if (timer) {
        clearTimeout(timer);
        timer = null;
      }
      if (es) {
        es.close();
        es = null;
      }
    },
  };
}

export { APIError };
