package main

import "time"

type Thumbs struct {
	Large    string `json:"large"`
	Original string `json:"original"`
	Small    string `json:"small"`
}

type Wallpaper struct {
	ID         string   `json:"id"`
	Path       string   `json:"path"`
	Resolution string   `json:"resolution"`
	Thumbs     *Thumbs  `json:"thumbs"`
	Source     string   `json:"source,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Colors     []string `json:"colors,omitempty"`
}

type APIResponse struct {
	Data []Wallpaper `json:"data"`
	Meta *struct {
		LastPage int `json:"last_page"`
	} `json:"meta"`
}

type HistoryEntry struct {
	Wallpaper Wallpaper `json:"wallpaper"`
	Monitor   string    `json:"monitor"`
	AppliedAt time.Time `json:"applied_at"`
}
