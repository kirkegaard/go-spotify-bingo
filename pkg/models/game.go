package models

import (
	"encoding/json"
	"time"
)

const (
	ContentTypeCombined = "combined"
	ContentTypeMixed    = "mixed"
	ContentTypeTracks   = "tracks"
	ContentTypeArtists  = "artists"
)

type Game struct {
	GameCode        string       `json:"game_code" db:"game_code"`
	CreatorID       string       `json:"creator_id" db:"creator_session_id"`
	PlayerCount     int          `json:"player_count" db:"player_count"`
	PlatesPerPlayer int          `json:"plates_per_player" db:"plates_per_player"`
	ContentType     string       `json:"content_type" db:"content_type"`
	PlaylistData    PlaylistData `json:"playlist_data" db:"playlist_data"`
	CreatedAt       time.Time    `json:"created_at" db:"created_at"`
}

type PlaylistData struct {
	PlaylistID   string  `json:"playlist_id"`
	PlaylistName string  `json:"playlist_name"`
	Tracks       []Track `json:"tracks"`
}

type Track struct {
	Name    string   `json:"name"`
	Artists []string `json:"artists"`
	ID      string   `json:"id"`
}

type Plate struct {
	ID            int         `json:"id" db:"id"`
	GameCode      string      `json:"game_code" db:"game_code"`
	UserSessionID string      `json:"user_session_id" db:"user_session_id"`
	PlateNumber   int         `json:"plate_number" db:"plate_number"`
	Fields        PlateFields `json:"fields" db:"fields"`
}

type PlateFields struct {
	Grid [3][9]BingoField `json:"grid"`
}

type BingoField struct {
	Content string `json:"content"`
	Type    string `json:"type"`
	Marked  bool   `json:"marked"`
}

type UserSession struct {
	SessionID    string    `json:"session_id" db:"session_id"`
	SpotifyToken string    `json:"spotify_token,omitempty" db:"spotify_token"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

func (pd PlaylistData) ToJSON() (string, error) {
	data, err := json.Marshal(pd)
	return string(data), err
}

func PlaylistDataFromJSON(data string) (PlaylistData, error) {
	var pd PlaylistData
	err := json.Unmarshal([]byte(data), &pd)
	return pd, err
}

func (pf PlateFields) ToJSON() (string, error) {
	data, err := json.Marshal(pf)
	return string(data), err
}

func PlateFieldsFromJSON(data string) (PlateFields, error) {
	var pf PlateFields
	err := json.Unmarshal([]byte(data), &pf)
	return pf, err
}
