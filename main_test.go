package main

import (
	"os"
	"strings"
	"testing"
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
}

func TestStartRoundRandomizesStartingPlayer(t *testing.T) {
	oldChoose := chooseRandomInt
	defer func() { chooseRandomInt = oldChoose }()

	calls := []int{2, 0} // first call picks starting player, second picks impostor from rotated order
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
	(&Server{}).startRound(room)

	wantOrder := []string{"p3", "p1", "p2"}
	if strings.Join(room.Order, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("room.Order = %v, want %v", room.Order, wantOrder)
	}
	if room.Phase != PhaseTurns {
		t.Fatalf("phase = %s, want %s", room.Phase, PhaseTurns)
	}
	if room.TurnIndex != 0 || room.Order[room.TurnIndex] != "p3" {
		t.Fatalf("first turn = %q at index %d, want p3 at index 0", room.Order[room.TurnIndex], room.TurnIndex)
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
