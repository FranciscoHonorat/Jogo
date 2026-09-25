<script setup>
// "Convide os jogadores": QR code + endereço para os alunos entrarem pelo celular.
// No computador que roda o jogo (localhost), usa o endereço da rede local informado pelo servidor;
// quando o jogo está na internet, usa o próprio endereço da página.
import { computed, ref, watchEffect, onMounted } from 'vue';
import QRCode from 'qrcode';

const props = defineProps({ room: { type: String, default: '' } });

const isHostMachine = ['localhost', '127.0.0.1', '[::1]'].includes(location.hostname);
const lan = ref([]);
const loaded = ref(!isHostMachine);
const svg = ref('');

onMounted(async () => {
  if (!isHostMachine) return;
  try { lan.value = (await (await fetch('/api/info')).json()).lan || []; } catch {}
  loaded.value = true;
});

const base = computed(() => (isHostMachine ? lan.value[0] || '' : location.origin));
const room = computed(() => props.room.trim().toUpperCase().replace(/[^A-Z0-9]/g, ''));
const url = computed(() => (base.value ? base.value + (room.value ? `/?sala=${room.value}` : '/') : ''));
const pretty = computed(() => url.value.replace(/^https?:\/\//, ''));

watchEffect(async () => {
  if (!url.value) return (svg.value = '');
  svg.value = await QRCode.toString(url.value, { type: 'svg', margin: 1, color: { dark: '#2a1f0e', light: '#ffffff' } });
});
</script>

<template>
  <div v-if="loaded" class="invite panel">
    <h3>Convide os jogadores</h3>
    <div v-if="url" class="invite-body">
      <div class="qr" v-html="svg"></div>
      <div class="invite-text">
        <ol>
          <li v-if="isHostMachine">Conecte o celular ou computador na <b>mesma rede Wi-Fi</b> deste computador.</li>
          <li>Aponte a câmera do celular para o QR code<span v-if="room">, ou digite o endereço abaixo</span>:</li>
        </ol>
        <div class="invite-url">{{ pretty }}</div>
        <div v-if="isHostMachine && lan.length > 1" class="invite-alt">
          Não funcionou? Tente: <span v-for="u in lan.slice(1)" :key="u">{{ u.replace('http://', '') }} </span>
        </div>
        <ol start="3">
          <li>Escolha <b>Time da Ordem</b>, <b>Time do Caos</b> ou <b>Juiz(a)</b>.</li>
        </ol>
      </div>
    </div>
    <p v-else class="invite-warn">
      Este computador não está conectado a nenhuma rede. Conecte no Wi-Fi e recarregue a página
      para os alunos conseguirem entrar.
    </p>
  </div>
</template>
