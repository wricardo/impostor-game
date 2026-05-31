package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
)

const defaultWSURL = "ws://localhost:8000/ws"

type viewState struct {
	Type          string              `json:"type"`
	Toast         string              `json:"toast"`
	PlayerID      string              `json:"playerId"`
	RoomCode      string              `json:"roomCode"`
	Phase         Phase               `json:"phase"`
	Category      string              `json:"category"`
	Categories    []string            `json:"categories"`
	RoundLimit    int                 `json:"roundLimit"`
	RoundNumber   int                 `json:"roundNumber"`
	Players       []Player            `json:"players"`
	IsHost        bool                `json:"isHost"`
	SecretWord    string              `json:"secretWord"`
	IsImpostor    bool                `json:"isImpostor"`
	CurrentTurnID string              `json:"currentTurnId"`
	TurnIndex     int                 `json:"turnIndex"`
	DraftVote     string              `json:"draftVote"`
	ImpostorName  string              `json:"impostorName"`
	ResultWord    string              `json:"resultWord"`
	PlayersWin    bool                `json:"playersWin"`
	VoteSummary   []map[string]string `json:"voteSummary"`
}

type serverMessage struct {
	viewState
	Message string `json:"message"`
}

type wsGameClient struct {
	conn *websocket.Conn
}

func newWSGameClient(server string) (*wsGameClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(server, nil)
	if err != nil {
		return nil, err
	}
	return &wsGameClient{conn: conn}, nil
}

func (c *wsGameClient) Close() error           { return c.conn.Close() }
func (c *wsGameClient) Send(msg Message) error { return c.conn.WriteJSON(msg) }

func (c *wsGameClient) Read() (serverMessage, error) {
	var msg serverMessage
	err := c.conn.ReadJSON(&msg)
	return msg, err
}

func commandMode(args []string) string {
	if len(args) == 0 {
		return "serve"
	}
	return args[0]
}

func runCLI(args []string) error {
	switch commandMode(args) {
	case "serve":
		fs := flag.NewFlagSet("serve", flag.ExitOnError)
		port := fs.String("port", "", "port to listen on")
		if len(args) > 0 {
			_ = fs.Parse(args[1:])
		}
		return runServer(*port)
	case "tui":
		return runTUICommand(args[1:])
	case "create", "join", "state", "vote", "start", "done-turn", "set-category", "set-rounds", "play-again", "back-lobby":
		return runNonInteractive(args[0], args[1:])
	case "help", "-h", "--help":
		fmt.Print(cliUsage())
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", commandMode(args), cliUsage())
	}
}

func cliUsage() string {
	return `Usage:
  impostor-game serve [--port 8000]
  impostor-game tui [--server ws://localhost:8000/ws] [--local] [--port 8000]
  impostor-game create --name NAME [--server URL] [--json]
  impostor-game join --room CODE --name NAME [--server URL] [--json]
  impostor-game state --room CODE --player-id ID [--server URL] [--json]
  impostor-game start|done-turn|play-again|back-lobby --room CODE --player-id ID [--server URL] [--json]
  impostor-game vote --room CODE --player-id ID --target-id ID [--server URL] [--json]
  impostor-game set-category --room CODE --player-id ID --category CATEGORY [--server URL] [--json]
  impostor-game set-rounds --room CODE --player-id ID --rounds N [--server URL] [--json]
`
}

func runNonInteractive(action string, args []string) error {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	server := fs.String("server", defaultWSURL, "websocket server URL")
	name := fs.String("name", "", "player name")
	room := fs.String("room", "", "room code")
	playerID := fs.String("player-id", "", "player ID")
	targetID := fs.String("target-id", "", "target player ID")
	category := fs.String("category", "", "category")
	rounds := fs.Int("rounds", 0, "round count")
	jsonOut := fs.Bool("json", false, "print JSON output")
	_ = fs.Parse(args)

	client, err := newWSGameClient(*server)
	if err != nil {
		return err
	}
	defer client.Close()
	// connected message
	_, _ = client.Read()

	var resumedState viewState
	if action != "create" && action != "join" {
		if *room == "" || *playerID == "" {
			return errors.New("--room and --player-id are required")
		}
		if err := client.Send(Message{Type: "resume", RoomCode: *room, PlayerID: *playerID}); err != nil {
			return err
		}
		var err error
		resumedState, err = waitForState(client, 3*time.Second)
		if err != nil {
			return err
		}
	}

	var msg Message
	switch action {
	case "create":
		if *name == "" {
			return errors.New("--name is required")
		}
		msg = Message{Type: "createRoom", Name: *name}
	case "join":
		if *room == "" || *name == "" {
			return errors.New("--room and --name are required")
		}
		msg = Message{Type: "joinRoom", RoomCode: *room, Name: *name}
	case "state":
		return printState(resumedState, *jsonOut)
	case "start":
		msg = Message{Type: "startGame"}
	case "done-turn":
		msg = Message{Type: "doneTurn"}
	case "play-again":
		msg = Message{Type: "playAgain"}
	case "back-lobby":
		msg = Message{Type: "backLobby"}
	case "vote":
		if *targetID == "" {
			return errors.New("--target-id is required")
		}
		msg = Message{Type: "vote", TargetID: *targetID}
	case "set-category":
		if *category == "" {
			return errors.New("--category is required")
		}
		msg = Message{Type: "setCategory", Category: *category}
	case "set-rounds":
		if *rounds == 0 {
			return errors.New("--rounds is required")
		}
		msg = Message{Type: "setRounds", Rounds: *rounds}
	}
	if err := client.Send(msg); err != nil {
		return err
	}
	state, err := waitForState(client, 3*time.Second)
	if err != nil {
		return err
	}
	return printState(state, *jsonOut)
}

func waitForState(client *wsGameClient, timeout time.Duration) (viewState, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_ = client.conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		msg, err := client.Read()
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				continue
			}
			return viewState{}, err
		}
		if msg.Type == "error" {
			return viewState{}, errors.New(msg.Message)
		}
		if msg.Type == "state" {
			return msg.viewState, nil
		}
	}
	return viewState{}, errors.New("timed out waiting for state")
}

func printState(state viewState, asJSON bool) error {
	if asJSON {
		b, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	fmt.Printf("Room %s · %s · %s · Round %d/%d\n", strings.ToUpper(state.RoomCode), state.Phase, state.Category, state.RoundNumber, state.RoundLimit)
	fmt.Printf("Player ID: %s\n", state.PlayerID)
	for _, p := range state.Players {
		mark := " "
		if p.ID == state.PlayerID {
			mark = "*"
		}
		fmt.Printf("%s %s %s voted=%v online=%v\n", mark, p.ID, p.Name, p.Voted, p.Online)
	}
	return nil
}

func runTUICommand(args []string) error {
	fs := flag.NewFlagSet("tui", flag.ExitOnError)
	server := fs.String("server", defaultWSURL, "websocket server URL")
	local := fs.Bool("local", false, "start an embedded local server")
	port := fs.String("port", "8000", "local server port")
	_ = fs.Parse(args)
	if *local {
		srv := &http.Server{Addr: ":" + *port, Handler: newServerHandler()}
		go func() { _ = srv.ListenAndServe() }()
		defer srv.Shutdown(context.Background())
		*server = "ws://localhost:" + *port + "/ws"
		time.Sleep(100 * time.Millisecond)
	}
	return runTUI(*server)
}

type tuiScreen int

const (
	screenHome tuiScreen = iota
	screenCreate
	screenJoinRoom
	screenJoinName
	screenGame
	screenHelp
)

type tuiModel struct {
	server        string
	client        *wsGameClient
	state         viewState
	screen        tuiScreen
	input         textinput.Model
	joinCode      string
	width, height int
	status        string
	err           string
	voteCursor    int
	showWord      bool
}

type connectedMsg struct {
	client *wsGameClient
	err    error
}
type wsMsg struct {
	msg serverMessage
	err error
}

type sentMsg struct{ err error }

func runTUI(server string) error {
	m := newTUIModel(server)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func newTUIModel(server string) tuiModel {
	ti := textinput.New()
	ti.Placeholder = "Your name"
	ti.CharLimit = 20
	ti.Width = 30
	ti.Focus()
	return tuiModel{server: server, screen: screenHome, input: ti, status: "c create · j join · h help · q quit"}
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case connectedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.client = msg.client
		cmds = append(cmds, readWS(m.client))
	case sendAfterConnectMsg:
		if m.client == nil {
			m.err = "not connected"
			return m, nil
		}
		cmds = append(cmds, func() tea.Msg { return sentMsg{err: m.client.Send(msg.Message)} })
	case sentMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		}
	case wsMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		if msg.msg.Type == "error" {
			m.err = msg.msg.Message
		} else if msg.msg.Type == "state" {
			if msg.msg.viewState.Phase != m.state.Phase {
				m.showWord = false
			}
			m.state = msg.msg.viewState
			m.screen = screenGame
			m.err = ""
		}
		if m.client != nil {
			cmds = append(cmds, readWS(m.client))
		}
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" || key == "q" {
			return m, tea.Quit
		}
		if m.screen == screenHelp {
			m.screen = screenHome
			return m, nil
		}
		if m.screen == screenHome {
			switch key {
			case "c":
				m.screen = screenCreate
				m.input.SetValue("")
				m.input.Placeholder = "Host name"
				return m, nil
			case "j":
				m.screen = screenJoinRoom
				m.input.SetValue("")
				m.input.Placeholder = "Room code"
				return m, nil
			case "h":
				m.screen = screenHelp
				return m, nil
			}
		}
		if m.screen == screenCreate || m.screen == screenJoinRoom || m.screen == screenJoinName {
			if key == "esc" {
				m.screen = screenHome
				return m, nil
			}
			if key == "enter" {
				return m.handleInputEnter()
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		if m.screen == screenGame {
			return m.updateGameKeys(key)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m tuiModel) handleInputEnter() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.input.Value())
	if val == "" {
		m.err = "Please type a value."
		return m, nil
	}
	if m.screen == screenCreate {
		return m, tea.Sequence(connectCmd(m.server), func() tea.Msg { return sendAfterConnectMsg{Message{Type: "createRoom", Name: val}} })
	}
	if m.screen == screenJoinRoom {
		m.joinCode = val
		m.screen = screenJoinName
		m.input.SetValue("")
		m.input.Placeholder = "Your name"
		return m, nil
	}
	if m.screen == screenJoinName {
		room := m.joinCode
		return m, tea.Sequence(connectCmd(m.server), func() tea.Msg { return sendAfterConnectMsg{Message{Type: "joinRoom", RoomCode: room, Name: val}} })
	}
	return m, nil
}

type sendAfterConnectMsg struct{ Message Message }

func (m tuiModel) updateGameKeys(key string) (tea.Model, tea.Cmd) {
	if m.client == nil {
		return m, nil
	}
	send := func(msg Message) tea.Cmd { return func() tea.Msg { return sentMsg{err: m.client.Send(msg)} } }
	switch key {
	case "w":
		m.showWord = !m.showWord
		return m, nil
	case "s":
		return m, send(Message{Type: "startGame"})
	case "d", "enter":
		return m, send(Message{Type: "doneTurn"})
	case "n":
		return m, send(Message{Type: "playAgain"})
	case "b":
		return m, send(Message{Type: "backLobby"})
	case "up", "k":
		if m.voteCursor > 0 {
			m.voteCursor--
		}
	case "down", "j":
		if m.voteCursor < len(m.state.Players)-1 {
			m.voteCursor++
		}
	case "v":
		if len(m.state.Players) > 0 {
			return m, send(Message{Type: "vote", TargetID: m.state.Players[m.voteCursor].ID})
		}
	case "[":
		return m, send(Message{Type: "setCategory", Category: m.prevCategory()})
	case "]":
		return m, send(Message{Type: "setCategory", Category: m.nextCategory()})
	case "-":
		if m.state.RoundLimit > 1 {
			return m, send(Message{Type: "setRounds", Rounds: m.state.RoundLimit - 1})
		}
	case "+", "=":
		if m.state.RoundLimit < 5 {
			return m, send(Message{Type: "setRounds", Rounds: m.state.RoundLimit + 1})
		}
	}
	return m, nil
}

func connectCmd(server string) tea.Cmd {
	return func() tea.Msg {
		c, err := newWSGameClient(server)
		if err == nil {
			_, _ = c.Read()
		}
		return connectedMsg{c, err}
	}
}

func readWS(c *wsGameClient) tea.Cmd {
	return func() tea.Msg { msg, err := c.Read(); return wsMsg{msg: msg, err: err} }
}

func (m tuiModel) View() string {
	body := ""
	switch m.screen {
	case screenHome:
		body = m.viewHome()
	case screenCreate, screenJoinRoom, screenJoinName:
		body = m.viewInput()
	case screenGame:
		body = m.viewGame()
	case screenHelp:
		body = m.viewHelp()
	}
	if m.err != "" {
		body += "\n" + errorStyle.Render("Error: "+m.err)
	}
	return lipgloss.Place(max(m.width, 60), max(m.height, 20), lipgloss.Center, lipgloss.Center, body)
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Padding(0, 1)
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(1, 2).Width(70)
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	goodStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
)

func (m tuiModel) viewHome() string {
	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("🎭 Impostor Game TUI"), "", "c  Create room", "j  Join room", "h  Help", "q  Quit", "", dimStyle.Render("Server: "+m.server)))
}
func (m tuiModel) viewInput() string {
	prompt := ""
	if m.screen == screenCreate {
		prompt = "Create room: enter host name"
	} else if m.screen == screenJoinRoom {
		prompt = "Join room: enter room code"
	} else {
		prompt = "Join room: enter your name"
	}
	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(prompt), "", m.input.View(), "", dimStyle.Render("enter submit · esc back")))
}
func (m tuiModel) viewHelp() string {
	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("Help"), "", "Lobby: [/] category · +/- rounds · s start", "Role: host presses s to start turns", "Turns: d or enter when your spoken clue is done", "Voting: ↑/↓ choose player · v vote", "Results: n next/play again · b back to lobby", "", "Random picks one real category for the round.", "Press any key to go back."))
}

func (m tuiModel) viewGame() string {
	lines := []string{titleStyle.Render(fmt.Sprintf("Room %s · %s · %s", strings.ToUpper(m.state.RoomCode), m.state.Phase, m.state.Category))}
	lines = append(lines, fmt.Sprintf("Round %d/%d · Player ID %s", m.state.RoundNumber, m.state.RoundLimit, m.state.PlayerID), "")
	lines = append(lines, m.renderPhase())
	lines = append(lines, "", m.renderPlayers())
	if m.state.Toast != "" {
		lines = append(lines, "", goodStyle.Render(m.state.Toast))
	}
	lines = append(lines, "", dimStyle.Render(m.gameHelp()))
	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m tuiModel) renderPhase() string {
	switch m.state.Phase {
	case PhaseLobby:
		return fmt.Sprintf("Lobby setup: %s · %d rounds", m.state.Category, m.state.RoundLimit)
	case PhaseTurns:
		name := m.playerName(m.state.CurrentTurnID)
		var line string
		if m.state.CurrentTurnID == m.state.PlayerID {
			line = goodStyle.Render("Your turn: say one related word out loud.")
		} else {
			line = "Wait for " + name + " to say one word."
		}
		if m.state.SecretWord != "" {
			if m.showWord {
				line += "\n" + dimStyle.Render("Word: "+strings.ToUpper(m.state.SecretWord))
			} else {
				line += "\n" + dimStyle.Render("Word: "+strings.Repeat("█", len(m.state.SecretWord))+" (w to peek)")
			}
		}
		return line
	case PhaseVoting:
		return "Vote for the impostor: \n" + m.renderVotePicker()
	case PhaseResults:
		winner := "Impostor wins!"
		if m.state.PlayersWin {
			winner = "Players win!"
		}
		return fmt.Sprintf("%s\nImpostor: %s\nSecret word: %s\n%s", winner, m.state.ImpostorName, m.state.ResultWord, m.renderVotes())
	default:
		return "Waiting..."
	}
}

func (m tuiModel) renderPlayers() string {
	lines := []string{"Players:"}
	for _, p := range m.state.Players {
		tags := []string{}
		if p.Host {
			tags = append(tags, "host")
		}
		if p.Voted {
			tags = append(tags, "voted")
		}
		if !p.Online {
			tags = append(tags, "offline")
		}
		marker := " "
		if p.ID == m.state.PlayerID {
			marker = "*"
		}
		lines = append(lines, fmt.Sprintf("%s %s %s", marker, p.Name, dimStyle.Render(strings.Join(tags, ", "))))
	}
	return strings.Join(lines, "\n")
}
func (m tuiModel) renderVotePicker() string {
	var lines []string
	for i, p := range m.state.Players {
		cur := "  "
		if i == m.voteCursor {
			cur = "> "
		}
		lines = append(lines, cur+p.Name)
	}
	return strings.Join(lines, "\n")
}
func (m tuiModel) renderVotes() string {
	if len(m.state.VoteSummary) == 0 {
		return ""
	}
	var lines []string
	for _, v := range m.state.VoteSummary {
		lines = append(lines, v["voter"]+" → "+v["target"])
	}
	return strings.Join(lines, "\n")
}
func (m tuiModel) gameHelp() string {
	switch m.state.Phase {
	case PhaseLobby:
		return "[/] category · +/- rounds · s start · q quit"
	case PhaseTurns:
		return "d done turn · w peek word · q quit"
	case PhaseVoting:
		return "↑/↓ select · v vote · q quit"
	case PhaseResults:
		return "n next/play again · b lobby · q quit"
	}
	return "q quit"
}
func (m tuiModel) playerName(id string) string {
	for _, p := range m.state.Players {
		if p.ID == id {
			return p.Name
		}
	}
	return "them"
}
func (m tuiModel) nextCategory() string {
	return adjacentCategory(m.state.Categories, m.state.Category, 1)
}
func (m tuiModel) prevCategory() string {
	return adjacentCategory(m.state.Categories, m.state.Category, -1)
}
func adjacentCategory(cats []string, current string, delta int) string {
	if len(cats) == 0 {
		return current
	}
	idx := 0
	for i, c := range cats {
		if c == current {
			idx = i
			break
		}
	}
	return cats[(idx+delta+len(cats))%len(cats)]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m tuiModel) update(msg tea.Msg) (tuiModel, tea.Cmd) {
	model, cmd := m.Update(msg)
	return model.(tuiModel), cmd
}
