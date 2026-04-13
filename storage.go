package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func initDirs() {
	home, _ := os.UserHomeDir()
	cacheDir = filepath.Join(home, ".cache", "wallgtk")
	wallpaperHor = filepath.Join(home, "Wallpapers", "backgrounds", "hor")
	wallpaperVert = filepath.Join(home, "Wallpapers", "backgrounds", "vert")

	os.MkdirAll(cacheDir, 0755)
	os.MkdirAll(wallpaperHor, 0755)
	os.MkdirAll(wallpaperVert, 0755)

	SyncFavorites()
}

// SyncFavorites читает физические папки на диске и добавляет их в UI
func SyncFavorites() {
	syncedFavorites := make(map[string]Wallpaper)

	loadDir := func(dir string, isPort bool) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
				id := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
				res := "1920x1080"
				if isPort {
					res = "1080x1920"
				}
				syncedFavorites[id] = Wallpaper{
					ID:         id,
					Path:       filepath.Join(dir, e.Name()), // Локальный путь к файлу
					Resolution: res,
				}
			}
		}
	}

	loadDir(wallpaperHor, false)
	loadDir(wallpaperVert, true)

	favMutex.Lock()
	for id, wp := range pendingFavs {
		syncedFavorites[id] = wp
	}
	favorites = syncedFavorites
	favMutex.Unlock()
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
	existing, exists := favorites[wp.ID]
	if exists {
		delete(favorites, wp.ID)
		delete(pendingFavs, wp.ID)
	}
	favMutex.Unlock()

	if exists {
		// Если было в избранном, удаляем локальный файл на диске
		os.Remove(existing.Path)
		os.Remove(existing.Path + ".tmp")
		SyncFavorites()
		return false
	} else {
		// Добавляем файл на диск в зависимости от пропорций
		dir := wallpaperHor
		if isPortrait(wp.Resolution) {
			dir = wallpaperVert
		}
		path := filepath.Join(dir, wp.ID+".jpg")

		// Оптимистично записываем в мапу для мгновенного изменения UI
		pending := Wallpaper{
			ID:         wp.ID,
			Path:       path,
			Resolution: wp.Resolution,
		}
		favMutex.Lock()
		favorites[wp.ID] = pending
		pendingFavs[wp.ID] = pending
		favMutex.Unlock()

		go func() {
			downloadOK := true
			if strings.HasPrefix(wp.Path, "http") {
				downloadOK = download(wp.Path, path)
			}

			favMutex.Lock()
			current, stillPending := pendingFavs[wp.ID]
			if stillPending && current.Path == path {
				delete(pendingFavs, wp.ID)
			}
			favMutex.Unlock()

			if !downloadOK || !stillPending || current.Path != path {
				os.Remove(path)
				os.Remove(path + ".tmp")
			}
			SyncFavorites()
		}()
		return true
	}
}
