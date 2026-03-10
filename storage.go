package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func initDirs() {
	home, _ := os.UserHomeDir()
	cacheDir = filepath.Join(home, ".cache", "wallgtk")
	favFile = filepath.Join(cacheDir, "favorites.json")
	wallpaperHor = filepath.Join(home, "Wallpapers", "backgrounds", "hor")
	wallpaperVert = filepath.Join(home, "Wallpapers", "backgrounds", "vert")

	os.MkdirAll(cacheDir, 0755)
	os.MkdirAll(wallpaperHor, 0755)
	os.MkdirAll(wallpaperVert, 0755)

	if data, err := os.ReadFile(favFile); err == nil {
		json.Unmarshal(data, &favorites)
	}
}

func saveFavorites() {
	favMutex.RLock()
	data, _ := json.MarshalIndent(favorites, "", "  ")
	favMutex.RUnlock()
	os.WriteFile(favFile, data, 0644)
}

func isPortrait(res string) bool {
	parts := strings.Split(res, "x")
	if len(parts) != 2 {
		return false
	}
	w, _ := strconv.Atoi(parts[0])
	h, _ := strconv.Atoi(parts[1])
	return h > w
}

func toggleFav(wp *Wallpaper) bool {
	favMutex.Lock()
	_, exists := favorites[wp.ID]
	if exists {
		delete(favorites, wp.ID)
	} else {
		favorites[wp.ID] = *wp
	}
	favMutex.Unlock()
	go saveFavorites()

	if !exists {
		go func() {
			dir := wallpaperHor
			if isPortrait(wp.Resolution) {
				dir = wallpaperVert
			}
			path := filepath.Join(dir, wp.ID+".jpg")
			download(wp.Path, path)
		}()
	}
	return !exists
}
