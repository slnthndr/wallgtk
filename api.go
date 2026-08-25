package main

import (
	"encoding/json"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const userAgent = "wallgtk (+https://github.com/d754b/wallgtk)"

// download скачивает url в dest ровно один раз: параллельные вызовы для одного
// и того же назначения ждут первую загрузку и получают её результат.
func download(url, dest string) bool {
	if fileExists(dest) {
		return true
	}

	downloadMu.Lock()
	if state, exists := activeDownloads[dest]; exists {
		downloadMu.Unlock()
		<-state.done
		return state.ok
	}
	state := &downloadState{done: make(chan struct{})}
	activeDownloads[dest] = state
	downloadMu.Unlock()

	state.ok = downloadWithRetry(url, dest, 3)

	downloadMu.Lock()
	delete(activeDownloads, dest)
	downloadMu.Unlock()
	close(state.done)
	return state.ok
}

func downloadWithRetry(url, dest string, attempts int) bool {
	if fileExists(dest) {
		return true
	}

	downloadSem <- struct{}{}
	defer func() { <-downloadSem }()

	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 400 * time.Millisecond)
		}
		if fetchToFile(url, dest) {
			return true
		}
	}
	logf("[NETWORK] download failed after %d attempts: %s", attempts, url)
	return false
}

func fetchToFile(url, dest string) bool {
	resp, err := httpGet(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}

	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return false
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(tmp)
		return false
	}
	if os.Rename(tmp, dest) != nil {
		os.Remove(tmp)
		return false
	}
	return true
}

func httpGet(rawurl string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, rawurl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	return httpClient.Do(req)
}

func fetchPage(query, sorting, ratio, atleast, purity string, page int) ([]Wallpaper, int) {
	params := neturl.Values{}
	params.Set("categories", "100")
	params.Set("purity", purity)
	params.Set("sorting", sorting)
	params.Set("order", "desc")
	params.Set("ratios", ratio)
	params.Set("atleast", atleast)
	params.Set("page", strconv.Itoa(page))
	if query != "" {
		params.Set("q", query)
	}
	if apiKey := getWallhavenAPIKey(); apiKey != "" {
		params.Set("apikey", apiKey)
	}

	// Ключ в лог не попадает намеренно.
	logf("[NETWORK] search page=%d q=%q sort=%s ratio=%s atleast=%s purity=%s",
		page, query, sorting, ratio, atleast, purity)

	resp, err := httpGet(APIBase + "?" + params.Encode())
	if err != nil {
		return nil, 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logf("[NETWORK] search returned %s", resp.Status)
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
	rawurl := "https://wallhaven.cc/api/v1/w/" + neturl.PathEscape(id)
	if apiKey := getWallhavenAPIKey(); apiKey != "" {
		rawurl += "?apikey=" + neturl.QueryEscape(apiKey)
	}
	resp, err := httpGet(rawurl)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

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

// purityNeedsAPIKey сообщает, требует ли выбранный фильтр NSFW-доступа,
// который Wallhaven отдаёт только по API-ключу.
func purityNeedsAPIKey(purity string) bool {
	return len(purity) >= 3 && purity[2] == '1'
}
