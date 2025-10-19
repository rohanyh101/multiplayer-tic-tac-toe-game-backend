package game

import (
	"errors"
	"tictactoe-server/models"
)

// GameEngine handles the game logic
type GameEngine struct{}

// NewGameEngine creates a new game engine
func NewGameEngine() *GameEngine {
	return &GameEngine{}
}

// IsValidMove checks if a move is valid
func (ge *GameEngine) IsValidMove(game *models.Game, playerID string, position int) error {
	if game.Status != models.STATUS_PLAYING {
		return errors.New("game is not in playing state")
	}

	if position < 0 || position > 8 {
		return errors.New("invalid position")
	}

	if game.Board[position] != "" {
		return errors.New("position already occupied")
	}

	// Check if it's the player's turn
	var playerSymbol string
	if game.PlayerX != nil && game.PlayerX.ID == playerID {
		playerSymbol = "X"
	} else if game.PlayerO != nil && game.PlayerO.ID == playerID {
		playerSymbol = "O"
	} else {
		return errors.New("player not in this game")
	}

	if game.CurrentTurn != playerSymbol {
		return errors.New("not your turn")
	}

	return nil
}

// MakeMove executes a move and updates the game state
func (ge *GameEngine) MakeMove(game *models.Game, playerID string, position int) error {
	if err := ge.IsValidMove(game, playerID, position); err != nil {
		return err
	}

	// Make the move
	game.Board[position] = game.CurrentTurn

	// Check for winner
	winner := ge.CheckWinner(game.Board)
	if winner != "" {
		game.Status = models.STATUS_FINISHED
		game.Winner = winner
		ge.updatePlayerStats(game)
	} else if ge.IsBoardFull(game.Board) {
		game.Status = models.STATUS_FINISHED
		game.Winner = "draw"
		ge.updatePlayerStats(game)
	} else {
		// Switch turns
		if game.CurrentTurn == "X" {
			game.CurrentTurn = "O"
		} else {
			game.CurrentTurn = "X"
		}
	}

	return nil
}

// CheckWinner checks if there's a winner on the board
func (ge *GameEngine) CheckWinner(board [9]string) string {
	// Winning combinations
	winningCombos := [][]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // Rows
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // Columns
		{0, 4, 8}, {2, 4, 6}, // Diagonals
	}

	for _, combo := range winningCombos {
		if board[combo[0]] != "" &&
			board[combo[0]] == board[combo[1]] &&
			board[combo[1]] == board[combo[2]] {
			return board[combo[0]]
		}
	}

	return ""
}

// IsBoardFull checks if the board is full
func (ge *GameEngine) IsBoardFull(board [9]string) bool {
	for _, cell := range board {
		if cell == "" {
			return false
		}
	}
	return true
}

// updatePlayerStats updates player statistics after a game
func (ge *GameEngine) updatePlayerStats(game *models.Game) {
	if game.PlayerX == nil || game.PlayerO == nil {
		return
	}

	switch game.Winner {
	case "X":
		game.PlayerX.Wins++
		game.PlayerO.Losses++
		ge.updateRating(game.PlayerX, game.PlayerO, 1.0) // X wins
	case "O":
		game.PlayerO.Wins++
		game.PlayerX.Losses++
		ge.updateRating(game.PlayerX, game.PlayerO, 0.0) // O wins
	case "draw":
		game.PlayerX.Draws++
		game.PlayerO.Draws++
		ge.updateRating(game.PlayerX, game.PlayerO, 0.5) // Draw
	}
}

// updateRating updates player ratings using a simplified ELO system
func (ge *GameEngine) updateRating(playerX, playerO *models.Player, score float64) {
	const K = 32 // ELO K-factor

	expectedX := 1.0 / (1.0 + float64(10.0^((playerO.Rating-playerX.Rating)/400.0)))

	ratingChangeX := int(K * (score - expectedX))
	ratingChangeO := int(K * ((1.0 - score) - (1.0 - expectedX)))

	playerX.Rating += ratingChangeX
	playerO.Rating += ratingChangeO

	// Ensure ratings don't go below 0
	if playerX.Rating < 0 {
		playerX.Rating = 0
	}
	if playerO.Rating < 0 {
		playerO.Rating = 0
	}
}

// GetGameStateForPlayer returns the game state from a player's perspective
func (ge *GameEngine) GetGameStateForPlayer(game *models.Game, playerID string) map[string]interface{} {
	var mySymbol string
	var opponentName string

	if game.PlayerX != nil && game.PlayerX.ID == playerID {
		mySymbol = "X"
		if game.PlayerO != nil {
			opponentName = game.PlayerO.Name
		}
	} else if game.PlayerO != nil && game.PlayerO.ID == playerID {
		mySymbol = "O"
		if game.PlayerX != nil {
			opponentName = game.PlayerX.Name
		}
	}

	return map[string]interface{}{
		"gameId":       game.ID,
		"board":        game.Board,
		"currentTurn":  game.CurrentTurn,
		"status":       game.Status,
		"winner":       game.Winner,
		"mySymbol":     mySymbol,
		"opponentName": opponentName,
		"isMyTurn":     game.CurrentTurn == mySymbol && game.Status == models.STATUS_PLAYING,
	}
}
