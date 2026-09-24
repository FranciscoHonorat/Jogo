// Ordem & Caos — servidor do MVP (Node puro, sem dependências)
// Tempo real via Server-Sent Events; ações via POST /api/action.

const http = require('http');
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const CARDS = require('./cards');

const PORT = process.env.PORT || 3000;
const THINK_MS = 30 * 1000;
const SPEAK_MS = 120 * 1000;
const JUDGE_MS = 180 * 1000;
const TEAMS = ['ordem', 'caos'];

const rooms = new Map(); // code -> room
const streams = new Map(); // token -> Set<res>

function shuffle(arr) {
  const a = arr.map((c, i) => ({ ...c, id: i }));
  for (let i = a.length - 1; i > 0; i--) {
    const j = crypto.randomInt(i + 1);
    [a[i], a[j]] = [a[j], a[i]];
  }
  return a;
}

function newRoom(code) {
  return {
    code,
    players: new Map(), // token -> { name, role }
    teams: { ordem: { name: '' }, caos: { name: '' } },
    ...freshGame(),
  };
}

function freshGame() {
  return {
    phase: 'lobby', // lobby | draw | think | speak | judging | result | finished
    round: 0,
    turn: 0, // índice em order (0 ou 1)
    order: ['ordem', 'caos'],
    endsAt: null,
    timer: null,
    decks: { ordem: shuffle(CARDS.ordem), caos: shuffle(CARDS.caos) },
    current: { ordem: null, caos: null },
    votes: {}, // token -> 'ordem' | 'caos' | 'empate'
    lastResult: null,
    score: { ordem: 0, caos: 0, empate: 0 },
    history: [],
  };
}

function getRoom(code) {
  if (!rooms.has(code)) rooms.set(code, newRoom(code));
  return rooms.get(code);
}

function judges(room) {
  return [...room.players.entries()].filter(([, p]) => p.role === 'juiz');
}

function activeTeam(room) {
  return room.order[room.turn];
}

function setPhase(room, phase, ms) {
  clearTimeout(room.timer);
  room.timer = null;
  room.phase = phase;
  room.endsAt = ms ? Date.now() + ms : null;
  if (ms) room.timer = setTimeout(() => onTimeout(room), ms);
}

function onTimeout(room) {
  room.timer = null;
  if (room.phase === 'think') setPhase(room, 'speak', SPEAK_MS);
  else if (room.phase === 'speak') endTurn(room);
  else if (room.phase === 'judging') closeVoting(room);
  broadcast(room);
}

function startRound(room) {
  room.round += 1;
  // alterna quem começa: rodadas ímpares a Ordem, pares o Caos
  room.order = room.round % 2 === 1 ? ['ordem', 'caos'] : ['caos', 'ordem'];
  room.turn = 0;
  room.current = { ordem: null, caos: null };
  room.votes = {};
  room.lastResult = null;
  setPhase(room, 'draw');
}

function endTurn(room) {
  if (room.turn === 0) {
    room.turn = 1;
    setPhase(room, 'draw');
  } else {
    setPhase(room, 'judging', JUDGE_MS);
  }
}

function closeVoting(room) {
  const tally = { ordem: 0, caos: 0, empate: 0 };
  for (const v of Object.values(room.votes)) tally[v]++;
  const winner = tally.ordem > tally.caos ? 'ordem' : tally.caos > tally.ordem ? 'caos' : 'empate';
  room.score[winner]++;
  room.lastResult = { winner, tally };
  room.history.push({
    round: room.round,
    ordem: room.current.ordem && room.current.ordem.nome,
    caos: room.current.caos && room.current.caos.nome,
    winner,
    tally,
  });
  setPhase(room, 'result');
}

// ---------- ações ----------

function act(room, token, body) {
  const me = room.players.get(token);
  if (!me) throw new Error('Você não está nesta sala.');
  const isJudge = me.role === 'juiz';
  const a = body.action;

  switch (a) {
    case 'start':
      if (!isJudge) throw new Error('Só juízes podem iniciar.');
      if (room.phase !== 'lobby') throw new Error('A partida já começou.');
      if (!room.teams.ordem.name || !room.teams.caos.name) throw new Error('Os dois times precisam entrar antes.');
      startRound(room);
      break;

    case 'draw': {
      const team = activeTeam(room);
      if (room.phase !== 'draw' || me.role !== team) throw new Error('Não é a vez do seu time de puxar.');
      const card = room.decks[team].pop();
      if (!card) throw new Error('Deck vazio.');
      room.current[team] = card;
      setPhase(room, 'think', THINK_MS);
      break;
    }

    case 'skip': // time encerra o tempo de pensar/falar antes do fim
      if (!['think', 'speak'].includes(room.phase)) throw new Error('Nada para avançar agora.');
      if (me.role !== activeTeam(room)) throw new Error('Não é a vez do seu time.');
      if (room.phase === 'think') setPhase(room, 'speak', SPEAK_MS);
      else endTurn(room);
      break;

    case 'vote':
      if (!isJudge) throw new Error('Só juízes votam.');
      if (room.phase !== 'judging') throw new Error('A votação não está aberta.');
      if (!['ordem', 'caos', 'empate'].includes(body.choice)) throw new Error('Voto inválido.');
      room.votes[token] = body.choice;
      if (judges(room).every(([t]) => room.votes[t])) closeVoting(room);
      break;

    case 'close':
      if (!isJudge || room.phase !== 'judging') throw new Error('Não é possível encerrar agora.');
      closeVoting(room);
      break;

    case 'next':
      if (!isJudge || room.phase !== 'result') throw new Error('Não é possível avançar agora.');
      if (room.decks.ordem.length === 0 || room.decks.caos.length === 0) setPhase(room, 'finished');
      else startRound(room);
      break;

    case 'end':
      if (!isJudge) throw new Error('Só juízes podem encerrar.');
      setPhase(room, 'finished');
      break;

    case 'reset':
      if (!isJudge) throw new Error('Só juízes podem reiniciar.');
      clearTimeout(room.timer);
      Object.assign(room, freshGame());
      break;

    default:
      throw new Error('Ação desconhecida.');
  }
}

// ---------- estado enviado ao cliente ----------

function view(room, token) {
  const me = room.players.get(token) || null;
  const roster = { ordem: [], caos: [], juiz: [], plateia: [] };
  for (const p of room.players.values()) roster[p.role].push(p.name);
  const js = judges(room);
  return {
    now: Date.now(),
    me,
    code: room.code,
    teams: room.teams,
    roster,
    phase: room.phase,
    round: room.round,
    totalRounds: CARDS.ordem.length,
    active: room.phase === 'lobby' || room.phase === 'finished' ? null : activeTeam(room),
    order: room.order,
    endsAt: room.endsAt,
    durations: { think: THINK_MS, speak: SPEAK_MS, judging: JUDGE_MS },
    deckLeft: { ordem: room.decks.ordem.length, caos: room.decks.caos.length },
    current: room.current,
    votesIn: Object.keys(room.votes).length,
    judgeCount: js.length,
    myVote: room.votes[token] || null,
    lastResult: room.lastResult,
    score: room.score,
    history: room.history,
  };
}

function send(res, data) {
  res.write(`data: ${JSON.stringify(data)}\n\n`);
}

function broadcast(room) {
  for (const token of room.players.keys()) {
    const set = streams.get(token);
    if (set) for (const res of set) send(res, view(room, token));
  }
}

// ---------- HTTP ----------

function readJson(req) {
  return new Promise((resolve, reject) => {
    let raw = '';
    req.on('data', (c) => {
      raw += c;
      if (raw.length > 1e5) req.destroy();
    });
    req.on('end', () => {
      try { resolve(raw ? JSON.parse(raw) : {}); } catch (e) { reject(new Error('JSON inválido')); }
    });
  });
}

function json(res, status, obj) {
  res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' });
  res.end(JSON.stringify(obj));
}

const clean = (s, n) => String(s || '').trim().slice(0, n);
const roomCode = (s) => clean(s, 20).toUpperCase().replace(/[^A-Z0-9]/g, '') || 'SALA1';

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, 'http://x');

  try {
    if (req.method === 'POST' && url.pathname === '/api/join') {
      const b = await readJson(req);
      const room = getRoom(roomCode(b.room));
      const role = ['ordem', 'caos', 'juiz', 'plateia'].includes(b.role) ? b.role : null;
      const name = clean(b.name, 40);
      if (!role || !name) return json(res, 400, { error: 'Informe nome e papel.' });
      const token = typeof b.token === 'string' && b.token.length >= 16 ? b.token : crypto.randomUUID();
      if (TEAMS.includes(role)) {
        const team = room.teams[role];
        if (!team.name) {
          const tn = clean(b.teamName, 40);
          if (!tn) return json(res, 400, { error: 'Informe o nome do time.' });
          team.name = tn;
        }
      }
      room.players.set(token, { name, role });
      broadcast(room);
      return json(res, 200, { token, room: room.code });
    }

    if (req.method === 'POST' && url.pathname === '/api/action') {
      const b = await readJson(req);
      const room = rooms.get(roomCode(b.room));
      if (!room) return json(res, 404, { error: 'Sala não encontrada.' });
      act(room, b.token, b);
      broadcast(room);
      return json(res, 200, { ok: true });
    }

    if (req.method === 'GET' && url.pathname === '/api/teams') {
      const room = rooms.get(roomCode(url.searchParams.get('room')));
      return json(res, 200, room ? room.teams : { ordem: { name: '' }, caos: { name: '' } });
    }

    if ((req.method === 'GET' || req.method === 'HEAD') && url.pathname === '/events') {
      const room = rooms.get(roomCode(url.searchParams.get('room')));
      const token = url.searchParams.get('token');
      if (!room || !room.players.has(token)) {
        res.writeHead(404);
        return res.end();
      }
      if (req.method === 'HEAD') {
        res.writeHead(200);
        return res.end();
      }
      res.writeHead(200, {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache, no-transform',
        Connection: 'keep-alive',
        'X-Accel-Buffering': 'no',
      });
      res.write('retry: 2000\n\n');
      if (!streams.has(token)) streams.set(token, new Set());
      streams.get(token).add(res);
      send(res, view(room, token));
      const ping = setInterval(() => res.write(': ping\n\n'), 15000);
      req.on('close', () => {
        clearInterval(ping);
        streams.get(token)?.delete(res);
      });
      return;
    }

    if (req.method === 'GET' && (url.pathname === '/' || url.pathname === '/index.html')) {
      res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
      return fs.createReadStream(path.join(__dirname, 'public', 'index.html')).pipe(res);
    }

    if (req.method === 'GET' && url.pathname === '/health') return json(res, 200, { ok: true });

    res.writeHead(404);
    res.end('Not found');
  } catch (e) {
    json(res, 400, { error: e.message });
  }
});

server.listen(PORT, () => console.log(`Ordem & Caos rodando em http://localhost:${PORT}`));
