package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestReadRouteSupportsCodeQueryParam(t *testing.T) {
	content, err := os.ReadFile("index.html")
	if err != nil {
		t.Fatal(err)
	}
	js := string(content)
	if !strings.Contains(js, "params.get('room') || params.get('code')") {
		t.Fatalf("readRoute should use the code query param as a fallback for room prefill")
	}
	if !strings.Contains(js, "setRoute({ roomCode: route.roomCode })") {
		t.Fatalf("home screen should preserve room/code route so Join Room can prefill it")
	}
}

func TestStartRoundRandomizesStartingPlayer(t *testing.T) {
	oldChoose := chooseRandomInt
	defer func() { chooseRandomInt = oldChoose }()

	calls := []int{0, 2, 0} // word, starting player, impostor from rotated order
	chooseRandomInt = func(max int) int {
		if len(calls) == 0 {
			t.Fatalf("unexpected chooseRandomInt call with max %d", max)
		}
		v := calls[0]
		calls = calls[1:]
		if v >= max {
			t.Fatalf("test random value %d is outside max %d", v, max)
		}
		return v
	}

	room := testRoom()
	room.Category = "Food"
	(&Server{}).startRound(room)

	wantOrder := []string{"p3", "p1", "p2"}
	if strings.Join(room.Order, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("room.Order = %v, want %v", room.Order, wantOrder)
	}
	if room.Phase != PhaseRole {
		t.Fatalf("phase = %s, want %s", room.Phase, PhaseRole)
	}
	if room.TurnIndex != 0 || room.Order[room.TurnIndex] != "p3" {
		t.Fatalf("first turn = %q at index %d, want p3 at index 0", room.Order[room.TurnIndex], room.TurnIndex)
	}
}

func TestRandomWordSelectsFromARealCategory(t *testing.T) {
	oldChoose := chooseRandomInt
	defer func() { chooseRandomInt = oldChoose }()

	calls := []int{1, 2} // Animals, then elephant
	chooseRandomInt = func(max int) int {
		if len(calls) == 0 {
			t.Fatalf("unexpected chooseRandomInt call with max %d", max)
		}
		v := calls[0]
		calls = calls[1:]
		if v >= max {
			t.Fatalf("test random value %d is outside max %d", v, max)
		}
		return v
	}

	if got := randomWord("Random"); got != "elephant" {
		t.Fatalf("randomWord(Random) = %q, want elephant", got)
	}
	if _, ok := words["Random"]; ok {
		t.Fatalf("Random should not have its own word list")
	}
}

func TestRandomCategoryIsSelectable(t *testing.T) {
	room := testRoom()
	s := &Server{rooms: map[string]*Room{room.Code: room}, clients: map[*Client]bool{}}
	host := &Client{id: "p1", room: room}

	room.Category = "Food"
	s.handleMessage(host, Message{Type: "setCategory", Category: "Random"})

	if room.Category != "Random" {
		t.Fatalf("room.Category = %q, want Random", room.Category)
	}
}

func TestHostCanRemovePlayerFromLobby(t *testing.T) {
	room := testRoom()
	s := &Server{rooms: map[string]*Room{room.Code: room}, clients: map[*Client]bool{}}
	host := &Client{id: "p1", room: room}

	s.handleMessage(host, Message{Type: "removePlayer", TargetID: "p2"})

	if _, ok := room.Players["p2"]; ok {
		t.Fatalf("p2 should have been removed from room.Players")
	}
	if strings.Contains(strings.Join(room.Order, ","), "p2") {
		t.Fatalf("p2 should have been removed from room.Order: %v", room.Order)
	}
	if _, ok := room.Players["p1"]; !ok {
		t.Fatalf("host should remain in the room")
	}
}

func TestNonHostCannotRemovePlayerFromLobby(t *testing.T) {
	room := testRoom()
	s := &Server{rooms: map[string]*Room{room.Code: room}, clients: map[*Client]bool{}}
	nonHost := &Client{id: "p2", room: room}

	s.handleMessage(nonHost, Message{Type: "removePlayer", TargetID: "p3"})

	if _, ok := room.Players["p3"]; !ok {
		t.Fatalf("non-host should not be able to remove p3")
	}
}

func TestHostStartTurnsAdvancesFromRole(t *testing.T) {
	room := testRoom()
	room.Phase = PhaseRole
	s := &Server{rooms: map[string]*Room{room.Code: room}, clients: map[*Client]bool{}}

	s.handleMessage(&Client{id: "p1", room: room}, Message{Type: "startGame"})

	if room.Phase != PhaseTurns {
		t.Fatalf("phase = %s, want %s", room.Phase, PhaseTurns)
	}
}

func TestNonHostCannotStartTurnsFromRole(t *testing.T) {
	room := testRoom()
	room.Phase = PhaseRole
	s := &Server{rooms: map[string]*Room{room.Code: room}, clients: map[*Client]bool{}}

	s.handleMessage(&Client{id: "p2", room: room}, Message{Type: "startGame"})

	if room.Phase != PhaseRole {
		t.Fatalf("non-host advanced phase to %s, should stay %s", room.Phase, PhaseRole)
	}
}

func TestFinalTurnMovesDirectlyToVoting(t *testing.T) {
	room := testRoom()
	room.Phase = PhaseTurns
	room.TurnIndex = len(room.Order) - 1
	s := &Server{rooms: map[string]*Room{room.Code: room}, clients: map[*Client]bool{}}
	lastPlayer := &Client{id: room.Order[room.TurnIndex], room: room}

	s.handleMessage(lastPlayer, Message{Type: "doneTurn"})

	if room.Phase != PhaseVoting {
		t.Fatalf("phase = %s, want %s", room.Phase, PhaseVoting)
	}
}

func TestDraftVoteDuringTurnsCanChangeAndIsNotFinal(t *testing.T) {
	room := testRoom()
	room.Phase = PhaseTurns
	s := &Server{rooms: map[string]*Room{room.Code: room}, clients: map[*Client]bool{}}
	player := &Client{id: "p1", room: room}

	s.handleMessage(player, Message{Type: "draftVote", TargetID: "p2"})
	s.handleMessage(player, Message{Type: "draftVote", TargetID: "p3"})

	if room.Players["p1"].DraftVote != "p3" {
		t.Fatalf("draft vote = %q, want p3", room.Players["p1"].DraftVote)
	}
	if room.Players["p1"].Voted {
		t.Fatalf("draft vote should not mark player as voted")
	}
	if len(room.Votes) != 0 {
		t.Fatalf("draft vote should not create final votes: %v", room.Votes)
	}
}

func TestDraftVoteCanChangeDuringVotingUntilFinalVote(t *testing.T) {
	room := testRoom()
	room.Phase = PhaseVoting
	s := &Server{rooms: map[string]*Room{room.Code: room}, clients: map[*Client]bool{}}
	player := &Client{id: "p1", room: room}

	s.handleMessage(player, Message{Type: "draftVote", TargetID: "p2"})
	s.handleMessage(player, Message{Type: "vote", TargetID: "p3"})
	s.handleMessage(player, Message{Type: "draftVote", TargetID: "p2"})

	if room.Players["p1"].DraftVote != "p2" {
		t.Fatalf("draft vote = %q, want p2", room.Players["p1"].DraftVote)
	}
	if got := room.Votes["p1"]; got != "p3" {
		t.Fatalf("final vote = %q, want p3", got)
	}
	if !room.Players["p1"].Voted {
		t.Fatalf("final vote should mark player as voted")
	}
}

func TestAllFinalVotesMoveToResults(t *testing.T) {
	room := testRoom()
	room.Phase = PhaseVoting
	s := &Server{rooms: map[string]*Room{room.Code: room}, clients: map[*Client]bool{}}

	for _, id := range room.Order {
		s.handleMessage(&Client{id: id, room: room}, Message{Type: "vote", TargetID: "p3"})
	}

	if room.Phase != PhaseResults {
		t.Fatalf("phase = %s, want %s", room.Phase, PhaseResults)
	}
}

func TestPlayersWin(t *testing.T) {
	cases := []struct {
		name     string
		phase    Phase
		impostor string
		votes    map[string]string
		want     bool
	}{
		{"impostor has majority", PhaseResults, "p3", map[string]string{"p1": "p3", "p2": "p3", "p3": "p1"}, true},
		{"tie including impostor wins for players", PhaseResults, "p3", map[string]string{"p1": "p3", "p2": "p2", "p3": "p1"}, true},
		{"innocent has majority", PhaseResults, "p3", map[string]string{"p1": "p2", "p2": "p2", "p3": "p1"}, false},
		{"no votes", PhaseResults, "p3", map[string]string{}, false},
		{"not results phase", PhaseVoting, "p3", map[string]string{"p1": "p3", "p2": "p3"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			room := testRoom()
			room.Phase = tc.phase
			room.ImpostorID = tc.impostor
			room.Votes = tc.votes
			if got := playersWin(room); got != tc.want {
				t.Fatalf("playersWin() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPrivateWord(t *testing.T) {
	cases := []struct {
		name     string
		phase    Phase
		clientID string
		want     string
	}{
		{"innocent sees word during role", PhaseRole, "p1", "pizza"},
		{"innocent sees word during turns", PhaseTurns, "p1", "pizza"},
		{"impostor does not see word during role", PhaseRole, "p3", ""},
		{"hidden in lobby", PhaseLobby, "p1", ""},
		{"hidden in voting", PhaseVoting, "p1", ""},
		{"hidden in results", PhaseResults, "p1", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			room := testRoom()
			room.Phase = tc.phase
			room.Word = "pizza"
			room.ImpostorID = "p3"
			if got := privateWord(&Client{id: tc.clientID}, room); got != tc.want {
				t.Fatalf("privateWord() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResultWord(t *testing.T) {
	room := testRoom()
	room.Word = "pizza"

	room.Phase = PhaseVoting
	if got := resultWord(room); got != "" {
		t.Fatalf("resultWord before results = %q, want empty", got)
	}

	room.Phase = PhaseResults
	if got := resultWord(room); got != "pizza" {
		t.Fatalf("resultWord during results = %q, want pizza", got)
	}
}

func TestResetToLobby(t *testing.T) {
	room := testRoom()
	room.Phase = PhaseResults
	room.Word = "pizza"
	room.ImpostorID = "p3"
	room.TurnIndex = 2
	room.Votes = map[string]string{"p1": "p3"}
	for _, p := range room.Players {
		p.Voted = true
		p.LastVote = "p3"
		p.DraftVote = "p2"
	}

	(&Server{}).resetToLobby(room)

	if room.Phase != PhaseLobby || room.Word != "" || room.ImpostorID != "" || room.TurnIndex != 0 {
		t.Fatalf("room was not reset: %+v", room)
	}
	if len(room.Votes) != 0 {
		t.Fatalf("votes = %v, want empty", room.Votes)
	}
	for id, p := range room.Players {
		if p.Voted || p.LastVote != "" || p.DraftVote != "" {
			t.Fatalf("player %s was not reset: %+v", id, p)
		}
	}
}

func TestViewForSerializesState(t *testing.T) {
	room := testRoom()
	room.Phase = PhaseResults
	room.Word = "pizza"
	room.ImpostorID = "p3"
	room.RoundNumber = 1
	room.TurnIndex = 1
	room.Players["p1"].DraftVote = "p3"
	room.Votes = map[string]string{"p2": "p3", "p1": "p3", "p3": "p1"}

	view := (&Server{}).viewFor(&Client{id: "p1"}, room, "done")

	if view["type"] != "state" || view["toast"] != "done" || view["playerId"] != "p1" || view["roomCode"] != "abc" {
		t.Fatalf("unexpected identity fields: %#v", view)
	}
	if view["isHost"] != true || view["secretWord"] != "" || view["isImpostor"] != false {
		t.Fatalf("unexpected player fields: %#v", view)
	}
	if view["impostorName"] != "Mia" || view["resultWord"] != "pizza" || view["playersWin"] != true {
		t.Fatalf("unexpected result fields: %#v", view)
	}
	voteSummary, ok := view["voteSummary"].([]map[string]string)
	if !ok || len(voteSummary) != 3 {
		t.Fatalf("voteSummary = %#v, want three entries", view["voteSummary"])
	}
	gotSummary := voteSummary[0]["voter"] + ":" + voteSummary[0]["target"] + "," + voteSummary[1]["voter"] + ":" + voteSummary[1]["target"] + "," + voteSummary[2]["voter"] + ":" + voteSummary[2]["target"]
	if gotSummary != "Ana:Mia,Bob:Mia,Mia:Ana" {
		t.Fatalf("voteSummary order = %s", gotSummary)
	}
}

func TestViewForCurrentTurnAndSecretWord(t *testing.T) {
	room := testRoom()
	room.Phase = PhaseTurns
	room.Word = "pizza"
	room.ImpostorID = "p3"
	room.TurnIndex = 1

	view := (&Server{}).viewFor(&Client{id: "p1"}, room, "")

	if view["currentTurnId"] != "p2" || view["secretWord"] != "pizza" || view["draftVote"] != "" {
		t.Fatalf("unexpected turn fields: %#v", view)
	}
}

func TestHandleMessageCreateRoom(t *testing.T) {
	s := &Server{rooms: map[string]*Room{}, clients: map[*Client]bool{}}
	c := &Client{id: "p1"}

	s.handleMessage(c, Message{Type: "createRoom", Name: " Ana "})

	if c.room == nil || len(s.rooms) != 1 {
		t.Fatalf("room was not created: client=%+v rooms=%v", c, s.rooms)
	}
	p := c.room.Players["p1"]
	if p == nil || p.Name != "Ana" || !p.Host || !p.Online {
		t.Fatalf("host player = %+v", p)
	}
}

func TestHandleMessageJoinRoom(t *testing.T) {
	room := testRoom()
	s := &Server{rooms: map[string]*Room{room.Code: room}, clients: map[*Client]bool{}}
	c := &Client{id: "p4"}

	s.handleMessage(c, Message{Type: "joinRoom", RoomCode: " ABC ", Name: "Lee"})

	if c.room != room || room.Players["p4"].Name != "Lee" || room.Order[len(room.Order)-1] != "p4" {
		t.Fatalf("join failed: client=%+v room=%+v", c, room)
	}
}

func TestHandleMessageJoinRoomRejectsDuplicateName(t *testing.T) {
	room := testRoom()
	s := &Server{rooms: map[string]*Room{room.Code: room}, clients: map[*Client]bool{}}
	c, cleanup := testClientWithConn(t, "p4")
	defer cleanup()

	s.handleMessage(c, Message{Type: "joinRoom", RoomCode: "abc", Name: "ana"})

	if _, ok := room.Players["p4"]; ok {
		t.Fatalf("duplicate name should not have joined")
	}
}

func TestHandleMessageJoinRoomRejectsStartedGame(t *testing.T) {
	room := testRoom()
	room.Phase = PhaseTurns
	s := &Server{rooms: map[string]*Room{room.Code: room}, clients: map[*Client]bool{}}
	c, cleanup := testClientWithConn(t, "p4")
	defer cleanup()

	s.handleMessage(c, Message{Type: "joinRoom", RoomCode: "abc", Name: "Lee"})

	if _, ok := room.Players["p4"]; ok {
		t.Fatalf("player should not join after game starts")
	}
}

func TestHandleMessageJoinRoomRejectsMissingRoom(t *testing.T) {
	s := &Server{rooms: map[string]*Room{}, clients: map[*Client]bool{}}
	c, cleanup := testClientWithConn(t, "p4")
	defer cleanup()

	s.handleMessage(c, Message{Type: "joinRoom", RoomCode: "bad", Name: "Lee"})

	if c.room != nil {
		t.Fatalf("client joined missing room: %+v", c.room)
	}
}


func TestHandleMessagePlayAgainStartsNextRound(t *testing.T) {
	oldChoose := chooseRandomInt
	defer func() { chooseRandomInt = oldChoose }()
	calls := []int{0, 0, 1}
	chooseRandomInt = testChooser(t, &calls)

	room := testRoom()
	room.Phase = PhaseResults
	room.Category = "Food"
	room.RoundNumber = 1
	room.RoundLimit = 2
	s := &Server{rooms: map[string]*Room{room.Code: room}, clients: map[*Client]bool{}}

	s.handleMessage(&Client{id: "p1", room: room}, Message{Type: "playAgain"})

	if room.RoundNumber != 2 || room.Phase != PhaseRole || room.Word != "pizza" || room.ImpostorID != "p2" {
		t.Fatalf("next round failed: %+v", room)
	}
}

func TestHandleMessagePlayAgainAfterFinalRoundResetsLobby(t *testing.T) {
	room := testRoom()
	room.Phase = PhaseResults
	room.RoundNumber = 2
	room.RoundLimit = 2
	room.Word = "pizza"
	room.ImpostorID = "p3"
	s := &Server{rooms: map[string]*Room{room.Code: room}, clients: map[*Client]bool{}}

	s.handleMessage(&Client{id: "p1", room: room}, Message{Type: "playAgain"})

	if room.RoundNumber != 0 || room.Phase != PhaseLobby || room.Word != "" || room.ImpostorID != "" {
		t.Fatalf("final playAgain should reset lobby: %+v", room)
	}
}

func TestHandleMessageBackLobby(t *testing.T) {
	room := testRoom()
	room.Phase = PhaseResults
	room.RoundNumber = 1
	room.Word = "pizza"
	s := &Server{rooms: map[string]*Room{room.Code: room}, clients: map[*Client]bool{}}

	s.handleMessage(&Client{id: "p1", room: room}, Message{Type: "backLobby"})

	if room.RoundNumber != 0 || room.Phase != PhaseLobby || room.Word != "" {
		t.Fatalf("backLobby should reset lobby: %+v", room)
	}
}

func TestCleanName(t *testing.T) {
	if got := cleanName("  Ana  "); got != "Ana" {
		t.Fatalf("cleanName trim = %q", got)
	}
	if got := cleanName("abcdefghijklmnopqrstuvwxyz"); got != "abcdefghijklmnopqrst" {
		t.Fatalf("cleanName truncation = %q", got)
	}
}

func TestIsCategory(t *testing.T) {
	if !isCategory("Random") || !isCategory("Food") {
		t.Fatalf("expected Random and Food to be categories")
	}
	if isCategory("Not a category") {
		t.Fatalf("unexpected valid category")
	}
}

func TestRotateStartingPlayerEdgeCases(t *testing.T) {
	cases := []struct {
		name  string
		order []string
		start int
		want  string
	}{
		{"empty", nil, 2, ""},
		{"zero", []string{"p1", "p2"}, 0, "p1,p2"},
		{"negative", []string{"p1", "p2"}, -1, "p1,p2"},
		{"too large", []string{"p1", "p2"}, 2, "p1,p2"},
		{"middle", []string{"p1", "p2", "p3"}, 1, "p2,p3,p1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			room := &Room{Order: append([]string{}, tc.order...)}
			rotateStartingPlayer(room, tc.start)
			if got := strings.Join(room.Order, ","); got != tc.want {
				t.Fatalf("order = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPlayerName(t *testing.T) {
	room := testRoom()
	if got := playerName(room, "p1"); got != "Ana" {
		t.Fatalf("playerName = %q, want Ana", got)
	}
	if got := playerName(room, "missing"); got != "" {
		t.Fatalf("missing playerName = %q, want empty", got)
	}
}

func TestNewRoomCodeSkipsCollisions(t *testing.T) {
	oldChoose := chooseRandomInt
	defer func() { chooseRandomInt = oldChoose }()
	calls := []int{0, 0, 0, 1, 1, 1}
	chooseRandomInt = testChooser(t, &calls)

	s := &Server{rooms: map[string]*Room{"aaa": {}}, clients: map[*Client]bool{}}
	if got := s.newRoomCode(); got != "bbb" {
		t.Fatalf("newRoomCode = %q, want bbb", got)
	}
}

func TestRandomID(t *testing.T) {
	oldChoose := chooseRandomInt
	defer func() { chooseRandomInt = oldChoose }()
	calls := []int{0, 25, 26, 35}
	chooseRandomInt = testChooser(t, &calls)

	if got := randomID(4); got != "az09" {
		t.Fatalf("randomID = %q, want az09", got)
	}
}

func testChooser(t *testing.T, calls *[]int) func(int) int {
	t.Helper()
	return func(max int) int {
		if len(*calls) == 0 {
			t.Fatalf("unexpected chooseRandomInt call with max %d", max)
		}
		v := (*calls)[0]
		*calls = (*calls)[1:]
		if v >= max {
			t.Fatalf("test random value %d is outside max %d", v, max)
		}
		return v
	}
}

func testClientWithConn(t *testing.T, id string) (*Client, func()) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	u, err := url.Parse(server.URL)
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	u.Scheme = "ws"
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	return &Client{id: id, conn: conn}, func() {
		conn.Close()
		server.Close()
	}
}

func testRoom() *Room {
	return &Room{
		Code:       "abc",
		Phase:      PhaseLobby,
		Category:   "Random",
		RoundLimit: 1,
		Players: map[string]*Player{
			"p1": {ID: "p1", Name: "Ana", Host: true, Online: true},
			"p2": {ID: "p2", Name: "Bob", Online: true},
			"p3": {ID: "p3", Name: "Mia", Online: true},
		},
		Order: []string{"p1", "p2", "p3"},
		Votes: map[string]string{},
	}
}
