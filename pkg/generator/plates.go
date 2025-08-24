package generator

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/kirkegaard/go-spotify-bingo/pkg/models"
)

type Generator struct {
	rng *rand.Rand
}

func New() *Generator {
	return &Generator{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (g *Generator) GeneratePlates(playlistData models.PlaylistData, count int, contentType string) ([]models.PlateFields, error) {
	requiredTracks := count * 15 // Each plate has 15 fields (5 per row × 3 rows)
	if len(playlistData.Tracks) < requiredTracks {
		return nil, fmt.Errorf("playlist must have at least %d tracks for %d plates (need %d unique fields)", requiredTracks, count, requiredTracks)
	}

	var plates []models.PlateFields
	usedCombinations := make(map[string]bool)

	for range count {
		plate, err := g.generateSinglePlate(playlistData.Tracks, usedCombinations, contentType)
		if err != nil {
			return nil, err
		}
		plates = append(plates, plate)
	}

	return plates, nil
}

func (g *Generator) generateSinglePlate(tracks []models.Track, usedCombinations map[string]bool, contentType string) (models.PlateFields, error) {
	var plate models.PlateFields
	usedContent := make(map[string]bool)

	for row := 0; row < 3; row++ {
		fieldsInRow := g.getRandomPositionsForRow()

		for _, col := range fieldsInRow {
			content, fieldType := g.getRandomContent(tracks, usedContent, contentType)
			plate.Grid[row][col] = models.BingoField{
				Content: content,
				Type:    fieldType,
				Marked:  false,
			}
			usedContent[content] = true
		}
	}

	return plate, nil
}

func (g *Generator) getRandomPositionsForRow() []int {
	positions := make([]int, 9)
	for i := range positions {
		positions[i] = i
	}

	g.rng.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})

	return positions[:5]
}

func (g *Generator) getRandomContent(tracks []models.Track, used map[string]bool, contentType string) (string, string) {
	maxAttempts := 100
	for attempts := 0; attempts < maxAttempts; attempts++ {
		track := tracks[g.rng.Intn(len(tracks))]

		switch contentType {
		case models.ContentTypeTracks:
			// Only use track names
			if !used[track.Name] && track.Name != "" {
				return track.Name, "track"
			}
		case models.ContentTypeArtists:
			// Only use artist names
			if len(track.Artists) > 0 {
				artist := track.Artists[g.rng.Intn(len(track.Artists))]
				if !used[artist] && artist != "" {
					return artist, "artist"
				}
			}
		case models.ContentTypeCombined:
			// Combine track name and artist(s)
			if len(track.Artists) > 0 && track.Name != "" {
				artistName := track.Artists[0] // Use the first artist
				if len(track.Artists) > 1 {
					// If multiple artists, join with &
					artistName = track.Artists[0]
					for i := 1; i < len(track.Artists) && i < 2; i++ { // Limit to 2 artists max
						artistName += " & " + track.Artists[i]
					}
				}
				combined := track.Name + " - " + artistName
				if !used[combined] {
					return combined, "combined"
				}
			}
		default:
			// Mixed mode (default behavior)
			useTrackName := g.rng.Float32() < 0.5
			if useTrackName {
				if !used[track.Name] && track.Name != "" {
					return track.Name, "track"
				}
			} else {
				if len(track.Artists) > 0 {
					artist := track.Artists[g.rng.Intn(len(track.Artists))]
					if !used[artist] && artist != "" {
						return artist, "artist"
					}
				}
			}
		}
	}

	// Fallback - return track name or artist based on content type
	track := tracks[g.rng.Intn(len(tracks))]
	switch contentType {
	case models.ContentTypeArtists:
		if len(track.Artists) > 0 {
			return track.Artists[0], "artist"
		}
	case models.ContentTypeCombined:
		if len(track.Artists) > 0 && track.Name != "" {
			return track.Name + " - " + track.Artists[0], "combined"
		}
	}
	return track.Name, "track"
}

func GenerateGameCode() string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := ""
	for i := 0; i < 6; i++ {
		code += fmt.Sprintf("%d", rng.Intn(10))
	}
	return code
}
