package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// pruneCache держит размер ~/.cache/wallgtk в пределах cacheSizeLimit,
// удаляя самые старые картинки. Служебные файлы (история, язык) не трогаем.
func pruneCache() {
	if cacheDir == "" {
		return
	}
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return
	}

	type cached struct {
		path    string
		size    int64
		modTime time.Time
	}

	var files []cached
	var total int64
	for _, e := range entries {
		if e.IsDir() || !isCacheArtifact(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, cached{
			path:    filepath.Join(cacheDir, e.Name()),
			size:    info.Size(),
			modTime: info.ModTime(),
		})
		total += info.Size()
	}

	// Недописанные загрузки прошлых запусков смысла не имеют.
	for _, f := range files {
		if strings.HasSuffix(f.path, ".tmp") && time.Since(f.modTime) > time.Hour {
			os.Remove(f.path)
			total -= f.size
		}
	}

	if total <= cacheSizeLimit {
		return
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	removed := 0
	for _, f := range files {
		if total <= cacheSizeLimit {
			break
		}
		if os.Remove(f.path) == nil {
			dropTexture(f.path)
			total -= f.size
			removed++
		}
	}
	logf("[CACHE] pruned %d files, %d MiB left", removed, total>>20)
}

func isCacheArtifact(name string) bool {
	lower := strings.ToLower(name)
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp", ".tmp"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}
