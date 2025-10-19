package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"tictactoe-server/game"
	"tictactoe-server/models"

	"github.com/gorilla/websocket"
)

// GameServer manages all game sessions and players
type GameServer struct {
	clients     map[*websocket.Conn]*models.Player
	games       map[string]*models.Game
	players     map[string]*models.Player
	matchmaking []string // Queue of player IDs waiting for a match
	gameEngine  *game.GameEngine
	upgrader    websocket.Upgrader
	mutex       sync.RWMutex
	broadcast   chan *models.GameMessage
}

// NewGameServer creates a new game server
func NewGameServer() *GameServer {
	return &GameServer{
		clients:     make(map[*websocket.Conn]*models.Player),
		games:       make(map[string]*models.Game),
		players:     make(map[string]*models.Player),
		matchmaking: make([]string, 0),
		gameEngine:  game.NewGameEngine(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for development and production
				// In production, you could restrict this to specific domains
				return true
			},
		},
		broadcast: make(chan *models.GameMessage, 256),
	}
}

// Run starts the game server
func (gs *GameServer) Run() {
	go gs.handleBroadcast()
}

// HandleWebSocket handles WebSocket connections
func (gs *GameServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := gs.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Get player name from query parameter
	playerName := r.URL.Query().Get("name")
	if playerName == "" {
		playerName = "Anonymous"
	}

	// Create or get existing player
	player := models.NewPlayer(playerName)

	gs.mutex.Lock()
	gs.clients[conn] = player
	gs.players[player.ID] = player
	gs.mutex.Unlock()

	log.Printf("New player connected: %s (ID: %s)", player.Name, player.ID)

	// Send player info
	gs.sendToClient(conn, &models.GameMessage{
		Type: models.MSG_PLAYER_UPDATE,
		Data: player,
	})

	// Send current leaderboard
	gs.sendLeaderboard(conn)

	// Handle messages
	for {
		var msg models.GameMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		gs.handleMessage(conn, &msg)
	}

	// Clean up on disconnect
	gs.handleDisconnect(conn)
}

// handleMessage processes incoming WebSocket messages
func (gs *GameServer) handleMessage(conn *websocket.Conn, msg *models.GameMessage) {
	gs.mutex.Lock()
	player, exists := gs.clients[conn]
	gs.mutex.Unlock()

	if !exists {
		return
	}

	msg.PlayerID = player.ID

	switch msg.Type {
	case models.MSG_JOIN_QUEUE:
		gs.handleJoinQueue(player)
	case models.MSG_LEAVE_QUEUE:
		gs.handleLeaveQueue(player)
	case models.MSG_MAKE_MOVE:
		gs.handleMakeMove(msg)
	case models.MSG_LEADERBOARD:
		gs.sendLeaderboard(conn)
	}
}

// handleJoinQueue adds a player to the matchmaking queue
func (gs *GameServer) handleJoinQueue(player *models.Player) {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	// Check if player is already in queue
	for _, playerID := range gs.matchmaking {
		if playerID == player.ID {
			log.Printf("Player %s (%s) already in queue", player.Name, player.ID)
			return
		}
	}

	// Add to queue
	gs.matchmaking = append(gs.matchmaking, player.ID)
	log.Printf("Player %s (%s) added to queue. Queue size: %d", player.Name, player.ID, len(gs.matchmaking))

	// Try to match players
	if len(gs.matchmaking) >= 2 {
		log.Printf("Attempting to create match with %d players in queue", len(gs.matchmaking))
		gs.createMatch()
	}
}

// handleLeaveQueue removes a player from the matchmaking queue
func (gs *GameServer) handleLeaveQueue(player *models.Player) {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	for i, playerID := range gs.matchmaking {
		if playerID == player.ID {
			gs.matchmaking = append(gs.matchmaking[:i], gs.matchmaking[i+1:]...)
			break
		}
	}
}

// createMatch creates a new game between two players
func (gs *GameServer) createMatch() {
	if len(gs.matchmaking) < 2 {
		log.Printf("Not enough players in queue: %d", len(gs.matchmaking))
		return
	}

	// Get first two players from queue
	player1ID := gs.matchmaking[0]
	player2ID := gs.matchmaking[1]
	gs.matchmaking = gs.matchmaking[2:]

	player1, exists1 := gs.players[player1ID]
	player2, exists2 := gs.players[player2ID]

	if !exists1 || !exists2 {
		log.Printf("One or both players not found: player1=%v, player2=%v", exists1, exists2)
		return
	}

	// Create new game
	newGame := models.NewGame()
	newGame.PlayerX = player1
	newGame.PlayerO = player2
	newGame.Status = models.STATUS_PLAYING
	player1.Symbol = "X"
	player2.Symbol = "O"

	gs.games[newGame.ID] = newGame

	log.Printf("Created game %s between %s (X) and %s (O)", newGame.ID, player1.Name, player2.Name)

	// Notify both players
	gameFoundMsg := &models.GameMessage{
		Type:   models.MSG_GAME_FOUND,
		Data:   gs.gameEngine.GetGameStateForPlayer(newGame, player1.ID),
		GameID: newGame.ID,
	}

	log.Printf("Sending game found message to player1: %s", player1.Name)
	gs.sendToPlayer(player1.ID, gameFoundMsg)

	gameFoundMsg.Data = gs.gameEngine.GetGameStateForPlayer(newGame, player2.ID)
	log.Printf("Sending game found message to player2: %s", player2.Name)
	gs.sendToPlayer(player2.ID, gameFoundMsg)
}

// handleMakeMove processes a player's move
func (gs *GameServer) handleMakeMove(msg *models.GameMessage) {
	var moveData map[string]interface{}
	moveBytes, _ := json.Marshal(msg.Data)
	json.Unmarshal(moveBytes, &moveData)

	gameID, ok := moveData["gameId"].(string)
	if !ok {
		return
	}

	position, ok := moveData["position"].(float64)
	if !ok {
		return
	}

	gs.mutex.Lock()
	gameInstance, exists := gs.games[gameID]
	gs.mutex.Unlock()

	if !exists {
		gs.sendError(msg.PlayerID, "Game not found")
		return
	}

	// Make the move
	err := gs.gameEngine.MakeMove(gameInstance, msg.PlayerID, int(position))
	if err != nil {
		gs.sendError(msg.PlayerID, err.Error())
		return
	}

	// Send game update to both players
	gs.sendGameUpdate(gameInstance)

	// If game is finished, update leaderboard
	if gameInstance.Status == models.STATUS_FINISHED {
		now := time.Now()
		gameInstance.EndTime = &now
		gs.broadcastLeaderboard()
	}
}

// sendGameUpdate sends game state to both players
func (gs *GameServer) sendGameUpdate(gameInstance *models.Game) {
	if gameInstance.PlayerX != nil {
		updateMsg := &models.GameMessage{
			Type:   models.MSG_GAME_UPDATE,
			Data:   gs.gameEngine.GetGameStateForPlayer(gameInstance, gameInstance.PlayerX.ID),
			GameID: gameInstance.ID,
		}
		gs.sendToPlayer(gameInstance.PlayerX.ID, updateMsg)
	}

	if gameInstance.PlayerO != nil {
		updateMsg := &models.GameMessage{
			Type:   models.MSG_GAME_UPDATE,
			Data:   gs.gameEngine.GetGameStateForPlayer(gameInstance, gameInstance.PlayerO.ID),
			GameID: gameInstance.ID,
		}
		gs.sendToPlayer(gameInstance.PlayerO.ID, updateMsg)
	}
}

// sendToPlayer sends a message to a specific player
func (gs *GameServer) sendToPlayer(playerID string, msg *models.GameMessage) {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()

	log.Printf("Attempting to send %s message to player %s", msg.Type, playerID)

	found := false
	for conn, player := range gs.clients {
		if player.ID == playerID {
			log.Printf("Found player %s, sending message", playerID)
			gs.sendToClient(conn, msg)
			found = true
			break
		}
	}

	if !found {
		log.Printf("ERROR: Player %s not found in clients map!", playerID)
	}
}

// sendToClient sends a message to a WebSocket connection
func (gs *GameServer) sendToClient(conn *websocket.Conn, msg *models.GameMessage) {
	err := conn.WriteJSON(msg)
	if err != nil {
		log.Printf("WebSocket write error: %v", err)
		conn.Close()
	} else {
		log.Printf("Message %s sent successfully", msg.Type)
	}
}

// sendError sends an error message to a player
func (gs *GameServer) sendError(playerID string, errorMsg string) {
	msg := &models.GameMessage{
		Type: models.MSG_ERROR,
		Data: map[string]string{"error": errorMsg},
	}
	gs.sendToPlayer(playerID, msg)
}

// sendLeaderboard sends the leaderboard to a specific connection
func (gs *GameServer) sendLeaderboard(conn *websocket.Conn) {
	leaderboard := gs.getLeaderboard()
	msg := &models.GameMessage{
		Type: models.MSG_LEADERBOARD,
		Data: leaderboard,
	}
	gs.sendToClient(conn, msg)
}

// broadcastLeaderboard sends the leaderboard to all connected players
func (gs *GameServer) broadcastLeaderboard() {
	leaderboard := gs.getLeaderboard()
	msg := &models.GameMessage{
		Type: models.MSG_LEADERBOARD,
		Data: leaderboard,
	}
	gs.broadcast <- msg
}

// getLeaderboard returns the top players sorted by rating
func (gs *GameServer) getLeaderboard() []*models.Player {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()

	players := make([]*models.Player, 0, len(gs.players))
	for _, player := range gs.players {
		// Only include players who have played at least one game
		if player.Wins+player.Losses+player.Draws > 0 {
			players = append(players, player)
		}
	}

	// Sort by rating (descending)
	sort.Slice(players, func(i, j int) bool {
		return players[i].Rating > players[j].Rating
	})

	// Return top 10
	if len(players) > 10 {
		players = players[:10]
	}

	return players
}

// handleBroadcast processes broadcast messages
func (gs *GameServer) handleBroadcast() {
	for msg := range gs.broadcast {
		gs.mutex.RLock()
		for conn := range gs.clients {
			gs.sendToClient(conn, msg)
		}
		gs.mutex.RUnlock()
	}
}

// handleDisconnect cleans up when a player disconnects
func (gs *GameServer) handleDisconnect(conn *websocket.Conn) {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	player, exists := gs.clients[conn]
	if !exists {
		return
	}

	// Remove from queue if present
	for i, playerID := range gs.matchmaking {
		if playerID == player.ID {
			gs.matchmaking = append(gs.matchmaking[:i], gs.matchmaking[i+1:]...)
			break
		}
	}

	// Update last seen time
	player.LastSeen = time.Now()

	delete(gs.clients, conn)
}
