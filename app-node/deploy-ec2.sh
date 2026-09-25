#!/usr/bin/env bash
# Envia o app para uma instância EC2 (Ubuntu 24.04) e sobe com Docker Compose na porta 80.
# Uso: ./deploy-ec2.sh caminho/da-chave.pem IP_PUBLICO_DA_EC2
set -euo pipefail

KEY="${1:?Informe o caminho da chave .pem}"
HOST="${2:?Informe o IP público da EC2}"
SSH="ssh -i $KEY -o StrictHostKeyChecking=accept-new ubuntu@$HOST"
DIR="$(cd "$(dirname "$0")" && pwd)"

chmod 400 "$KEY"

echo "==> Enviando arquivos para $HOST"
tar -C "$DIR" -czf - Dockerfile docker-compose.yml .dockerignore package.json server.js cards.js public \
  | $SSH 'mkdir -p ~/ordem-caos && tar -C ~/ordem-caos -xzf -'

echo "==> Instalando Docker (se necessário) e subindo o app"
$SSH 'bash -s' <<'EOF'
set -e
if ! command -v docker >/dev/null || ! docker compose version >/dev/null 2>&1; then
  sudo apt-get update -qq
  sudo apt-get install -y -qq docker.io docker-compose-v2
  sudo systemctl enable --now docker
fi
cd ~/ordem-caos
sudo PORT=80 docker compose up -d --build
sleep 3
curl -fs localhost/health && echo " <- app respondendo"
EOF

echo "==> Pronto: http://$HOST/"
