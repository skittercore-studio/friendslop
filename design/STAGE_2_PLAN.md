# Stage 2 — Phone Screen Port: Agent Scoping

This document is the contract for the four Stage 2 agents. Each agent rewrites
**exactly one screen file**, consumes the shared store + atoms that already
exist, and must NOT touch anything outside its declared scope.

## What we're doing

Porting the four interactive phone screens from the design handoff
(`design/handoff-v1/`) into the live Preact codebase. The atoms layer
(`frontend/src/components/atoms/`) was built in Stage 1 — every visual
primitive the screens need is already there.

The screens consume real, server-authoritative data via `store.ts` (Preact
signals) and submit via `api.ts`. **Do not invent local game state.** Read
the snapshot, render it, post on submit, let SSE refresh push the next
state.

## What's already done (DO NOT REDO)

- `frontend/src/tokens.css` — full design system (colours, type, atoms).
- `frontend/src/components/atoms/` — `TimerRing`, `CharCard`, `PlayerAvatar`,
  `SpeechBubble`, `IndexCard`, `StatusPill`, `RoomCode`, plus the palette /
  accent / glyph helpers in `palette.ts`. Import from `../components/atoms`.
- Google Fonts in `index.html` (Lilita One, Space Grotesk, JetBrains Mono).
- `store.ts` exposes `now` (a 1Hz-ticking signal) — use this for timer
  countdowns; do NOT spawn your own `setInterval`.
- `screens/Game.tsx` is now a thin dispatcher that picks `AnswerPhase` /
  `GuessPhase` based on `room.value.state`.

## Agent assignments

Each agent owns ONE file. No exceptions.

### Agent A — `screens/Lobby.tsx`

Replaces the existing Lobby. Design ref: `design/handoff-v1/friendslop-screens.jsx`
→ `LobbyScreen`.

Required behaviour:
- Show the `RoomCode` huge at the top.
- Render the player ring around a centre `n / 8` counter. Use the lobby
  ring layout from the design (8 slots in a circle, dashed border, slot
  avatars filled vs empty).
- Players are `room.value.players.filter(p => !p.left)`. Each player gets
  an `accentForPlayer(p.id)` colour. Host gets a star badge.
- Show pool source and mode as `StatusPill`s (one is `--on` for the active
  setting — though the lobby is post-creation so they're informational
  only, not toggles).
- Host sees a primary `START THE SLOP →` button, disabled when
  `activePlayers.length < 4`.
- Non-hosts see "waiting for host to start" copy.
- Show a small "leave room" affordance (ghost button) and host-only
  "abandon" button — keep these from the existing screen, restyled.

Calls: `api.startGame(code)`, `api.abandonRoom(code)`, `leaveAndReset(code)`.

### Agent B — `screens/CharCreate.tsx`

Replaces existing CharCreate. Design ref: `CharcreateScreen`.

Required behaviour:
- Top: title + `StatusPill` showing `submitted / total` from
  `charcreateProgress` signal (fall back to `me.your_authored_character_id`
  presence for own counter contribution).
- Two stacked card-style fields: name (1-60 chars) and blurb (20-300 chars).
  Live-character-count chip in the corner of the blurb.
- Below the form: a live-preview `CharCard` rendering exactly what others
  will see when this character is in the pool. Use `accentFor(<temporary
  preview id>)` — pick a stable preview id like `"preview"` or use the
  player's id so the preview stays the same colour across keystrokes.
- After submit: collapse the form, show a "locked in, waiting on M and J"
  state with mini avatars of players who haven't submitted yet (those are
  players in `room.players` whose count hasn't bumped — for MVP just show
  the counter and a generic waiting state; deriving who-hasn't-submitted
  from public data isn't currently possible).
- Validation matches existing rules; preserve them.

Calls: `api.submitCharacter(code, name, blurb)`.

### Agent C — `screens/AnswerPhase.tsx`

NEW file. Design ref: `AnsweringScreen`. Mounted by Game.tsx when
`room.value.state === "answering"`.

Required behaviour:
- Top row: `TimerRing` (left) + meta block (right) with round number, mini
  player avatars showing who has submitted (`answer.submitted` SSE bumps
  the count; for MVP we can show all `n / total` only — see below).
- Question card: gradient background, `THE QUESTION` tiny label, then the
  question text in `fs-display`. Source: `room.value.current_round.question_text`.
- "YOU ARE PLAYING" — render a `CharCard` for `me.your_character`. The
  description is `me.your_character.blurb`. Sticky accent via
  `accentFor(me.your_character.id)`.
- Below: an answer textarea styled per the design (mono font, dark card
  shell). Submits via `api.submitAnswer(code, round_number, text)`.
  Disable + show "submitted, waiting" state once `me.your_answer_for_current_round`
  is non-null.
- Timer: derive remaining seconds from
  `room.value.current_round.answer_deadline - now.value` (clamped). Mark
  `urgent={true}` when remaining ≤ 10s. Total = `answer_timeout_seconds`
  if exposed; otherwise compute from `answer_deadline - first_seen_at` —
  for MVP **just pass `total = remaining_at_first_render` cached in a ref**.
  Document the limitation.
- Submitted-count chip: backend doesn't expose answer-submitted-count
  publicly. Show "your answer locked" once you've submitted; don't try to
  fake a per-player tracker. (`SSEAnswerSubmitted` only carries player_id —
  if you want a counter, count distinct events received since round start.
  For MVP, just show your own state.)

### Agent D — `screens/GuessPhase.tsx`

NEW file. Design ref: `RevealScreen` + `GridScreen` combined into a single
screen, separated visually. Mounted by Game.tsx when
`room.value.state === "guessing"` or `"scoring"`.

Required behaviour:
- Top: countdown `TimerRing` from `current_round.guess_deadline - now`.
- **Reveal section** (top half): each entry in
  `current_round.answers` rendered as a `CharCard` (mini) + `SpeechBubble`.
  Stagger fade-in per entry (use `fs-anim-slide` with `animationDelay:
  0.15s * i`). Highlight the most-recently-arrived bubble with
  `accent={accentFor(answer.character_id)}` for the freshness glow.
  Look up character by `id` against `room.characters`.
- **Grid section** (bottom half): the existing `GuessGrid` logic
  rebuilt as the corkboard pattern from `GridScreen`:
  - Rows: `room.players.filter(p => p.id !== me.player_id && !p.left)`.
  - Columns: each round (past columns from `me.your_past_guesses` showing
    locked-in cells with their correct count in the column header; current
    column editable when `state === "guessing"` and not yet submitted).
  - Cells: `IndexCard` with the character name. Past cells get a green pin
    (correct) or grey pin (wrong) based on whether the past guess matched
    the still-private true assignment — but **we don't have the truth yet
    in mid-game** — so only colour past cells by per-round correct count
    distribution (we know the count, not which cells were right). For MVP,
    just show all past pins as neutral and rely on the column-header score
    pill (e.g. `2/3`).
  - Editable column: tap-to-pick. The design shows drag-and-drop; for MVP
    do tap-to-cycle through available characters (`<select>`-equivalent
    behaviour, but visually rendered as an empty `IndexCard` waiting for a
    pin, opening a picker on tap).
  - 1:1 enforcement client-side (already in existing GuessGrid; copy that
    validation logic).
- Submit button: "LOCK GUESSES →" — disabled until valid + not submitted.
  After submit: badge shows "guess locked in for R{n}".

Calls: `api.submitGuess(code, round_number, mapping)`.

## Conventions

- **Preact:** use `class=` not `className=`. The repo mixes both for
  historical reasons; new code should standardise on `class=`.
- **Strict TS:** `noUnusedLocals`/`noUnusedParameters` are on. Prefix
  unused params with `_`.
- **No new dependencies.** Everything you need is in `node_modules`
  already.
- **No new globals on `store.ts`.** If a screen needs a derived value,
  compute it locally with `useMemo`.
- **Styles:** prefer the `fs-*` utility classes in tokens.css. Add
  per-screen tweaks via inline `style={...}` rather than editing
  `styles.css`. The legacy `styles.css` is dead-code-in-waiting; do
  not modify it.
- **Layout:** mobile-first. Stage 3 (separate agent) handles desktop. Don't
  add `@media (min-width: 1024px)` blocks here.

## Validation

Each agent must, before declaring done:
1. `cd frontend && npm run typecheck` → clean.
2. `cd frontend && npm run build` → success.
3. Smoke-render its screen by patching the dispatcher temporarily — but
   revert the patch before commit. (We'll wire up real game testing after
   octopus merge.)

## Commit hygiene

- Branch name: `stage2/<screen>` (e.g. `stage2/lobby`).
- Single commit per agent unless the change is genuinely
  multi-feature. Co-author footer per repo convention.
- Don't push. Vex octopus-merges from local worktrees.
