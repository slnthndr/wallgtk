package main

import (
	"crypto/sha1"
	"encoding/hex"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	glibv2 "github.com/diamondburned/gotk4/pkg/glib/v2"
)

func initLibraryDirs() {
	home, _ := os.UserHomeDir()
	libraryHor = filepath.Join(home, "Wallpapers", "library", "hor")
	libraryVert = filepath.Join(home, "Wallpapers", "library", "vert")
	os.MkdirAll(libraryHor, 0755)
	os.MkdirAll(libraryVert, 0755)
}

func isSupportedImage(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	default:
		return false
	}
}

func localWallpaperFromPath(path string, source string) (Wallpaper, bool) {
	if !isSupportedImage(path) {
		return Wallpaper{}, false
	}
	cfg, err := readImageConfig(path)
	if err != nil {
		return Wallpaper{}, false
	}
	resolution := "1920x1080"
	if cfg.Width > 0 && cfg.Height > 0 {
		resolution = strconv.Itoa(cfg.Width) + "x" + strconv.Itoa(cfg.Height)
	}
	id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if id == "" {
		sum := sha1.Sum([]byte(path))
		id = hex.EncodeToString(sum[:8])
	}
	wp := Wallpaper{
		ID:         id,
		Path:       path,
		Resolution: resolution,
		Source:     source,
	}
	return wp, true
}

func matchesLocalQuery(wp Wallpaper, query string) bool {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return true
	}
	haystack := strings.ToLower(strings.Join([]string{
		wp.ID,
		wp.Path,
		wp.Resolution,
		wp.Source,
		strings.Join(wp.Colors, " "),
	}, " "))
	return strings.Contains(haystack, query)
}

func readImageConfig(path string) (image.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return image.Config{}, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	return cfg, err
}

func importLocalFile(src string) (Wallpaper, bool) {
	wp, ok := localWallpaperFromPath(src, "library")
	if !ok {
		return Wallpaper{}, false
	}

	destDir := libraryHor
	if isPortrait(wp.Resolution) {
		destDir = libraryVert
	}
	destPath := filepath.Join(destDir, filepath.Base(src))
	if filepath.Clean(src) != filepath.Clean(destPath) {
		if !copyFile(src, destPath) {
			return Wallpaper{}, false
		}
	}
	return localWallpaperFromPath(destPath, "library")
}

func importDroppedURIList(data string) []Wallpaper {
	var imported []Wallpaper
	for _, raw := range glibv2.URIListExtractURIs(data) {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "file" {
			continue
		}
		path, err := url.PathUnescape(u.Path)
		if err != nil {
			continue
		}
		if wp, ok := importLocalFile(path); ok {
			imported = append(imported, wp)
		}
	}
	return imported
}
