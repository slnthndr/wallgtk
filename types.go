package main

type Thumbs struct {
	Large    string `json:"large"`
	Original string `json:"original"`
	Small    string `json:"small"`
}

type Wallpaper struct {
	ID         string  `json:"id"`
	Path       string  `json:"path"`
	Resolution string  `json:"resolution"`
	Thumbs     *Thumbs `json:"thumbs"`
}

type APIResponse struct {
	Data []Wallpaper `json:"data"`
	Meta *struct {
		LastPage int `json:"last_page"`
	} `json:"meta"`
}
