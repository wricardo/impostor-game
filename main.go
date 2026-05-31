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
	PhaseLobby   Phase = "lobby"
	PhaseTurns   Phase = "turns"
	PhaseVoting  Phase = "voting"
	PhaseResults Phase = "results"
)

type Player struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Host      bool   `json:"host"`
	Voted     bool   `json:"voted"`
	Online    bool   `json:"online"`
	LastVote  string `json:"-"`
	DraftVote string `json:"-"`
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
	"Food":       {"pizza", "burger", "taco", "banana", "pasta", "cookie", "sushi", "popcorn", "cheese", "ice cream", "pancake", "apple", "sandwich", "noodles", "cupcake", "salad", "donut", "fries", "cereal", "watermelon", "waffle", "strawberry", "blueberry", "pineapple", "mango", "pretzel", "bagel", "muffin", "brownie", "nachos", "smoothie", "lemonade", "hot dog", "burrito", "ramen", "meatball", "toast", "yogurt", "avocado", "carrot"},
	"Animals":    {"lion", "penguin", "elephant", "shark", "giraffe", "monkey", "rabbit", "turtle", "dolphin", "zebra", "tiger", "panda", "koala", "horse", "frog", "owl", "bear", "kangaroo", "whale", "fox", "sloth", "otter", "seal", "parrot", "flamingo", "peacock", "raccoon", "hedgehog", "llama", "camel", "octopus", "seahorse", "butterfly", "ladybug", "bee", "snail", "goat", "sheep", "moose", "cheetah"},
	"Places":     {"beach", "school", "park", "hotel", "airport", "amusement park", "museum", "stadium", "science center", "zoo", "farm", "ice cream shop", "restaurant", "town square", "hospital", "movie theater", "aquarium", "bakery", "campsite", "theater", "carnival", "circus", "water park", "skate park", "arcade", "treehouse", "planetarium", "harbor", "train station", "fire station", "post office", "grocery store", "bookstore", "garden", "picnic area", "observatory", "lighthouse", "toy store", "orchard", "pet shop"},
	"Objects":    {"phone", "chair", "rubber duck", "lamp", "toy car", "clock", "bicycle", "camera", "blanket", "key", "umbrella", "toothbrush", "mirror", "balloon", "suitcase", "flashlight", "scissors", "helmet", "spoon", "towel", "kite", "yo-yo", "skateboard", "scooter", "teddy bear", "puzzle", "snow globe", "magnifying glass", "compass", "whistle", "binoculars", "paintbrush", "drum", "guitar", "microphone", "remote", "bookmark", "sticker", "marbles", "water bottle"},
	"Sports":     {"soccer", "baseball", "basketball", "tennis", "swimming", "running", "boxing", "golf", "skating", "volleyball", "football", "hockey", "gymnastics", "skiing", "surfing", "cycling", "bowling", "karate", "frisbee", "climbing", "badminton", "table tennis", "dodgeball", "kickball", "jump rope", "archery", "canoeing", "kayaking", "snowboarding", "sledding", "rollerblading", "cheerleading", "track", "fencing", "lacrosse", "softball", "rugby", "diving", "sailing", "mini golf"},
	"School":     {"teacher", "homework", "pencil", "recess", "math", "book", "desk", "science", "bus", "cafeteria", "backpack", "principal", "marker", "notebook", "classroom", "library", "quiz", "art", "music", "playground", "crayon", "glue", "ruler", "eraser", "whiteboard", "locker", "field trip", "spelling", "history", "geography", "reading", "experiment", "calendar", "bell", "folder", "worksheet", "computer", "assembly", "lunchbox", "storytime"},
	"Nature":     {"rainbow", "river", "mountain", "flower", "tree", "cloud", "ocean", "desert", "volcano", "waterfall", "sunshine", "moon", "stars", "leaf", "cave", "meadow", "thunder", "snow", "island", "forest", "breeze", "raindrop", "seashell", "coral", "sunset", "sunrise", "acorn", "pinecone", "mushroom", "fern", "cactus", "glacier", "lagoon", "pond", "stream", "comet", "meteor", "cliff", "valley", "wildflower"},
	"Jobs":       {"doctor", "chef", "firefighter", "artist", "pilot", "farmer", "professor", "dentist", "builder", "nurse", "scientist", "baker", "librarian", "police", "mechanic", "musician", "gardener", "coach", "mail carrier", "veterinarian", "astronaut", "park ranger", "zookeeper", "photographer", "author", "illustrator", "dancer", "actor", "engineer", "architect", "plumber", "electrician", "barber", "florist", "translator", "lifeguard", "referee", "bus driver", "tour guide", "meteorologist"},
	"Activities": {"camping", "painting", "singing", "dancing", "puzzle solving", "fishing", "cooking", "gardening", "hiking", "shopping", "drawing", "baking", "sleeping", "building", "cleaning", "jumping", "fort building", "traveling", "playing", "bubble blowing", "stargazing", "birdwatching", "picnicking", "hopscotch", "skipping", "juggling", "crafting", "collecting", "exploring", "storytelling", "roller skating", "kite flying", "treasure hunting", "puddle jumping", "sandcastle building", "origami", "caroling", "volunteering", "daydreaming", "drumming"},
	"Fantasy":    {"dragon", "castle", "wizard", "treasure", "unicorn", "knight", "fairy", "giant", "mermaid", "phoenix", "spell", "potion", "crown", "quest", "goblin", "elf", "crystal", "sword", "portal", "kingdom", "griffin", "pegasus", "troll", "sprite", "lantern", "map", "riddle", "enchanted forest", "magic carpet", "wishing well", "royal banquet", "secret door", "hidden cave", "golden apple", "moonstone", "starship", "time machine", "friendly ghost", "talking tree", "rainbow bridge"},
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

var chooseRandomInt = randomInt

func main() {
	if err := runCLI(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func runServer(port string) error {
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8000"
	}
	log.Printf("Impostor Game running at http://localhost:%s", port)
	return http.ListenAndServe(":"+port, newServerHandler())
}

func newServerHandler() http.Handler {
	s := &Server{rooms: map[string]*Room{}, clients: map[*Client]bool{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFileFS(w, r, content, "index.html")
	})
	mux.HandleFunc("/ws", s.handleWS)
	return mux
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
		if isCategory(msg.Category) {
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

	case "removePlayer":
		if !s.isHost(c) || c.room.Phase != PhaseLobby || msg.TargetID == c.id {
			return
		}
		room := c.room
		p := room.Players[msg.TargetID]
		if p == nil || p.Host {
			return
		}
		delete(room.Players, msg.TargetID)
		room.Order = removeID(room.Order, msg.TargetID)
		for other := range s.clients {
			if other.room == room && other.id == msg.TargetID {
				other.room = nil
				other.send(map[string]any{"type": "removed", "message": "The host removed you from the room."})
			}
		}
		s.broadcast(room, p.Name+" was removed.")

	case "startGame":
		if !s.isHost(c) {
			return
		}
		if c.room.Phase == PhaseLobby {
			if len(c.room.Players) < 3 {
				return
			}
			room := c.room
			room.RoundNumber = 1
			s.startRound(room)
			s.broadcast(room, "Game started.")
		}

	case "doneTurn":
		if c.room == nil || c.room.Phase != PhaseTurns || c.room.Order[c.room.TurnIndex] != c.id {
			return
		}
		c.room.TurnIndex++
		if c.room.TurnIndex >= len(c.room.Order) {
			c.room.Phase = PhaseVoting
		}
		s.broadcast(c.room, "")

	case "draftVote":
		if c.room == nil || c.room.Players[msg.TargetID] == nil {
			return
		}
		p := c.room.Players[c.id]
		if p == nil || (c.room.Phase != PhaseTurns && (c.room.Phase != PhaseVoting || p.Voted)) {
			return
		}
		p.DraftVote = msg.TargetID
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
	room.Phase = PhaseTurns
	room.Word = randomWord(room.Category)
	rotateStartingPlayer(room, chooseRandomInt(len(room.Order)))
	room.ImpostorID = room.Order[chooseRandomInt(len(room.Order))]
	room.TurnIndex = 0
	room.Votes = map[string]string{}
	for _, p := range room.Players {
		p.Voted = false
		p.LastVote = ""
		p.DraftVote = ""
	}
}

func (s *Server) resetToLobby(room *Room) {
	room.Phase = PhaseLobby
	room.Word = ""
	room.ImpostorID = ""
	room.TurnIndex = 0
	room.Votes = map[string]string{}
	for _, p := range room.Players {
		p.Voted = false
		p.LastVote = ""
		p.DraftVote = ""
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
		"draftVote":     room.Players[c.id].DraftVote,
		"impostorName":  playerName(room, room.ImpostorID),
		"resultWord":    resultWord(room),
		"playersWin":    playersWin(room),
		"voteSummary":   voteSummary,
	}
}

func privateWord(c *Client, room *Room) string {
	if room.Phase == PhaseTurns && room.ImpostorID != c.id {
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

func rotateStartingPlayer(room *Room, start int) {
	if len(room.Order) == 0 || start <= 0 || start >= len(room.Order) {
		return
	}
	rotated := append([]string{}, room.Order[start:]...)
	rotated = append(rotated, room.Order[:start]...)
	room.Order = rotated
}

func removeID(ids []string, target string) []string {
	out := ids[:0]
	for _, id := range ids {
		if id != target {
			out = append(out, id)
		}
	}
	return out
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
	return append([]string{"Random"}, wordCategories()...)
}

func wordCategories() []string {
	return []string{"Food", "Animals", "Places", "Objects", "Sports", "School", "Nature", "Jobs", "Activities", "Fantasy"}
}

func isCategory(category string) bool {
	if category == "Random" {
		return true
	}
	_, ok := words[category]
	return ok
}

func randomWord(category string) string {
	if _, ok := words[category]; !ok {
		realCategories := wordCategories()
		category = realCategories[chooseRandomInt(len(realCategories))]
	}
	list := words[category]
	return list[chooseRandomInt(len(list))]
}

func randomID(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[chooseRandomInt(len(alphabet))]
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
