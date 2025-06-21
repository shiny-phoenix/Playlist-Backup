package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func getPlaylistTitles(apiKey, playlistID string) ([]string, error) {
	var titles []string
	baseURL := "https://www.googleapis.com/youtube/v3/playlistItems"
	pageToken := ""
	for {
		reqURL := fmt.Sprintf("%s?part=snippet&maxResults=50&playlistId=%s&key=%s&pageToken=%s",
			baseURL, playlistID, apiKey, pageToken)
		resp, err := http.Get(reqURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		var result struct {
			NextPageToken string `json:"nextPageToken"`
			Items         []struct {
				Snippet struct {
					Title string `json:"title"`
				} `json:"snippet"`
			} `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}
		for _, item := range result.Items {
			titles = append(titles, item.Snippet.Title)
		}
		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}
	return titles, nil
}

func getGistFiles(gistID, pat string) (map[string]string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/gists/"+gistID, nil)
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var gist struct {
		Files map[string]struct {
			Filename string `json:"filename"`
			Content  string `json:"content"`
		} `json:"files"`
	}
	if err := json.NewDecoder(res.Body).Decode(&gist); err != nil {
		return nil, err
	}
	files := make(map[string]string)
	for name, file := range gist.Files {
		files[name] = file.Content
	}
	return files, nil
}

func diffPlaylist(oldContent string, currentTitles []string, playlistName string) string {
	oldSongs := make(map[string]bool)
	currentSet := make(map[string]bool)

	// Extract old songs from the Gist content
	if oldContent != "" {
		lines := strings.Split(oldContent, "\n")
		for _, line := range lines[1:] { // skip header
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "##") {
				continue
			}

			// Remove index prefix like "1. "
			parts := strings.SplitN(line, ".", 2)
			if len(parts) != 2 {
				continue
			}
			song := parts[1][5:]

			song = cleanSong(song)
			oldSongs[song] = true
		}
	}

	// Build current songs set
	for _, title := range currentTitles {
		currentSet[cleanSong(title)] = true
	}

	var outputLines []string
	outputLines = append(outputLines, "## Playlist: "+playlistName)
	index := 1

	// Step 1: Show removed songs (in old but not in current)
	for oldTitle := range oldSongs {
		if !currentSet[oldTitle] {
			outputLines = append(outputLines, fmt.Sprintf("%d. ❌ %s", index, oldTitle))
			index++
		}
	}

	// Step 2: Show all current songs with appropriate markers
	for _, title := range currentTitles {
		if oldSongs[title] {
			// Song exists in both ✅ marker
			outputLines = append(outputLines, fmt.Sprintf("%d. ✅ %s", index, title))
		} else {
			// New song - add ➕ marker
			outputLines = append(outputLines, fmt.Sprintf("%d. ➕ %s", index, title))
		}
		index++
	}

	return strings.Join(outputLines, "\n") + "\n"
}

func updateGist(gistID, pat string, updatedFiles map[string]string, oldFiles map[string]string) error {
	client := &http.Client{}

	// Step 1: Delete all existing .md files
	for filename := range oldFiles {
		if strings.HasSuffix(filename, ".md") {
			fmt.Println("🗑️ Deleting file:", filename)

			deletePayload := map[string]interface{}{
				"files": map[string]interface{}{
					filename: nil,
				},
			}
			deleteJSON, _ := json.Marshal(deletePayload)

			req, _ := http.NewRequest("PATCH", "https://api.github.com/gists/"+gistID, bytes.NewBuffer(deleteJSON))
			req.Header.Set("Authorization", "Bearer "+pat)
			req.Header.Set("Accept", "application/vnd.github+json")

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("failed to delete %s: %v", filename, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 300 {
				body, _ := ioutil.ReadAll(resp.Body)
				return fmt.Errorf("delete failed for %s: %s", filename, body)
			}
		}
	}

	// Step 2: Create each new file fresh
	for filename, content := range updatedFiles {
		fmt.Println("📝 Creating file:", filename)

		createPayload := map[string]interface{}{
			"files": map[string]interface{}{
				filename: map[string]string{"content": content},
			},
		}
		createJSON, _ := json.Marshal(createPayload)

		req, _ := http.NewRequest("PATCH", "https://api.github.com/gists/"+gistID, bytes.NewBuffer(createJSON))
		req.Header.Set("Authorization", "Bearer "+pat)
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to create %s: %v", filename, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			body, _ := ioutil.ReadAll(resp.Body)
			return fmt.Errorf("create failed for %s: %s", filename, body)
		}
	}

	return nil
}

func cleanSong(title string) string {
	// Remove known invisible characters
	title = strings.TrimSpace(title)
	title = strings.TrimPrefix(title, "\uFEFF")    // Byte Order Mark (BOM)
	title = strings.ReplaceAll(title, "\u200B", "") // Zero-width space
	title = strings.ReplaceAll(title, "\r", "")     // Windows carriage return
	return title
}
