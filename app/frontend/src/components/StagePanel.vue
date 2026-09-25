<script setup>
// Fase atual, cronômetro e os botões de ação de cada papel.
import { computed } from 'vue';
import { store, me, now, teamName, action, PHASE } from '../store.js';

const S = computed(() => store.S);
const isJudge = computed(() => me.value.role === 'juiz');
const act = computed(() => S.value.active);
const mine = computed(() => me.value.role === act.value);
const teamTurn = computed(() => act.value && ['draw', 'think', 'speak'].includes(S.value.phase));
const bothTeamsIn = computed(() => S.value.teams.ordem.name && S.value.teams.caos.name);
const lastRound = computed(() => S.value.deckLeft.ordem === 0 || S.value.deckLeft.caos === 0);
const list = (a) => (a.length ? a.join(', ') : '—');

// limitado à duração da fase: a diferença entre os relógios não deixa o timer mostrar 3:01
const left = computed(() => Math.min(S.value.durations[S.value.phase] || Infinity, Math.max(0, S.value.endsAt - (now.value + store.offset))));
const clock = computed(() => {
  const s = Math.ceil(left.value / 1000);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
});
const low = computed(() => Math.ceil(left.value / 1000) <= 10);
const barWidth = computed(() => `${(left.value / (S.value.durations[S.value.phase] || 1)) * 100}%`);

const finalWinner = computed(() => {
  const { ordem, caos } = S.value.score;
  return ordem > caos ? 'ordem' : caos > ordem ? 'caos' : null;
});

function reset() {
  if (confirm('Começar uma nova partida? O placar será zerado.')) action('reset');
}
</script>

<template>
  <div class="stage">
    <div class="ph">{{ PHASE[S.phase] }}<template v-if="teamTurn"> · {{ teamName(act) }}</template></div>

    <template v-if="S.endsAt">
      <div class="timer" :class="{ low }">{{ clock }}</div>
      <div class="bar"><i :style="{ width: barWidth }"></i></div>
    </template>

    <div class="hint" style="margin-top:8px">
      <template v-if="S.phase === 'lobby'">
        Ordem: {{ list(S.roster.ordem) }} · Caos: {{ list(S.roster.caos) }} · Juízes: {{ list(S.roster.juiz) }}
        <template v-if="!isJudge"><br>Um(a) juiz(a) inicia a partida quando os dois times estiverem na sala.</template>
      </template>
      <template v-else-if="S.phase === 'draw'">Vez de <b>{{ teamName(act) }}</b> puxar uma carta.</template>
      <template v-else-if="S.phase === 'think'"><b>{{ teamName(act) }}</b> está preparando o argumento.</template>
      <template v-else-if="S.phase === 'speak'"><b>{{ teamName(act) }}</b> está argumentando.</template>
      <template v-else-if="S.phase === 'judging'">Quem venceu o argumento? · Votos: {{ S.votesIn }}/{{ S.judgeCount }}</template>
      <template v-else-if="S.phase === 'result'">
        <div class="result" :class="S.lastResult.winner">
          {{ S.lastResult.winner === 'empate' ? 'Empate!' : teamName(S.lastResult.winner) + ' venceu a rodada!' }}
        </div>
        Votos — {{ teamName('ordem') }}: {{ S.lastResult.tally.ordem }} · {{ teamName('caos') }}: {{ S.lastResult.tally.caos }} · Empate: {{ S.lastResult.tally.empate }}
      </template>
      <template v-else-if="S.phase === 'finished'">
        <div class="result" :class="finalWinner">
          {{ finalWinner ? teamName(finalWinner) + ' venceu a partida!' : 'A partida terminou empatada!' }}
        </div>
        {{ S.score.ordem }} × {{ S.score.caos }} ({{ S.score.empate }} empate{{ S.score.empate === 1 ? '' : 's' }})
      </template>
    </div>

    <div class="actions">
      <button v-if="S.phase === 'lobby' && isJudge" class="big" data-a="start" :disabled="!bothTeamsIn" @click="action('start')">Iniciar partida</button>

      <button v-if="S.phase === 'draw' && mine" class="big" :class="act" data-a="draw" @click="action('draw')">
        Puxar carta ({{ S.deckLeft[act] }} no deck)
      </button>
      <button v-if="S.phase === 'think' && mine" :class="act" data-a="skip" @click="action('skip')">Pronto — começar a argumentar</button>
      <button v-if="S.phase === 'speak' && mine" :class="act" data-a="skip" @click="action('skip')">Terminamos</button>

      <template v-if="S.phase === 'judging' && isJudge">
        <div class="votes">
          <button class="ordem big" :class="{ sel: S.myVote === 'ordem' }" data-a="vote" data-c="ordem" @click="action('vote', { choice: 'ordem' })">{{ teamName('ordem') }}</button>
          <button class="big ghost" :class="{ sel: S.myVote === 'empate' }" data-a="vote" data-c="empate" @click="action('vote', { choice: 'empate' })">Empate</button>
          <button class="caos big" :class="{ sel: S.myVote === 'caos' }" data-a="vote" data-c="caos" @click="action('vote', { choice: 'caos' })">{{ teamName('caos') }}</button>
        </div>
        <button class="ghost" data-a="close" @click="action('close')">Encerrar votação agora</button>
      </template>

      <button v-if="S.phase === 'result' && isJudge" class="big" data-a="next" @click="action('next')">
        {{ lastRound ? 'Ver resultado final' : 'Próxima rodada' }}
      </button>
      <button v-if="S.phase === 'finished' && isJudge" data-a="reset" @click="reset">Nova partida</button>
    </div>
  </div>
</template>

<style scoped>
.actions:empty { display: none; }
</style>
