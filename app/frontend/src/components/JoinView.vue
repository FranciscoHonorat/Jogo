<script>
// fica no módulo (e não no componente) para valer uma vez por visita, mesmo ao sair e voltar
let roached = false;
</script>

<script setup>
import { onMounted, ref, watch } from 'vue';
import { store, LS, ROLE_LABEL, join, fetchTeams } from '../store.js';
import { spawnRoach } from '../roach.js';
import InviteBox from './InviteBox.vue';

// no computador que roda o jogo, mostra o QR code para os alunos
const isHostMachine = ['localhost', '127.0.0.1', '[::1]'].includes(location.hostname);

const ROLES = [
  { id: 'ordem', label: 'Time da Ordem' },
  { id: 'caos', label: 'Time do Caos' },
  { id: 'juiz', label: ROLE_LABEL.juiz },
  { id: 'plateia', label: ROLE_LABEL.plateia },
];

const room = ref(new URLSearchParams(location.search).get('sala') || LS.get('oc_room') || 'SALA1');
const name = ref(LS.get('oc_name') || '');
const role = ref(null);
const teamName = ref('');
const existingTeam = ref(''); // nome do time, se alguém já criou
const error = ref(store.joinError);
const sending = ref(false);

const isTeam = () => role.value === 'ordem' || role.value === 'caos';

// Gregor atravessa a tela de entrada uma vez por visita
onMounted(() => {
  if (!roached) { roached = true; spawnRoach(1500); }
});

let teamsReq = 0;
watch([role, room], async () => {
  existingTeam.value = '';
  if (!isTeam()) return;
  const req = ++teamsReq;
  try {
    const teams = await fetchTeams(room.value);
    if (req === teamsReq) existingTeam.value = teams[role.value]?.name || '';
  } catch {}
});

async function submit() {
  if (!role.value) return (error.value = 'Escolha um papel.');
  sending.value = true;
  try {
    store.joinError = '';
    await join({ room: room.value.trim(), name: name.value.trim(), role: role.value, teamName: teamName.value.trim() });
  } catch (e) {
    error.value = e.message;
  } finally {
    sending.value = false;
  }
}
</script>

<template>
  <form class="join" @submit.prevent="submit">
    <h1>Ordem &amp; Caos</h1>
    <div class="sub">RPG Literário · A Metamorfose · Franz Kafka</div>

    <label for="room">Código da sala</label>
    <input id="room" v-model="room" maxlength="20">

    <label for="name">Seu nome</label>
    <input id="name" v-model="name" maxlength="40" placeholder="Ex.: Maria">

    <label>Entrar como</label>
    <div class="roles">
      <button v-for="r in ROLES" :key="r.id" type="button" :data-role="r.id"
              :class="[r.id, { sel: role === r.id }]" @click="role = r.id">{{ r.label }}</button>
    </div>

    <template v-if="isTeam()">
      <label for="teamName">Nome do time</label>
      <template v-if="existingTeam">
        <div class="sc" style="font-size:22px">{{ existingTeam }}</div>
        <div style="color:var(--muted);font-size:14px">Time já criado — você entra nele.</div>
      </template>
      <input v-else id="teamName" v-model="teamName" maxlength="40" placeholder="Ex.: Os Samsa">
    </template>

    <div style="margin-top:18px">
      <button id="go" type="submit" class="big" style="width:100%" :disabled="sending">Entrar</button>
    </div>
    <div class="err">{{ error }}</div>
  </form>
  <div v-if="isHostMachine" class="join-invite">
    <InviteBox :room="room" />
  </div>
</template>
