package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/kirkegaard/go-spotify-bingo/pkg/config"
	"github.com/kirkegaard/go-spotify-bingo/pkg/database"
	"github.com/kirkegaard/go-spotify-bingo/pkg/handlers"
)

func main() {
	// Load .env file if it exists
	godotenv.Load()

	cfg := config.Load()

	// Debug: Check if Spotify credentials are loaded
	if cfg.SpotifyID == "" {
		log.Fatal("SPOTIFY_CLIENT_ID is not set. Please check your .env file or environment variables.")
	}
	if cfg.SpotifySecret == "" {
		log.Fatal("SPOTIFY_CLIENT_SECRET is not set. Please check your .env file or environment variables.")
	}

	log.Printf("Loaded config - Base URL: %s, Port: %s", cfg.BaseURL, cfg.Port)

	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	authHandler := handlers.NewAuthHandler(db, cfg)
	gameHandler := handlers.NewGameHandler(db)

	http.Handle("GET /", http.FileServer(http.Dir("./www/")))

	http.HandleFunc("GET /auth/spotify", authHandler.SpotifyLogin)
	http.HandleFunc("GET /auth/callback", authHandler.SpotifyCallback)
	http.HandleFunc("GET /api/user", authHandler.UserInfo)

	http.HandleFunc("POST /api/games", gameHandler.CreateGame)
	http.HandleFunc("GET /api/games/join", gameHandler.JoinGame)
	http.HandleFunc("GET /api/games/all-plates", gameHandler.GetAllPlates)

	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
