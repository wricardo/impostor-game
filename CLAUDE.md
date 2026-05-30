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

**Game phases (in order):** `lobby → role → turns → voting → results`

Phase transitions:
- `startGame` → role
- all players `ready` → turns
- players may send `draftVote` during turns to save/change a non-final guess
- all turns done → voting
- all votes cast → results
- host `playAgain` → role (next round) or lobby (rounds exhausted)
- host `backLobby` → lobby

**View model:** `viewFor` computes a per-client state snapshot on every broadcast. The secret word is stripped for the impostor; the impostor identity is stripped for non-impostors until results phase.

**No persistence** — all state is in-memory. Disconnecting marks a player offline but keeps them in the room. No reconnect support.


<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
