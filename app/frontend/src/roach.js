// Gregor: uma barata que atravessa a tela e foge do mouse.
// Vive fora da árvore do Vue (direto no <body>) para não ser interrompida pelas re-renderizações.

const ROACH_SVG = `<svg viewBox="-40 -30 80 60" width="64" height="48">
  <g stroke="#2b1606" stroke-width="2.2" stroke-linecap="round" fill="none">
    <g class="la"><path d="M10 -6 L16 -14 L22 -22"/><path d="M0 6 L-2 15 L2 24"/><path d="M-10 -6 L-18 -14 L-28 -22"/></g>
    <g class="lb"><path d="M10 6 L16 14 L22 22"/><path d="M0 -6 L-2 -15 L2 -24"/><path d="M-10 6 L-18 14 L-28 22"/></g>
    <path d="M24 -2 Q34 -10 40 -26" stroke-width="1.2"/><path d="M24 2 Q34 10 40 26" stroke-width="1.2"/>
    <path d="M-24 -3 L-34 -7" stroke-width="1.4"/><path d="M-24 3 L-34 7" stroke-width="1.4"/>
  </g>
  <ellipse cx="-5" cy="0" rx="20" ry="11" fill="#5a3214"/>
  <path d="M-24 0 L12 0" stroke="#2b1606" stroke-width="1"/>
  <ellipse cx="-5" cy="-4" rx="14" ry="3" fill="#8a5a2c" opacity=".45"/>
  <ellipse cx="12" cy="0" rx="8" ry="9" fill="#6b3d18"/>
  <circle cx="21" cy="0" r="4.5" fill="#3a1f0a"/>
</svg>`;

const roaches = new Set();
const mouse = { x: -999, y: -999 };
const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
document.addEventListener('mousemove', (e) => { mouse.x = e.clientX; mouse.y = e.clientY; });
document.addEventListener('touchstart', (e) => { const t = e.touches[0]; mouse.x = t.clientX; mouse.y = t.clientY; }, { passive: true });

// getPhase: função que devolve a fase atual — a barata nunca aparece durante a argumentação
export function spawnRoach(delay = 0, getPhase = () => null) {
  if (reduceMotion || getPhase() === 'speak') return;
  setTimeout(() => {
    if (getPhase() === 'speak') return;
    const W = innerWidth, H = innerHeight;
    const fromLeft = Math.random() < 0.5;
    const el = document.createElement('div');
    el.className = 'roach';
    el.innerHTML = ROACH_SVG;
    document.body.appendChild(el);
    const r = { el, x: fromLeft ? -50 : W + 50, y: H * (0.2 + Math.random() * 0.6), h: fromLeft ? 0 : Math.PI, goal: fromLeft ? 0 : Math.PI, flee: 0, t: 0 };
    roaches.add(r);
    if (roaches.size === 1) requestAnimationFrame(step);
  }, delay);
}

export function killRoaches() {
  for (const r of roaches) r.el.remove();
  roaches.clear();
}

let lastStep = 0;
function step(now) {
  const dt = Math.min(0.05, (now - (lastStep || now)) / 1000);
  lastStep = now;
  const W = innerWidth, H = innerHeight;
  for (const r of roaches) {
    r.t += dt;
    const dx = r.x - mouse.x, dy = r.y - mouse.y;
    if (dx * dx + dy * dy < 150 * 150) { r.goal = Math.atan2(dy, dx); r.flee = 0.7; }
    r.flee = Math.max(0, r.flee - dt);
    // passos nervosos: vai virando aos poucos em direção ao objetivo, com tremidas
    r.goal += (Math.random() - 0.5) * 0.25;
    const diff = Math.atan2(Math.sin(r.goal - r.h), Math.cos(r.goal - r.h));
    r.h += diff * Math.min(1, dt * (r.flee ? 14 : 4));
    const pause = !r.flee && Math.sin(r.t * 2.3) > 0.93; // paradinhas
    const speed = r.flee ? 560 : pause ? 0 : 190;
    r.x += Math.cos(r.h) * speed * dt;
    r.y += Math.sin(r.h) * speed * dt;
    r.goal = Math.atan2(Math.sin(r.goal), Math.cos(r.goal));
    if (r.y < 30) r.goal = Math.abs(r.goal); // não sai por cima/baixo
    if (r.y > H - 30) r.goal = -Math.abs(r.goal);
    r.el.classList.toggle('fast', r.flee > 0);
    r.el.style.transform = `translate(${r.x - 32}px, ${r.y - 24}px) rotate(${r.h}rad)`;
    if (r.t > 25 || (r.t > 1 && (r.x < -80 || r.x > W + 80))) { r.el.remove(); roaches.delete(r); }
  }
  if (roaches.size) requestAnimationFrame(step);
  else lastStep = 0;
}
