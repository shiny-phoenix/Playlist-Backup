package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type Playlist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}


func main() {

	apiKey := os.Getenv("YOUTUBE_API_KEY")
	if apiKey == "" {
		log.Fatal("YOUTUBE_API_KEY is not set")
	}
	gistID := os.Getenv("GIST_ID")
	if gistID == "" {
		log.Fatal("GIST_ID is not set")
	}
	pat := os.Getenv("PAT")
	if pat == "" {
		log.Fatal("PAT is not set")
	}

	// Load playlists.json
	var playlists []Playlist
	data, err := ioutil.ReadFile("playlists.json")
	if err != nil {
		log.Fatalf("Error reading playlists.json: %v", err)
	}
	if err := json.Unmarshal(data, &playlists); err != nil {
		log.Fatalf("Error parsing playlists.json: %v", err)
	}

	// Get existing gist files
	oldFilesContent, err := getGistFiles(gistID, pat)
	if err != nil {
		log.Fatalf("Error fetching Gist: %v", err)
	}

	updatedFiles := make(map[string]string)
	oldFilenames := make(map[string]string)
	for filename, content := range oldFilesContent {
		oldFilenames[filename] = content
	}

	for _, pl := range playlists {
		fmt.Printf("▶ Processing playlist: %q\n", pl.Name)
		titles, err := getPlaylistTitles(apiKey, pl.ID)
		if err != nil {
			log.Printf("❌ Error fetching playlist %s: %v", pl.Name, err)
			continue
		}
		fmt.Printf("✔ Fetched %d videos from YouTube\n", len(titles))

		filename := pl.Name + ".md"
		oldContent := oldFilesContent[filename] // empty if not exist
		newContent := diffPlaylist(oldContent, titles, pl.Name)
		updatedFiles[filename] = newContent

		fmt.Printf("📄 Prepared %d lines for %s\n\n", strings.Count(newContent, "\n"), filename)
	}

	for key, value := range updatedFiles{
		fmt.Println(key + " : ")
		fmt.Println(value)
	}

	if err := updateGist(gistID, pat, updatedFiles, oldFilenames); err != nil {
		log.Fatalf("Failed to update Gist: %v", err)
	}

	fmt.Println("✅ Gist successfully updated with all playlists.")
}
