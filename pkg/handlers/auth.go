package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/kirkegaard/go-spotify-bingo/pkg/config"
	"github.com/kirkegaard/go-spotify-bingo/pkg/database"
	"github.com/kirkegaard/go-spotify-bingo/pkg/models"
	"github.com/kirkegaard/go-spotify-bingo/pkg/spotify"
)

type AuthHandler struct {
	db          *database.DB
	spotifyAuth *spotify.AuthConfig
}

func NewAuthHandler(db *database.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		db: db,
		spotifyAuth: &spotify.AuthConfig{
			ClientID:     cfg.SpotifyID,
			ClientSecret: cfg.SpotifySecret,
			RedirectURI:  cfg.BaseURL + "/auth/callback",
		},
	}
}

func (h *AuthHandler) SpotifyLogin(w http.ResponseWriter, r *http.Request) {
	sessionID := generateSessionID()

	session := models.UserSession{
		SessionID: sessionID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	_, err := h.db.Exec(`INSERT INTO user_sessions (session_id, created_at, expires_at) VALUES (?, ?, ?)`,
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

	authURL := h.spotifyAuth.GetAuthURL(sessionID)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) SpotifyCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		http.Error(w, "Authorization code missing", http.StatusBadRequest)
		return
	}

	tokenResp, err := h.spotifyAuth.ExchangeCodeForToken(code)
	if err != nil {
		http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	_, err = h.db.Exec(`UPDATE user_sessions SET spotify_token = ?, expires_at = ? WHERE session_id = ?`,
		tokenResp.AccessToken, expiresAt, state)
	if err != nil {
		http.Error(w, "Failed to store token", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

type UserInfoResponse struct {
	Authenticated bool           `json:"authenticated"`
	SessionID     string         `json:"session_id,omitempty"`
	Playlists     []PlaylistInfo `json:"playlists,omitempty"`
}

type PlaylistInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *AuthHandler) UserInfo(w http.ResponseWriter, r *http.Request) {
	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		json.NewEncoder(w).Encode(UserInfoResponse{Authenticated: false})
		return
	}

	var session models.UserSession
	err = h.db.QueryRow(`SELECT session_id, spotify_token, expires_at FROM user_sessions WHERE session_id = ?`,
		sessionCookie.Value).Scan(&session.SessionID, &session.SpotifyToken, &session.ExpiresAt)

	if err != nil || session.SpotifyToken == "" || time.Now().After(session.ExpiresAt) {
		json.NewEncoder(w).Encode(UserInfoResponse{Authenticated: false})
		return
	}

	client := spotify.NewClient(session.SpotifyToken)
	playlists, err := client.GetUserPlaylists()
	if err != nil {
		json.NewEncoder(w).Encode(UserInfoResponse{
			Authenticated: true,
			SessionID:     session.SessionID,
		})
		return
	}

	var playlistInfos []PlaylistInfo
	for _, playlist := range playlists {
		playlistInfos = append(playlistInfos, PlaylistInfo{
			ID:   playlist.ID,
			Name: playlist.Name,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UserInfoResponse{
		Authenticated: true,
		SessionID:     session.SessionID,
		Playlists:     playlistInfos,
	})
}

func generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
