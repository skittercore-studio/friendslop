# Friendslop — Monetisation Analysis

A clear-eyed read on whether Friendslop (working title, likely Inkognito) can be
monetised, written for a one-developer studio with no marketing budget and no
appetite for self-promotion.

## 1. Honest read on the revenue ceiling

Friend-group party games span four orders of magnitude. The Jackbox ceiling
is irrelevant — Jackbox sits on a decade of brand, a TV-attached host screen,
and dozens of minigames. Among Us is off the table: it caught a streamer wave
that no longer exists for a text game. Realistic reference points are Werewords,
small Jackbox-likes, and indie web party games like Skribbl.io or Gartic Phone.
The web ones make essentially no money (beloved, ad-supported, low ARPU). The
Steam ones depend on streamer pickup Friendslop will not get.

Compounding factors:

- **Text-only.** No streamer-friendly visual hook. No clip-bait without
  someone reading answers aloud.
- **Friends-required.** Not a game for strangers. Zero K-factor from random
  matchmaking — grows only when friend groups invite friend groups.
- **Voice-call companion required.** A meta-game wrapped around a Discord
  call. Gates the audience hard.
- **No UA budget, no marketing, no Play Store presence.**

Realistic upper bound, calibrated against Skittercore's existing portfolio
("moderately successful in the way Orb Unravel is moderately successful"):
**£0-500/year, lifetime, plausibly £0**. A surprise viral moment could push
that to four figures across a single quarter, but it is not a base case to
build a business model around. Friendslop is not a revenue product. Treating
it as one will produce decisions that damage the game.

## 2. Models ranked by effort:fit ratio

Ranked best fit first.

### Tip jar / "buy the dev a coffee" — **best fit**
Implementation: a Ko-fi or Stripe Payment Link in the footer and on the
endgame screen. Zero backend work. No accounts, no entitlements, no Play
Billing. Friction: zero. Conversion: ~0.1-0.5% of *enjoying* groups, of which
maybe 50/year exist. Expected revenue: £20-200/year. Effort: an afternoon.
**The ratio here is unbeatable.**

### Pay-once host unlock (Jackbox model) — **viable, but niche-gated**
Stripe Checkout, one-shot £3-5, host token flips a flag on the host's session
cookie or a row in sqlite. Guests free. Conversion: maybe 5-10% of *repeat*
host groups, of which there will be very few until distribution exists.
Expected revenue: £50-500/year. Effort: 2-4 days. The catch: only works if
free games are capped (e.g. 3 free, then host pays). Capping a still-fragile,
unknown game kills discovery before it starts.

### Cosmetic-only — **wrong shape for this game**
Custom room codes, character template flair, profile colours. Friendslop has
no profile system and no persistent identity by design (cookie-based
ephemeral sessions only). Adding accounts purely to support cosmetics is a
huge architectural delta for a vanity payoff. Skip.

### Premium themed question packs (DLC-ish one-offs) — **plausible if content moat is real**
"Spicy pack", "office party pack", "horror movie pack". £2 each, host owns,
guests inherit per room. Implementation lift: medium — entitlement table,
question bank tagging (already partially there via the JSON), a minimal store
UI. Conversion: maybe 3-5% of repeat host groups. Worth doing *only* if the
content moat (section 4) is real. **Conditional yes.**

### Subscription content packs — **wrong shape, do not build**
Subs require persistent identity, monthly retention, and a content treadmill.
A one-developer studio with no marketing cannot fill the treadmill, and the
audience for a niche text party game is the worst possible substrate for
recurring billing. Skip.

### Patreon "supporter" community with roadmap influence — **maybe, if community materialises**
Zero implementation cost (it lives entirely on Patreon). Only works if there
*is* a community, which is not currently the case. Park this — revisit in
12 months if the game has organic word-of-mouth.

### Ads (interstitial between rounds) — **do not ship**
The whole point of Friendslop is the social rhythm between answer-reveal and
guess-grid. An interstitial mid-round will torch the experience. AdMob
NO_FILL on BazaarBoss is also a real signal: ads at this scale don't pay
enough to justify the UX cost. Hard no.

## 3. The "host pays" model on this stack

Mechanically clean. Stripe Checkout one-shot is half a day of work:

1. Host clicks "Create Premium Room" → server creates a pending room id,
   redirects to Stripe Checkout.
2. Webhook flips `rooms.paid_unlock = 1` in sqlite, returns the room code.
3. Host's session cookie carries the unlock for that room only.

No Play Billing required: the web app is the canonical surface. Play Store
presence would force Play Billing (15-30% cut + review surface) for in-app
purchase. Stay on web, ship Stripe. The Go + sqlite + SSE stack is ideal —
single binary, single file, webhook is one chi route.

**But:** host-pays only generates revenue with distribution. There is none.
Build the *technical capability* in a day; do not gate the game on it.

## 4. Content moats

Honest answer: **no, not really.** The current question bank is ~60 prompts.
The DESIGN_BRIEF prizes cross-character prompts ("morning routine", "worst
Monday") — exactly the prompts a friend group can generate in 90 seconds on a
whiteboard. Player-written characters are already first-class in the design;
the game already encodes that *players bring the content*.

Themed packs with genuine craft (horror, workplace, sci-fi) could be a moat
*only* if writing quality is markedly above casual user output. Toy can write
that, but a 1,460-prompt year-pool is months of work. A 50-prompt pack at £2
sold to 20 hosts/year is £40. Economics don't support paid authoring at this
volume. Better play: keep the bank open-source, accept PRs, let the community
moat itself.

## 5. Hard truth verdict

**Do not monetise Friendslop as a revenue product. Treat it as a portfolio
and community piece that drives traffic to the paid Skittercore games.**

The studio's scarce resource is Toy's time, not money. Every hour on
entitlement plumbing, Stripe webhooks, refund flows, and content packs is an
hour not spent on Sigil or Word-Mashup payments — both of which sit on better
revenue substrates (mobile, Play Store, solo play, no friend-coordination
bottleneck). Friendslop's job is to be the *good* game in the portfolio —
remembered, linked, used as a reason to look at the studio. That is worth
more on this roadmap than the £200/year a host-pay model would produce.

One exception: tip jar + footer link to the rest of Skittercore. An afternoon
of work, compounds forever, costs the game nothing.

## Recommended next concrete step (next 1-2 weeks)

Ship a **tip jar + cross-promo footer** on the endgame screen. Concretely:

1. Add a Ko-fi or Stripe Payment Link (no webhook, no sqlite changes, no
   accounts) — 30 minutes.
2. On the endgame / win screen, after the dramatic reveal, add a small,
   unobtrusive panel: "Liked Friendslop? Skittercore made these too →" with
   links to Orb Unravel, BazaarBoss, Word-Mashup. Use the moment of maximum
   goodwill (the laugh after the win) to surface the rest of the catalogue.
3. Add basic privacy-respecting telemetry — just a count of completed games
   per week, no per-player tracking. This is the single missing input you
   need to know whether *any* monetisation question is worth re-asking in
   six months.

If after three months that telemetry shows >50 completed games per week
without any marketing push, revisit host-pay. Until then, the right call is
to ship the game, link the catalogue, and move on.
