package main

import (
	"log"
	"net/http"
	"os"

	"tictactoe-server/handlers"

	"github.com/rs/cors"
)

func main() {
	// Create game server
	gameServer := handlers.NewGameServer()
	gameServer.Run()

	// Set up HTTP routes
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", gameServer.HandleWebSocket)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Enable CORS for cross-origin requests (frontend will be on different domain)
	// Get allowed origins from environment variable for security
	allowedOrigins := []string{"http://localhost:3000"} // Default for local development
	if frontendURL := os.Getenv("FRONTEND_URL"); frontendURL != "" {
		allowedOrigins = []string{frontendURL}
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(mux)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🎮 Multiplayer Tic-Tac-Toe Server starting on port %s", port)
	log.Printf("🌐 Allowed CORS origins: %v", allowedOrigins)
	log.Printf("✅ Health check: /health | WebSocket: /ws")
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
