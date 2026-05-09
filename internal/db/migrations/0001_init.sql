-- 0001_init.sql
-- Initial schema for friendslop. Mirrors SPEC.md §3.
-- All tables, indexes and constraints required for MVP.

PRAGMA foreign_keys = ON;

-- Game session
CREATE TABLE IF NOT EXISTS rooms (
  id                          TEXT PRIMARY KEY,
  code                        TEXT UNIQUE NOT NULL,
  state                       TEXT NOT NULL,
  mode                        TEXT NOT NULL,
  pool_source                 TEXT NOT NULL,
  charcreate_timeout_seconds  INTEGER,
  host_player_id              TEXT,
  round_number                INTEGER NOT NULL DEFAULT 0,
  answer_timeout_seconds      INTEGER,
  guess_timeout_seconds       INTEGER,
  inter_round_pause_seconds   INTEGER NOT NULL DEFAULT 10,
  question_bank               TEXT NOT NULL DEFAULT 'default',
  character_pool              TEXT NOT NULL DEFAULT 'default',
  created_at                  INTEGER NOT NULL,
  started_at                  INTEGER,
  ended_at                    INTEGER,
  winner_player_id            TEXT,
  last_activity_at            INTEGER NOT NULL,
  CHECK (state IN ('lobby','charcreate','answering','guessing','scoring','won','abandoned')),
  CHECK (mode IN ('live','async')),
  CHECK (pool_source IN ('curated','playerwritten'))
);
CREATE INDEX IF NOT EXISTS idx_rooms_code ON rooms(code);
CREATE INDEX IF NOT EXISTS idx_rooms_last_activity ON rooms(last_activity_at);

-- Joined participant
CREATE TABLE IF NOT EXISTS players (
  id              TEXT PRIMARY KEY,
  room_id         TEXT NOT NULL REFERENCES rooms(id),
  name            TEXT NOT NULL,
  character_id    TEXT,
  session_token   TEXT UNIQUE NOT NULL,
  is_host         INTEGER NOT NULL DEFAULT 0,
  joined_at       INTEGER NOT NULL,
  left_at         INTEGER,
  UNIQUE (room_id, name)
);
CREATE INDEX IF NOT EXISTS idx_players_room ON players(room_id);
CREATE INDEX IF NOT EXISTS idx_players_session ON players(session_token);

-- Characters in a specific room's rolled pool (snapshot at game start)
-- Invariant: exactly one of (template_id, author_player_id) is non-null per row.
CREATE TABLE IF NOT EXISTS room_characters (
  id                TEXT PRIMARY KEY,
  room_id           TEXT NOT NULL REFERENCES rooms(id),
  template_id       TEXT,
  author_player_id  TEXT REFERENCES players(id),
  name              TEXT NOT NULL,
  blurb             TEXT NOT NULL,
  CHECK (
    (template_id IS NOT NULL AND author_player_id IS NULL) OR
    (template_id IS NULL AND author_player_id IS NOT NULL)
  )
);
CREATE INDEX IF NOT EXISTS idx_room_chars ON room_characters(room_id);

-- One per round
CREATE TABLE IF NOT EXISTS rounds (
  id              TEXT PRIMARY KEY,
  room_id         TEXT NOT NULL REFERENCES rooms(id),
  number          INTEGER NOT NULL,
  question_text   TEXT NOT NULL,
  state           TEXT NOT NULL,
  answer_deadline INTEGER,
  guess_deadline  INTEGER,
  started_at      INTEGER NOT NULL,
  closed_at       INTEGER,
  UNIQUE (room_id, number),
  CHECK (state IN ('answering','guessing','scoring','done'))
);
CREATE INDEX IF NOT EXISTS idx_rounds_room ON rounds(room_id);

-- Per-player answer in a round
CREATE TABLE IF NOT EXISTS answers (
  round_id      TEXT NOT NULL REFERENCES rounds(id),
  player_id     TEXT NOT NULL REFERENCES players(id),
  text          TEXT NOT NULL,
  submitted_at  INTEGER NOT NULL,
  PRIMARY KEY (round_id, player_id)
);

-- Per-player full guess in a round
CREATE TABLE IF NOT EXISTS guesses (
  id                 TEXT PRIMARY KEY,
  round_id           TEXT NOT NULL REFERENCES rounds(id),
  guesser_player_id  TEXT NOT NULL REFERENCES players(id),
  correct_count      INTEGER NOT NULL,
  submitted_at       INTEGER NOT NULL,
  UNIQUE (round_id, guesser_player_id)
);

-- One row per (other player → guessed character) within a guess
CREATE TABLE IF NOT EXISTS guess_entries (
  guess_id          TEXT NOT NULL REFERENCES guesses(id),
  target_player_id  TEXT NOT NULL REFERENCES players(id),
  character_id      TEXT NOT NULL REFERENCES room_characters(id),
  PRIMARY KEY (guess_id, target_player_id)
);
