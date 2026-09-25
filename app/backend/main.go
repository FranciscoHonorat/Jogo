package main

// Ordem & Caos — servidor HTTP.
// Tempo real via Server-Sent Events (GET /events); ações via POST /api/action.
// O front-end Vue compilado (dist/) vai embutido no binário.

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

//go:embed all:dist
var distFS embed.FS

var (
	roomsMu sync.Mutex
	rooms   = map[string]*Room{}
)

func getRoom(code string, create bool) *Room {
	roomsMu.Lock()
	defer roomsMu.Unlock()
	r := rooms[code]
	if r == nil && create {
		r = newRoom(code)
		rooms[code] = r
	}
	return r
}

// ---------- utilitários ----------

func clean(s string, n int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > n {
		s = string([]rune(s)[:n])
	}
	return s
}

var notCode = regexp.MustCompile(`[^A-Z0-9]`)

func roomCode(s string) string {
	c := notCode.ReplaceAllString(strings.ToUpper(clean(s, 20)), "")
	if c == "" {
		return "SALA1"
	}
	return c
}

func newToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func readBody(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 100_000)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		fail(w, http.StatusBadRequest, "JSON inválido")
		return false
	}
	return true
}

// ---------- handlers ----------

func handleJoin(w http.ResponseWriter, r *http.Request) {
	var b struct{ Room, Name, Role, TeamName, Token string }
	if !readBody(w, r, &b) {
		return
	}
	switch b.Role {
	case "ordem", "caos", "juiz", "plateia":
	default:
		b.Role = ""
	}
	name := clean(b.Name, 40)
	if b.Role == "" || name == "" {
		fail(w, http.StatusBadRequest, "Informe nome e papel.")
		return
	}
	token := b.Token
	if len(token) < 16 || len(token) > 64 {
		token = newToken()
	}

	room := getRoom(roomCode(b.Room), true)
	room.mu.Lock()
	defer room.mu.Unlock()
	if team := room.teams[b.Role]; team != nil && team.Name == "" {
		tn := clean(b.TeamName, 40)
		if tn == "" {
			fail(w, http.StatusBadRequest, "Informe o nome do time.")
			return
		}
		team.Name = tn
	}
	if _, ok := room.players[token]; !ok {
		room.order = append(room.order, token)
	}
	room.players[token] = &Player{Name: name, Role: b.Role}
	room.broadcast()
	writeJSON(w, http.StatusOK, map[string]string{"token": token, "room": room.code})
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	var b struct{ Room, Token, Action, Choice string }
	if !readBody(w, r, &b) {
		return
	}
	room := getRoom(roomCode(b.Room), false)
	if room == nil {
		fail(w, http.StatusNotFound, "Sala não encontrada.")
		return
	}
	room.mu.Lock()
	defer room.mu.Unlock()
	if err := room.act(b.Token, b.Action, b.Choice); err != nil {
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	room.broadcast()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleTeams(w http.ResponseWriter, r *http.Request) {
	room := getRoom(roomCode(r.URL.Query().Get("room")), false)
	if room == nil {
		writeJSON(w, http.StatusOK, map[string]Team{"ordem": {}, "caos": {}})
		return
	}
	room.mu.Lock()
	defer room.mu.Unlock()
	writeJSON(w, http.StatusOK, room.teams)
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	room := getRoom(roomCode(r.URL.Query().Get("room")), false)
	token := r.URL.Query().Get("token")
	if room == nil {
		http.NotFound(w, r)
		return
	}
	room.mu.Lock()
	_, member := room.players[token]
	room.mu.Unlock()
	if !member {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming não suportado", http.StatusInternalServerError)
		return
	}

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	ch := make(chan []byte, 4)
	room.mu.Lock()
	if room.subs[token] == nil {
		room.subs[token] = map[chan []byte]struct{}{}
	}
	room.subs[token][ch] = struct{}{}
	first, _ := json.Marshal(room.view(token))
	room.mu.Unlock()

	defer func() {
		room.mu.Lock()
		delete(room.subs[token], ch)
		if len(room.subs[token]) == 0 {
			delete(room.subs, token)
		}
		room.mu.Unlock()
	}()

	w.Write([]byte("retry: 2000\n\n"))
	w.Write([]byte("data: " + string(first) + "\n\n"))
	flusher.Flush()

	ping := time.NewTicker(15 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			if _, err := w.Write([]byte("data: " + string(msg) + "\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case <-ping.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func main() {
	setupConsole()
	port := os.Getenv("PORT")
	fixedPort := port != ""
	if port == "" {
		port = "3000"
	}
	// no computador do professor (fora do Docker): abre o navegador e mostra instruções
	local := !inContainer() && os.Getenv("NO_BROWSER") == ""

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil && local && alreadyRunning(port) {
		// deram dois cliques de novo: o jogo já está aberto, só abre o navegador
		fmt.Println("O jogo já está aberto neste computador. Abrindo o navegador...")
		openBrowser("http://localhost:" + port + "/")
		time.Sleep(3 * time.Second)
		return
	}
	for p := 3001; err != nil && !fixedPort && p <= 3010; p++ {
		port = strconv.Itoa(p) // porta ocupada por outro programa: tenta a próxima
		ln, err = net.Listen("tcp", ":"+port)
	}
	if err != nil {
		fmt.Println("Não foi possível iniciar o jogo:", err)
		pauseBeforeExit()
		os.Exit(1)
	}

	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		log.Fatal(err)
	}
	static := http.FileServer(http.FS(dist))

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/join", handleJoin)
	mux.HandleFunc("POST /api/action", handleAction)
	mux.HandleFunc("GET /api/teams", handleTeams)
	mux.HandleFunc("GET /api/info", func(w http.ResponseWriter, r *http.Request) {
		lan := []string{}
		if !inContainer() {
			lan = lanURLs(port)
		}
		writeJSON(w, http.StatusOK, map[string]any{"lan": lan})
	})
	mux.HandleFunc("GET /events", handleEvents) // GET também atende HEAD
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	mux.Handle("GET /", static)

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		// sem WriteTimeout: as conexões SSE ficam abertas a partida inteira
	}
	if local {
		printWelcome(port, lanURLs(port))
		openBrowser("http://localhost:" + port + "/")
	} else {
		log.Printf("Ordem & Caos rodando em http://localhost:%s", port)
	}
	if err := srv.Serve(ln); err != nil {
		fmt.Println("O jogo parou:", err)
		pauseBeforeExit()
	}
}

// alreadyRunning confere se quem ocupa a porta é o próprio jogo.
func alreadyRunning(port string) bool {
	c := http.Client{Timeout: time.Second}
	r, err := c.Get("http://localhost:" + port + "/health")
	if err != nil {
		return false
	}
	r.Body.Close()
	return r.StatusCode == http.StatusOK
}
