<script setup>
import { onMounted, ref } from 'vue';
import { store, resume } from './store.js';
import JoinView from './components/JoinView.vue';
import GameView from './components/GameView.vue';

const ready = ref(false);
onMounted(async () => {
  await resume();
  ready.value = true;
});
</script>

<template>
  <div class="wrap">
    <template v-if="ready">
      <GameView v-if="store.session && store.S" />
      <JoinView v-else-if="!store.session" />
    </template>
  </div>
  <div class="toast" v-show="store.toast">{{ store.toast }}</div>
</template>
