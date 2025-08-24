package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func New(dbPath string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{sqlDB}

	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

func (db *DB) createTables() error {
	// First, create tables with original schema
	queries := []string{
		`CREATE TABLE IF NOT EXISTS games (
			game_code TEXT PRIMARY KEY,
			creator_session_id TEXT NOT NULL,
			player_count INTEGER NOT NULL,
			playlist_data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS plates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_code TEXT NOT NULL,
			user_session_id TEXT NOT NULL,
			plate_number INTEGER NOT NULL,
			fields TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (game_code) REFERENCES games(game_code),
			UNIQUE(game_code, user_session_id, plate_number)
		)`,
		`CREATE TABLE IF NOT EXISTS user_sessions (
			session_id TEXT PRIMARY KEY,
			spotify_token TEXT,
			expires_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %s, error: %w", query, err)
		}
	}

	// Run migrations for existing databases
	if err := db.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func (db *DB) runMigrations() error {
	// Check existing columns in games table
	rows, err := db.Query("PRAGMA table_info(games)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasContentType := false
	hasPlatesPerPlayer := false
	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue interface{}
		var pk int

		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			continue
		}

		if name == "content_type" {
			hasContentType = true
		}
		if name == "plates_per_player" {
			hasPlatesPerPlayer = true
		}
	}

	// Add content_type column if it doesn't exist
	if !hasContentType {
		_, err = db.Exec("ALTER TABLE games ADD COLUMN content_type TEXT NOT NULL DEFAULT 'mixed'")
		if err != nil {
			return fmt.Errorf("failed to add content_type column: %w", err)
		}
	}

	// Add plates_per_player column if it doesn't exist
	if !hasPlatesPerPlayer {
		_, err = db.Exec("ALTER TABLE games ADD COLUMN plates_per_player INTEGER NOT NULL DEFAULT 3")
		if err != nil {
			return fmt.Errorf("failed to add plates_per_player column: %w", err)
		}
	}

	return nil
}
