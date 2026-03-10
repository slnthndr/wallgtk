package main

import (
	"net/http"
	"sync"
	"time"
)

const (
	APIKey          = "xi8JF5JCUKK3CUW0jXmDQ4UPdDHGrDT5" // Ваш ключ (взят из логов)
	APIBase         = "https://wallhaven.cc/api/v1/search"
	TileSpacing     = 8 
	ScrollThreshold = 600

	LandscapeTileW = 380
	LandscapeTileH = 214
	PortraitTileW  = 180
	PortraitTileH  = 320
)

var (
	MonitorNames   = []string{"Все", "Основной монитор", "Второй монитор"}
	SortOptions    = []string{"date_added", "relevance", "random", "views", "favorites", "toplist"}
	
	// Теперь используем правильные значения по умолчанию ("landscape" и "portrait")
	LandscapeRatios      = []string{"landscape", "16x9", "16x10", "21x9"}
	LandscapeRatiosNames = []string{"Широкие (Любые)", "16:9", "16:10", "21:9"}
	LandscapeRes         = []string{"2560x1440", "1920x1080", "3840x2160"}
	LandscapeResNames    = []string{">= 2K (1440p)", ">= Full HD (1080p)", ">= 4K (2160p)"}

	PortraitRatios       = []string{"portrait", "9x16", "10x16"}
	PortraitRatiosNames  = []string{"Вертикальные (Любые)", "9:16", "10:16"}
	PortraitRes          = []string{"1440x2560", "1080x1920", "2160x3840"}
	PortraitResNames     = []string{">= 2K (1440p)", ">= Full HD (1080p)", ">= 4K (2160p)"}

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
