<script setup>
import { computed } from 'vue';
import { store, me, teamName, action, leave, ROLE_LABEL } from '../store.js';
import ScoreBoard from './ScoreBoard.vue';
import PhaseBanner from './PhaseBanner.vue';
import StagePanel from './StagePanel.vue';
import GameCard from './GameCard.vue';
import JudgeCriteria from './JudgeCriteria.vue';
import HistoryPanel from './HistoryPanel.vue';
import InviteBox from './InviteBox.vue';

const S = computed(() => store.S);
const isJudge = computed(() => me.value.role === 'juiz');
// na sala de espera, a tela projetada (juiz/plateia) mostra o QR code para quem ainda vai entrar
const showInvite = computed(() => S.value.phase === 'lobby' && ['juiz', 'plateia'].includes(me.value.role));
const inPlay = computed(() => !['lobby', 'finished'].includes(S.value.phase));

function endGame() {
  if (confirm('Encerrar a partida agora?')) action('end');
}
</script>

<template>
  <ScoreBoard />
  <PhaseBanner />
  <StagePanel />
  <InviteBox v-if="showInvite" :room="S.code" />
  <JudgeCriteria v-if="isJudge && S.phase === 'judging'" />

  <div v-if="S.phase !== 'lobby'" class="board">
    <div v-for="team in S.order" :key="team">
      <GameCard v-if="S.current[team]" :key="S.round + team + S.current[team].id" :card="S.current[team]" :team="team" />
      <div v-else class="slot">{{ inPlay ? `${teamName(team)} ainda não puxou a carta` : 'Carta' }}</div>
    </div>
  </div>

  <HistoryPanel v-if="S.history.length" />
  <JudgeCriteria v-if="isJudge && S.phase !== 'judging'" />

  <div class="foot">
    <span>
      <span class="conn" :class="{ off: !store.connected }"></span>
      Sala <b>{{ S.code }}</b> · você: {{ me.name }} ({{ ROLE_LABEL[me.role] || '—' }})
    </span>
    <span>
      <button v-if="isJudge && inPlay" class="ghost" data-a="end" style="font-size:14px;padding:6px 10px" @click="endGame">Encerrar partida</button>
      <button class="ghost" id="leave" style="font-size:14px;padding:6px 10px" @click="leave">Sair</button>
    </span>
  </div>
</template>
