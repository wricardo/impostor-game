# Impostor Game

A simple web-based party game inspired by *Impostor*. Players join the same room on their phones, receive a secret word, and take turns saying one related word out loud. One player is secretly the impostor and must bluff their way through the round without knowing the word.

The game is designed to be easy for kids and adults to play at a party: no accounts, no setup, big buttons, short instructions, and one phone per player.

## How the Game Works

1. One player creates a room.
2. Friends join the room using a short 3-character room code.
3. Each player enters a name that must be unique in that room.
4. The host chooses a word category and number of rounds.
5. The host starts the game.
6. Most players see the same secret word on their phone.
7. One player is randomly chosen as the impostor.
8. The impostor does **not** see the secret word. They only see `IMPOSTOR`.
9. Players take turns saying one word related to the secret word.
10. The impostor listens, guesses what the word might be, and tries to say something believable.
11. After all turns, players discuss who they think the impostor is.
12. Everyone votes.
13. The impostor is revealed.

## Winning

- The players win if they correctly vote for the impostor.
- The impostor wins if they avoid getting the most votes.

## Example Round

Secret word: `Pizza`

Normal players might say:

- Cheese
- Slice
- Oven
- Pepperoni

The impostor does not know the word, so they have to listen carefully and say something that sounds related.

For example, the impostor might say:

- Hot

That might fit `Pizza`, but it could also fit many other words, which makes the game fun.

## Room Codes

Rooms use simple 3-character codes made from:

```txt
a-z 0-9
```

Examples:

```txt
a7k
p9x
2bc
```

Room codes are treated as lowercase, even if someone types uppercase letters.

## Player Names

Each player joins a room with a name.

Rules:

- Names must be unique inside the room.
- Names can be reused in different rooms.
- No account or login is required.

If a player tries to join with a name already used in that room, they will see:

> That name is already taken in this room.

## Screens

### Home Screen

The first screen players see when opening the website.

Players can choose:

- `Create Room`
- `Join Room`

### Create Room Screen

The host enters their name and creates a new room.

After creating a room:

- A random 3-character room code is generated.
- The creator becomes the host.
- The host is moved to the lobby.

### Join Room Screen

Players enter:

- Room code
- Name

Then they tap `Join`.

### Lobby Screen

Players wait here before the game starts.

The lobby shows:

- Room code
- Player list
- Category selector for the host
- Start button for the host

The game requires at least 3 players to start.

If there are fewer than 3 players, the lobby shows:

> Need at least 3 players to start.

### Category and Rounds Selector

Before starting the game, the host chooses a category and how many rounds to play.

The number of rounds can be set from 1 to 5.

Current categories:

- Random
- Food
- Animals
- Places
- Movies
- Objects
- Sports
- School

Each category uses a simple built-in word list.

### Role Screen

After the game starts, every player sees a private screen.

Normal players see:

> Your word is:

Then the secret word.

The impostor sees:

> You are the:

Then:

> IMPOSTOR

Players should hide their phones during this screen.

### Turn Screen

Players take turns saying one word out loud.

The current player sees:

> It's your turn. Say one word out loud.

Everyone else sees:

> Wait for Ana to say one word.

After speaking, the current player taps `Done`.

### Discussion Screen

After everyone has said one word, players discuss who they think the impostor is.

The host can start voting when the group is ready.

### Voting Screen

Each player privately votes for who they think the impostor is.

After voting, they wait for everyone else.

### Results Screen

The game reveals:

- Who the impostor was
- What the secret word was
- Whether the players or impostor won
- Who each player voted for

The host can then start another round or go back to the lobby.

## Current Features

- Single-page web app
- Go web server
- WebSocket-based realtime multiplayer
- Create room
- Join room
- 3-character room codes
- Unique names per room
- Host-controlled game start
- Category selection
- Configurable rounds from 1 to 5
- Random word selection
- Random impostor selection
- Private role screen
- Ready check before turns begin
- Turn-based gameplay
- Discussion phase
- Private voting
- Results reveal
- Play again / back to lobby
- Mobile-friendly high-fidelity UI

## Tech Stack

- Go
- HTML
- CSS
- JavaScript
- WebSockets
- [`github.com/gorilla/websocket`](https://github.com/gorilla/websocket)

The frontend is served as one single HTML file: `index.html`.

The backend server is implemented in: `main.go`.

## Running Locally

Install dependencies:

```bash
go mod tidy
```

Run the server:

```bash
go run .
```

Open the game in your browser:

```txt
http://localhost:8080
```

To test multiplayer locally, open the same URL in multiple browser tabs or on multiple devices connected to the same network.

## UI Preview Mode

The single-page frontend includes a static preview mode for quickly reviewing every screen without creating rooms or using multiple browser tabs.

Run the server, then open:

```txt
http://localhost:8080/?preview=1
```

Preview mode shows a developer toolbar with sample states for:

- Home
- Create Room
- Join Room
- Lobby as host
- Lobby as player
- Role screen with word
- Role screen as impostor
- Turn screen, current player
- Turn screen, waiting player
- Discussion
- Voting before and after voting
- Results, players win
- Results, impostor wins

This is useful for UI design, screenshots, and quick mobile layout checks.

## Development

Run tests/build check:

```bash
go test ./...
```

## Project Files

- `main.go` - Go server, room state, game state, WebSocket handling
- `index.html` - single-page frontend with all screens and UI logic
- `SCREENS.md` - detailed screen plan and wireframe notes
- `README.md` - project overview and usage
- `go.mod` - Go module definition

## Future Ideas

- Custom word lists
- More than one impostor for large groups
- Timer for turns
- Timer for discussion
- QR code for joining a room
- Host can kick players from the lobby
- Better reconnect support after refresh
- Score tracking across rounds
- Sound effects
- Animations for the reveal screen
- Family-friendly category packs

## License

Add a license before publishing or sharing publicly.
