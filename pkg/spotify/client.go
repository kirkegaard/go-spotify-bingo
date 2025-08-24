package spotify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"spotify-bingo/pkg/models"
)

type Client struct {
	accessToken string
	httpClient  *http.Client
}

type PlaylistResponse struct {
	Items []PlaylistItem `json:"items"`
}

type PlaylistItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PlaylistTracksResponse struct {
	Items []TrackItem `json:"items"`
	Next  *string     `json:"next"`
}

type TrackItem struct {
	Track Track `json:"track"`
}

type Track struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Artists []Artist `json:"artists"`
}

type Artist struct {
	Name string `json:"name"`
}

func NewClient(accessToken string) *Client {
	return &Client{
		accessToken: accessToken,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) GetUserPlaylists() ([]PlaylistItem, error) {
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me/playlists?limit=50", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get playlists: status %d", resp.StatusCode)
	}

	var playlistResp PlaylistResponse
	if err := json.NewDecoder(resp.Body).Decode(&playlistResp); err != nil {
		return nil, err
	}

	return playlistResp.Items, nil
}

func (c *Client) GetPlaylistTracks(playlistID string) (models.PlaylistData, error) {
	var allTracks []models.Track
	url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks?limit=100", playlistID)

	for url != "" {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return models.PlaylistData{}, err
		}

		req.Header.Set("Authorization", "Bearer "+c.accessToken)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return models.PlaylistData{}, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return models.PlaylistData{}, fmt.Errorf("failed to get playlist tracks: status %d", resp.StatusCode)
		}

		var tracksResp PlaylistTracksResponse
		if err := json.NewDecoder(resp.Body).Decode(&tracksResp); err != nil {
			return models.PlaylistData{}, err
		}

		for _, item := range tracksResp.Items {
			if item.Track.ID != "" {
				var artistNames []string
				for _, artist := range item.Track.Artists {
					artistNames = append(artistNames, artist.Name)
				}

				allTracks = append(allTracks, models.Track{
					ID:      item.Track.ID,
					Name:    cleanTrackName(item.Track.Name),
					Artists: artistNames,
				})
			}
		}

		if tracksResp.Next != nil {
			url = *tracksResp.Next
		} else {
			url = ""
		}
	}

	return models.PlaylistData{
		PlaylistID: playlistID,
		Tracks:     allTracks,
	}, nil
}

func (c *Client) GetPlaylistByID(playlistID string) (*PlaylistItem, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.spotify.com/v1/playlists/%s", playlistID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get playlist: status %d", resp.StatusCode)
	}

	var playlist PlaylistItem
	if err := json.NewDecoder(resp.Body).Decode(&playlist); err != nil {
		return nil, err
	}

	return &playlist, nil
}

// cleanTrackName removes common suffixes from track names to make bingo fields cleaner
func cleanTrackName(name string) string {
	// Remove common patterns in parentheses or after dashes
	patterns := []string{
		// Remix/Edit variations (parentheses)
		`\s*\(.*?[Rr]adio [Ee]dit.*?\)`,
		`\s*\(.*?[Rr]adio [Vv]ersion.*?\)`,
		`\s*\(.*?[Oo]riginal.*?[Rr]adio.*?[Ee]dit.*?\)`,
		`\s*\(.*?[Oo]riginal [Mm]ix.*?\)`,
		`\s*\([Oo]riginal\)`,
		`\s*\(.*?[Aa]lbum [Vv]ersion.*?\)`,
		`\s*\(.*?[Ss]ingle [Vv]ersion.*?\)`,
		`\s*\(.*?[Ee]xtended [Mm]ix.*?\)`,
		`\s*\(.*?[Cc]lub [Mm]ix.*?\)`,
		`\s*\(.*?[Ss]hort [Mm]ix.*?\)`,
		`\s*\(.*?[Vv]ideo [Mm]ix.*?\)`,
		`\s*\(.*?\d+["'""]?\s*[Vv]ersion.*?\)`, // 12" Version, 7" Version, etc.
		`\s*\(.*?\d{4}.*?[Rr]emaster.*?\)`,     // 2005 Remaster, etc.
		`\s*\(.*?[Rr]emix.*?\)`,
		`\s*\(.*?[Ee]dit.*?\)`,
		`\s*\(.*?[Vv]ersion.*?\)`,
		`\s*\(.*?[Rr]emaster.*?\)`,
		`\s*\(.*?[Mm]ix.*?\)`, // Generic mix patterns

		// Dash variations
		`\s*-\s*[Rr]adio [Ee]dit.*$`,
		`\s*-\s*[Rr]adio [Vv]ersion.*$`,
		`\s*-\s*[Oo]riginal.*?[Rr]adio.*?[Ee]dit.*$`,
		`\s*-\s*[Oo]riginal [Mm]ix.*$`,
		`\s*-\s*[Oo]riginal [Aa]lbum [Vv]ersion.*$`,
		`\s*-\s*[Oo]riginal$`,
		`\s*-\s*[Aa]lbum [Vv]ersion.*$`,
		`\s*-\s*[Ss]ingle [Vv]ersion.*$`,
		`\s*-\s*[Ee]xtended [Mm]ix.*$`,
		`\s*-\s*[Cc]lub [Mm]ix.*$`,
		`\s*-\s*[Ss]hort [Mm]ix.*$`,
		`\s*-\s*[Vv]ideo [Mm]ix.*$`,
		`\s*-\s*\d+["'""]?\s*[Vv]ersion.*$`, // 12" Version, 7" Version, etc.
		`\s*-\s*\d{4}.*?[Rr]emaster.*$`,     // 2005 Remaster, etc.
		`\s*-\s*[Rr]emix.*$`,
		`\s*-\s*[Ee]dit.*$`,
		`\s*-\s*[Vv]ersion.*$`,
		`\s*-\s*[Rr]emaster.*$`,
		`\s*-\s*.*?[Mm]ix.*$`, // Generic mix patterns

		// Additional common patterns
		`\s*\([Ff]eat\..*?\)`,
		`\s*\([Ff]t\..*?\)`,
		`\s*\(.*?[Yy]ear.*?\)`,
		`\s*\(.*?\d{4}.*?\)`, // Years

		// Complex combinations with slashes (e.g., "Radio Edit / Remastered 2025")
		`\s*-\s*.*?[Rr]adio [Ee]dit\s*/\s*[Rr]emaster.*?\d{4}.*$`,
		`\s*-\s*.*?[Ee]dit\s*/\s*[Rr]emaster.*?\d{4}.*$`,
		`\s*-\s*.*?[Mm]ix\s*/\s*[Rr]emaster.*?\d{4}.*$`,
		`\s*-\s*.*?[Vv]ersion\s*/\s*[Rr]emaster.*?\d{4}.*$`,

		// Catch any remaining parentheses with mix/edit/version
		`\s*\(.*?(?i:mix|edit|version|remix|remaster).*?\)`,
	}

	cleaned := name
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		cleaned = re.ReplaceAllString(cleaned, "")
	}

	// Clean up extra whitespace
	cleaned = strings.TrimSpace(cleaned)
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")

	// If we accidentally removed everything, return original
	if cleaned == "" {
		return name
	}

	return cleaned
}
