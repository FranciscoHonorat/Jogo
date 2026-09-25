// Estado global: sessão, conexão SSE com o servidor e ações do jogo.
import { reactive, ref, computed } from 'vue';
import { spawnRoach, killRoaches } from './roach.js';

export const LS = {
  get(k) { try { return localStorage.getItem(k); } catch { return null; } },
  set(k, v) { try { localStorage.setItem(k, v); } catch {} },
  del(k) { try { localStorage.removeItem(k); } catch {} },
};

export const ROLE_LABEL = { ordem: 'Ordem', caos: 'Caos', juiz: 'Juiz(a)', plateia: 'Plateia' };
export const MISSION = {
  ordem: 'Defenda que a família e a rotina resistem à metamorfose.',
  caos: 'Mostre que a metamorfose revela e destrói a família.',
};
export const PHASE = {
  lobby: 'Aguardando jogadores',
  draw: 'Puxar carta',
  think: 'Tempo para pensar',
  speak: 'Argumentação',
  judging: 'Juízes decidem',
  result: 'Resultado da rodada',
  finished: 'Fim de jogo',
};

export const store = reactive({
  session: null,   // { room, token }
  S: null,         // último estado recebido do servidor
  offset: 0,       // relógio do servidor - relógio local
  connected: false,
  joinError: '',
  toast: '',
});

// relógio local para os timers (atualiza 4x por segundo)
export const now = ref(Date.now());
setInterval(() => (now.value = Date.now()), 250);

export const teamName = (t) => store.S?.teams[t]?.name || (t === 'ordem' ? 'Ordem' : 'Caos');
export const me = computed(() => store.S?.me || {});

let es = null;
let lastPhaseKey = '';
let toastTimer = null;

export function showToast(msg) {
  store.toast = msg;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => (store.toast = ''), 3500);
}

async function post(url, body) {
  const r = await fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  const j = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(j.error || 'Erro');
  return j;
}

export async function action(name, extra = {}) {
  try { await post('/api/action', { room: store.session.room, token: store.session.token, action: name, ...extra }); }
  catch (e) { showToast(e.message); }
}

export async function join({ room, name, role, teamName }) {
  const r = await post('/api/join', { room, name, role, teamName, token: LS.get('oc_token') });
  LS.set('oc_token', r.token); LS.set('oc_room', r.room); LS.set('oc_name', name);
  store.session = { room: r.room, token: r.token };
  connect();
}

export function leave() {
  if (es) es.close();
  LS.del('oc_room');
  store.session = null;
  store.S = null;
  lastPhaseKey = '';
  setDecay(0);
}

export async function fetchTeams(room) {
  return (await fetch('/api/teams?room=' + encodeURIComponent(room))).json();
}

function beep(freq = 660, ms = 180) {
  try {
    const ctx = beep.ctx || (beep.ctx = new (window.AudioContext || window.webkitAudioContext)());
    const o = ctx.createOscillator(), g = ctx.createGain();
    o.frequency.value = freq; o.connect(g); g.connect(ctx.destination);
    g.gain.setValueAtTime(0.15, ctx.currentTime);
    g.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + ms / 1000);
    o.start(); o.stop(ctx.currentTime + ms / 1000);
  } catch {}
}

// o quanto a página já "se metamorfoseou": 0 = Ordem no controle, 1 = Caos 5+ rodadas à frente
function setDecay(d) { document.body.style.setProperty('--decay', d.toFixed(2)); }
function updateDecay(S) {
  setDecay(S.phase !== 'lobby' ? Math.max(0, Math.min(1, (S.score.caos - S.score.ordem) / 5)) : 0);
}

const currentPhase = () => store.S?.phase;

export function connect() {
  if (es) es.close();
  const { room, token } = store.session;
  es = new EventSource(`/events?room=${encodeURIComponent(room)}&token=${encodeURIComponent(token)}`);
  es.onopen = () => (store.connected = true);
  es.onmessage = (e) => {
    const prev = store.S;
    const S = JSON.parse(e.data);
    store.S = S;
    store.offset = S.now - Date.now();
    store.connected = true;
    const key = S.phase + S.round + S.active;
    const changed = lastPhaseKey && key !== lastPhaseKey;
    if (changed) beep(S.phase === 'result' ? 880 : 660);
    lastPhaseKey = key;
    updateDecay(S);
    if (S.phase === 'speak') killRoaches();
    if (changed && prev) {
      // Caos acabou de puxar uma carta
      if (S.phase === 'think' && S.active === 'caos') spawnRoach(400, currentPhase);
      // Caos venceu a rodada / a partida
      if (S.phase === 'result' && S.lastResult.winner === 'caos') { spawnRoach(300, currentPhase); spawnRoach(1100, currentPhase); }
      if (S.phase === 'finished' && S.score.caos > S.score.ordem) for (let i = 0; i < 6; i++) spawnRoach(i * 450, currentPhase);
    }
  };
  es.onerror = () => {
    store.connected = false;
    // 404 = a sala não existe mais (servidor reiniciou): volta à entrada
    if (es.readyState === EventSource.CLOSED) {
      leave();
      store.joinError = 'Conexão perdida ou sala reiniciada. Entre novamente.';
    }
  };
}

// retoma a sessão salva, se a sala ainda existir no servidor
export async function resume() {
  const room = new URLSearchParams(location.search).get('sala') || LS.get('oc_room');
  const token = LS.get('oc_token');
  if (!room || !token || !LS.get('oc_name')) return false;
  const code = room.toUpperCase().replace(/[^A-Z0-9]/g, '');
  try {
    const r = await fetch(`/events?room=${encodeURIComponent(code)}&token=${encodeURIComponent(token)}`, { method: 'HEAD' });
    if (!r.ok) return false;
  } catch { return false; }
  store.session = { room: code, token };
  connect();
  return true;
}
