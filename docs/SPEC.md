# Friendslop — Spec (v0.1 draft)

Working title. Text-based social-deduction party game with Mastermind feedback. Friend-group target.

---

## 1. Game Design Summary

- N players (5-7 sweet spot, hard min 4, soft max 8) join a room.
- Room is configured with a `pool_source`: either `curated` (deal from default pool) or `playerwritten` (players write the pool during a setup phase).
- A pool of N characters is revealed publicly. Authorship is never displayed.
- Each player is secretly assigned one character (1:1, fixed for the game).
- Game proceeds in rounds:
  1. **Question phase:** server posts a question. All players answer in-character.
  2. **Reveal phase:** answers are shown **attributed to the character that produced them**, NOT to the player. The player identity behind each character is the puzzle.
  3. **Guess phase:** every player submits a full mapping `{player → character}` for all *other* players (excluding self).
  4. **Score phase:** each player privately sees their own correct count; everyone sees the leaderboard counts (but not the contents of others' guesses).
- **Win condition:** first player to submit a fully-correct mapping (N-1 / N-1 since self is excluded) at the end of a guess phase. Tie-break: earliest submission timestamp.
- If no one wins, advance to next round with a new question. Assignments are unchanged.

### Key design decisions
- **Assignments fixed across game.** Mastermind logic only works with a stable target.
- **Self excluded from your own guess slate.** Free correct on yourself would inflate scores meaninglessly.
- **Score feedback: count only, not which ones.** Per Toy's rule. Iterative deduction emerges from round-over-round count comparison.
- **Answers attributed to character, not player.** The character is the costume; the puzzle is whose voice bleeds through it. Friend groups know each other's writing — the character constraint is the deception layer that makes that recognition non-trivial. Strategic surface includes voice-spoofing (writing in another player's style to misdirect) and over-/under-playing the character (so the costume itself becomes evidence).
- **Server must strip player_id from broadcast answer payloads.** Replace with `character_id` only. Otherwise the whole game is one DevTools-tab away from being trivially solved. Internal record retains player_id for scoring.
- **No submission timestamps on revealed answers.** `answer.submitted` events fire publicly during the answering phase ("player p1 has submitted") with timestamps. If the reveal payload also carries `submitted_at` per answer, a player can correlate the two and unmask via timing. The reveal must contain `{character_id, text}` only.
- **Reveal order is sorted by `character_id`, not submission order.** Otherwise the order itself leaks the mapping (the third character to submit is the third character in the list).
- *Anonymous mode* (no attribution at all — answers shown as "Answer A/B/C…", players must infer both authorship AND which character was being played) is a planned difficulty toggle, not MVP.
- **Pool size = player count.** Decoy characters (pool > player count) is a planned difficulty toggle, not MVP.

### Player-written characters (`pool_source: playerwritten`)

When the host configures `pool_source = playerwritten` at room creation, the start sequence inserts a **character creation phase** before round 1:

- After `host:start`, room enters state `CHARCREATE` instead of `ANSWERING`.
- Each player privately submits exactly one character (`name` + `blurb`).
- When all N have submitted, server shuffles into a single pool and reveals it publicly. **Authorship is hidden in the public view.** Each player knows only which one *they* wrote (returned via `/me`).
- Server then deals 1:1 random assignments. Authors are not specially excluded from being dealt their own character — random uniform assignment.
- Room transitions `CHARCREATE → ANSWERING (round 1)`, normal play resumes.

**Information-leak acknowledgement.** Each author has private soft information about the character they wrote. They know its style/intent, and once another player's answers betray a matching voice they can lock in that mapping early. This is intentional — it's a small per-player tilt, not a flaw, and rewards writing distinctive characters. To eliminate the leak entirely we would need pool dilution (every player submits 2, server picks N) — deferred to v2.

### What's deliberately out of MVP
- Spectators (leak risk; defer)
- Mid-game player swap (breaks Mastermind state space)
- Custom question banks / character pools (uploaded by host) — defer to v2
- Anonymous-answer mode — defer
- Decoy characters — defer
- Per-character side-pot scoring (reward for "your character was correctly identified") — defer

---

## 2. Room Lifecycle / State Machine

Top-level room state:

```
LOBBY ──host:start──┬─pool_source=curated──────────────▶ ANSWERING (R1)
                    └─pool_source=playerwritten──▶ CHARCREATE ──all submitted──▶ ANSWERING (R1)

ANSWERING ──▶ GUESSING ──▶ SCORING ──┬─winner──▶ WON
                                     └─no winner──▶ ANSWERING (next round)

(any state) ──host:abandon / 7d idle──▶ ABANDONED
```

`IN_ROUND` has substates:

```
ANSWERING ──all answers OR timeout──▶ GUESSING ──all guesses OR timeout──▶ SCORING
```

### Transition triggers
- `LOBBY → CHARCREATE`: host calls `/start` with ≥4 players AND `pool_source=playerwritten`. Server transitions to CHARCREATE, awaiting one character submission per player.
- `CHARCREATE → ANSWERING (R1)`: all N players have submitted a character. Server shuffles the pool, reveals it publicly (authorship hidden), deals 1:1 random assignments, advances `round_number` to 1.
- `LOBBY → ANSWERING (R1)`: host calls `/start` with ≥4 players AND `pool_source=curated`. Server rolls N characters from default pool, deals 1:1 assignments, advances `round_number` to 1.
- `ANSWERING → GUESSING`: all players have a non-null answer for the current round, OR (if `mode=live`) the answer timer expired. Missing answers auto-fill as `"[no answer]"`.
- `GUESSING → SCORING`: all players have submitted a guess for the current round, OR (if `mode=live`) the guess timer expired. Missing guesses score 0.
- `SCORING → WON`: any player's guess for the just-closed round equals the true assignment for all other players. If multiple, earliest `submitted_at` wins.
- `SCORING → ANSWERING (next round)`: no winner. After a brief inter-round pause (configurable, default 10s), advance `round_number` and post a new question.
- `* → ABANDONED`: host explicitly abandons, or no activity for 7 days, or all players have left.

### Modes
- **`live`**: timers enforced. `answer_timeout_seconds` and `guess_timeout_seconds` non-null at room creation. Default 120s answer / 120s guess.
- **`async`**: timers nullable but defaulted to 24h. Round closes only when all submit or 24h pass.

---

## 3. Data Model (SQLite)

```sql
-- Game session
CREATE TABLE rooms (
  id              TEXT PRIMARY KEY,         -- uuid
  code            TEXT UNIQUE NOT NULL,     -- 4-letter join code, e.g. "BRSK"
  state           TEXT NOT NULL,            -- lobby|charcreate|answering|guessing|scoring|won|abandoned
  mode            TEXT NOT NULL,            -- live|async
  pool_source     TEXT NOT NULL,            -- curated|playerwritten
  charcreate_timeout_seconds INTEGER,       -- nullable; live mode default 300s
  host_player_id  TEXT,
  round_number    INTEGER NOT NULL DEFAULT 0,
  answer_timeout_seconds  INTEGER,          -- nullable
  guess_timeout_seconds   INTEGER,          -- nullable
  inter_round_pause_seconds INTEGER NOT NULL DEFAULT 10,
  question_bank   TEXT NOT NULL DEFAULT 'default',
  character_pool  TEXT NOT NULL DEFAULT 'default',
  created_at      INTEGER NOT NULL,         -- unix ms
  started_at      INTEGER,
  ended_at        INTEGER,
  winner_player_id TEXT,
  last_activity_at INTEGER NOT NULL
);
CREATE INDEX idx_rooms_code ON rooms(code);
CREATE INDEX idx_rooms_last_activity ON rooms(last_activity_at);

-- Joined participant
CREATE TABLE players (
  id              TEXT PRIMARY KEY,         -- uuid
  room_id         TEXT NOT NULL REFERENCES rooms(id),
  name            TEXT NOT NULL,
  character_id    TEXT,                     -- null until game start, then immutable
  session_token   TEXT UNIQUE NOT NULL,     -- httpOnly cookie
  is_host         INTEGER NOT NULL DEFAULT 0,
  joined_at       INTEGER NOT NULL,
  left_at         INTEGER,
  UNIQUE (room_id, name)                    -- no duplicate names per room
);
CREATE INDEX idx_players_room ON players(room_id);
CREATE INDEX idx_players_session ON players(session_token);

-- Characters in a specific room's rolled pool (snapshot at game start)
CREATE TABLE room_characters (
  id              TEXT PRIMARY KEY,         -- uuid (per room, not template id)
  room_id         TEXT NOT NULL REFERENCES rooms(id),
  template_id     TEXT,                     -- references global character pool; NULL if player-written
  author_player_id TEXT REFERENCES players(id),  -- NULL if curated; set for player-written. NEVER exposed in public room snapshot.
  name            TEXT NOT NULL,            -- snapshot or submission
  blurb           TEXT NOT NULL             -- snapshot or submission
);
CREATE INDEX idx_room_chars ON room_characters(room_id);
-- Invariant: exactly one of (template_id, author_player_id) is non-null per row.

-- One per round
CREATE TABLE rounds (
  id              TEXT PRIMARY KEY,
  room_id         TEXT NOT NULL REFERENCES rooms(id),
  number          INTEGER NOT NULL,
  question_text   TEXT NOT NULL,            -- snapshot from question bank
  state           TEXT NOT NULL,            -- answering|guessing|scoring|done
  answer_deadline INTEGER,                  -- unix ms, nullable
  guess_deadline  INTEGER,
  started_at      INTEGER NOT NULL,
  closed_at       INTEGER,
  UNIQUE (room_id, number)
);
CREATE INDEX idx_rounds_room ON rounds(room_id);

-- Per-player answer in a round
CREATE TABLE answers (
  round_id        TEXT NOT NULL REFERENCES rounds(id),
  player_id       TEXT NOT NULL REFERENCES players(id),
  text            TEXT NOT NULL,
  submitted_at    INTEGER NOT NULL,
  PRIMARY KEY (round_id, player_id)
);

-- Per-player full guess in a round
CREATE TABLE guesses (
  id              TEXT PRIMARY KEY,
  round_id        TEXT NOT NULL REFERENCES rounds(id),
  guesser_player_id TEXT NOT NULL REFERENCES players(id),
  correct_count   INTEGER NOT NULL,         -- computed at scoring
  submitted_at    INTEGER NOT NULL,
  UNIQUE (round_id, guesser_player_id)
);

-- One row per (other player → guessed character) within a guess
CREATE TABLE guess_entries (
  guess_id        TEXT NOT NULL REFERENCES guesses(id),
  target_player_id TEXT NOT NULL REFERENCES players(id),
  character_id    TEXT NOT NULL REFERENCES room_characters(id),
  PRIMARY KEY (guess_id, target_player_id)
);
```

### Static content (in-code or seed JSON, not DB-mutable in MVP)
- `characters_default.json` — array of `{template_id, name, blurb}`. Target: 60-100 entries.
- `questions_default.json` — array of `{id, text}`. Target: 40-60 entries.

---

## 4. HTTP API

All under `/api/v1/`. JSON in/out. Session cookie `slop_session=<token>` (httpOnly, SameSite=Lax) set on join.

### Public (no session)
```
POST   /api/v1/rooms
       body: {
         host_name,
         mode,                                  // live|async
         pool_source,                           // curated|playerwritten
         answer_timeout_seconds?,
         guess_timeout_seconds?,
         charcreate_timeout_seconds?            // only meaningful if pool_source=playerwritten and mode=live
       }
       → { room_code, session_token }   (also Set-Cookie)

POST   /api/v1/rooms/:code/join
       body: { name }
       → { room_code, player_id, session_token }   (also Set-Cookie)

GET    /api/v1/rooms/:code
       → public room snapshot (no secrets — see §4.1)
```

### Authenticated (session cookie required, must match room)
```
GET    /api/v1/rooms/:code/me
       → private player view (see §4.2 — includes own character)

POST   /api/v1/rooms/:code/leave
       → 204

POST   /api/v1/rooms/:code/start         (host only)
       → 204; transitions LOBBY → ANSWERING

POST   /api/v1/rooms/:code/abandon       (host only)
       → 204; transitions any → ABANDONED

POST   /api/v1/rooms/:code/character     (only valid in CHARCREATE state)
       body: { name, blurb }
       constraints: name 1-60 chars, blurb 20-300 chars (negotiable)
       → 204
       409 if already submitted (one per player; no edits in MVP)

POST   /api/v1/rooms/:code/answer
       body: { round_number, text }       (round_number guards stale submits)
       → 204

POST   /api/v1/rooms/:code/guess
       body: { round_number, mapping: { <target_player_id>: <character_id>, ... } }
       → 204
       400 if mapping is incomplete (must cover all other players)
       400 if any character_id is reused (must be 1:1 over the pool minus your own)

GET    /api/v1/rooms/:code/events
       SSE stream — see §5
```

### Static
```
GET    /api/v1/character-pools           → list pool ids + sample
GET    /api/v1/question-banks            → list bank ids + sample
GET    /healthz                          → 200 OK
```

### 4.1 Public room snapshot

```jsonc
{
  "code": "BRSK",
  "state": "guessing",          // lobby|charcreate|answering|guessing|scoring|won|abandoned
  "mode": "async",
  "pool_source": "playerwritten",
  "round_number": 3,
  "players": [
    { "id": "p1", "name": "bob",   "is_host": true,  "left": false },
    { "id": "p2", "name": "alice", "is_host": false, "left": false }
  ],
  "characters": [               // present after CHARCREATE completes (or immediately if curated)
                                  // authorship NEVER exposed here, even after game ends
    { "id": "c1", "name": "Hermione Granger", "blurb": "..." },
    { "id": "c2", "name": "Sherlock Holmes",  "blurb": "..." }
  ],
  "current_round": {
    "number": 3,
    "state": "guessing",
    "question_text": "Describe your worst Monday morning.",
    "answer_deadline": 1736500000000,
    "guess_deadline":  1736500120000,
    "answers": [                // present in guessing|scoring states only
                                  // attributed to character_id, NEVER to player_id
      { "character_id": "c1", "text": "..." },
      { "character_id": "c2", "text": "..." }
    ]
  },
  "scoreboard": {               // counts of correct guesses per player, per round
    "rounds": [
      { "round_number": 1, "scores": { "p1": 2, "p2": 1 } },
      { "round_number": 2, "scores": { "p1": 3, "p2": 2 } }
    ]
  },
  "winner_player_id": null
}
```

### 4.2 Private `/me` view

```jsonc
{
  "player_id": "p2",
  "name": "alice",
  "is_host": false,
  "your_character": { "id": "c1", "name": "Hermione Granger", "blurb": "..." },
  "your_authored_character_id": "c4",       // present iff pool_source=playerwritten AND CHARCREATE complete; the character this player wrote. Equal to your_character.id in the current implementation: every author plays the character they authored (deterministic assignment, no shuffle).
  "your_answer_for_current_round": "...",   // null if not submitted
  "your_guess_for_current_round": {         // null if not submitted
    "p1": "c2",
    "p3": "c4"
  },
  "your_past_guesses": [
    { "round_number": 1, "mapping": { "p1": "c2", "p3": "c4" }, "correct_count": 1 },
    { "round_number": 2, "mapping": { "p1": "c4", "p3": "c2" }, "correct_count": 2 }
  ]
}
```

This is the data behind the **guess grid** UI — enough to render historical guesses with scores so the player can do Mastermind reasoning.

---

## 5. SSE Events

Stream: `GET /api/v1/rooms/:code/events`. Server emits named events. Client uses `EventSource` and `addEventListener(name, ...)`.

```
event: state.changed
data: { "state": "guessing", "round_number": 3 }

event: player.joined
data: { "player_id": "p3", "name": "carol" }

event: player.left
data: { "player_id": "p3" }

event: charcreate.started
data: { "deadline": 173... | null }                 // only fires if pool_source=playerwritten

event: charcreate.submitted
data: { "submitted_count": 3, "total_players": 5 }   // aggregate, no player attribution

event: charcreate.completed
data: { "characters": [ { "id": "c1", "name": "...", "blurb": "..." }, ... ] }
// Pool revealed publicly. Authorship NOT included. Order is shuffled.

event: round.started
data: { "round_number": 3, "question_text": "...", "answer_deadline": 173... }

event: answer.submitted
data: { "player_id": "p1" }              // metadata only — text is delivered at reveal

event: round.answers_revealed
data: {
  "round_number": 3,
  "answers": [ { "character_id": "c1", "text": "..." }, ... ],   // character-attributed only
  "guess_deadline": 173...
}

event: guess.submitted
data: { "player_id": "p1" }

event: round.scored
data: {
  "round_number": 3,
  "public_scores": { "p1": 4, "p2": 2 },   // counts only, no mapping content
  "next_round_at": 173...                  // null if game ended this round
}

event: game.won
data: {
  "winner_player_id": "p1",
  "true_assignments": [ { "player_id": "p1", "character_id": "c2" }, ... ]
}

event: game.abandoned
data: { "reason": "host_quit" | "idle_timeout" | "all_players_left" }

event: heartbeat
data: { "ts": 173... }                     // every 25s, keep proxies happy
```

Private per-recipient counts (your own correct count for the round) are NOT emitted on SSE — clients re-fetch `/me` after `round.scored` to learn their own number. Keeps SSE stream identical for everyone, simplifies fan-out.

Alternative considered: per-player SSE channels. Rejected for MVP — extra plumbing, no benefit at expected scale.

---

## 6. UI Screens (frontend MVP)

Single-page Preact app. Routes via hash or simple state:

1. **Landing** — "Create room" / "Join with code". Creating asks for name + mode + pool_source (curated vs player-written) + (if live) timer values.
2. **Lobby** — list of joined players, room code prominent for sharing, host has "Start" button (disabled if <4 players). UI hint shows which `pool_source` was chosen.
3. **Character creation** (only if `pool_source=playerwritten`) — single textarea form with `name` + `blurb` fields and submit button. Live counter "3 of 5 players have submitted." After submit, player sees their own submission read-only and waits. When all in, transitions to game screen with the revealed pool.
4. **Game screen** — single layout, sections show/hide based on state:
   - **Top:** character pool (always visible once game started)
   - **Centre-left:** current question + answer input (during answering) OR all answers attributed **to characters** (during guessing/scoring) — e.g. a stack of cards labelled "Hermione Granger said:", "Sherlock Holmes said:", each with the answer text. Player attribution is the puzzle and never appears here.
   - **Centre-right:** **guess grid** — rows = other players, columns = past rounds, cells = your guessed character per round, with per-row "this round" dropdown + per-row "edit" for current round; right margin shows per-round correct count for *your* row of submissions
   - **Bottom:** scoreboard (round-over-round counts for all players)
5. **Endgame** — winner banner, true assignment table (player → character), "Play again" → spawns new room with same roster. **Character authorship is NOT revealed**, even at game end. Authors keep that info private permanently — preserves replay value within the same friend group and avoids embarrassment about who wrote the cursed one.

The guess grid is the killer feature. Design sketch:

```
                  R1     R2     R3   ← rounds (cols)
                  2/3    3/3    ?    ← your correct count per round

Bob       →   Hermione  Gandalf  [▼]
Alice     →   Gandalf   Hermione [▼]
Carol     →   Sherlock  Sherlock [▼]
                                  ↑ editable until guess submitted
```

When the player tweaks the current-round column and notices "swapping these two costs me a point" they're doing exactly the deduction the game wants.

---

## 7. Default Content

### Character pool — bias toward recognisability

Mix of:
- **Fiction:** Sherlock Holmes, Hermione Granger, Gandalf, Voldemort, Yoda, Walter White, Dwight Schrute, Hannibal Lecter, Captain Jack Sparrow, Wednesday Addams, …
- **Archetypes:** Conspiracy theorist, life coach, surly teen, dramatic theatre kid, doomsday prepper, midwestern mom, French philosopher, …
- **Memes / internet:** Gigachad, doomer, normie, cottagecore witch, finance bro, …

Target ~80 characters. Each entry: `{template_id, name, blurb}`. Blurb is one short line — enough to remind players who the character is, doesn't have to be in-depth.

### Question bank — bias toward voice-revealing prompts

- Describe your worst Monday morning.
- What's in your pockets right now?
- You're stuck in an elevator. What do you do?
- Describe your ideal first date.
- Write a haiku about your current job.
- You've been arrested. What did you do?
- Describe yourself in three words.
- What's the most embarrassing thing you own?
- You walk into a bar. Then what?
- Tell us about your most controversial opinion (food edition).
- What would you do with £1m and 24 hours?
- Describe the worst gift you've ever received.
- ...

Target 50 questions. Bias toward open-ended, story-prompting. Avoid factual / yes-no.

---

## 8. Tech Stack & Deploy

- **Backend:** Go (chi router), SQLite (WAL mode, single file), in-memory pubsub for SSE fan-out per room.
- **Frontend:** Preact + TypeScript + Vite. Single bundle, served from same Go server.
- **Container:** single Docker image, multi-stage build (Vite build → embed in Go binary via `embed.FS`).
- **Host:** new Hetzner cax11 (`skittercore-slop`, 10.10.0.6) on existing `skittercore-net`. ~€4/mo.
- **Routing:** add to `/etc/caddy/Caddyfile` on `skittercore-rp`:
  ```
  friendslop.skittercore.studio {
      reverse_proxy 10.10.0.6:8080
  }
  ```
- **Persistence:** SQLite file at `/data/slop.db`, daily rsync snapshot to forge if we care (probably don't — games are ephemeral).
- **Logs:** stdout → journald → loki/promtail later if observability stack ever lands.
- **Auth secrets:** none. Session tokens are random 32-byte base64 in cookie + DB.

---

## 9. MVP Scope Cut

**In:**
- Live mode + async mode (timers configurable at room create)
- Default character pool + question bank, baked into binary
- Full game loop (lobby → rounds → win/abandon)
- Guess grid UI with per-round score history
- Browser push for round transitions (defer if Service Worker bites)

**Out (v2+):**
- Anonymous-answer mode toggle (no attribution — neither player nor character shown on each answer)
- Decoy characters (pool > player count)
- Custom uploaded pools/banks
- Spectators
- Webhook integration to Discord/Telegram for round pings
- Side-pot scoring for "your character was correctly identified"
- Replay viewer / shareable game summary
- Account system / persistent stats

---

## 10. Open Questions

1. **Spelling: "Friendslop" vs "FriendSlop" vs new name?** Working title only.
2. **Domain — `friendslop.skittercore.studio` or buy a standalone?** Subdomain is free and pattern-consistent. Defaults to subdomain unless Toy wants branded.
3. **Min player count — 4 or 5?** 4 is mathematically tight (3! = 6 permutations, solved in 1-2 rounds). Lean toward hard-min 4 with a "you'll solve this fast" warning, encourage 5+.
4. **Inter-round pause — auto-advance after 10s, or require host click?** Auto-advance is friendlier for async; host-button is more controlled. Lean auto.
5. **What happens if the host leaves mid-game?** Promote earliest-joined remaining player, or freeze room? Lean: promote silently, log it.
6. **Character ID stability — if pool changes between game versions, do old game replays still resolve names?** `room_characters` snapshots names at game start, so replay works. But links to "what is this character" for shareable summaries need a stable template_id. Probably fine — just version the JSON.
7. **Profanity / abuse moderation in answers?** Friend-group context — trust roster. No moderation in MVP. If we ever open public matchmaking we'd need it.
8. **Minimum answer length / max length?** A player typing "I am Hermione" verbatim is unhelpful but technically legal. Lean toward soft min 20 chars + hard max 500 chars. Negotiable.
9. **Should answer.submitted events fire at all?** They give live "X of N submitted" feedback in the lobby (good UX) but they're a low-bandwidth timing oracle (correlate event timestamp with later round events). Mitigation: emit with debounced/jittered delays, or aggregate as "3 of 5 submitted" rather than naming the player. Lean toward aggregate-only for safety.
10. **Reveal authorship at endgame?** Spec'd current = NO. Keeping authorship hidden forever lets the same friend group replay the same set of player-written characters across nights. If we ever want a "post-game gossip" reveal it'd need an explicit host opt-in.
11. **Editable character submissions during CHARCREATE?** Currently no — first submit locks. Could allow edits until everyone has submitted, at small UX cost. Lean toward editable-until-locked for the next iteration.
12. **Bad-faith character submissions** (one-word, slurs, "asdf"). Friend-group context — same trust as answers. Soft min-length on the blurb (20 chars) catches most laziness. No moderation in MVP.

---

## 11. Build Order

Once Toy approves the shape:

1. Schema + Go skeleton + room/player CRUD + cookie session
2. Round state machine (no UI yet) — testable end-to-end via curl
3. SSE fan-out + event emission
4. Default content JSON (characters + questions)
5. Frontend skeleton: landing + lobby + bare game screen
6. Guess grid UI (the hero piece)
7. Score reveal + win detection
8. Polish: timers, push notifications, endgame screen
9. Deploy: cax11 provision, Docker, Caddy route, smoke test

Each stage shippable in isolation. Stages 1-4 can be one specialist's brief; 5-8 a frontend specialist or same hand. Stage 9 is ops.

---

_Spec author: Vex. v0.1 draft, awaiting Toy review._
