package models

import (
	"time"

	"github.com/google/uuid"
)

// Player represents a player in the game
type Player struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Symbol   string    `json:"symbol"` // "X" or "O"
	Wins     int       `json:"wins"`
	Losses   int       `json:"losses"`
	Draws    int       `json:"draws"`
	Rating   int       `json:"rating"`
	LastSeen time.Time `json:"lastSeen"`
}

// Game represents a Tic-Tac-Toe game
type Game struct {
	ID          string     `json:"id"`
	Board       [9]string  `json:"board"` // 0-8 positions, empty string means empty cell
	PlayerX     *Player    `json:"playerX"`
	PlayerO     *Player    `json:"playerO"`
	CurrentTurn string     `json:"currentTurn"` // "X" or "O"
	Status      string     `json:"status"`      // "waiting", "playing", "finished"
	Winner      string     `json:"winner"`      // "X", "O", "draw", or ""
	StartTime   time.Time  `json:"startTime"`
	EndTime     *time.Time `json:"endTime,omitempty"`
}

// Move represents a player's move
type Move struct {
	GameID   string `json:"gameId"`
	PlayerID string `json:"playerId"`
	Position int    `json:"position"` // 0-8
}

// GameMessage represents WebSocket messages
type GameMessage struct {
	Type     string      `json:"type"`
	Data     interface{} `json:"data"`
	GameID   string      `json:"gameId,omitempty"`
	PlayerID string      `json:"playerId,omitempty"`
}

// MessageTypes
const (
	MSG_JOIN_QUEUE    = "join_queue"
	MSG_LEAVE_QUEUE   = "leave_queue"
	MSG_GAME_FOUND    = "game_found"
	MSG_MAKE_MOVE     = "make_move"
	MSG_GAME_UPDATE   = "game_update"
	MSG_GAME_END      = "game_end"
	MSG_ERROR         = "error"
	MSG_LEADERBOARD   = "leaderboard"
	MSG_PLAYER_UPDATE = "player_update"
)

// GameStatus constants
const (
	STATUS_WAITING  = "waiting"
	STATUS_PLAYING  = "playing"
	STATUS_FINISHED = "finished"
)

// NewGame creates a new game instance
func NewGame() *Game {
	return &Game{
		ID:          uuid.New().String(),
		Board:       [9]string{},
		CurrentTurn: "X",
		Status:      STATUS_WAITING,
		StartTime:   time.Now(),
	}
}

// NewPlayer creates a new player
func NewPlayer(name string) *Player {
	return &Player{
		ID:       uuid.New().String(),
		Name:     name,
		Rating:   1000, // Starting rating
		LastSeen: time.Now(),
	}
}
