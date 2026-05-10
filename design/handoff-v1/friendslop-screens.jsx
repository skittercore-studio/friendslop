/* global React */
const { useState, useEffect, useRef } = React;

/* ───────────────────────────────────────────────────────────
   Shared atoms
   ─────────────────────────────────────────────────────────── */

// 12-hue palette, sticky per character
const PALETTE = [
  '#ffd60a','#ff3d8a','#00e8d4','#c4ff3a',
  '#ff7a3a','#b89cff','#4ad6ff','#e065ff',
  '#ff6b6b','#6effbe','#ffb13d','#7e9cff',
];

function StatusBar({ time = '9:41' }) {
  return (
    <>
      <div className="fs-notch"></div>
      <div className="fs-status">
        <span>{time}</span>
        <span className="right">
          <svg width="18" height="11" viewBox="0 0 18 11"><path d="M1 7 L1 9 M5 5 L5 9 M9 3 L9 9 M13 1 L13 9" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round"/></svg>
          <svg width="22" height="11" viewBox="0 0 22 11" fill="none"><rect x="1" y="1" width="18" height="9" rx="2" stroke="currentColor" strokeOpacity="0.6" strokeWidth="1"/><rect x="3" y="3" width="14" height="5" rx="1" fill="currentColor"/></svg>
        </span>
      </div>
    </>
  );
}

function HomeBar() { return <div className="fs-home"></div>; }

function TimerRing({ remaining = 92, total = 120, size = 64, urgent = false }) {
  const r = (size - 8) / 2;
  const c = 2 * Math.PI * r;
  const pct = Math.max(0, Math.min(1, remaining / total));
  const dash = c * pct;
  const m = Math.floor(remaining / 60);
  const s = String(remaining % 60).padStart(2, '0');
  const colour = urgent ? 'var(--fs-live)' : 'var(--fs-accent)';
  return (
    <div className="fs-ring" style={{ width: size, height: size }}>
      <svg width={size} height={size}>
        <circle cx={size/2} cy={size/2} r={r} stroke="var(--fs-line)" strokeWidth="3" fill="none"/>
        <circle cx={size/2} cy={size/2} r={r} stroke={colour} strokeWidth="3" fill="none"
          strokeDasharray={`${dash} ${c}`} strokeLinecap="round"
          style={{ transition: 'stroke-dasharray 0.4s linear, stroke 0.3s' }}/>
      </svg>
      <span className="fs-ring__time" style={{ color: urgent ? 'var(--fs-live)' : 'var(--fs-fg)' }}>
        {m}:{s}
      </span>
    </div>
  );
}

function CharCard({ name, glyph, desc, accent, tilt = 0, mini = false }) {
  return (
    <div className="fs-charcard"
         style={{ '--char-accent': accent, transform: `rotate(${tilt}deg)`, padding: mini ? 10 : 14 }}>
      <div className="fs-charcard__name">
        {glyph && <span className="glyph" style={{ marginRight: 8 }}>{glyph}</span>}
        {name}
      </div>
      {desc && <div className="fs-charcard__desc">{desc}</div>}
    </div>
  );
}

/* ───────────────────────────────────────────────────────────
   01 — LOBBY (code-hero ring)
   ─────────────────────────────────────────────────────────── */
function LobbyScreen() {
  const players = [
    { id: 'T', name: 'Toy',   accent: PALETTE[0], host: true },
    { id: 'M', name: 'Mira',  accent: PALETTE[1] },
    { id: 'A', name: 'Andie', accent: PALETTE[2] },
    { id: 'J', name: 'Jules', accent: PALETTE[3] },
  ];
  const ringR = 100;
  const slots = 8;
  return (
    <div className="fs-phone">
      <StatusBar/>
      <div className="fs-screen" style={{ padding: '10px 24px 0' }}>
        {/* top */}
        <div className="row between" style={{ paddingTop: 4 }}>
          <span className="tiny">room</span>
          <span className="fs-chip">friendslop.skittercore.studio</span>
        </div>
        <div className="fs-display tac" style={{ fontSize: 78, marginTop: 10, letterSpacing: '0.12em' }}>DDWR</div>
        <div className="lbl tac" style={{ marginTop: -2 }}>4-letter code · share with friends</div>

        {/* ring */}
        <div className="grow" style={{ position: 'relative', marginTop: 22 }}>
          {/* dashed ring */}
          <div style={{
            position: 'absolute', left: '50%', top: '50%',
            transform: 'translate(-50%,-50%)',
            width: ringR * 2, height: ringR * 2,
            border: '1.5px dashed var(--fs-line)',
            borderRadius: '50%',
          }}/>
          {/* center counter */}
          <div style={{
            position: 'absolute', left: '50%', top: '50%',
            transform: 'translate(-50%,-50%)',
            textAlign: 'center', width: 130,
          }}>
            <div className="fs-display" style={{ fontSize: 56 }}>{players.length}<span style={{ color: 'var(--fs-fg-faint)' }}>/{slots}</span></div>
            <div className="tiny" style={{ marginTop: 2 }}>in the room</div>
          </div>
          {/* slot avatars around ring */}
          {Array.from({ length: slots }).map((_, i) => {
            const angle = (i / slots) * 2 * Math.PI - Math.PI / 2;
            const x = Math.cos(angle) * ringR;
            const y = Math.sin(angle) * ringR;
            const p = players[i];
            const style = {
              position: 'absolute',
              left: `calc(50% + ${x}px)`,
              top:  `calc(50% + ${y}px)`,
              transform: 'translate(-50%, -50%)',
            };
            return (
              <div key={i} style={style}>
                {p ? (
                  <div className="col center" style={{ gap: 2 }}>
                    <div className={`fs-av fs-av--lg ${p.host ? 'fs-anim-pop' : 'fs-anim-pop'}`}
                         style={{ '--char-accent': p.accent, background: p.accent, color: '#1a1300', borderColor: 'transparent' }}>
                      {p.id}
                    </div>
                    <div className="tiny" style={{ fontSize: 10, marginTop: 2 }}>
                      {p.name}{p.host ? ' ★' : ''}
                    </div>
                  </div>
                ) : (
                  <div className="fs-av fs-av--lg fs-av--empty">?</div>
                )}
              </div>
            );
          })}
        </div>

        {/* mode + cta */}
        <div className="row between" style={{ marginBottom: 8 }}>
          <span className="lbl">mode</span>
          <div className="row" style={{ gap: 6 }}>
            <span className="fs-chip fs-chip--on">CURATED</span>
            <span className="fs-chip">PLAYER-WRITTEN</span>
          </div>
        </div>
        <button className="fs-btn fs-btn--primary" style={{ width: '100%' }}>START THE SLOP →</button>
        <div className="lbl tac" style={{ marginTop: 6 }}>host only · ≥4 players to begin</div>
      </div>
      <HomeBar/>
    </div>
  );
}

/* ───────────────────────────────────────────────────────────
   02 — CHARCREATE  (form ↑, live preview ↓)
   ─────────────────────────────────────────────────────────── */
function CharcreateScreen() {
  return (
    <div className="fs-phone">
      <StatusBar/>
      <div className="fs-screen" style={{ padding: '10px 22px 0' }}>
        <div className="row between" style={{ paddingTop: 4 }}>
          <span className="fs-display" style={{ fontSize: 26 }}>write a character</span>
          <span className="fs-chip fs-chip--on">3 / 5</span>
        </div>
        <div className="lbl" style={{ marginTop: 2 }}>everyone writes one. the others get assigned each other's, in secret.</div>

        {/* name field */}
        <label className="tiny" style={{ marginTop: 16 }}>name</label>
        <div className="fs-card" style={{ padding: '12px 14px', marginTop: 4 }}>
          <span style={{ fontFamily: "'Lilita One', sans-serif", fontSize: 22 }}>Sir Reginald the Unwell</span>
        </div>

        {/* description */}
        <label className="tiny" style={{ marginTop: 12 }}>description</label>
        <div className="fs-card" style={{ padding: '12px 14px', marginTop: 4, minHeight: 96, fontSize: 14, lineHeight: 1.45, color: 'var(--fs-fg)' }}>
          A Victorian hypochondriac who blames everyone within earshot for his many bizarre ailments. Refuses to remove his cravat. Suspects the cat.
          <span style={{ display: 'inline-block', width: 1, height: 14, background: 'var(--fs-accent)', marginLeft: 2, verticalAlign: 'middle', animation: 'fs-pulse-pink 1s steps(2) infinite' }}/>
        </div>
        <div className="row between" style={{ marginTop: 4 }}>
          <span className="tiny">tone tags <span className="faint">(optional)</span></span>
          <span className="tiny faint">128 / 240</span>
        </div>
        <div className="row" style={{ gap: 6, marginTop: 6, flexWrap: 'wrap' }}>
          <span className="fs-chip fs-chip--on">PETTY</span>
          <span className="fs-chip">VICTORIAN</span>
          <span className="fs-chip" style={{ borderStyle: 'dashed' }}>+ add</span>
        </div>

        {/* live preview */}
        <div className="row between" style={{ marginTop: 18 }}>
          <span className="tiny">↓ live preview · this is what others will see</span>
        </div>
        <div style={{ marginTop: 8 }}>
          <CharCard name="Sir Reginald" glyph="◈"
                    desc="A Victorian hypochondriac who blames everyone for his many bizarre ailments…"
                    accent={PALETTE[5]} tilt={-0.6}/>
        </div>

        <div className="grow"/>

        <button className="fs-btn fs-btn--primary" style={{ width: '100%' }}>SUBMIT TO POOL →</button>
        <div className="row center" style={{ gap: 8, marginTop: 6 }}>
          <span className="lbl">waiting on</span>
          <span className="fs-av" style={{ width: 22, height: 22, fontSize: 12 }}>M</span>
          <span className="fs-av" style={{ width: 22, height: 22, fontSize: 12 }}>J</span>
        </div>
      </div>
      <HomeBar/>
    </div>
  );
}

/* ───────────────────────────────────────────────────────────
   03 — ANSWERING (stage + ring)
   ─────────────────────────────────────────────────────────── */
function AnsweringScreen() {
  return (
    <div className="fs-phone">
      <StatusBar/>
      <div className="fs-screen" style={{ padding: '10px 20px 0' }}>
        {/* top: timer + meta */}
        <div className="row between" style={{ paddingTop: 4, alignItems: 'flex-start' }}>
          <TimerRing remaining={92} total={120} size={68}/>
          <div className="col" style={{ flex: 1, marginLeft: 12, gap: 2 }}>
            <span className="tiny">round 3 · question 3</span>
            <div className="row" style={{ gap: 4, marginTop: 2 }}>
              {['T','M','A','J'].map((id, i) => (
                <span key={id} className="fs-av" style={{
                  width: 22, height: 22, fontSize: 11,
                  background: i < 2 ? PALETTE[i] : 'transparent',
                  color: i < 2 ? '#1a1300' : 'var(--fs-fg-faint)',
                  borderColor: i < 2 ? 'transparent' : 'var(--fs-line)',
                  borderStyle: i < 2 ? 'solid' : 'dashed',
                }}>{id}</span>
              ))}
              <span className="tiny" style={{ marginLeft: 2 }}>2 / 4 done</span>
            </div>
          </div>
        </div>

        {/* question card */}
        <div className="fs-card" style={{
          marginTop: 14, padding: '16px 18px',
          background: 'linear-gradient(180deg, var(--fs-bg-2), var(--fs-bg-1))',
          borderColor: 'var(--fs-bg-3)',
        }}>
          <div className="tiny" style={{ color: 'var(--fs-accent)' }}>THE QUESTION</div>
          <div className="fs-display" style={{ fontSize: 26, marginTop: 6, lineHeight: 1.1 }}>
            You're stuck in an elevator with a stranger. What do you do?
          </div>
        </div>

        {/* you are */}
        <div className="tiny tac" style={{ marginTop: 16 }}>YOU ARE PLAYING</div>
        <div style={{ marginTop: 8 }}>
          <CharCard name="An Exhausted Barista" glyph="☕"
                    desc="Has worked 11 doubles. Apologises for everything. Spells your name 'Squrrl'."
                    accent={PALETTE[4]} tilt={-1}/>
        </div>

        {/* answer area */}
        <label className="tiny" style={{ marginTop: 16 }}>answer in their voice</label>
        <div className="fs-card" style={{
          marginTop: 6, padding: '12px 14px', minHeight: 110,
          fontFamily: "'JetBrains Mono', monospace", fontSize: 14, lineHeight: 1.5,
        }}>
          sorry — sorry, I — sorry. you can have my coffee, sorry it's mostly oat
          <span style={{ display:'inline-block', width:1, height:14, background:'var(--fs-accent)', marginLeft:2, verticalAlign:'middle', animation: 'fs-pulse-pink 1s steps(2) infinite' }}/>
        </div>

        <div className="grow"/>

        <button className="fs-btn fs-btn--primary" style={{ width: '100%' }}>LOCK IT IN →</button>
        <div className="lbl tac" style={{ marginTop: 4 }}>last 0:10 — ring goes pink. don't choke.</div>
      </div>
      <HomeBar/>
    </div>
  );
}

/* ───────────────────────────────────────────────────────────
   04 — REVEAL (talk-show drop 1-by-1)
   ─────────────────────────────────────────────────────────── */
function RevealScreen() {
  // 3 of 5 revealed. Most recent = WIZARD, glowing. 2 incoming.
  const revealed = [
    { name: 'A Pirate',  glyph: '☠', accent: PALETTE[6], quote: 'Yarr, I\'d be wantin\' to know their elevator opinions. Strong ones. Now.' },
    { name: 'Exhausted Barista', glyph: '☕', accent: PALETTE[4], quote: 'sorry — sorry, I — sorry. you can have my coffee, sorry it\'s mostly oat' },
    { name: 'A Wizard', glyph: '✦', accent: PALETTE[5], quote: 'I shall consult the ancient elevator-spirits. They bear bad tidings re: floor 4.', fresh: true },
  ];
  return (
    <div className="fs-phone">
      <StatusBar/>
      <div className="fs-screen" style={{ padding: '10px 20px 0' }}>
        <div className="row between" style={{ paddingTop: 4 }}>
          <span className="fs-display" style={{ fontSize: 26 }}>the answers</span>
          <span className="fs-chip">3 / 5 read</span>
        </div>
        <div className="lbl">"You're stuck in an elevator with a stranger…"</div>

        {/* stack */}
        <div className="col grow" style={{ gap: 12, marginTop: 14, overflow: 'hidden' }}>
          {revealed.map((r, i) => (
            <div key={i} className={r.fresh ? 'fs-anim-slide' : ''} style={{ animationDelay: `${i * 0.1}s`, opacity: r.fresh ? 1 : 0.78 }}>
              <div className="row" style={{ gap: 10, alignItems: 'flex-start' }}>
                <div style={{ flex: '0 0 96px' }}>
                  <CharCard name={r.name.split(' ').slice(-1)[0]} glyph={r.glyph} accent={r.accent} mini/>
                </div>
                <div className="grow">
                  <div className="tiny" style={{ marginBottom: 4 }}>
                    <span style={{ color: r.accent }}>{r.name.toUpperCase()}</span> says
                  </div>
                  <div className="fs-bubble" style={r.fresh ? { boxShadow: `0 0 0 1.5px ${r.accent}, 0 14px 40px -8px ${r.accent}66` } : {}}>
                    "{r.quote}"
                  </div>
                </div>
              </div>
            </div>
          ))}

          {/* incoming placeholder */}
          <div className="row" style={{ gap: 10, opacity: 0.45 }}>
            <div style={{ flex: '0 0 96px' }}>
              <div className="fs-charcard" style={{ '--char-accent': 'var(--fs-line)', padding: 10, borderStyle: 'dashed' }}>
                <div className="fs-charcard__name" style={{ fontSize: 14, color: 'var(--fs-fg-faint)' }}>?</div>
              </div>
            </div>
            <div className="grow col center" style={{ gap: 4, alignItems: 'flex-start' }}>
              <span className="tiny faint">incoming…</span>
              <span className="fs-chip fs-chip--live fs-pulse" style={{ alignSelf: 'flex-start' }}>● LIVE</span>
            </div>
          </div>
        </div>

        <button className="fs-btn fs-btn--disabled" style={{ width: '100%' }}>guess unlocks in 2 …</button>
      </div>
      <HomeBar/>
    </div>
  );
}

/* ───────────────────────────────────────────────────────────
   05 — GUESS GRID (corkboard)
   ─────────────────────────────────────────────────────────── */
function GridScreen() {
  // Other players' rows × rounds (R1, R2 locked; R3 live)
  const rows = [
    { name: 'Mira', cells: [
      { round: 1, char: 'PIRATE',  accent: PALETTE[6], correct: true },
      { round: 2, char: 'WIZARD',  accent: PALETTE[5], correct: false },
      { round: 3, empty: true },
    ]},
    { name: 'Andie', cells: [
      { round: 1, char: 'BARISTA', accent: PALETTE[4], correct: true },
      { round: 2, char: 'BARISTA', accent: PALETTE[4], correct: true },
      { round: 3, char: 'BARISTA', accent: PALETTE[4], staged: true },
    ]},
    { name: 'Jules', cells: [
      { round: 1, char: 'WIZARD',  accent: PALETTE[5], correct: false },
      { round: 2, char: 'PIRATE',  accent: PALETTE[6], correct: false },
      { round: 3, empty: true },
    ]},
  ];
  const remaining = [
    { char: 'PIRATE',   accent: PALETTE[6], dragging: true },
    { char: 'WIZARD',   accent: PALETTE[5] },
    { char: 'GIGACHAD', accent: PALETTE[3] },
  ];

  return (
    <div className="fs-phone">
      <StatusBar/>
      <div className="fs-screen" style={{ padding: '10px 18px 0' }}>
        <div className="row between" style={{ paddingTop: 4 }}>
          <div className="col" style={{ gap: 2 }}>
            <span className="fs-display" style={{ fontSize: 24 }}>the board</span>
            <span className="lbl">round 3 · drag to assign</span>
          </div>
          <TimerRing remaining={102} total={120} size={56}/>
        </div>

        {/* grid */}
        <div className="col" style={{ gap: 10, marginTop: 14 }}>
          {/* column header */}
          <div className="row" style={{ gap: 6, paddingLeft: 50 }}>
            <span className="tiny tac" style={{ flex: 1 }}>R1 <span className="fs-chip" style={{ padding:'1px 6px', fontSize:10, marginLeft:4 }}>2/3</span></span>
            <span className="tiny tac" style={{ flex: 1 }}>R2 <span className="fs-chip" style={{ padding:'1px 6px', fontSize:10, marginLeft:4 }}>1/3</span></span>
            <span className="tiny tac" style={{ flex: 1 }}>R3 <span className="fs-chip fs-chip--live" style={{ padding:'1px 6px', fontSize:10, marginLeft:4 }}>LIVE</span></span>
          </div>

          {rows.map((row, ri) => (
            <div key={ri} className="row" style={{ gap: 6 }}>
              <span className="lbl" style={{ width: 44, fontWeight: 600, color: 'var(--fs-fg)' }}>{row.name}</span>
              {row.cells.map((c, ci) => (
                <div key={ci} style={{ flex: 1 }}>
                  {c.empty ? (
                    <div style={{
                      borderRadius: 6, border: '1.5px dashed var(--fs-line)',
                      height: 56, display: 'grid', placeItems: 'center',
                      background: 'rgba(255,214,10,0.04)',
                    }}>
                      <span className="tiny faint" style={{ fontSize: 10 }}>tap to pin</span>
                    </div>
                  ) : (
                    <div className="fs-idx" style={{
                      transform: `rotate(${(ri + ci) % 2 === 0 ? -1.4 : 1.2}deg)`,
                      height: 56,
                      ...(c.staged ? { boxShadow: `0 0 0 2px ${c.accent}, 0 8px 22px -4px rgba(0,0,0,0.5)` } : {}),
                    }}>
                      {!c.staged && <div className="fs-pin" style={{ background: c.correct ? 'var(--c-lime)' : 'var(--fs-fg-faint)' }}/>}
                      {c.staged && <div className="fs-pin"/>}
                      <div style={{
                        fontSize: 10, fontWeight: 700, letterSpacing: '0.06em',
                        color: c.accent, textTransform: 'uppercase',
                      }}>R{c.round}{c.correct === true ? ' ✓' : c.correct === false ? ' ✗' : ''}</div>
                      <div style={{ fontFamily: "'Lilita One', sans-serif", fontSize: 14, marginTop: 14, color: '#2a2410' }}>
                        {c.char}
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          ))}
        </div>

        <div className="grow"/>

        {/* tray */}
        <div className="fs-card" style={{
          padding: 12, background: 'rgba(255,214,10,0.06)',
          borderColor: 'var(--fs-line)', borderStyle: 'dashed', marginBottom: 10,
        }}>
          <div className="row between">
            <span className="tiny">remaining characters</span>
            <span className="tiny faint">1:1 per round</span>
          </div>
          <div className="row" style={{ gap: 6, marginTop: 8, flexWrap: 'wrap' }}>
            {remaining.map((r, i) => (
              <div key={i} className="fs-idx" style={{
                transform: `rotate(${[-2,1,-1][i] || 0}deg)`,
                padding: '4px 10px',
                ...(r.dragging ? { boxShadow: `0 0 0 2px ${r.accent}, 0 12px 28px -4px rgba(0,0,0,0.7)`, transform: 'translate(6px,-6px) rotate(-3deg) scale(1.04)' } : {}),
              }}>
                <div style={{ fontSize: 9, color: r.accent, fontWeight: 700, letterSpacing: '0.08em' }}>·</div>
                <div style={{ fontFamily: "'Lilita One', sans-serif", fontSize: 13, marginTop: 0 }}>{r.char}</div>
              </div>
            ))}
          </div>
        </div>

        <button className="fs-btn fs-btn--primary" style={{ width: '100%' }}>LOCK GUESSES →</button>
      </div>
      <HomeBar/>
    </div>
  );
}

/* ───────────────────────────────────────────────────────────
   Export to window so the canvas script can mount them
   ─────────────────────────────────────────────────────────── */
Object.assign(window, {
  LobbyScreen, CharcreateScreen, AnsweringScreen, RevealScreen, GridScreen,
});
