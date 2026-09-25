<script setup>
// A carta anima (vira) só quando é montada, isto é, quando acabou de ser puxada.
defineProps({
  card: { type: Object, required: true },
  team: { type: String, required: true },
});
</script>

<template>
  <div class="card new" :class="[team, { especial: card.especial }]">
    <div class="hd">
      <div>
        <div class="nome"><span v-if="card.especial" class="star">★ </span>{{ card.nome }}</div>
        <div class="tipo">{{ card.tipo }} · {{ team === 'ordem' ? 'Ordem' : 'Caos' }}</div>
      </div>
      <div class="val">{{ card.valor }}</div>
    </div>
    <div class="arg"><b>Argumente que…</b> {{ card.argumente }}</div>
    <div class="nar-l">NARRAÇÃO SUGERIDA</div>
    <div class="nar">“{{ card.narracao }}”</div>
  </div>
</template>
