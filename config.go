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
	// НОВЫЕ ПЕРЕМЕННЫЕ ДЛЯ ЛОКАЛИЗАЦИИ
	MonitorKeys = []string{"mon_all", "mon_pri", "mon_sec"}
	SortKeys    = []string{"sort_date", "sort_rel", "sort_rnd", "sort_view", "sort_fav", "sort_top"}
	SortOptions = []string{"date_added", "relevance", "random", "views", "favorites", "toplist"}

	LandscapeRatios     = []string{"landscape", "16x9", "16x10", "21x9"}
	LandscapeRatiosKeys = []string{"ratio_any_l", "16:9", "16:10", "21:9"}
	LandscapeRes        = []string{"2560x1440", "1920x1080", "3840x2160"}
	LandscapeResKeys    = []string{"res_2k", "res_fhd", "res_4k"}

	PortraitRatios     = []string{"portrait", "9x16", "10x16"}
	PortraitRatiosKeys = []string{"ratio_any_p", "9:16", "10:16"}
	PortraitRes        = []string{"1440x2560", "1080x1920", "2160x3840"}
	PortraitResKeys    = []string{"res_2k", "res_fhd", "res_4k"}

	// Важно: Теперь используются внутренние ключи вместо локализованного текста
	MonitorOutputs = map[string]string{
		"mon_pri": "DP-2",
		"mon_sec": "DP-1",
	}

	cacheDir      string
	favFile       string
	wallpaperHor  string
	wallpaperVert string
	favorites     = make(map[string]Wallpaper)
	favMutex      sync.RWMutex
	httpClient    = &http.Client{Timeout: 30 * time.Second}
)
