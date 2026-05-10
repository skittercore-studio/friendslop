/* ───────────────────────────────────────────────────────────
   FRIENDSLOP — host (desktop / TV) screens, 1280×720
   Read-only spectator surfaces. No interactive controls.
   ─────────────────────────────────────────────────────────── */

/* shared palette (must match phone screens) */
const HOST_PALETTE = window.PALETTE || [
  '#ffd60a','#ff3d8a','#00e8d4','#c4ff3a',
  '#ff7a3a','#b89cff','#4ad6ff','#e065ff',
  '#ff6b6b','#6effbe','#ffb13d','#7e9cff',
];

function HostShell({ children, badge }) {
  return (
    <div style={{
      width: 1280, height: 720,
      background: 'radial-gradient(120% 80% at 50% 0%, rgba(126,156,255,0.10), transparent 60%), radial-gradient(120% 80% at 50% 100%, rgba(255,61,138,0.10), transparent 60%), #0c0a18',
      color: '#f6f3ff',
      fontFamily: "'Space Grotesk', system-ui, sans-serif",
      position: 'relative',
      overflow: 'hidden',
      borderRadius: 14,
      border: '1px solid #2a2654',
      boxShadow: '0 30px 80px -30px rgba(0,0,0,0.7)',
    }}>
      {/* subtle scanline noise */}
      <div style={{
        position:'absolute', inset:0, pointerEvents:'none', opacity:0.04,
        backgroundImage: 'repeating-linear-gradient(0deg, #fff 0 1px, transparent 1px 3px)',
      }}/>
      {/* top bar */}
      <div style={{
        position:'absolute', top:0, left:0, right:0, height:48,
        display:'flex', alignItems:'center', justifyContent:'space-between',
        padding: '0 32px', borderBottom: '1px solid #211e44',
        fontSize: 13, letterSpacing: '0.18em', textTransform: 'uppercase',
        color: '#a5a0d0', zIndex: 2,
      }}>
        <span style={{ fontFamily: "'Lilita One', sans-serif", fontSize: 22, letterSpacing: '0.06em', color: '#f6f3ff' }}>
          friend<span style={{ color: '#ffd60a' }}>slop</span>
        </span>
        <span>room <span style={{ color:'#f6f3ff', fontFamily:"'JetBrains Mono', monospace", fontWeight:700, letterSpacing:'0.3em', marginLeft:6 }}>DDWR</span></span>
        {badge}
      </div>
      <div style={{ position:'absolute', inset: '48px 0 0', display:'flex' }}>
        {children}
      </div>
    </div>
  );
}

/* ───────────────────────────────────────────────────────────
   01 — LOBBY  (host)
   ─────────────────────────────────────────────────────────── */
function LobbyHostScreen() {
  const players = [
    { id: 'T', name: 'Toy',   accent: HOST_PALETTE[0], host: true },
    { id: 'M', name: 'Mira',  accent: HOST_PALETTE[1] },
    { id: 'A', name: 'Andie', accent: HOST_PALETTE[2] },
    { id: 'J', name: 'Jules', accent: HOST_PALETTE[3] },
  ];
  return (
    <HostShell badge={<span>lobby · waiting on host</span>}>
      <div style={{ flex: 1.05, display:'flex', flexDirection:'column', justifyContent:'center', padding:'0 56px', borderRight:'1px solid #211e44' }}>
        <div className="tiny" style={{ color:'#a5a0d0' }}>JOIN AT</div>
        <div style={{ fontFamily:"'JetBrains Mono', monospace", fontSize:20, marginTop:6, color:'#f6f3ff' }}>friendslop.skittercore.studio</div>
        <div className="tiny" style={{ color:'#a5a0d0', marginTop:32 }}>OR DROP THE CODE</div>
        <div style={{
          fontFamily:"'Lilita One', sans-serif",
          fontSize: 220,
          letterSpacing: '0.12em',
          lineHeight: 0.9,
          marginTop: 4,
          background: 'linear-gradient(180deg, #ffd60a 0%, #ff7a3a 90%)',
          WebkitBackgroundClip: 'text', backgroundClip: 'text',
          color: 'transparent',
          textShadow: '0 0 60px rgba(255,214,10,0.25)',
        }}>DDWR</div>
        <div style={{ fontSize: 18, color:'#a5a0d0', marginTop: 18 }}>
          4 letters · case insensitive · expires in 2h
        </div>
      </div>
      <div style={{ flex: 1, display:'flex', flexDirection:'column', padding:'40px 56px', gap: 18 }}>
        <div style={{ display:'flex', alignItems:'baseline', justifyContent:'space-between' }}>
          <div style={{ fontFamily:"'Lilita One', sans-serif", fontSize:36 }}>in the room</div>
          <div style={{ fontFamily:"'Lilita One', sans-serif", fontSize:36, color:'#ffd60a' }}>{players.length}<span style={{ color:'#6b6798' }}>/8</span></div>
        </div>
        <div style={{ display:'grid', gridTemplateColumns:'1fr 1fr', gap: 12 }}>
          {Array.from({ length: 8 }).map((_, i) => {
            const p = players[i];
            return p ? (
              <div key={i} style={{
                display:'flex', alignItems:'center', gap: 14,
                padding: '14px 16px',
                background: 'linear-gradient(180deg, #1d1a3a, #14112a)',
                border: '1px solid #322e60',
                borderRadius: 14,
                boxShadow: `inset 0 0 0 1px ${p.accent}22, 0 8px 24px -12px ${p.accent}55`,
              }}>
                <div style={{
                  width:48, height:48, borderRadius:'50%',
                  background: p.accent, color:'#1a1300',
                  display:'grid', placeItems:'center',
                  fontFamily:"'Lilita One', sans-serif", fontSize: 24,
                }}>{p.id}</div>
                <div>
                  <div style={{ fontFamily:"'Lilita One', sans-serif", fontSize: 22 }}>{p.name}</div>
                  {p.host && <div className="tiny" style={{ color: p.accent }}>HOST ★</div>}
                </div>
              </div>
            ) : (
              <div key={i} style={{
                display:'grid', placeItems:'center',
                padding: '14px 16px', minHeight: 76,
                border: '1.5px dashed #322e60',
                borderRadius: 14,
                color:'#6b6798',
                fontSize: 14,
              }}>empty seat</div>
            );
          })}
        </div>
        <div style={{ marginTop:'auto', textAlign:'center', color:'#a5a0d0', fontSize: 14 }}>
          host taps START on their phone when ready · ≥4 needed
        </div>
      </div>
    </HostShell>
  );
}

/* ───────────────────────────────────────────────────────────
   02 — CHARCREATE (host)
   ─────────────────────────────────────────────────────────── */
function CharcreateHostScreen() {
  const players = [
    { id:'T', name:'Toy',   accent:HOST_PALETTE[0], submitted:true,  preview:'Sir Reginald the Unwell' },
    { id:'M', name:'Mira',  accent:HOST_PALETTE[1], submitted:true,  preview:'Coke (the cow)' },
    { id:'A', name:'Andie', accent:HOST_PALETTE[2], submitted:true,  preview:'Brad, who lifts' },
    { id:'J', name:'Jules', accent:HOST_PALETTE[3], submitted:false },
    { id:'X', name:'Sky',   accent:HOST_PALETTE[6], submitted:false },
  ];
  const submitted = players.filter(p => p.submitted).length;
  return (
    <HostShell badge={<span>character creation · {submitted} / {players.length} in the pool</span>}>
      <div style={{ flex: 1, padding: '48px 64px', display:'flex', flexDirection:'column', gap: 22 }}>
        <div style={{ display:'flex', alignItems:'baseline', justifyContent:'space-between' }}>
          <div>
            <div style={{ fontFamily:"'Lilita One', sans-serif", fontSize: 76, lineHeight:0.95 }}>
              everyone's <span style={{ color:'#ffd60a' }}>writing</span>…
            </div>
            <div style={{ color:'#a5a0d0', fontSize: 18, marginTop: 8 }}>
              each player invents one character. you'll be assigned each other's, in secret.
            </div>
          </div>
          <div style={{
            fontFamily:"'Lilita One', sans-serif",
            fontSize: 96, lineHeight: 0.9, color:'#ffd60a',
          }}>
            {submitted}<span style={{ color:'#6b6798' }}>/{players.length}</span>
          </div>
        </div>

        {/* progress dots */}
        <div style={{ display:'flex', gap: 6, height: 8 }}>
          {players.map(p => (
            <div key={p.id} style={{
              flex: 1, borderRadius: 4,
              background: p.submitted ? p.accent : '#211e44',
              boxShadow: p.submitted ? `0 0 16px ${p.accent}99` : 'none',
              transition: 'background 0.3s',
            }}/>
          ))}
        </div>

        {/* pool cards */}
        <div style={{ display:'grid', gridTemplateColumns:'repeat(3, 1fr)', gap: 18, marginTop: 8 }}>
          {players.map((p, i) => (
            <div key={p.id} style={{
              padding: 20,
              background: p.submitted ? 'linear-gradient(180deg, #1d1a3a, #14112a)' : 'transparent',
              border: p.submitted ? '1px solid #322e60' : '1.5px dashed #322e60',
              borderRadius: 18,
              minHeight: 200,
              position: 'relative',
              overflow: 'hidden',
              transform: p.submitted ? `rotate(${[-1.2, 0.8, -0.6, 1.4, -0.8][i] || 0}deg)` : 'none',
              boxShadow: p.submitted ? `inset 0 0 0 1px ${p.accent}33, 0 12px 30px -12px ${p.accent}55` : 'none',
            }}>
              {p.submitted && (
                <div style={{ position:'absolute', top:0, left:0, right:0, height: 6, background: p.accent, boxShadow:`0 0 24px ${p.accent}` }}/>
              )}
              <div style={{ display:'flex', alignItems:'center', gap: 10, marginBottom: 14 }}>
                <div style={{
                  width:32, height:32, borderRadius:'50%',
                  background: p.submitted ? p.accent : 'transparent',
                  border: p.submitted ? 'none' : '1.5px dashed #322e60',
                  color: p.submitted ? '#1a1300' : '#6b6798',
                  display:'grid', placeItems:'center',
                  fontFamily:"'Lilita One', sans-serif", fontSize: 16,
                }}>{p.id}</div>
                <span style={{ fontFamily:"'Lilita One', sans-serif", fontSize: 20, color: p.submitted ? '#f6f3ff' : '#6b6798' }}>{p.name}</span>
                {p.submitted ? (
                  <span className="tiny" style={{ color: p.accent, marginLeft:'auto' }}>SUBMITTED ✓</span>
                ) : (
                  <span className="tiny" style={{ color:'#6b6798', marginLeft:'auto' }}>WRITING…</span>
                )}
              </div>
              {p.submitted ? (
                <div>
                  <div style={{ fontFamily:"'Lilita One', sans-serif", fontSize: 28, lineHeight: 1.05 }}>{p.preview}</div>
                  <div style={{ color:'#a5a0d0', fontSize: 13, marginTop: 8, fontStyle:'italic' }}>(description hidden — only the assigned player sees it)</div>
                </div>
              ) : (
                <div>
                  <div style={{
                    height: 14, borderRadius: 6, background: '#211e44',
                    width: '78%', marginBottom: 8,
                  }}/>
                  <div style={{
                    height: 14, borderRadius: 6, background: '#211e44',
                    width: '52%',
                  }}/>
                  <div style={{ marginTop: 14 }}>
                    <span style={{
                      display:'inline-block', width:6, height:14, background:'#ff3d8a',
                      animation:'fs-pulse-pink 1s steps(2) infinite', verticalAlign:'middle',
                    }}/>
                    <span className="tiny" style={{ color:'#6b6798', marginLeft: 6 }}>typing…</span>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </HostShell>
  );
}

/* ───────────────────────────────────────────────────────────
   03 — ANSWERING (host) — pressure cooker
   ─────────────────────────────────────────────────────────── */
function AnsweringHostScreen() {
  const players = [
    { id:'T', name:'Toy',   accent:HOST_PALETTE[0], state:'done' },
    { id:'M', name:'Mira',  accent:HOST_PALETTE[1], state:'done' },
    { id:'A', name:'Andie', accent:HOST_PALETTE[2], state:'typing' },
    { id:'J', name:'Jules', accent:HOST_PALETTE[3], state:'typing' },
    { id:'X', name:'Sky',   accent:HOST_PALETTE[6], state:'idle' },
  ];
  // big timer ring
  const T = 120, R = 92, size = 200, r = (size-12)/2, c = 2*Math.PI*r;
  const dash = c * (R/T);
  return (
    <HostShell badge={<span>round 3 · question 3 of ?</span>}>
      <div style={{ flex: 1, padding: '40px 64px', display:'flex', flexDirection:'column' }}>
        {/* the question */}
        <div className="tiny" style={{ color:'#ffd60a' }}>THE QUESTION</div>
        <div style={{
          fontFamily:"'Lilita One', sans-serif",
          fontSize: 84, lineHeight: 1.02, marginTop: 6,
          textWrap: 'pretty',
        }}>
          You're stuck in an elevator with a stranger.<br/>
          <span style={{ color:'#ffd60a' }}>What do you do?</span>
        </div>
        <div style={{ color:'#a5a0d0', fontSize: 18, marginTop: 14 }}>
          phones only · everyone's writing in their secret character's voice
        </div>

        {/* bottom row — timer + status */}
        <div style={{ marginTop:'auto', display:'flex', alignItems:'center', gap: 32 }}>
          <div style={{ position:'relative', width:size, height:size }}>
            <svg width={size} height={size} style={{ transform:'rotate(-90deg)' }}>
              <circle cx={size/2} cy={size/2} r={r} stroke="#211e44" strokeWidth="6" fill="none"/>
              <circle cx={size/2} cy={size/2} r={r} stroke="#ffd60a" strokeWidth="6" fill="none"
                strokeDasharray={`${dash} ${c}`} strokeLinecap="round"/>
            </svg>
            <div style={{
              position:'absolute', inset:0, display:'grid', placeItems:'center',
              fontFamily:"'JetBrains Mono', monospace", fontSize: 56, fontWeight:700,
            }}>1:32</div>
          </div>
          <div style={{ flex:1, display:'flex', flexDirection:'column', gap: 10 }}>
            <div className="tiny" style={{ color:'#a5a0d0' }}>WRITING STATUS</div>
            <div style={{ display:'flex', gap: 12, flexWrap:'wrap' }}>
              {players.map(p => {
                const fill = p.state === 'done' ? p.accent : 'transparent';
                const fg   = p.state === 'done' ? '#1a1300' : p.accent;
                const border = p.state === 'idle' ? '1.5px dashed #322e60' : `2px solid ${p.accent}`;
                return (
                  <div key={p.id} style={{
                    display:'flex', alignItems:'center', gap: 10,
                    padding: '10px 16px', borderRadius: 999,
                    background: fill === 'transparent' ? '#14112a' : fill,
                    border, color: fg,
                    fontFamily:"'Lilita One', sans-serif", fontSize: 22,
                    boxShadow: p.state === 'done' ? `0 0 24px -4px ${p.accent}aa` : 'none',
                  }}>
                    <span>{p.name}</span>
                    {p.state === 'done' && <span style={{ fontSize: 16 }}>✓</span>}
                    {p.state === 'typing' && (
                      <span style={{
                        display:'inline-block', width:8, height:14, background:p.accent,
                        animation:'fs-pulse-pink 1s steps(2) infinite',
                      }}/>
                    )}
                    {p.state === 'idle' && <span style={{ fontSize: 13, color:'#6b6798' }}>·</span>}
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>
    </HostShell>
  );
}

/* ───────────────────────────────────────────────────────────
   04 — REVEAL (host) — 5-up wall
   ─────────────────────────────────────────────────────────── */
function RevealHostScreen() {
  const answers = [
    { name:'PIRATE',   glyph:'☠', accent:HOST_PALETTE[6], quote:'Yarr, I\'d be wantin\' their elevator opinions. Strong ones. Now.', state:'read' },
    { name:'BARISTA',  glyph:'☕', accent:HOST_PALETTE[4], quote:'sorry — sorry, I — sorry. you can have my coffee, sorry it\'s mostly oat', state:'read' },
    { name:'WIZARD',   glyph:'✦', accent:HOST_PALETTE[5], quote:'I shall consult the ancient elevator-spirits. They bear bad tidings re: floor 4.', state:'fresh' },
    { name:'GIGACHAD', glyph:'⚡', accent:HOST_PALETTE[3], state:'pending' },
    { name:'???',                  accent:'#322e60',    state:'pending' },
  ];
  return (
    <HostShell badge={<span>reveal · 3 of 5 read</span>}>
      <div style={{ flex:1, padding:'32px 56px', display:'flex', flexDirection:'column' }}>
        <div style={{ fontFamily:"'Lilita One', sans-serif", fontSize: 44, lineHeight: 1 }}>
          <span style={{ color:'#a5a0d0', fontSize: 22, letterSpacing:'0.18em', textTransform:'uppercase', display:'block', marginBottom: 6 }}>they said…</span>
          "You're stuck in an elevator with a stranger. What do you do?"
        </div>

        <div style={{
          flex:1, display:'grid',
          gridTemplateColumns:'repeat(5, 1fr)', gap: 18,
          marginTop: 24,
        }}>
          {answers.map((a, i) => {
            const isPending = a.state === 'pending';
            const isFresh   = a.state === 'fresh';
            return (
              <div key={i} style={{
                display:'flex', flexDirection:'column', gap: 10,
                opacity: isPending ? 0.35 : 1,
                transform: isFresh ? 'translateY(-6px)' : 'none',
                transition: 'transform 0.3s',
              }}>
                {/* big char header card */}
                <div style={{
                  background: 'linear-gradient(180deg, #1d1a3a, #14112a)',
                  border: `1px solid #322e60`,
                  borderRadius: 16,
                  padding: '18px 14px',
                  textAlign:'center',
                  position:'relative', overflow:'hidden',
                  boxShadow: isFresh ? `inset 0 0 0 2px ${a.accent}, 0 16px 40px -8px ${a.accent}aa` : 'none',
                }}>
                  <div style={{ position:'absolute', top:0, left:0, right:0, height:6, background: a.accent, boxShadow: `0 0 24px ${a.accent}` }}/>
                  <div style={{ fontSize: 40, color: a.accent, marginTop: 4 }}>{a.glyph}</div>
                  <div style={{ fontFamily:"'Lilita One', sans-serif", fontSize: 24, marginTop: 6, letterSpacing:'0.04em' }}>{a.name}</div>
                </div>

                {/* quote */}
                {a.quote ? (
                  <div style={{
                    flex: 1,
                    background: '#14112a',
                    border: `1px solid ${isFresh ? a.accent : '#322e60'}`,
                    borderRadius: 16,
                    padding: '14px 16px',
                    fontFamily:"'JetBrains Mono', monospace",
                    fontSize: 14, lineHeight: 1.5,
                    color: '#f6f3ff',
                    boxShadow: isFresh ? `0 12px 30px -8px ${a.accent}66` : 'none',
                  }}>
                    "{a.quote}"
                  </div>
                ) : (
                  <div style={{
                    flex:1, display:'grid', placeItems:'center',
                    border: '1.5px dashed #322e60', borderRadius: 16,
                    color:'#6b6798', fontSize: 13, letterSpacing:'0.18em', textTransform:'uppercase',
                  }}>incoming…</div>
                )}
              </div>
            );
          })}
        </div>

        <div style={{ marginTop: 16, color:'#a5a0d0', fontSize: 14, textAlign:'center' }}>
          ★ next answer drops in <span style={{ color:'#ff3d8a' }}>1.2s</span> · phones see the same wall, compact
        </div>
      </div>
    </HostShell>
  );
}

/* ───────────────────────────────────────────────────────────
   05 — GUESS GRID (host) — public deduction wall
   ─────────────────────────────────────────────────────────── */
function GridHostScreen() {
  const players = [
    { name:'Toy',   accent:HOST_PALETTE[0], rounds:['3/3','2/3','typing'] },
    { name:'Mira',  accent:HOST_PALETTE[1], rounds:['2/3','1/3','typing'] },
    { name:'Andie', accent:HOST_PALETTE[2], rounds:['3/3','3/3','SEALED'] },
    { name:'Jules', accent:HOST_PALETTE[3], rounds:['0/3','1/3','typing'] },
  ];
  const T = 120, R = 102, size = 76, r = (size-8)/2, c = 2*Math.PI*r;
  const dash = c * (R/T);
  return (
    <HostShell badge={<span>round 3 · guessing in progress</span>}>
      <div style={{ flex:1, padding:'32px 56px', display:'flex', flexDirection:'column', gap: 22 }}>
        <div style={{ display:'flex', alignItems:'flex-start', justifyContent:'space-between' }}>
          <div>
            <div style={{ fontFamily:"'Lilita One', sans-serif", fontSize: 56, lineHeight: 1 }}>
              the <span style={{ color:'#ffd60a' }}>deduction wall</span>
            </div>
            <div style={{ color:'#a5a0d0', fontSize: 16, marginTop: 6 }}>
              past totals visible · individual guesses hidden until endgame · sealed = locked
            </div>
          </div>
          <div style={{ position:'relative', width:size, height:size }}>
            <svg width={size} height={size} style={{ transform:'rotate(-90deg)' }}>
              <circle cx={size/2} cy={size/2} r={r} stroke="#211e44" strokeWidth="4" fill="none"/>
              <circle cx={size/2} cy={size/2} r={r} stroke="#ffd60a" strokeWidth="4" fill="none"
                strokeDasharray={`${dash} ${c}`} strokeLinecap="round"/>
            </svg>
            <div style={{
              position:'absolute', inset:0, display:'grid', placeItems:'center',
              fontFamily:"'JetBrains Mono', monospace", fontSize: 18, fontWeight:700,
            }}>1:42</div>
          </div>
        </div>

        {/* matrix */}
        <div style={{
          flex:1, background: '#14112a',
          border:'1px solid #322e60', borderRadius: 18,
          padding: 20, display:'flex', flexDirection:'column',
          backgroundImage:
            'radial-gradient(circle at 18% 28%, rgba(255,255,255,0.025) 1px, transparent 2px),' +
            'radial-gradient(circle at 72% 65%, rgba(255,255,255,0.02) 1px, transparent 2px)',
          backgroundSize: '32px 32px, 41px 41px',
        }}>
          {/* header row */}
          <div style={{ display:'grid', gridTemplateColumns:'140px 1fr 1fr 1.2fr', gap: 12, marginBottom: 14 }}>
            <div></div>
            <div className="tiny" style={{ textAlign:'center', color:'#a5a0d0' }}>ROUND 1</div>
            <div className="tiny" style={{ textAlign:'center', color:'#a5a0d0' }}>ROUND 2</div>
            <div className="tiny" style={{ textAlign:'center', color:'#ff3d8a' }}>ROUND 3 · LIVE</div>
          </div>

          {players.map((p, pi) => (
            <div key={p.name} style={{
              display:'grid', gridTemplateColumns:'140px 1fr 1fr 1.2fr', gap: 12,
              alignItems:'center',
              padding: '12px 0',
              borderTop: pi === 0 ? '1px solid #211e44' : '1px solid #211e44',
            }}>
              <div style={{ display:'flex', alignItems:'center', gap: 10 }}>
                <div style={{
                  width:36, height:36, borderRadius:'50%',
                  background: p.accent, color:'#1a1300',
                  display:'grid', placeItems:'center',
                  fontFamily:"'Lilita One', sans-serif", fontSize: 18,
                }}>{p.name[0]}</div>
                <div style={{ fontFamily:"'Lilita One', sans-serif", fontSize: 22 }}>{p.name}</div>
              </div>
              {p.rounds.map((cell, ri) => {
                const isLive = ri === 2;
                const isSealed = cell === 'SEALED';
                const isTyping = cell === 'typing';
                let bg = '#1d1a3a', fg = '#f6f3ff', border = '1px solid #322e60';
                if (isLive && isSealed) { bg = p.accent; fg = '#1a1300'; border = 'none'; }
                else if (isLive && isTyping) { bg = 'rgba(255,61,138,0.08)'; border = '1px solid #ff3d8a44'; }
                let display = cell;
                if (isTyping) display = '⌨ writing…';
                return (
                  <div key={ri} style={{
                    background: bg, color: fg, border, borderRadius: 12,
                    padding: '14px 18px', textAlign:'center',
                    fontFamily:"'Lilita One', sans-serif", fontSize: 22, letterSpacing: '0.04em',
                    boxShadow: isLive && isSealed ? `0 0 32px -4px ${p.accent}cc` : 'none',
                    transition: 'all 0.3s',
                  }}>
                    {display}
                  </div>
                );
              })}
            </div>
          ))}
        </div>
      </div>
    </HostShell>
  );
}

/* export */
Object.assign(window, {
  LobbyHostScreen, CharcreateHostScreen, AnsweringHostScreen, RevealHostScreen, GridHostScreen,
});
