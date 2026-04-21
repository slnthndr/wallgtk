package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func download(url, dest string) bool {
	downloadMu.Lock()
	if state, exists := activeDownloads[dest]; exists {
		downloadMu.Unlock()
		return <-state.done
	}
	state := &downloadState{done: make(chan bool, 1)}
	activeDownloads[dest] = state
	downloadMu.Unlock()

	ok := downloadWithRetry(url, dest, 3)

	downloadMu.Lock()
	delete(activeDownloads, dest)
	downloadMu.Unlock()
	state.done <- ok
	close(state.done)
	return ok
}

func downloadWithRetry(url, dest string, attempts int) bool {
	if _, err := os.Stat(dest); err == nil {
		return true
	}
	for attempt := 0; attempt < attempts; attempt++ {
		resp, err := httpClient.Get(url)
		if err != nil || resp.StatusCode != 200 {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			continue
		}

		f, err := os.Create(dest + ".tmp")
		if err != nil {
			resp.Body.Close()
			return false
		}
		_, copyErr := io.Copy(f, resp.Body)
		resp.Body.Close()
		closeErr := f.Close()
		if copyErr == nil && closeErr == nil && os.Rename(dest+".tmp", dest) == nil {
			return true
		}
		os.Remove(dest + ".tmp")
	}
	return false
}

func fetchPage(query, sorting, ratio, atleast, purity string, page int) ([]Wallpaper, int) {
	url := fmt.Sprintf("%s?categories=100&purity=%s&sorting=%s&order=desc&ratios=%s&atleast=%s&page=%d",
		APIBase, purity, sorting, ratio, atleast, page)

	if apiKey := getWallhavenAPIKey(); apiKey != "" {
		url += "&apikey=" + apiKey
	}
	if query != "" {
		url += "&q=" + query
	}

	fmt.Println("[NETWORK] Searching:", url)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, 0
	}

	var r APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, 0
	}

	lastPage := 1
	if r.Meta != nil {
		lastPage = r.Meta.LastPage
	}
	for i := range r.Data {
		r.Data[i].Source = "wallhaven"
	}
	return r.Data, lastPage
}

func fetchTags(id string) []string {
	url := "https://wallhaven.cc/api/v1/w/" + id
	if apiKey := getWallhavenAPIKey(); apiKey != "" {
		url += "?apikey=" + apiKey
	}
	resp, err := httpClient.Get(url)
	if err != nil || resp.StatusCode != 200 {
		return nil
	}
	defer resp.Body.Close()

	var r struct {
		Data struct {
			Tags []struct {
				Name string `json:"name"`
			} `json:"tags"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil
	}

	var tags []string
	for _, t := range r.Data.Tags {
		tags = append(tags, t.Name)
	}
	return tags
}

func getWallhavenAPIKey() string {
	envKey := strings.TrimSpace(os.Getenv("WALLHAVEN_API"))
	if envKey != "" {
		return envKey
	}
	if APIKey == "" || APIKey == "$WALLHAVEN_API" {
		return ""
	}
	return APIKey
}

func purityNeedsAPIKey(purity string) bool {
	return strings.Contains(purity, "1") && len(purity) >= 3 && purity[2] == '1'
}
