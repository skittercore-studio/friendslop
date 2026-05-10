# Handoff: Friendslop hi-fi UI pass

## Overview

Friendslop is a text-based social-deduction party game (4–8 friends, played live in a browser; Jackbox in spirit). The current production UI is functional but plain — dark theme, monospaced-feel, lots of static text. This handoff is the **hi-fi visual pass** for five of the core screens, in both **phone (390×844)** and **host / TV (1280×720)** viewports.

Public URL: `friendslop.skittercore.studio`
Stack (do not fight): Go + sqlite backend, Preact + Vite frontend, SSE for state sync, single-page app.

## About the design files

The `.html` and `.jsx` files in this bundle are **design references** — prototypes built in plain React + Babel-in-the-browser to show intended look, motion, and behaviour. They are **not production code to copy directly.**

The task is to recreate these designs in the existing **Preact + Vite** frontend, using its established component / state patterns and SSE wiring. Lift design tokens (colors, type, spacing, motion) verbatim; rebuild components in the codebase's idiom. All timers and game state are **server-authoritative** — the frontend just listens and renders.

## Fidelity

**High-fidelity.** Final colors, typography, layout proportions, accent assignments, and motion intent are all set. Pixel-match the visuals; the implementation framework is yours to choose within the existing stack.

## What's in this bundle

| File | Purpose |
| --- | --- |
| `Friendslop Hi-Fi v1.html` | Entry point. Loads everything below into a `<DesignCanvas>` so you can pan / zoom / fullscreen any screen. |
| `friendslop-tokens.css` | **All design tokens.** CSS custom properties for colors, surfaces, type, shadows, and a small library of utility classes (`.fs-btn`, `.fs-charcard`, `.fs-ring`, `.fs-bubble`, `.fs-idx`, etc). This is the source of truth — port these values into your codebase. |
| `friendslop-screens.jsx` | Phone screens (390×844): `LobbyScreen`, `CharcreateScreen`, `AnsweringScreen`, `RevealScreen`, `GridScreen`. |
| `friendslop-host-screens.jsx` | Host / TV screens (1280×720): `LobbyHostScreen`, `CharcreateHostScreen`, `AnsweringHostScreen`, `RevealHostScreen`, `GridHostScreen`. |
| `design-canvas.jsx` | Canvas viewer component used by the entry point. Not part of the game; ignore for implementation. |

To preview locally: open `Friendslop Hi-Fi v1.html` in a browser (no build step).

---

## Design tokens

All values live in `friendslop-tokens.css`. Highlights:

### Colors — surfaces
| Token | Hex | Use |
| --- | --- | --- |
| `--fs-bg` | `#0c0a18` | App background |
| `--fs-bg-1` | `#14112a` | Card surface |
| `--fs-bg-2` | `#1d1a3a` | Elevated card / button rest |
| `--fs-bg-3` | `#2a2654` | Highest elevation |
| `--fs-line` | `#322e60` | Default borders |
| `--fs-line-soft` | `#211e44` | Subtle dividers |

### Colors — text
| Token | Hex |
| --- | --- |
| `--fs-fg` | `#f6f3ff` |
| `--fs-fg-mute` | `#a5a0d0` |
| `--fs-fg-faint` | `#6b6798` |

### Colors — character accent palette (12 hues, sticky-per-character)
Assign one to each character on creation. The same character keeps the same accent across all rounds — this is the visual signal players use to track characters across the deduction grid.

```
#ffd60a  yellow      #ff3d8a  pink
#00e8d4  teal        #c4ff3a  lime
#ff7a3a  orange      #b89cff  lavender
#4ad6ff  sky         #e065ff  magenta
#ff6b6b  coral       #6effbe  mint
#ffb13d  gold        #7e9cff  periwinkle
```

Semantic aliases: `--fs-accent` = yellow (primary CTA), `--fs-live` = pink (urgency / typing / live), `--fs-positive` = lime (success / sealed).

### Typography
- **Display / character names:** `Lilita One` — chunky, friendly, big.
- **UI body:** `Space Grotesk` (400/500/600/700).
- **Quoted answers + monospace UI (codes, timers):** `JetBrains Mono` (400/500/700).

Load via Google Fonts. **Body text floor: 14px on phone, 18px on host.** **Display sizes go up to 220px** (the lobby code on the host view).

### Radii
4px (corkboard index card) · 12px (chips, small buttons) · 14px (buttons) · 16px (cards, bubbles) · 18px (large cards) · 44px (phone shell) · 999px (pills).

### Shadows
- `--fs-shadow-1` — default elevation
- `--fs-shadow-2` — modal / focused card
- `--fs-glow-accent` — yellow primary CTA glow
- `--fs-glow-pink` — live / urgency glow

Per-character cards get a custom glow: `0 12px 30px -12px <accent>55` plus an accent-colored top stripe with `box-shadow: 0 0 24px <accent>`.

### Motion
Three keyframes defined in tokens:
- `fs-pop` — card entry, 400ms, `cubic-bezier(0.34, 1.56, 0.64, 1)` (slight overshoot).
- `fs-slide-in` — answer reveal, 450ms, `cubic-bezier(0.2, 0.9, 0.3, 1)`.
- `fs-pulse-pink` — live/typing indicator, 1.2s ease-out infinite.

Hover scale on buttons: `translateY(-1px)` 120ms. Active: `translateY(1px)`.

---

## Screens — phone (390×844)

### 01 · Lobby — `LobbyScreen`
**Purpose:** Friends join with the 4-letter room code; host watches roster fill.

**Layout (top → bottom):**
1. Status bar + notch (cosmetic, drop in real app).
2. "ROOM" eyebrow label.
3. **Room code** — `Lilita One`, ~88px, yellow→orange gradient, `letter-spacing: 0.12em`. The single biggest element on the screen.
4. Tap-to-copy chip with the join URL.
5. Player ring — 8 slots arranged on a circle (`ringR = 100`). Filled slots show an accent-colored avatar with the player's first initial in `Lilita One`. Empty slots are dashed circles.
6. Roster list below the ring — name + "HOST ★" tag for the host.
7. Footer: primary CTA "START THE GAME" — disabled (greyed `--fs-bg-1`) until ≥4 players. When active, `--fs-glow-accent` shadow.

**Interactions:** new players animate in with `fs-pop` on the avatar + slide-in on the row.

### 02 · Charcreate — `CharcreateScreen`
**Purpose:** Player privately writes one original character (player-written mode only).

**Layout:**
1. Header: "3 of 5 in the pool" counter + 5 dot pips (filled = submitted, accent = self).
2. Two text fields:
   - **Character name** — large input, `Lilita One` placeholder.
   - **Description** — multi-line, `Space Grotesk`, ~3 line floor.
3. **Live preview card** — renders exactly what the other players will see. Updates as the user types. Uses the player's auto-assigned accent.
4. Submit button. After submit: card slides into a "holding area" with rotation, switch to "waiting for others" state showing the dot grid.

### 03 · Answering — `AnsweringScreen`
**Purpose:** Player writes an answer to the prompt **in their secretly-assigned character's voice.**

**Layout:**
1. **Question card** — top, highlighted with yellow accent stripe, `Lilita One` ~28px, `text-wrap: pretty`.
2. **Timer ring** — top-right, 64px diameter, fills as time progresses. Switches to `--fs-live` (pink) and pulses in last 10 seconds.
3. **"You are playing…" character card** — center, accent-striped, with character name + description. This is the player's secret assignment.
4. **Answer textarea** — `JetBrains Mono`, 14px line, full-width.
5. **Submit** primary CTA at the bottom.
6. **Other players' status** — small avatar row at the very edge, "3 of 4 done" label.

### 04 · Reveal — `RevealScreen`
**Purpose:** All answers are revealed publicly, attributed to characters but **not** to players.

**Layout:** scroll list of quote cards. Each card contains:
- Character header strip (accent stripe + glyph + name in `Lilita One`).
- Speech bubble (`.fs-bubble`) containing the answer in `JetBrains Mono`.
- The most recently dropped card has `--fs-glow-pink` and a slight upward `translateY(-4px)`.

Cards drop in one at a time with `fs-slide-in`, ~1.2s gap between drops. Already-shown cards settle into the list.

### 05 · Guess grid — `GridScreen` (corkboard variant)
**Purpose:** Player matches every other player to a character. **The puzzle screen.**

**Layout — corkboard metaphor:**
- Background is a textured dark felt (`#14112a` with subtle radial-gradient noise).
- **Columns = rounds**, **rows = other players**. Past columns show locked-in guesses with their "2/3" pill in the header. Current column is editable.
- Cells render as **index cards** (`.fs-idx` — cream background, ruled line, push-pin top-center). Past cards are pinned and slightly rotated; current-column cards are upright and tappable.
- Below the grid: **character tray** — drag-source row of character chips. One chip is shown mid-drag with elevated shadow + tilt.
- Tap-to-assign OR drag-and-drop. Each character can appear at most once per column (1:1 enforcement).

---

## Screens — host / TV (1280×720)

All host views share `HostShell`: 48px top bar with "friendslop" wordmark, room code, and a status badge on the right. Background uses a subtle scanline noise overlay (`repeating-linear-gradient`, opacity 0.04) for the TV feel.

### Host 01 · Lobby — `LobbyHostScreen`
Two-column. Left: huge gradient room code (220px), join URL above. Right: 8-slot 2×4 grid of player cards (or empty seats with dashed borders). Footer hint: "host taps START on their phone when ready · ≥4 needed".

### Host 02 · Charcreate — `CharcreateHostScreen`
Hero "everyone's writing…" + giant "n / N" counter. Progress dots full-width. 3-column grid of character pool cards: submitted ones use `Lilita One` for the character name, dashed/skeleton placeholders for not-yet-submitted, with a blinking caret + "typing…" label. **Descriptions are deliberately hidden** ("only the assigned player sees it") — only names are public on the host view.

### Host 03 · Answering — `AnsweringHostScreen`
Question at hero scale (84px). Bottom row: 200px timer ring + status pills. Pills states:
- **done** — filled accent, dark text, ✓, accent glow.
- **typing** — outlined accent, blinking-caret marker.
- **idle** — dashed border, faint text.

### Host 04 · Reveal — `RevealHostScreen`
5-up grid of answer cards. Each column = (character header card, then speech bubble below). Most recently revealed lifts (`translateY(-6px)`) and gets accent inner-stroke + glow. Pending columns are dimmed (`opacity: 0.35`) with "incoming…" placeholder.

### Host 05 · Guess grid — `GridHostScreen` (public deduction wall)
**This is the spectator-mode grid — totals only, individual guesses hidden until endgame.**
Header row: "ROUND 1", "ROUND 2", "ROUND 3 · LIVE" (pink). Per-player rows show:
- Past round cells: their "n/3" total in `Lilita One` on dark background.
- Live round cell: either pink-tinted "⌨ writing…" OR — once sealed — full accent fill with strong accent glow and "SEALED".
Big timer ring top-right. Subtle dot pattern on the wall background (the felt texture).

---

## Game flow & server-authoritative state

The frontend renders state pushed via SSE. State machine roughly:

```
LOBBY → CHARCREATE? → ANSWERING ⇄ REVEAL → GRID → SCORE → (ANSWERING | ENDGAME)
```

Per-screen state needs (frontend-side):
- **Lobby:** `roomCode`, `players[]`, `isHost`, `mode` (curated | player-written).
- **Charcreate:** `players[].submitted`, `myCharacter` draft (local), submitted set.
- **Answering:** `currentQuestion`, `myAssignedCharacter` (private), `myAnswer` (local draft until submit), `timerRemaining`, `timerTotal`, `othersDone[]`.
- **Reveal:** ordered `revealedAnswers[]` with `{character, quote}`, `revealCursor` (index of latest drop).
- **Grid:** `roundsHistory[].guesses[]` (locked, with totals on host view), `currentGuess` (editable), `availableCharacters[]`, `othersStatus[]`.

**Critical:** never animate timers off `Date.now()` alone — interpolate between server ticks to avoid drift.

---

## Components catalogue (in the codebase)

Map roughly 1:1 from the JSX scaffolds. Suggested file layout in your codebase:

```
components/
  CharCard.tsx           // accent-striped character card (main + mini variants)
  TimerRing.tsx          // SVG ring, urgent flag flips color + pulse
  SpeechBubble.tsx       // .fs-bubble equivalent
  IndexCard.tsx          // corkboard cell (.fs-idx + pin)
  RoomCode.tsx           // gradient display code
  PlayerAvatar.tsx       // accent-filled circle, sizes sm/md/lg/xl
  StatusPill.tsx         // done / typing / idle states
screens/
  LobbyScreen.tsx
  CharcreateScreen.tsx
  AnsweringScreen.tsx
  RevealScreen.tsx
  GridScreen.tsx
host/
  LobbyHost.tsx
  CharcreateHost.tsx
  AnsweringHost.tsx
  RevealHost.tsx
  GridHost.tsx
```

The host views and phone views share the **same tokens and the same character-accent assignments** but have totally different layouts — don't try to share components beyond atoms (avatar, ring, bubble).

## Open items (not in this hi-fi pass)

The following screens are scoped but not yet hi-fi'd. Wireframes exist in `Friendslop Wireframes.html` (in the parent project, not this bundle):
- **Landing / room create** — mode-choice cards (curated vs player-written) as the lead.
- **Score** — between-rounds public scoreboard, no spoilers.
- **Endgame / win** — confetti, true assignments revealed dramatically one-by-one.

When you implement these, follow the same token system + character-accent stickiness.

## Assets

No bitmap or vector assets shipped — all visual elements are CSS / inline SVG. Character "glyphs" used in mocks (☠, ☕, ✦, ⚡) are unicode placeholders; the brief calls for the host to optionally pick a symbol per character — treat that as a future feature.

Fonts: load `Lilita One`, `Space Grotesk` (400/500/600/700), `JetBrains Mono` (400/500/700) from Google Fonts.
