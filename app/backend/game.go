package main

// Regras da partida: fases, turnos, votação e o estado enviado a cada cliente.

import (
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"errors"
	"math/big"
	"sync"
	"time"
)

const (
	thinkDur = 30 * time.Second
	speakDur = 120 * time.Second
	judgeDur = 180 * time.Second
)

type Card struct {
	ID        int    `json:"id"`
	Nome      string `json:"nome"`
	Tipo      string `json:"tipo"`
	Valor     string `json:"valor"`
	Especial  bool   `json:"especial,omitempty"`
	Argumente string `json:"argumente"`
	Narracao  string `json:"narracao"`
}

//go:embed cards.json
var cardsJSON []byte

var decks struct {
	Ordem []Card `json:"ordem"`
	Caos  []Card `json:"caos"`
}

func init() {
	if err := json.Unmarshal(cardsJSON, &decks); err != nil {
		panic("cards.json inválido: " + err.Error())
	}
}

func shuffled(src []Card) []Card {
	a := make([]Card, len(src))
	for i, c := range src {
		c.ID = i
		a[i] = c
	}
	for i := len(a) - 1; i > 0; i-- {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		j := int(n.Int64())
		a[i], a[j] = a[j], a[i]
	}
	return a
}

type Player struct {
	Name string `json:"name"`
	Role string `json:"role"` // ordem | caos | juiz | plateia
}

type Team struct {
	Name string `json:"name"`
}

type Tally struct {
	Ordem  int `json:"ordem"`
	Caos   int `json:"caos"`
	Empate int `json:"empate"`
}

type Result struct {
	Winner string `json:"winner"`
	Tally  Tally  `json:"tally"`
}

type HistoryEntry struct {
	Round  int    `json:"round"`
	Ordem  string `json:"ordem"`
	Caos   string `json:"caos"`
	Winner string `json:"winner"`
	Tally  Tally  `json:"tally"`
}

type Room struct {
	mu      sync.Mutex
	code    string
	players map[string]*Player // token -> jogador
	order   []string           // ordem de jogadores na sala, para o roster sair estável
	teams   map[string]*Team
	subs    map[string]map[chan []byte]struct{} // token -> conexões SSE abertas
	game
}

// game é tudo que é zerado em "Nova partida".
type game struct {
	phase   string // lobby | draw | think | speak | judging | result | finished
	round   int
	turn    int      // índice em turnOrder (0 ou 1)
	turnOrd []string // quem joga primeiro nesta rodada
	endsAt  *int64
	timer   *time.Timer
	seq     int // incrementa a cada troca de fase; invalida timers antigos
	decks   map[string][]Card
	current map[string]*Card
	votes   map[string]string // token -> ordem | caos | empate
	last    *Result
	score   Tally
	history []HistoryEntry
}

func freshGame() game {
	return game{
		phase:   "lobby",
		turnOrd: []string{"ordem", "caos"},
		decks:   map[string][]Card{"ordem": shuffled(decks.Ordem), "caos": shuffled(decks.Caos)},
		current: map[string]*Card{"ordem": nil, "caos": nil},
		votes:   map[string]string{},
		history: []HistoryEntry{},
	}
}

func newRoom(code string) *Room {
	return &Room{
		code:    code,
		players: map[string]*Player{},
		teams:   map[string]*Team{"ordem": {}, "caos": {}},
		subs:    map[string]map[chan []byte]struct{}{},
		game:    freshGame(),
	}
}

func (r *Room) activeTeam() string { return r.turnOrd[r.turn] }

func (r *Room) judges() []string {
	var out []string
	for _, t := range r.order {
		if r.players[t].Role == "juiz" {
			out = append(out, t)
		}
	}
	return out
}

// setPhase troca de fase e, se houver duração, agenda o avanço automático.
func (r *Room) setPhase(phase string, d time.Duration) {
	if r.timer != nil {
		r.timer.Stop()
		r.timer = nil
	}
	r.seq++
	r.phase = phase
	r.endsAt = nil
	if d > 0 {
		end := time.Now().Add(d).UnixMilli()
		r.endsAt = &end
		seq := r.seq
		r.timer = time.AfterFunc(d, func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			if r.seq != seq {
				return // a fase já mudou por uma ação
			}
			r.onTimeout()
			r.broadcast()
		})
	}
}

func (r *Room) onTimeout() {
	switch r.phase {
	case "think":
		r.setPhase("speak", speakDur)
	case "speak":
		r.endTurn()
	case "judging":
		r.closeVoting()
	}
}

func (r *Room) startRound() {
	r.round++
	// alterna quem começa: rodadas ímpares a Ordem, pares o Caos
	if r.round%2 == 1 {
		r.turnOrd = []string{"ordem", "caos"}
	} else {
		r.turnOrd = []string{"caos", "ordem"}
	}
	r.turn = 0
	r.current = map[string]*Card{"ordem": nil, "caos": nil}
	r.votes = map[string]string{}
	r.last = nil
	r.setPhase("draw", 0)
}

func (r *Room) endTurn() {
	if r.turn == 0 {
		r.turn = 1
		r.setPhase("draw", 0)
	} else {
		r.setPhase("judging", judgeDur)
	}
}

func (r *Room) closeVoting() {
	var t Tally
	for _, v := range r.votes {
		switch v {
		case "ordem":
			t.Ordem++
		case "caos":
			t.Caos++
		case "empate":
			t.Empate++
		}
	}
	winner := "empate"
	if t.Ordem > t.Caos {
		winner = "ordem"
	} else if t.Caos > t.Ordem {
		winner = "caos"
	}
	switch winner {
	case "ordem":
		r.score.Ordem++
	case "caos":
		r.score.Caos++
	default:
		r.score.Empate++
	}
	r.last = &Result{Winner: winner, Tally: t}
	name := func(c *Card) string {
		if c == nil {
			return ""
		}
		return c.Nome
	}
	r.history = append(r.history, HistoryEntry{
		Round: r.round, Ordem: name(r.current["ordem"]), Caos: name(r.current["caos"]), Winner: winner, Tally: t,
	})
	r.setPhase("result", 0)
}

// act aplica uma ação de um jogador. Chamar com r.mu travado.
func (r *Room) act(token, action, choice string) error {
	me := r.players[token]
	if me == nil {
		return errors.New("Você não está nesta sala.")
	}
	isJudge := me.Role == "juiz"

	switch action {
	case "start":
		if !isJudge {
			return errors.New("Só juízes podem iniciar.")
		}
		if r.phase != "lobby" {
			return errors.New("A partida já começou.")
		}
		if r.teams["ordem"].Name == "" || r.teams["caos"].Name == "" {
			return errors.New("Os dois times precisam entrar antes.")
		}
		r.startRound()

	case "draw":
		team := r.activeTeam()
		if r.phase != "draw" || me.Role != team {
			return errors.New("Não é a vez do seu time de puxar.")
		}
		d := r.decks[team]
		if len(d) == 0 {
			return errors.New("Deck vazio.")
		}
		card := d[len(d)-1]
		r.decks[team] = d[:len(d)-1]
		r.current[team] = &card
		r.setPhase("think", thinkDur)

	case "skip": // o time encerra o tempo de pensar/falar antes do fim
		if r.phase != "think" && r.phase != "speak" {
			return errors.New("Nada para avançar agora.")
		}
		if me.Role != r.activeTeam() {
			return errors.New("Não é a vez do seu time.")
		}
		if r.phase == "think" {
			r.setPhase("speak", speakDur)
		} else {
			r.endTurn()
		}

	case "vote":
		if !isJudge {
			return errors.New("Só juízes votam.")
		}
		if r.phase != "judging" {
			return errors.New("A votação não está aberta.")
		}
		if choice != "ordem" && choice != "caos" && choice != "empate" {
			return errors.New("Voto inválido.")
		}
		r.votes[token] = choice
		all := true
		for _, j := range r.judges() {
			if r.votes[j] == "" {
				all = false
				break
			}
		}
		if all {
			r.closeVoting()
		}

	case "close":
		if !isJudge || r.phase != "judging" {
			return errors.New("Não é possível encerrar agora.")
		}
		r.closeVoting()

	case "next":
		if !isJudge || r.phase != "result" {
			return errors.New("Não é possível avançar agora.")
		}
		if len(r.decks["ordem"]) == 0 || len(r.decks["caos"]) == 0 {
			r.setPhase("finished", 0)
		} else {
			r.startRound()
		}

	case "end":
		if !isJudge {
			return errors.New("Só juízes podem encerrar.")
		}
		r.setPhase("finished", 0)

	case "reset":
		if !isJudge {
			return errors.New("Só juízes podem reiniciar.")
		}
		r.setPhase("lobby", 0) // para o timer atual
		seq := r.seq
		r.game = freshGame()
		r.seq = seq
	default:
		return errors.New("Ação desconhecida.")
	}
	return nil
}

// ---------- estado enviado ao cliente ----------

type View struct {
	Now         int64               `json:"now"`
	Me          *Player             `json:"me"`
	Code        string              `json:"code"`
	Teams       map[string]*Team    `json:"teams"`
	Roster      map[string][]string `json:"roster"`
	Phase       string              `json:"phase"`
	Round       int                 `json:"round"`
	TotalRounds int                 `json:"totalRounds"`
	Active      *string             `json:"active"`
	Order       []string            `json:"order"`
	EndsAt      *int64              `json:"endsAt"`
	Durations   map[string]int64    `json:"durations"`
	DeckLeft    map[string]int      `json:"deckLeft"`
	Current     map[string]*Card    `json:"current"`
	VotesIn     int                 `json:"votesIn"`
	JudgeCount  int                 `json:"judgeCount"`
	MyVote      *string             `json:"myVote"`
	LastResult  *Result             `json:"lastResult"`
	Score       Tally               `json:"score"`
	History     []HistoryEntry      `json:"history"`
}

func (r *Room) view(token string) View {
	roster := map[string][]string{"ordem": {}, "caos": {}, "juiz": {}, "plateia": {}}
	for _, t := range r.order {
		p := r.players[t]
		roster[p.Role] = append(roster[p.Role], p.Name)
	}
	var active *string
	if r.phase != "lobby" && r.phase != "finished" {
		a := r.activeTeam()
		active = &a
	}
	var myVote *string
	if v, ok := r.votes[token]; ok {
		myVote = &v
	}
	return View{
		Now:         time.Now().UnixMilli(),
		Me:          r.players[token],
		Code:        r.code,
		Teams:       r.teams,
		Roster:      roster,
		Phase:       r.phase,
		Round:       r.round,
		TotalRounds: len(decks.Ordem),
		Active:      active,
		Order:       r.turnOrd,
		EndsAt:      r.endsAt,
		Durations:   map[string]int64{"think": thinkDur.Milliseconds(), "speak": speakDur.Milliseconds(), "judging": judgeDur.Milliseconds()},
		DeckLeft:    map[string]int{"ordem": len(r.decks["ordem"]), "caos": len(r.decks["caos"])},
		Current:     r.current,
		VotesIn:     len(r.votes),
		JudgeCount:  len(r.judges()),
		MyVote:      myVote,
		LastResult:  r.last,
		Score:       r.score,
		History:     r.history,
	}
}

// broadcast envia o estado atualizado a todas as conexões. Chamar com r.mu travado.
func (r *Room) broadcast() {
	for token, chans := range r.subs {
		msg, err := json.Marshal(r.view(token))
		if err != nil {
			continue
		}
		for ch := range chans {
			push(ch, msg)
		}
	}
}

// push nunca bloqueia: se o cliente estiver lento, descarta o estado antigo e fica só com o mais novo.
func push(ch chan []byte, msg []byte) {
	select {
	case ch <- msg:
	default:
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- msg:
		default:
		}
	}
}
