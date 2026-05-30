# Impostor Game Screens

This document describes the simple screen flow for the web-based Impostor Game.

The goal is to keep the UI easy for kids and adults to use at a party: big buttons, short text, no accounts, and simple room codes.

## Room Codes

Rooms use a simple 3-character code made from:

```txt
a-z 0-9
```

Examples:

```txt
a7k
p9x
2bc
```

Room codes should be treated as lowercase even if the user types uppercase letters.

---

## 1. Home Screen

This is the first screen players see when loading the website.

### Shows

Title:

> Impostor Game

Main buttons:

- `Create Room`
- `Join Room`

### Join Room Form

When the player chooses `Join Room`, show:

- Room code input
- Name input
- `Join` button

### Rules

- The room code must be 3 characters.
- Room codes can contain letters `a-z` and numbers `0-9`.
- The player name must be unique inside the room.
- If the room does not exist, show an error.
- If the name is already taken in that room, show an error.

Example errors:

> Room not found.

> That name is already taken in this room.

---

## 2. Create Room Screen

This screen lets a player create a new room.

### Shows

- Name input: `Your name`
- `Create Room` button

### Behavior

When the player creates a room:

1. Generate a random 3-character room code.
2. Add the creator as the first player.
3. Make the creator the host.
4. Move the player to the lobby screen.

---

## 3. Lobby Screen

This is where players wait before the round starts.

### Shows

Room code, large and easy to read:

> Room Code: `k3p`

Helpful instruction:

> Tell your friends to open the website and join with this code.

Player list:

- Ana
- Bob
- Carlos
- Mia

### Host Controls

The host sees:

- `Start Game` button
- Category selector

### Everyone Sees

- `Leave Room` button

### Category Selector

Before starting the game, the host can choose a word category.

Possible categories:

- Random
- Food
- Animals
- Places
- Objects
- Sports
- School
- Nature
- Jobs
- Activities
- Fantasy

For the first version, real categories can be simple hardcoded word lists with PG, kid-friendly words. Random chooses one of the real categories for the round.

### Rules

- Minimum 3 players required to start.
- If there are fewer than 3 players, disable `Start Game` and show:

> Need at least 3 players to start.

---

## 4. Your Role Screen

After the host starts the game, each player sees their private role screen.

Players should hide their phone from others.

### Normal Player View

Shows:

> Your word is:

Big word:

> Pizza

Button:

- `I'm ready`

### Impostor View

Shows:

> You are the:

Big text:

> IMPOSTOR

Button:

- `I'm ready`

### Rules

- Most players see the same secret word.
- One player sees `IMPOSTOR` instead of the word.
- The impostor must bluff during the round.

---

## 5. Turn Screen

After everyone is ready, players take turns saying one word out loud.

### Shows

Current turn:

> Ana's turn

Instruction:

> Say one word related to your secret word.

### Current Player View

If it is your turn, show:

> It's your turn. Say one word out loud.

Button:

- `Done`

### Other Player View

If it is not your turn, show:

> Wait for Ana to say a word.

### Optional Display

Show the turn order:

- Ana
- Bob
- Carlos
- Mia

### Rules

- Each player says exactly one word out loud.
- The impostor should try to say something believable.
- After each player taps `Done`, the turn moves to the next player.

---

## 6. Voting During Turns

While turns are happening, each player can privately save a draft guess for who they think the impostor is.

### Shows

Current guess section:

> Your current guess

Player buttons:

- Ana
- Bob
- Carlos
- Mia

### Rules

- Draft guesses are optional.
- Players can change their draft guess while turns continue.
- Draft guesses do not count as final votes.

---

## 7. Final Voting Screen

After the last turn, each player votes privately for who they think the impostor is.

### Shows

Title for players without a draft guess:

> Vote now

Title for players with a draft guess:

> Confirm your vote

Player buttons:

- Ana
- Bob
- Carlos
- Mia

Players with a draft guess can submit that vote or change it first.

After submitting a final vote, show:

> Vote locked in. Waiting for everyone else...

### Rules

- Each player gets one final vote.
- A player can change their draft guess until submitting.
- A final submitted vote is locked.
- Voting for yourself can be allowed for simplicity.

---

## 8. Results Screen

After everyone votes, reveal the result.

### Shows

Impostor reveal:

> The impostor was:

Big name:

> Carlos

Secret word reveal:

> The secret word was:

Big word:

> Pizza

Outcome:

> Players win!

or:

> Impostor wins!

### Vote Summary

Show who voted for who:

- Ana voted Carlos
- Bob voted Carlos
- Carlos voted Ana
- Mia voted Carlos

### Buttons

- `Play Again`
- `Back to Lobby`

### Win Rules

- If the impostor receives the most votes, the players win.
- If the impostor does not receive the most votes, the impostor wins.

---

## 9. Room Not Found / Error Screen

If a player enters a room code that does not exist, show an error screen or inline error.

### Shows

> Room not found.

Buttons:

- `Try Again`
- `Create Room`

---

## Simple UI Guidelines

The game should be easy to use during a party.

### Design Goals

- Big readable text
- Big buttons
- Mobile-first layout
- No accounts
- No complicated menus
- Minimal typing
- Clear room code display
- Clear current action on every screen

### Recommended Button Style

Each screen should have one obvious primary action, such as:

- `Create Room`
- `Join`
- `Start Game`
- `I'm ready`
- `Done`
- `Vote`
- `Play Again`

---

## Extra Features for Later

These are not required for the first version, but could improve the game later.

- Custom word lists
- More than one impostor for large groups
- Timer for turns
- QR code for joining a room
- Host can kick players from the lobby
- Rejoin support after refresh
- Different game modes
- Score tracking across rounds
