# Multiplayer Tic-Tac-Toe Server

A high-performance, server-authoritative multiplayer Tic-Tac-Toe game backend built with Go and WebSockets.

## Features

- **Server-Authoritative Architecture**: All game logic runs on the server to prevent cheating
- **Real-time WebSocket Communication**: Instant game updates and player interactions
- **Player Rating System**: ELO-like rating system with win/loss tracking
- **Live Leaderboard**: Real-time player rankings
- **Automatic Matchmaking**: Queue-based player matching system
- **Concurrent Game Sessions**: Support for multiple simultaneous games
- **Health Monitoring**: Built-in health check endpoint for deployment monitoring

## Technology Stack

- **Go 1.21**: High-performance backend language
- **Gorilla WebSocket**: Reliable WebSocket implementation
- **CORS Middleware**: Cross-origin request handling
- **UUID**: Unique game and player identification
- **Docker**: Containerized deployment

## Architecture

- **Game Engine**: Pure game logic implementation
- **WebSocket Handlers**: Real-time communication management  
- **Player Management**: Session and rating tracking
- **Matchmaking System**: Queue-based player pairing
- **Concurrent Safety**: Mutex-protected shared state

Built for scalability and reliability in production environments.
