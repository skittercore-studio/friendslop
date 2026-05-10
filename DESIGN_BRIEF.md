# Friendslop — Design Brief

## What is Friendslop?

Friendslop is a text-based social-deduction party game for 4–8 friends, played live in a browser. The conceit is simple and weird: at the start of a round everyone is secretly assigned a fictional character — Cersei Lannister, an Exhausted Barista, Lord Voldemort, etc. Everybody answers the same goofy question **in their character's voice**, the answers are revealed with the *character* attached but not the player, and then everyone has to guess which of their friends is playing which character. Get all the others right and you win.

It's Jackbox in spirit — phone in hand, friends in voice chat, a host's screen optionally on the TV. But the gameplay is closer to Mastermind crossed with a low-stakes improv exercise. The funniest answers are the ones that lean hard into the character.

There are two modes for character pools:

- **Curated** — characters come from a built-in list (Cersei, Voldemort, an Exhausted Barista, a Gigachad…). Fast, low effort, great for first-time groups.
- **Player-written** — before the game starts, every player writes one original character (name + one-paragraph description) which is added to the pool. The other players are then secretly assigned each other's creations. This is where the "in-jokes between friends" magic happens.

Public URL: **friendslop.skittercore.studio**

## Core game loop

1. **Lobby** — host creates a room, gets a 4-letter code (e.g. `DDWR`). Friends join with the code + a display name. Min 4 players to start, no hard max but 5–7 is the sweet spot.
2. **(Optional) Charcreate** — only in player-written mode. Each player privately writes a character. UI shows a live counter ("3 of 5 submitted").
3. **Answering** — a question appears ("You're stuck in an elevator with a stranger. What do you do?"). Each player sees only *their own* secretly-assigned character and writes an answer in that voice. Timer (default 2 min, configurable).
4. **Reveal & Guess** — all answers are shown publicly, attributed to characters but **not** to players. Everyone now has to fill in a grid: for each *other* player, which character do you think they were playing? Timer (default 2 min).
5. **Score** — server reveals each player's correct count (e.g. "you got 2 of 3 right"). Past rounds' guesses stay visible as a history grid — every column is a round, every row is a player, and the cells fill in over time. This is the deduction signal: if Sir Reginald's writing style was clearly the same person across rounds 1–3, you can pin them down.
6. **Win** — first person to get a *perfect* guess (all other players correctly mapped in a single round) wins immediately. Otherwise rounds keep coming until someone cracks it.

## What the brief is asking for

The current UI is functional but plain — dark theme, monospaced-feel, lots of static text. **We want it to feel more like a Jackbox game** — visually expressive, animated, *eventful*. Specifically:

- **The question reveal should land**. Right now it's just text appearing. We want the question to swoop in, characters to feel like cards being dealt, the whole thing to feel like a TV game show.
- **Character cards should have personality**. Right now they're rectangles with a name and a blurb. Could they tilt slightly? Have a subtle gradient per character? Maybe an emoji/symbol the host picks?
- **Answer reveals should feel theatrical**. Each answer dropping in with the character avatar attached, one at a time, with a beat between. Not a static list.
- **The guess grid is the puzzle screen**. Right now it's a vanilla HTML table. It should feel like *evidence pinned to a corkboard* — past rounds locked in, current round editable, drag-and-drop or tap-to-assign for putting characters next to player names.
- **Timer urgency**. The current "Answer by 0:30" text is correct but flat. Give us a ring/bar that *fills* and pulses red in the last 10 seconds.
- **Win celebration**. When someone wins, the whole screen should erupt — confetti, the winner's character revealed, the true assignments slide-show'd one by one ("It was YOU all along, Cersei…").

## Screens to design

1. **Landing / room create** — currently a tabbed form. The "create room" path should **lead with mode choice** (curated vs player-written) as visually distinct cards rather than a dropdown. Player-written should look enticing — show example character cards as a preview. Currently this is buried and Toy himself didn't realize it was an option.
2. **Lobby** — show the room code huge, list joined players as cards animating in. "Waiting for host to start" state for non-hosts. Host gets a big START button that's disabled until ≥4 players.
3. **Charcreate** (player-written only) — split screen: left is the form ("Character name" + "Description"), right is a live preview card showing exactly how it'll appear to the others. "3 of 5 submitted" counter at the top. When you submit, your card slides into a holding area and a "waiting for others" state appears.
4. **Answering screen** — top: question card (highlighted, central), with a countdown ring. Center: your private "you are playing X" card showing the character you're inhabiting. Below: a text area for your answer. Other players' state shown as small avatars at the edge ("3 of 4 done").
5. **Reveal + Guess screen** — answers slide in one by one as character-attributed quote cards (like comic-book speech bubbles maybe?). Then transition into the guess grid view.
6. **Guess grid** — horizontal columns are rounds, vertical rows are *other* players. Past columns show locked-in guesses with their correct counts (e.g. "2/3" pill in the header). Current column is editable — tap a cell, get a character picker. 1:1 enforcement: each character can only appear once per column.
7. **Score screen** (between rounds) — quick scoreboard, public correct counts only (no spoilers about the true assignments — those are revealed only at game end). Inter-round pause ~10s, configurable.
8. **Endgame / win screen** — winner announced, true assignments revealed dramatically one at a time, share-the-room button.

## Visual language we're chasing

- **Energetic & playful** without being childish. The wit lives in the writing; the visuals should give it a stage.
- **Reads on a phone in landscape OR portrait**. Most plays will be on phones with a desktop "host screen" optional.
- **Dark theme primary**, with strong saturated accent colors (think neon yellow / hot pink / electric teal). One bright accent per character — pick from a palette so the same character keeps the same colour across rounds.
- **Type:** big and friendly for character names, monospace or distinct face for player-written answers (so they read as "quoted").
- **Microinteractions everywhere**: hover scales, tap pulses, card flips, ring fills. The site should feel alive even when nothing's happening.

## Technical constraints (don't fight these)

- **Stack:** Go + sqlite backend (stable), Preact + Vite frontend (small bundle). State syncs via SSE.
- **Single-page app**. Routes are screen states, no server-side rendering.
- **No 3D / no WebGL** — CSS animations + maybe Lottie/SVG for confetti is the budget.
- **All timers and game state are server-authoritative**. Frontend just listens and renders. Don't design anything that assumes the client controls round flow.
- **No accounts.** Cookie-based ephemeral sessions; players are distinguished only by the display name they pick on join.

## What "good" looks like

If a friend who's never played sees a 30-second clip of someone playing Friendslop, they should think:
1. "Oh that's like Jackbox, I get it."
2. "That looks fun, who's the kept-livestock-flagged player named Coke?"
3. "I want to play that *now*."

The current UI gets stuck on (1) and never hits (3). That's the gap to close.
