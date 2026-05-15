package main

import (
	"crypto/rand"
	"embed"
	"log"
	"math/big"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

//go:embed index.html
var content embed.FS

const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

type Phase string

const (
	PhaseLobby      Phase = "lobby"
	PhaseRole       Phase = "role"
	PhaseTurns      Phase = "turns"
	PhaseDiscussion Phase = "discussion"
	PhaseVoting     Phase = "voting"
	PhaseResults    Phase = "results"
)

type Player struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Host     bool   `json:"host"`
	Ready    bool   `json:"ready"`
	Voted    bool   `json:"voted"`
	Online   bool   `json:"online"`
	LastVote string `json:"-"`
}

type Room struct {
	Code        string
	Phase       Phase
	Category    string
	RoundLimit  int
	RoundNumber int
	Word        string
	ImpostorID  string
	Players     map[string]*Player
	Order       []string
	TurnIndex   int
	Votes       map[string]string
}

type Client struct {
	id   string
	conn *websocket.Conn
	room *Room
}

type Server struct {
	mu      sync.Mutex
	rooms   map[string]*Room
	clients map[*Client]bool
}

type Message struct {
	Type     string `json:"type"`
	RoomCode string `json:"roomCode,omitempty"`
	Name     string `json:"name,omitempty"`
	Category string `json:"category,omitempty"`
	Rounds   int    `json:"rounds,omitempty"`
	TargetID string `json:"targetId,omitempty"`
	PlayerID string `json:"playerId,omitempty"`
}

var words = map[string][]string{
	"Random":  {"pizza", "beach", "dragon", "school", "soccer", "banana", "castle", "music", "robot", "winter"},
	"Food":    {"pizza", "burger", "taco", "banana", "pasta", "cookie", "sushi", "popcorn", "cheese", "ice cream"},
	"Animals": {"lion", "penguin", "elephant", "shark", "giraffe", "monkey", "rabbit", "turtle", "dolphin", "zebra"},
	"Places":  {"beach", "school", "park", "castle", "airport", "forest", "museum", "stadium", "library", "zoo"},
	"Movies":  {"superhero", "wizard", "alien", "princess", "pirate", "detective", "monster", "robot", "dragon", "spy"},
	"Objects": {"phone", "chair", "backpack", "lamp", "pencil", "clock", "bicycle", "camera", "blanket", "key"},
	"Sports":  {"soccer", "baseball", "basketball", "tennis", "swimming", "running", "boxing", "golf", "skating", "volleyball"},
	"School":  {"teacher", "homework", "pencil", "recess", "math", "book", "desk", "science", "bus", "cafeteria"},
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func main() {
	s := &Server{rooms: map[string]*Room{}, clients: map[*Client]bool{}}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFileFS(w, r, content, "index.html")
	})
	http.HandleFunc("/ws", s.handleWS)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	log.Printf("Impostor Game running at http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	c := &Client{id: randomID(12), conn: conn}
	s.mu.Lock()
	s.clients[c] = true
	s.mu.Unlock()
	c.send(map[string]any{"type": "connected", "playerId": c.id})

	defer s.disconnect(c)
	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		s.handleMessage(c, msg)
	}
}

func (s *Server) handleMessage(c *Client, msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch msg.Type {
	case "resume":
		code := strings.ToLower(strings.TrimSpace(msg.RoomCode))
		playerID := strings.TrimSpace(msg.PlayerID)
		room := s.rooms[code]
		if room == nil {
			c.sendError("Room not found. Please create or join a room.")
			return
		}
		p := room.Players[playerID]
		if p == nil {
			c.sendError("Could not restore your seat. Please join again.")
			return
		}
		c.id = playerID
		c.room = room
		p.Online = true
		s.broadcast(room, p.Name+" reconnected.")

	case "createRoom":
		name := cleanName(msg.Name)
		if name == "" {
			c.sendError("Please enter your name.")
			return
		}
		code := s.newRoomCode()
		room := &Room{Code: code, Phase: PhaseLobby, Category: "Random", RoundLimit: 1, Players: map[string]*Player{}, Votes: map[string]string{}}
		p := &Player{ID: c.id, Name: name, Host: true, Online: true}
		room.Players[c.id] = p
		room.Order = []string{c.id}
		s.rooms[code] = room
		c.room = room
		s.broadcast(room, "Room created.")

	case "joinRoom":
		name := cleanName(msg.Name)
		code := strings.ToLower(strings.TrimSpace(msg.RoomCode))
		room := s.rooms[code]
		if room == nil {
			c.sendError("Room not found.")
			return
		}
		if room.Phase != PhaseLobby {
			c.sendError("This game already started. Wait for the next round.")
			return
		}
		if name == "" {
			c.sendError("Please enter your name.")
			return
		}
		for _, p := range room.Players {
			if strings.EqualFold(p.Name, name) {
				c.sendError("That name is already taken in this room.")
				return
			}
		}
		room.Players[c.id] = &Player{ID: c.id, Name: name, Online: true}
		room.Order = append(room.Order, c.id)
		c.room = room
		s.broadcast(room, name+" joined.")

	case "setCategory":
		if !s.isHost(c) || c.room.Phase != PhaseLobby {
			return
		}
		if _, ok := words[msg.Category]; ok {
			c.room.Category = msg.Category
			s.broadcast(c.room, "Category changed.")
		}

	case "setRounds":
		if !s.isHost(c) || c.room.Phase != PhaseLobby {
			return
		}
		if msg.Rounds >= 1 && msg.Rounds <= 5 {
			c.room.RoundLimit = msg.Rounds
			s.broadcast(c.room, "Rounds changed.")
		}

	case "startGame":
		if !s.isHost(c) || c.room.Phase != PhaseLobby || len(c.room.Players) < 3 {
			return
		}
		room := c.room
		room.RoundNumber = 1
		s.startRound(room)
		s.broadcast(room, "Game started.")

	case "ready":
		if c.room == nil || c.room.Phase != PhaseRole {
			return
		}
		c.room.Players[c.id].Ready = true
		all := true
		for _, id := range c.room.Order {
			if !c.room.Players[id].Ready {
				all = false
			}
		}
		if all {
			c.room.Phase = PhaseTurns
		}
		s.broadcast(c.room, "")

	case "doneTurn":
		if c.room == nil || c.room.Phase != PhaseTurns || c.room.Order[c.room.TurnIndex] != c.id {
			return
		}
		c.room.TurnIndex++
		if c.room.TurnIndex >= len(c.room.Order) {
			c.room.Phase = PhaseDiscussion
		}
		s.broadcast(c.room, "")

	case "startVoting":
		if !s.isHost(c) || c.room.Phase != PhaseDiscussion {
			return
		}
		c.room.Phase = PhaseVoting
		s.broadcast(c.room, "")

	case "vote":
		if c.room == nil || c.room.Phase != PhaseVoting || c.room.Players[msg.TargetID] == nil {
			return
		}
		p := c.room.Players[c.id]
		if p.Voted {
			return
		}
		p.Voted = true
		p.LastVote = msg.TargetID
		c.room.Votes[c.id] = msg.TargetID
		if len(c.room.Votes) == len(c.room.Players) {
			c.room.Phase = PhaseResults
		}
		s.broadcast(c.room, "")

	case "playAgain":
		if !s.isHost(c) || c.room.Phase != PhaseResults {
			return
		}
		room := c.room
		if room.RoundNumber < room.RoundLimit {
			room.RoundNumber++
			s.startRound(room)
			s.broadcast(room, "Next round started.")
			return
		}
		room.RoundNumber = 0
		s.resetToLobby(room)
		s.broadcast(room, "All rounds complete.")

	case "backLobby":
		if !s.isHost(c) || c.room.Phase != PhaseResults {
			return
		}
		room := c.room
		room.RoundNumber = 0
		s.resetToLobby(room)
		s.broadcast(room, "Back to lobby.")
	}
}

func (s *Server) startRound(room *Room) {
	room.Phase = PhaseRole
	room.Word = randomWord(room.Category)
	room.ImpostorID = room.Order[randomInt(len(room.Order))]
	room.TurnIndex = 0
	room.Votes = map[string]string{}
	for _, p := range room.Players {
		p.Ready = false
		p.Voted = false
		p.LastVote = ""
	}
}

func (s *Server) resetToLobby(room *Room) {
	room.Phase = PhaseLobby
	room.Word = ""
	room.ImpostorID = ""
	room.TurnIndex = 0
	room.Votes = map[string]string{}
	for _, p := range room.Players {
		p.Ready = false
		p.Voted = false
		p.LastVote = ""
	}
}

func (s *Server) disconnect(c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, c)
	if c.room != nil {
		stillConnected := false
		for other := range s.clients {
			if other != c && other.room == c.room && other.id == c.id {
				stillConnected = true
				break
			}
		}
		if !stillConnected {
			if p := c.room.Players[c.id]; p != nil {
				p.Online = false
			}
			s.broadcast(c.room, "")
		}
	}
	c.conn.Close()
}

func (s *Server) broadcast(room *Room, toast string) {
	for c := range s.clients {
		if c.room == room {
			c.send(s.viewFor(c, room, toast))
		}
	}
}

func (s *Server) viewFor(c *Client, room *Room, toast string) map[string]any {
	players := make([]*Player, 0, len(room.Order))
	for _, id := range room.Order {
		if p := room.Players[id]; p != nil {
			players = append(players, p)
		}
	}
	currentTurnID := ""
	if room.Phase == PhaseTurns && room.TurnIndex < len(room.Order) {
		currentTurnID = room.Order[room.TurnIndex]
	}
	voteSummary := []map[string]string{}
	if room.Phase == PhaseResults {
		for voter, target := range room.Votes {
			voteSummary = append(voteSummary, map[string]string{"voter": room.Players[voter].Name, "target": room.Players[target].Name})
		}
		sort.Slice(voteSummary, func(i, j int) bool { return voteSummary[i]["voter"] < voteSummary[j]["voter"] })
	}
	return map[string]any{
		"type":          "state",
		"toast":         toast,
		"playerId":      c.id,
		"roomCode":      room.Code,
		"phase":         room.Phase,
		"category":      room.Category,
		"categories":    categories(),
		"roundLimit":    room.RoundLimit,
		"roundNumber":   room.RoundNumber,
		"players":       players,
		"isHost":        room.Players[c.id] != nil && room.Players[c.id].Host,
		"secretWord":    privateWord(c, room),
		"isImpostor":    room.ImpostorID == c.id,
		"currentTurnId": currentTurnID,
		"turnIndex":     room.TurnIndex,
		"impostorName":  playerName(room, room.ImpostorID),
		"resultWord":    resultWord(room),
		"playersWin":    playersWin(room),
		"voteSummary":   voteSummary,
	}
}

func privateWord(c *Client, room *Room) string {
	if (room.Phase == PhaseRole || room.Phase == PhaseTurns) && room.ImpostorID != c.id {
		return room.Word
	}
	return ""
}

func resultWord(room *Room) string {
	if room.Phase == PhaseResults {
		return room.Word
	}
	return ""
}

func playersWin(room *Room) bool {
	if room.Phase != PhaseResults {
		return false
	}
	counts := map[string]int{}
	for _, target := range room.Votes {
		counts[target]++
	}
	maxVotes := 0
	for _, count := range counts {
		if count > maxVotes {
			maxVotes = count
		}
	}
	return counts[room.ImpostorID] == maxVotes && maxVotes > 0
}

func playerName(room *Room, id string) string {
	if p := room.Players[id]; p != nil {
		return p.Name
	}
	return ""
}

func (s *Server) isHost(c *Client) bool {
	return c.room != nil && c.room.Players[c.id] != nil && c.room.Players[c.id].Host
}

func (s *Server) newRoomCode() string {
	for {
		code := randomID(3)
		if s.rooms[code] == nil {
			return code
		}
	}
}

func (c *Client) send(v any) {
	_ = c.conn.WriteJSON(v)
}

func (c *Client) sendError(message string) {
	c.send(map[string]any{"type": "error", "message": message})
}

func cleanName(name string) string {
	name = strings.TrimSpace(name)
	if len(name) > 20 {
		name = name[:20]
	}
	return name
}

func categories() []string {
	return []string{"Random", "Food", "Animals", "Places", "Movies", "Objects", "Sports", "School"}
}

func randomWord(category string) string {
	list := words[category]
	if len(list) == 0 {
		list = words["Random"]
	}
	return list[randomInt(len(list))]
}

func randomID(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[randomInt(len(alphabet))]
	}
	return string(b)
}

func randomInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(err)
	}
	return int(n.Int64())
}
