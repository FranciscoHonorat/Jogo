<script setup>
import { computed } from 'vue';
import { store, teamName } from '../store.js';

const rows = computed(() => store.S.history.slice().reverse());
</script>

<template>
  <div class="panel">
    <h3>Histórico</h3>
    <table>
      <tr><th>#</th><th>{{ teamName('ordem') }}</th><th>{{ teamName('caos') }}</th><th>Vencedor</th></tr>
      <tr v-for="h in rows" :key="h.round">
        <td>{{ h.round }}</td>
        <td>{{ h.ordem || '—' }}</td>
        <td>{{ h.caos || '—' }}</td>
        <td :class="'w-' + h.winner">{{ h.winner === 'empate' ? 'Empate' : teamName(h.winner) }}</td>
      </tr>
    </table>
  </div>
</template>
