# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go run .          # run server at http://localhost:8000
go test ./...     # build check (no tests yet)
go mod tidy       # sync dependencies
```

Preview all UI screens without multiplayer:
```
http://localhost:8000/?preview=1
```

## Architecture

Single-file Go server + single-file frontend. No frameworks, no build step.

- `main.go` — entire backend: HTTP server, WebSocket handling, room/game state, all game logic
- `index.html` — entire frontend: all screens, CSS, JS, WebSocket client; embedded into binary via `//go:embed`

**State model:** `Server` holds a map of `Room`s and connected `Client`s under a single `sync.Mutex`. Every WebSocket message goes through `handleMessage`, which acquires the mutex and mutates room state, then calls `broadcast` to push updated view state to all clients in the room.

**Game phases (in order):** `lobby → role → turns → discussion → voting → results`

Phase transitions:
- `startGame` → role
- all players `ready` → turns
- all turns done → discussion
- host `startVoting` → voting
- all votes cast → results
- host `playAgain` → role (next round) or lobby (rounds exhausted)
- host `backLobby` → lobby

**View model:** `viewFor` computes a per-client state snapshot on every broadcast. The secret word is stripped for the impostor; the impostor identity is stripped for non-impostors until results phase.

**No persistence** — all state is in-memory. Disconnecting marks a player offline but keeps them in the room. No reconnect support.
