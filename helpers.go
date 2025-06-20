package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type PlaylistConfig struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type GistResponse struct {
	Files map[string]struct {
		Content string `json:"content"`
	} `json:"files"`
}

func LoadPlaylists(path string) ([]PlaylistConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg []PlaylistConfig
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

type VideoItem struct {
	Title string `json:"title"`
}

func FetchPlaylistItems(apiKey, playlistId string) ([]string, error) {
	var titles []string
	baseURL := "https://www.googleapis.com/youtube/v3/playlistItems"
	pageToken := ""

	for {
		url := fmt.Sprintf(
			"%s?part=snippet&maxResults=50&playlistId=%s&key=%s&pageToken=%s",
			baseURL, playlistId, apiKey, pageToken,
		)

		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		var result struct {
			NextPageToken string `json:"nextPageToken"`
			Items         []struct {
				Snippet struct {
					Title string `json:"title"`
				} `json:"snippet"`
			} `json:"items"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
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

func GetGist(gistID, token string) (GistResponse, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/gists/"+gistID, nil)
	req.Header.Set("Authorization", "token "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return GistResponse{}, err
	}
	defer resp.Body.Close()

	var gist GistResponse
	err = json.NewDecoder(resp.Body).Decode(&gist)
	return gist, err
}

func UpdateGist(gistID string, updates map[string]string, token string) error {
	files := map[string]map[string]string{}
	for k, v := range updates {
		files[k] = map[string]string{"content": v}
	}
	body := map[string]interface{}{"files": files}
	jsonData, _ := json.Marshal(body)

	req, _ := http.NewRequest("PATCH", "https://api.github.com/gists/"+gistID, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Gist update failed: %s", string(b))
	}
	return nil
}

func CompareSongs(oldList, newList []string) string {
	oldMap := make(map[string]bool)
	newMap := make(map[string]bool)
	var output strings.Builder

	for _, s := range oldList {
		oldMap[s] = true
	}
	for _, s := range newList {
		newMap[s] = true
	}

	for _, s := range oldList {
		if !newMap[s] {
			output.WriteString(fmt.Sprintf("- ❌ ~~%s~~\n", s))
		} else {
			output.WriteString(fmt.Sprintf("- %s\n", s))
		}
	}
	for _, s := range newList {
		if !oldMap[s] {
			output.WriteString(fmt.Sprintf("- ➕ **%s**\n", s))
		}
	}
	return output.String()
}

