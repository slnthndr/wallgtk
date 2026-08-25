package main

import (
	"sync"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

// Распаковка JPEG/PNG стоит дорого, а одни и те же миниатюры перерисовываются
// при каждом обновлении вкладки. Держим небольшой LRU распакованных текстур.
var (
	texMu    sync.Mutex
	texCache = make(map[string]*gdk.Texture)
	texOrder []string
)

func lookupTexture(path string) (*gdk.Texture, bool) {
	texMu.Lock()
	defer texMu.Unlock()
	tex, ok := texCache[path]
	return tex, ok
}

func storeTexture(path string, tex *gdk.Texture) {
	texMu.Lock()
	defer texMu.Unlock()
	if _, exists := texCache[path]; exists {
		return
	}
	texCache[path] = tex
	texOrder = append(texOrder, path)
	for len(texOrder) > textureCacheLimit {
		oldest := texOrder[0]
		texOrder = texOrder[1:]
		delete(texCache, oldest)
	}
}

func dropTexture(path string) {
	texMu.Lock()
	defer texMu.Unlock()
	if _, ok := texCache[path]; !ok {
		return
	}
	delete(texCache, path)
	for i, p := range texOrder {
		if p == path {
			texOrder = append(texOrder[:i], texOrder[i+1:]...)
			break
		}
	}
}

// loadTextureAsync распаковывает картинку в фоне и отдаёт готовую текстуру
// в главный поток. Уже закэшированная текстура применяется сразу, без IdleAdd,
// чтобы не мигало при повторном показе вкладки.
func loadTextureAsync(path string, apply func(*gdk.Texture)) {
	if path == "" {
		return
	}
	if tex, ok := lookupTexture(path); ok {
		apply(tex)
		return
	}
	go func() {
		tex, err := gdk.NewTextureFromFilename(path)
		if err != nil {
			logf("[TEXTURE] %s: %v", path, err)
			return
		}
		glib.IdleAdd(func() bool {
			storeTexture(path, tex)
			apply(tex)
			return false
		})
	}()
}
