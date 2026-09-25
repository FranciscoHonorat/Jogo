# 1) front-end Vue -> backend/dist
FROM node:22-alpine AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

# 2) servidor Go com o front embutido
FROM golang:1.24-alpine AS api
WORKDIR /src/backend
COPY backend/ ./
COPY --from=web /src/backend/dist ./dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /ordemcaos .

# 3) imagem final: só o binário
FROM alpine:3.21
RUN adduser -D -H app
USER app
COPY --from=api /ordemcaos /ordemcaos
ENV PORT=3000 NO_BROWSER=1
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost:3000/health || exit 1
CMD ["/ordemcaos"]
