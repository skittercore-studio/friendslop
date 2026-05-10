# Rename plan: Friendslop → Inkognito

Captured 2026-05-10 ahead of the playtest round. Working title is
**Inkognito** (pun on incognito + ink — captures the "secretly writing
as someone" mechanic). Lock-in deferred until after a few playtest
sessions confirm the game feel; this doc is the punch list for when
that switch flips.

## Scope

`grep -rn "[Ff]riendslop"` finds **~104 occurrences across 43 files** at
HEAD `5552702`. They split into five mechanical categories:

### 1. Go module path (invasive, requires GitHub repo rename)

- `go.mod` — `module github.com/skittercore-studio/friendslop`
- All Go internal imports (~14 files in `internal/` and `cmd/`)
- `cmd/friendslop/main.go` — directory rename to `cmd/inkognito/`

The Go module rename is mechanical but touches every Go file. Order:
1. Rename GitHub repo `skittercore-studio/friendslop` → `inkognito` (GitHub keeps a redirect).
2. `go mod edit -module github.com/skittercore-studio/inkognito`
3. `find . -name '*.go' -exec sed -i 's|github.com/skittercore-studio/friendslop|github.com/skittercore-studio/inkognito|g' {} +`
4. `git mv cmd/friendslop cmd/inkognito`
5. `go build ./... && go test ./...`

### 2. Display strings (cheap, user-facing)

These are what players actually see:

- `frontend/index.html` — `<title>` and meta description
- `frontend/src/screens/Landing.tsx` — hero copy
- `frontend/src/screens/AnswerPhase.tsx` — possibly a stray brand mention
- `frontend/src/components/atoms/palette.ts` — comment header
- `frontend/src/styles.css`, `frontend/src/tokens.css` — comment headers
- `README.md` — top heading + first paragraph

**Abstraction opportunity:** introduce a single `BRAND` constant in the
frontend (e.g. `frontend/src/brand.ts` exporting `name = "Inkognito"`,
`tagline = "..."`, `domain = "inkognito.skittercore.studio"`). Most
display strings reduce to template literals. This is the one piece of
preemptive abstraction worth doing before lock-in — it's ~10 lines and
makes the eventual rename a 1-line change for the user-visible copy.

### 3. Storage / cache keys (one line, but breaks active sessions)

- `frontend/src/store.ts` — `STORAGE_KEY = "friendslop:active"`

If we change this naively, every active player's sessionStorage entry
becomes orphaned and they'll lose their room rejoin. Either:
- Rename + accept the one-time invalidation (it's sessionStorage, low
  blast radius).
- Read both old and new keys for a migration window.

For Inkognito launch, just rename + accept invalidation.

### 4. Infrastructure / deploy artefacts

- `Dockerfile` — image tag `friendslop:latest`
- `Makefile` — binary name `friendslop`
- `frontend/package.json` — `"name": "friendslop-frontend"`
- `frontend/package-lock.json` — derived from package.json
- skittercore-slop hostname (Hetzner) — keep as-is or rename via hcloud
- `/etc/systemd/system/friendslop.service` on the box — rename to `inkognito.service`
- DNS — add `inkognito.skittercore.studio` CNAME, leave `friendslop.skittercore.studio` as a redirect for a month then drop
- Caddyfile block on `skittercore-rp` — add new subdomain block; keep old as `redir https://inkognito.skittercore.studio{uri}`

### 5. Design assets / docs (leave as-is)

- `design/handoff-v1/*` — design source, historically named, leave for archaeology
- `design/STAGE_2_PLAN.md`, `DESIGN_BRIEF.md` — historical, mention "Friendslop" in past tense
- `docs/SPEC.md` — update if it's a living doc, leave if archived

## Recommended sequencing

1. **Now (cheap):** create `frontend/src/brand.ts` with a single `BRAND`
   constant, route 5-10 user-visible strings through it. Zero behaviour
   change; pre-positions the rename. Single commit.
2. **At lock-in:** GitHub repo rename + Go module sed (one PR), then
   frontend BRAND constant value flip + Docker/systemd rename (second
   PR), then DNS + Caddy add new domain + redirect old (third PR).
3. **+1 month:** drop the friendslop.skittercore.studio redirect.

Do NOT do step 1 speculatively yet — Toy explicitly deferred. This doc
exists so when the trigger pulls, the full surface is mapped.

## Open question

Should the GitHub repo also rename, or stay as `friendslop` for
historical commit URLs? Strong argument for renaming: the slug shows
up in `go.mod` and every clone command. Weak argument against:
permalinks to early commits. GitHub auto-redirects old URLs, so
permalinks survive. **Default: rename the repo at lock-in.**
