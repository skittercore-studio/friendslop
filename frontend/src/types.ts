// Type definitions matching the spec at /docs/SPEC.md §4-5.
// Wire shapes only — keep these in lock-step with the Go backend.

export type RoomState =
  | "lobby"
  | "charcreate"
  | "answering"
  | "guessing"
  | "scoring"
  | "won"
  | "abandoned";

// Async mode was dropped pre-launch. The field stays on the wire for
// backwards-compat but only ever carries "live".
export type RoomMode = "live";
export type PoolSource = "curated" | "playerwritten";

export interface PublicPlayer {
  id: string;
  name: string;
  is_host: boolean;
  left: boolean;
}

export interface PublicCharacter {
  id: string;
  name: string;
  blurb: string;
}

export interface RevealedAnswer {
  character_id: string;
  text: string;
}

export interface CurrentRound {
  number: number;
  state: "answering" | "guessing" | "scoring" | "done";
  question_text: string;
  answer_deadline: number | null;
  guess_deadline: number | null;
  answers?: RevealedAnswer[];
}

export interface ScoreboardRound {
  round_number: number;
  scores: Record<string, number>; // player_id → correct_count
}

export interface Scoreboard {
  rounds: ScoreboardRound[];
}

// GET /api/v1/rooms/:code  → public snapshot
export interface PublicRoomSnapshot {
  code: string;
  state: RoomState;
  mode: RoomMode;
  pool_source: PoolSource;
  round_number: number;
  players: PublicPlayer[];
  characters?: PublicCharacter[];
  current_round: CurrentRound | null;
  scoreboard: Scoreboard;
  winner_player_id: string | null;
}

export interface PastGuess {
  round_number: number;
  mapping: Record<string, string>; // target_player_id → character_id
  correct_count: number;
}

// GET /api/v1/rooms/:code/me  → private view
export interface PrivateMeView {
  player_id: string;
  name: string;
  is_host: boolean;
  your_character: PublicCharacter | null;
  your_authored_character_id: string | null;
  your_answer_for_current_round: string | null;
  your_guess_for_current_round: Record<string, string> | null;
  your_past_guesses: PastGuess[];
}

// POST /api/v1/rooms — create payload
export interface CreateRoomRequest {
  host_name: string;
  pool_source: PoolSource;
  answer_timeout_seconds?: number | null;
  guess_timeout_seconds?: number | null;
  charcreate_timeout_seconds?: number | null;
}

export interface CreateRoomResponse {
  room_code: string;
  session_token: string;
}

// POST /api/v1/rooms/:code/join
export interface JoinRoomRequest {
  name: string;
}

export interface JoinRoomResponse {
  room_code: string;
  player_id: string;
  session_token: string;
}

// SSE event payloads (§5)
export interface SSEStateChanged {
  state: RoomState;
  round_number: number;
}

export interface SSEPlayerJoined {
  player_id: string;
  name: string;
}

export interface SSEPlayerLeft {
  player_id: string;
}

export interface SSECharcreateStarted {
  deadline: number | null;
}

export interface SSECharcreateSubmitted {
  submitted_count: number;
  total_players: number;
}

export interface SSECharcreateCompleted {
  characters: PublicCharacter[];
}

export interface SSERoundStarted {
  round_number: number;
  question_text: string;
  answer_deadline: number | null;
}

export interface SSEAnswerSubmitted {
  player_id: string;
}

export interface SSERoundAnswersRevealed {
  round_number: number;
  answers: RevealedAnswer[];
  guess_deadline: number | null;
}

export interface SSEGuessSubmitted {
  player_id: string;
}

export interface SSERoundScored {
  round_number: number;
  public_scores: Record<string, number>;
  next_round_at: number | null;
}

export interface SSETrueAssignment {
  player_id: string;
  character_id: string;
}

export interface SSEGameWon {
  winner_player_id: string;
  true_assignments: SSETrueAssignment[];
}

export interface SSEGameAbandoned {
  reason: "host_quit" | "idle_timeout" | "all_players_left";
}

export interface SSEHeartbeat {
  ts: number;
}

export type SSEEventMap = {
  "state.changed": SSEStateChanged;
  "player.joined": SSEPlayerJoined;
  "player.left": SSEPlayerLeft;
  "charcreate.started": SSECharcreateStarted;
  "charcreate.submitted": SSECharcreateSubmitted;
  "charcreate.completed": SSECharcreateCompleted;
  "round.started": SSERoundStarted;
  "answer.submitted": SSEAnswerSubmitted;
  "round.answers_revealed": SSERoundAnswersRevealed;
  "guess.submitted": SSEGuessSubmitted;
  "round.scored": SSERoundScored;
  "game.won": SSEGameWon;
  "game.abandoned": SSEGameAbandoned;
  heartbeat: SSEHeartbeat;
};

export type SSEEventName = keyof SSEEventMap;
