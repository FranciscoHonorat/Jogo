<script setup>
// Faixa grande dizendo a cada pessoa o que fazer agora.
import { computed } from 'vue';
import { store, me, teamName, MISSION } from '../store.js';

const banner = computed(() => {
  const S = store.S;
  const role = me.value.role;
  const act = S.active;
  const isTeam = role === 'ordem' || role === 'caos';
  const mine = role === act;
  const actName = act ? teamName(act) : '';
  const speaksLater = S.order[1] === role;
  let cls = '', text = '', sub = '';

  switch (S.phase) {
    case 'lobby':
      if (role === 'juiz') {
        const ready = S.teams.ordem.name && S.teams.caos.name;
        [cls, text] = ready ? ['juiz pulse', 'Tudo pronto! Clique em “Iniciar partida”.'] : ['', 'Aguardando os dois times entrarem…'];
      } else text = 'Aguarde: um(a) juiz(a) vai iniciar a partida.';
      if (isTeam) sub = 'Sua missão: ' + MISSION[role];
      break;
    case 'draw':
      if (mine) [cls, text, sub] = [role + ' pulse', 'Sua vez! Puxe uma carta.', 'Clique no botão abaixo.'];
      else text = `Aguarde: ${actName} vai puxar uma carta.`;
      break;
    case 'think':
      if (mine) [cls, text, sub] = [role, 'Preparem o argumento!', 'Leiam a carta e combinem o que vão dizer. Quando estiverem prontos, cliquem em “Pronto”.'];
      else if (role === 'juiz') [text, sub] = [`${actName} está se preparando.`, 'Você vota no fim da rodada, depois que os dois times falarem.'];
      else if (isTeam) text = speaksLater ? `${actName} está se preparando. Vocês falam depois.` : 'Vocês já falaram. Aguardem o outro time.';
      else text = `${actName} está se preparando.`;
      break;
    case 'speak':
      if (mine) [cls, text, sub] = [role + ' pulse', 'Falem agora!', 'Usem o livro para defender a carta de vocês.'];
      else if (role === 'juiz') [cls, text, sub] = ['juiz', `Escute ${actName}.`, 'Você vota no fim da rodada.'];
      else if (isTeam) [text, sub] = [`Escutem ${actName}.`, speaksLater ? 'Anotem pontos para rebater quando for a vez de vocês.' : ''];
      else text = `${actName} está argumentando.`;
      break;
    case 'judging':
      if (role === 'juiz') [cls, text, sub] = S.myVote
        ? ['', 'Voto registrado!', 'Aguardando os outros juízes. Você ainda pode trocar o voto.']
        : ['juiz pulse', 'Vote agora: quem argumentou melhor?', 'Use os critérios abaixo.'];
      else text = 'Os juízes estão decidindo…';
      break;
    case 'result': {
      const w = S.lastResult.winner;
      if (role === 'juiz') text = 'Clique em “Próxima rodada” quando todos estiverem prontos.';
      else if (isTeam && w !== 'empate') [cls, text] = w === role ? [role, 'Vocês venceram a rodada!'] : ['', 'Vocês perderam esta rodada. Próxima!'];
      else text = w === 'empate' ? 'Rodada empatada!' : `${teamName(w)} venceu a rodada!`;
      break;
    }
    case 'finished':
      text = 'Fim de jogo!';
      break;
  }
  return { cls, text, sub };
});
</script>

<template>
  <div class="banner" :class="banner.cls">
    {{ banner.text }}
    <small v-if="banner.sub">{{ banner.sub }}</small>
  </div>
</template>
