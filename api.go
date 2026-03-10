package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

func download(url, dest string) bool {
	if _, err := os.Stat(dest); err == nil {
		return true
	}
	resp, err := httpClient.Get(url)
	if err != nil || resp.StatusCode != 200 {
		return false
	}
	defer resp.Body.Close()

	f, err := os.Create(dest + ".tmp")
	if err != nil {
		return false
	}
	io.Copy(f, resp.Body)
	f.Close()
	return os.Rename(dest+".tmp", dest) == nil
}

func fetchPage(query, sorting, ratio, atleast string, page int) ([]Wallpaper, int) {
	url := fmt.Sprintf("%s?categories=100&purity=100&sorting=%s&order=desc&ratios=%s&atleast=%s&page=%d",
		APIBase, sorting, ratio, atleast, page)

	if APIKey != "" && APIKey != "$WALLHAVEN_API" {
		url += "&apikey=" + APIKey
	}
	if query != "" {
		url += "&q=" + query
	}

	fmt.Println("[СЕТЬ] Ищу обои:", url)
	resp, err := httpClient.Get(url)
	if err != nil { return nil, 0 }
	defer resp.Body.Close()

	if resp.StatusCode != 200 { return nil, 0 }

	var r APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil { return nil, 0 }

	lastPage := 1
	if r.Meta != nil {
		lastPage = r.Meta.LastPage
	}
	return r.Data, lastPage
}

func fetchTags(id string) []string {
	url := "https://wallhaven.cc/api/v1/w/" + id
	if APIKey != "" && APIKey != "$WALLHAVEN_API" {
		url += "?apikey=" + APIKey
	}
	resp, err := httpClient.Get(url)
	if err != nil || resp.StatusCode != 200 { return nil }
	defer resp.Body.Close()

	var r struct {
		Data struct {
			Tags []struct {
				Name string `json:"name"`
			} `json:"tags"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil { return nil }

	var tags []string
	for _, t := range r.Data.Tags {
		tags = append(tags, t.Name)
	}
	return tags
}

func setWallpaper(path, monitor string) {
	exec.Command("swww-daemon").Start()
	time.Sleep(100 * time.Millisecond)
	args := []string{"img", path, "--transition-type", "grow", "--transition-fps", "120", "--transition-duration", "1"}
	if out, ok := MonitorOutputs[monitor]; ok {
		args = append(args, "--outputs", out)
	}
	exec.Command("swww", args...).Run()
}
