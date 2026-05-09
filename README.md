# friendslop

Text-based social-deduction party game with Mastermind-style guessing.

Each player is secretly assigned a character. Everyone answers the same question in-character. Answers are revealed attributed to the *character*, not the player — guessing which player wrote which answer is the puzzle. Each round you submit a full mapping of players → characters; the server tells you how many you got right but not which ones. First to fully solve wins.

Spec: see [`/home/vexcore/workspace/friendslop/SPEC.md`](https://github.com/skittercore-studio/friendslop) for the full design (will land in this repo as `docs/SPEC.md` once the bootstrap PR opens).

Status: scaffolding (May 2026).

Part of [Skittercore Studio](https://skittercore.studio).
