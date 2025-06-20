package main

import (
	"fmt"
	"os"
	"strings"
)


func main() {
	apiKey := os.Getenv("YOUTUBE_API_KEY")
	gistID := os.Getenv("GIST_ID")
	gistToken := os.Getenv("GIST_TOKEN")

	playlists, err := loadPlaylists("playlists.json")
	if err != nil {
		panic(err)
	}

	gistData, err := getGist(gistID, gistToken)
	if err != nil {
		panic(err)
	}

	updates := make(map[string]string)

	for _, p := range playlists {
		newList, err := fetchPlaylistItems(apiKey, p.ID)
		if err != nil {
			fmt.Printf("Failed fetching %s: %v\n", p.Name, err)
			continue
		}

		oldContent := gistData.Files[p.Name+".txt"].Content
		oldList := strings.Split(strings.TrimSpace(oldContent), "\n")
		updates[p.Name+".txt"] = compareSongs(oldList, newList)
	}

	if err := updateGist(gistID, updates, gistToken); err != nil {
		panic(err)
	}
}
