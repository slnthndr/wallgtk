package main

import (
	"net/http"
	"sync"
	"time"
)

const (
	APIKey          = "" //get your apiKey on https://wallhaven.cc/settings/account
	APIBase         = "https://wallhaven.cc/api/v1/search"
	TileSpacing     = 8
	ScrollThreshold = 600

	// Жесткие размеры
	LandW = 384
	LandH = 216
	PortW = 198 // Увеличенный размер для вертикальных обоев
	PortH = 352
	Gap   = 4   // Минимальный зазор между плитками
)

var (
	MonitorNames = []string{"Все", "Основной монитор", "Второй монитор"}
	SortOptions  = []string{"date_added", "relevance", "random", "views", "favorites", "toplist"}

	LandscapeRatios      = []string{"landscape", "16x9", "16x10", "21x9"}
	LandscapeRatiosNames = []string{"Широкие (Любые)", "16:9", "16:10", "21:9"}
	LandscapeRes         = []string{"2560x1440", "1920x1080", "3840x2160"}
	LandscapeResNames    = []string{">= 2K (1440p)", ">= Full HD (1080p)", ">= 4K (2160p)"}

	PortraitRatios      = []string{"portrait", "9x16", "10x16"}
	PortraitRatiosNames = []string{"Вертикальные (Любые)", "9:16", "10:16"}
	PortraitRes         = []string{"1440x2560", "1080x1920", "2160x3840"}
	PortraitResNames    = []string{">= 2K (1440p)", ">= Full HD (1080p)", ">= 4K (2160p)"}

	MonitorOutputs = map[string]string{
		"Основной монитор": "DP-2",
		"Второй монитор":   "DP-1",
	}

	cacheDir      string
	favFile       string
	wallpaperHor  string
	wallpaperVert string
	favorites     = make(map[string]Wallpaper)
	favMutex      sync.RWMutex
	httpClient    = &http.Client{Timeout: 30 * time.Second}
)
