package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/kirkegaard/go-spotify-bingo/pkg/database"
	"github.com/kirkegaard/go-spotify-bingo/pkg/generator"
	"github.com/kirkegaard/go-spotify-bingo/pkg/models"
	"github.com/kirkegaard/go-spotify-bingo/pkg/spotify"
)

type GameHandler struct {
	db        *database.DB
	generator *generator.Generator
}

func NewGameHandler(db *database.DB) *GameHandler {
	return &GameHandler{
		db:        db,
		generator: generator.New(),
	}
}

type CreateGameRequest struct {
	PlaylistID      string `json:"playlist_id"`
	PlaylistURL     string `json:"playlist_url"`
	PlayerCount     int    `json:"player_count"`
	PlatesPerPlayer int    `json:"plates_per_player"`
	ContentType     string `json:"content_type"`
}

type CreateGameResponse struct {
	GameCode string         `json:"game_code"`
	Plates   []models.Plate `json:"plates"`
}

func (h *GameHandler) CreateGame(w http.ResponseWriter, r *http.Request) {
	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	var req CreateGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.PlayerCount <= 0 || req.PlayerCount > 20 {
		http.Error(w, "Player count must be between 1 and 20", http.StatusBadRequest)
		return
	}

	// Validate and set default plates per player
	if req.PlatesPerPlayer <= 0 {
		req.PlatesPerPlayer = 3
	}
	if req.PlatesPerPlayer > 10 {
		http.Error(w, "Plates per player must be 10 or fewer", http.StatusBadRequest)
		return
	}

	// Validate content type
	if req.ContentType == "" {
		req.ContentType = models.ContentTypeMixed
	}
	if req.ContentType != models.ContentTypeMixed && req.ContentType != models.ContentTypeTracks && req.ContentType != models.ContentTypeArtists && req.ContentType != models.ContentTypeCombined {
		http.Error(w, "Invalid content type", http.StatusBadRequest)
		return
	}

	var session models.UserSession
	err = h.db.QueryRow(`SELECT session_id, spotify_token, expires_at FROM user_sessions WHERE session_id = ?`,
		sessionCookie.Value).Scan(&session.SessionID, &session.SpotifyToken, &session.ExpiresAt)

	if err != nil || session.SpotifyToken == "" || time.Now().After(session.ExpiresAt) {
		http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
		return
	}

	client := spotify.NewClient(session.SpotifyToken)

	var playlistID string
	if req.PlaylistURL != "" {
		playlistID = extractPlaylistIDFromURL(req.PlaylistURL)
	} else {
		playlistID = req.PlaylistID
	}

	if playlistID == "" {
		http.Error(w, "Invalid playlist", http.StatusBadRequest)
		return
	}

	playlist, err := client.GetPlaylistByID(playlistID)
	if err != nil {
		log.Printf("Error fetching playlist by ID %s: %v", playlistID, err)
		http.Error(w, "Failed to fetch playlist info", http.StatusInternalServerError)
		return
	}

	playlistData, err := client.GetPlaylistTracks(playlistID)
	if err != nil {
		log.Printf("Error fetching playlist tracks for ID %s: %v", playlistID, err)
		http.Error(w, "Failed to fetch playlist tracks", http.StatusInternalServerError)
		return
	}

	playlistData.PlaylistName = playlist.Name

	requiredTracks := req.PlayerCount * req.PlatesPerPlayer * 5
	totalPlates := req.PlayerCount * req.PlatesPerPlayer
	if len(playlistData.Tracks) < requiredTracks {
		http.Error(w, fmt.Sprintf("Playlist must have at least %d tracks for %d players (%d plates total)", requiredTracks, req.PlayerCount, totalPlates), http.StatusBadRequest)
		return
	}

	gameCode := generator.GenerateGameCode()
	playlistJSON, _ := playlistData.ToJSON()

	game := models.Game{
		GameCode:        gameCode,
		CreatorID:       session.SessionID,
		PlayerCount:     req.PlayerCount,
		PlatesPerPlayer: req.PlatesPerPlayer,
		ContentType:     req.ContentType,
		PlaylistData:    playlistData,
		CreatedAt:       time.Now(),
	}

	_, err = h.db.Exec(`INSERT INTO games (game_code, creator_session_id, player_count, plates_per_player, content_type, playlist_data, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		game.GameCode, game.CreatorID, game.PlayerCount, game.PlatesPerPlayer, game.ContentType, playlistJSON, game.CreatedAt)
	if err != nil {
		log.Printf("Error creating game in database: %v", err)
		http.Error(w, "Failed to create game", http.StatusInternalServerError)
		return
	}

	// Generate plates for all players (reuse totalPlates from validation above)
	var plateFields []models.PlateFields
	plateFields, err = h.generator.GeneratePlates(playlistData, totalPlates, req.ContentType)
	if err != nil {
		http.Error(w, "Failed to generate plates", http.StatusInternalServerError)
		return
	}

	var creatorPlates []models.Plate
	plateNumber := 1

	for playerNum := 1; playerNum <= req.PlayerCount; playerNum++ {
		for plateInSet := 1; plateInSet <= req.PlatesPerPlayer; plateInSet++ {
			fields := plateFields[plateNumber-1]
			fieldsJSON, _ := fields.ToJSON()

			var userSessionID string
			if playerNum == 1 {
				// First player is the creator
				userSessionID = session.SessionID
			} else {
				// Use placeholder for unassigned players
				userSessionID = fmt.Sprintf("PLAYER_%d", playerNum)
			}

			plate := models.Plate{
				GameCode:      gameCode,
				UserSessionID: userSessionID,
				PlateNumber:   plateInSet,
				Fields:        fields,
			}

			_, err = h.db.Exec(`INSERT INTO plates (game_code, user_session_id, plate_number, fields) VALUES (?, ?, ?, ?)`,
				plate.GameCode, plate.UserSessionID, plate.PlateNumber, fieldsJSON)
			if err != nil {
				http.Error(w, "Failed to save plates", http.StatusInternalServerError)
				return
			}

			// Only add creator's plates to the response
			if playerNum == 1 {
				creatorPlates = append(creatorPlates, plate)
			}
			plateNumber++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateGameResponse{
		GameCode: gameCode,
		Plates:   creatorPlates,
	})
}

type JoinGameResponse struct {
	GameCode     string         `json:"game_code"`
	PlaylistName string         `json:"playlist_name"`
	Plates       []models.Plate `json:"plates"`
}

func (h *GameHandler) JoinGame(w http.ResponseWriter, r *http.Request) {
	gameCode := r.URL.Query().Get("code")
	if gameCode == "" {
		http.Error(w, "Game code required", http.StatusBadRequest)
		return
	}

	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		sessionID := generateSessionID()
		session := models.UserSession{
			SessionID: sessionID,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		_, err = h.db.Exec(`INSERT INTO user_sessions (session_id, created_at, expires_at) VALUES (?, ?, ?)`,
			session.SessionID, session.CreatedAt, session.ExpiresAt)
		if err != nil {
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    sessionID,
			Expires:  session.ExpiresAt,
			HttpOnly: true,
			Path:     "/",
		})
		sessionCookie = &http.Cookie{Value: sessionID}
	}

	var existingPlates []models.Plate
	rows, err := h.db.Query(`SELECT id, game_code, user_session_id, plate_number, fields FROM plates WHERE game_code = ? AND user_session_id = ?`,
		gameCode, sessionCookie.Value)
	if err == nil {
		for rows.Next() {
			var plate models.Plate
			var fieldsJSON string
			err := rows.Scan(&plate.ID, &plate.GameCode, &plate.UserSessionID, &plate.PlateNumber, &fieldsJSON)
			if err == nil {
				plate.Fields, _ = models.PlateFieldsFromJSON(fieldsJSON)
				existingPlates = append(existingPlates, plate)
			}
		}
		rows.Close()
	}

	if len(existingPlates) > 0 {
		var game models.Game
		var playlistJSON string
		err = h.db.QueryRow(`SELECT game_code, playlist_data FROM games WHERE game_code = ?`, gameCode).
			Scan(&game.GameCode, &playlistJSON)
		if err != nil {
			http.Error(w, "Game not found", http.StatusNotFound)
			return
		}

		playlistData, _ := models.PlaylistDataFromJSON(playlistJSON)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(JoinGameResponse{
			GameCode:     gameCode,
			PlaylistName: playlistData.PlaylistName,
			Plates:       existingPlates,
		})
		return
	}

	var game models.Game
	var playlistJSON string
	var contentType string
	var platesPerPlayer int
	err = h.db.QueryRow(`SELECT game_code, content_type, plates_per_player, playlist_data FROM games WHERE game_code = ?`, gameCode).
		Scan(&game.GameCode, &contentType, &platesPerPlayer, &playlistJSON)
	if err != nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	playlistData, err := models.PlaylistDataFromJSON(playlistJSON)
	if err != nil {
		http.Error(w, "Invalid game data", http.StatusInternalServerError)
		return
	}

	// Find the first available unassigned player slot
	var assignedPlates []models.Plate
	rows, err = h.db.Query(`SELECT user_session_id, plate_number, fields FROM plates WHERE game_code = ? AND user_session_id LIKE 'PLAYER_%' ORDER BY user_session_id LIMIT ?`, gameCode, platesPerPlayer)
	if err != nil {
		http.Error(w, "Failed to find available plates", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var platesToAssign []models.Plate
	for rows.Next() {
		var userID string
		var plateNumber int
		var fieldsJSON string
		err := rows.Scan(&userID, &plateNumber, &fieldsJSON)
		if err != nil {
			continue
		}

		fields, err := models.PlateFieldsFromJSON(fieldsJSON)
		if err != nil {
			continue
		}

		plate := models.Plate{
			GameCode:      gameCode,
			UserSessionID: userID, // Keep original placeholder for now
			PlateNumber:   plateNumber,
			Fields:        fields,
		}
		platesToAssign = append(platesToAssign, plate)

		if len(platesToAssign) >= platesPerPlayer {
			break
		}
	}

	if len(platesToAssign) < platesPerPlayer {
		http.Error(w, "Game is full - no available player slots", http.StatusBadRequest)
		return
	}

	// Assign the plates to the joining player
	for i, plate := range platesToAssign {
		_, err = h.db.Exec(`UPDATE plates SET user_session_id = ? WHERE game_code = ? AND user_session_id = ? AND plate_number = ?`,
			sessionCookie.Value, gameCode, plate.UserSessionID, plate.PlateNumber)
		if err != nil {
			http.Error(w, "Failed to assign plates", http.StatusInternalServerError)
			return
		}

		// Update the plate for response
		plate.UserSessionID = sessionCookie.Value
		plate.PlateNumber = i + 1 // Renumber for this player (1, 2, 3, etc.)
		assignedPlates = append(assignedPlates, plate)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(JoinGameResponse{
		GameCode:     gameCode,
		PlaylistName: playlistData.PlaylistName,
		Plates:       assignedPlates,
	})
}

func extractPlaylistIDFromURL(url string) string {
	if len(url) > 31 && url[:31] == "https://open.spotify.com/playlist/" {
		end := len(url)
		if qIndex := findChar(url, '?'); qIndex != -1 {
			end = qIndex
		}
		return url[34:end]
	}
	return ""
}

func findChar(s string, char rune) int {
	for i, c := range s {
		if c == char {
			return i
		}
	}
	return -1
}

type AllPlatesResponse struct {
	GameCode     string         `json:"game_code"`
	PlaylistName string         `json:"playlist_name"`
	IsCreator    bool           `json:"is_creator"`
	AllPlates    []PlayerPlates `json:"all_plates"`
}

type PlayerPlates struct {
	PlayerID string         `json:"player_id"`
	Plates   []models.Plate `json:"plates"`
}

func (h *GameHandler) GetAllPlates(w http.ResponseWriter, r *http.Request) {
	gameCode := r.URL.Query().Get("code")
	if gameCode == "" {
		http.Error(w, "Game code required", http.StatusBadRequest)
		return
	}

	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	// Check if user is the game creator
	var game models.Game
	var playlistJSON string
	var creatorID string
	err = h.db.QueryRow(`SELECT game_code, creator_session_id, playlist_data FROM games WHERE game_code = ?`, gameCode).
		Scan(&game.GameCode, &creatorID, &playlistJSON)
	if err != nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	if creatorID != sessionCookie.Value {
		http.Error(w, "Only game creator can view all plates", http.StatusForbidden)
		return
	}

	playlistData, err := models.PlaylistDataFromJSON(playlistJSON)
	if err != nil {
		http.Error(w, "Invalid game data", http.StatusInternalServerError)
		return
	}

	// Get all plates for this game
	rows, err := h.db.Query(`SELECT user_session_id, plate_number, fields FROM plates WHERE game_code = ? ORDER BY user_session_id, plate_number`, gameCode)
	if err != nil {
		http.Error(w, "Failed to fetch plates", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	playerPlatesMap := make(map[string][]models.Plate)
	for rows.Next() {
		var userID string
		var plateNumber int
		var fieldsJSON string
		err := rows.Scan(&userID, &plateNumber, &fieldsJSON)
		if err != nil {
			continue
		}

		fields, err := models.PlateFieldsFromJSON(fieldsJSON)
		if err != nil {
			continue
		}

		plate := models.Plate{
			GameCode:      gameCode,
			UserSessionID: userID,
			PlateNumber:   plateNumber,
			Fields:        fields,
		}

		playerPlatesMap[userID] = append(playerPlatesMap[userID], plate)
	}

	// Convert to response format
	var allPlates []PlayerPlates
	for userID, plates := range playerPlatesMap {
		// Generate a friendly name for each player
		var playerName string
		if userID == creatorID {
			playerName = "Game Creator"
		} else if strings.HasPrefix(userID, "PLAYER_") {
			// Extract player number from placeholder
			playerNum := strings.TrimPrefix(userID, "PLAYER_")
			playerName = fmt.Sprintf("Player %s (Not joined)", playerNum)
		} else {
			// Real player who has joined
			playerName = fmt.Sprintf("Player (Joined)")
		}

		allPlates = append(allPlates, PlayerPlates{
			PlayerID: playerName,
			Plates:   plates,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AllPlatesResponse{
		GameCode:     gameCode,
		PlaylistName: playlistData.PlaylistName,
		IsCreator:    true,
		AllPlates:    allPlates,
	})
}
