package main

import (
	"net/http"
	"sync"
	"time"
)

const (
	APIKey  = "" //get your apiKey on https://wallhaven.cc/settings/account
	APIBase = "https://wallhaven.cc/api/v1/search"

	// Жесткие размеры
	LandW = 384
	LandH = 216
	PortW = 192
	PortH = 288
	Gap   = 4 // Минимальный зазор между плитками

	// Сколько загрузок разрешено выполнять одновременно.
	maxParallelDownloads = 6
	// Верхняя граница размера кэша картинок, байт.
	cacheSizeLimit = 512 << 20
	// Сколько распакованных текстур держим в памяти.
	textureCacheLimit = 192
)

type MonitorEntry struct {
	Key      string
	Output   string
	Portrait bool
}

var (
	ModeKeys      = []string{"mode_single", "mode_pair"}
	PurityKeys    = []string{"purity_sfw", "purity_sketchy", "purity_nsfw", "purity_all"}
	SortKeys      = []string{"sort_date", "sort_rel", "sort_rnd", "sort_view", "sort_fav", "sort_top"}
	PreviewKeys   = []string{"preview_crop", "preview_fit", "preview_fill", "preview_center"}
	SortOptions   = []string{"date_added", "relevance", "random", "views", "favorites", "toplist"}
	PurityOptions = []string{"100", "010", "001", "111"}

	LandscapeRatios     = []string{"landscape", "16x9", "16x10", "21x9"}
	LandscapeRatiosKeys = []string{"ratio_any_l", "16:9", "16:10", "21:9"}
	LandscapeRes        = []string{"2560x1440", "1920x1080", "3840x2160"}
	LandscapeResKeys    = []string{"res_2k", "res_fhd", "res_4k"}

	PortraitRatios     = []string{"portrait", "9x16", "10x16"}
	PortraitRatiosKeys = []string{"ratio_any_p", "9:16", "10:16"}
	PortraitRes        = []string{"1440x2560", "1080x1920", "2160x3840"}
	PortraitResKeys    = []string{"res_2k", "res_fhd", "res_4k"}

	MonitorOutputs = map[string]string{}
	MonitorEntries = []MonitorEntry{{Key: "mon_all"}}

	cacheDir        string
	historyFile     string
	libraryHor      string
	libraryVert     string
	wallpaperHor    string
	wallpaperVert   string
	favorites       = make(map[string]Wallpaper)
	pendingFavs     = make(map[string]Wallpaper)
	historyItems    []HistoryEntry
	historyMu       sync.RWMutex
	favMutex        sync.RWMutex
	backendName     string
	backendOverride string

	httpClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        32,
			MaxIdleConnsPerHost: maxParallelDownloads,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	downloadMu      sync.Mutex
	activeDownloads = make(map[string]*downloadState)
	downloadSem     = make(chan struct{}, maxParallelDownloads)
)

type downloadState struct {
	done chan struct{}
	ok   bool
}
