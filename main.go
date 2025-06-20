package main

import (
	"fmt"
	"sort"
	"os"
	"strings"
)

func main() {
	apiKey := os.Getenv("YOUTUBE_API_KEY")
	gistID := os.Getenv("GIST_ID")
	gistToken := os.Getenv("PAT")

	playlists, err := LoadPlaylists("playlists.json")
	if err != nil {
		panic(err)
	}

	gistData, err := GetGist(gistID, gistToken)
	if err != nil {
		panic(err)
	}

	updates := make(map[string]string)

	for _, p := range playlists {
		newList, err := FetchPlaylistItems(apiKey, p.ID)
		if err != nil {
			fmt.Printf("⚠️ Failed fetching %s: %v\n", p.Name, err)
			continue
		}

		fileName := p.Name + ".md"
		oldContent := ""
		if fileMeta, exists := gistData.Files[fileName]; exists {
			oldContent = fileMeta.Content
		}

		existingSongs := map[string]bool{}
		var finalLines []string

		if strings.TrimSpace(oldContent) != "" {
			for _, line := range strings.Split(oldContent, "\n") {
				clean := cleanLine(line)
				if clean != "" && !existingSongs[clean] {
					existingSongs[clean] = true
					finalLines = append(finalLines, "- "+clean)
				}
			}
		}

		for _, song := range newList {
			song = strings.TrimSpace(song)
			if _, exists := existingSongs[song]; !exists {
				existingSongs[song] = true
				finalLines = append(finalLines, "- "+song)
			}
		}

		sort.Strings(finalLines)
		updates[fileName] = "## Playlist: " + p.Name + "\n\n" + strings.Join(finalLines, "\n")
	}

	if err := UpdateGist(gistID, updates, gistToken); err != nil {
		panic(err)
	}

	fmt.Println("✅ Gist updated successfully!")
}
