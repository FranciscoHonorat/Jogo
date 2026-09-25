<script setup>
import { computed } from 'vue';
import { store, teamName, MISSION } from '../store.js';

const S = computed(() => store.S);
const TEAMS = [
  { id: 'ordem', label: 'ORDEM · A FAMÍLIA' },
  { id: 'caos', label: 'CAOS · A METAMORFOSE' },
];
// destaca o time que está jogando agora
const isOn = (t) => S.value.active === t && ['draw', 'think', 'speak'].includes(S.value.phase);
</script>

<template>
  <div class="top">
    <template v-for="(t, i) in TEAMS" :key="t.id">
      <div class="team" :class="[t.id, { on: isOn(t.id) }]">
        <div class="lbl">{{ t.label }}</div>
        <div class="nm">{{ teamName(t.id) }}</div>
        <div class="mis">{{ MISSION[t.id] }}</div>
        <div class="pts">{{ S.score[t.id] }}</div>
        <div class="deck">{{ S.deckLeft[t.id] }} cartas no deck</div>
      </div>
      <div v-if="i === 0" class="mid">
        <div class="sc" style="font-size:22px">Rodada {{ S.round || '—' }}</div>
        <div class="rd">de {{ S.totalRounds }}</div>
        <div class="emp">Empates: {{ S.score.empate }}</div>
      </div>
    </template>
  </div>
</template>
