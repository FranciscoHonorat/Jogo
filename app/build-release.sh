#!/usr/bin/env bash
# Gera os programas "clique duas vezes" para Windows, Mac e Linux em release/.
# Cada .zip leva o executável + o guia em PDF + um LEIA-ME.
# Uso: ./build-release.sh
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
OUT="$DIR/release"
GUIA="$DIR/docs/Guia - Como jogar.pdf"

echo "==> Compilando o front-end (Vue)"
(cd "$DIR/frontend" && npm ci --no-audit --no-fund --silent && npm run build --silent)

rm -rf "$OUT" && mkdir -p "$OUT"

# nome do zip | GOOS | GOARCH | nome do executável | texto do LEIA-ME
build() {
  local zipname=$1 goos=$2 goarch=$3 exe=$4 how=$5
  local tmp="$OUT/.tmp/$zipname"
  mkdir -p "$tmp"
  echo "==> $zipname"
  (cd "$DIR/backend" && CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build -trimpath -ldflags="-s -w" -o "$tmp/$exe" .)
  chmod +x "$tmp/$exe"
  [ -f "$GUIA" ] && cp "$GUIA" "$tmp/"
  cat > "$tmp/LEIA-ME.txt" <<EOF
ORDEM & CAOS - A Metamorfose

COMO ABRIR
$how

Uma janela com textos vai aparecer: NAO FECHE essa janela durante o jogo.
O navegador abre sozinho, com um QR code para os alunos entrarem pelo celular.
Os alunos precisam estar na MESMA rede Wi-Fi deste computador.

Se o navegador nao abrir, digite no navegador: http://localhost:3000

Passo a passo com imagens: "Guia - Como jogar.pdf"
EOF
  (cd "$OUT/.tmp" && zip -qr "$OUT/$zipname.zip" "$zipname")
}

build "OrdemECaos-Windows" windows amd64 "OrdemECaos.exe" \
"1. Clique com o botao direito neste .zip e escolha \"Extrair tudo\".
2. Abra a pasta extraida e de dois cliques em OrdemECaos.exe.
3. Se aparecer \"O Windows protegeu o computador\": clique em
   \"Mais informacoes\" e depois em \"Executar assim mesmo\".
4. Se o Firewall perguntar, clique em \"Permitir acesso\"."

build "OrdemECaos-Mac-AppleSilicon" darwin arm64 "OrdemECaos" \
"(Para Macs com chip M1, M2, M3 ou M4. Mac antigo com Intel: use a versao Mac-Intel.)
1. De dois cliques neste .zip para extrair.
2. Clique com o botao direito em OrdemECaos e escolha \"Abrir\", depois \"Abrir\" de novo.
3. Se o Mac nao deixar: abra Ajustes do Sistema > Privacidade e Seguranca,
   role ate o fim e clique em \"Abrir Mesmo Assim\"."

build "OrdemECaos-Mac-Intel" darwin amd64 "OrdemECaos" \
"(Para Macs com processador Intel. Mac com chip M1/M2/M3/M4: use a versao Mac-AppleSilicon.)
1. De dois cliques neste .zip para extrair.
2. Clique com o botao direito em OrdemECaos e escolha \"Abrir\", depois \"Abrir\" de novo.
3. Se o Mac nao deixar: abra Ajustes do Sistema > Privacidade e Seguranca,
   role ate o fim e clique em \"Abrir Mesmo Assim\"."

build "OrdemECaos-Linux" linux amd64 "OrdemECaos" \
"1. Extraia este .zip.
2. De dois cliques em OrdemECaos (ou rode ./OrdemECaos no terminal)."

rm -rf "$OUT/.tmp"
echo
echo "Pronto! Arquivos em release/:"
ls -lh "$OUT"
