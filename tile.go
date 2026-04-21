package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (a *App) CreateTile(wp Wallpaper, onFav func()) *gtk.Box {
	tile := gtk.NewBox(gtk.OrientationVertical, 0)
	tile.SetHAlign(gtk.AlignCenter)
	tile.SetVAlign(gtk.AlignCenter)

	isPort := isPortrait(wp.Resolution)
	if a.currentMonitorFilter() != "mon_all" {
		isPort = a.selectedMonitorIsPortrait()
	}

	tw, th := LandW, LandH
	if isPort {
		tw, th = PortW, PortH
		tile.AddCSSClass("tile-p")
	} else {
		tile.AddCSSClass("tile-l")
	}

	tile.SetSizeRequest(tw, th)
	tile.SetHExpand(false)
	tile.SetVExpand(false)

	overlay := gtk.NewOverlay()
	overlay.SetOverflow(gtk.OverflowHidden)
	overlay.AddCSSClass("tile-clip")

	strut := gtk.NewDrawingArea()
	strut.SetSizeRequest(tw, th)
	overlay.SetChild(strut)

	picture := gtk.NewPicture()
	picture.SetContentFit(gtk.ContentFitCover)
	picture.SetCanShrink(true)
	picture.SetHAlign(gtk.AlignFill)
	picture.SetVAlign(gtk.AlignFill)

	overlay.AddOverlay(picture)

	heart := gtk.NewButton()
	heart.AddCSSClass("heart-btn")
	heart.SetVAlign(gtk.AlignEnd)
	heart.SetHAlign(gtk.AlignEnd)
	heart.SetMarginEnd(6)
	heart.SetMarginBottom(6)

	favMutex.RLock()
	_, isFav := favorites[wp.ID]
	favMutex.RUnlock()
	if isFav {
		heart.SetLabel("❤️")
	} else {
		heart.SetLabel("🤍")
	}
	overlay.AddOverlay(heart)

	if wp.Resolution != "" && !strings.HasPrefix(wp.Path, "/home") {
		lbl := gtk.NewLabel(wp.Resolution)
		lbl.AddCSSClass("res-label")
		lbl.SetHAlign(gtk.AlignStart)
		lbl.SetVAlign(gtk.AlignEnd)
		lbl.SetMarginBottom(6)
		lbl.SetMarginStart(6)
		overlay.AddOverlay(lbl)
	}

	spinner := gtk.NewSpinner()
	spinner.SetHAlign(gtk.AlignCenter)
	spinner.SetVAlign(gtk.AlignCenter)
	spinner.SetVisible(false)
	overlay.AddOverlay(spinner)

	tile.Append(overlay)

	actualPort := isPortrait(wp.Resolution)
	suffix := "_thumb.jpg"
	if actualPort {
		suffix = "_thumb_p.jpg"
	}
	thumbPath := filepath.Join(cacheDir, wp.ID+suffix)

	loadThumb := func(path string) {
		if _, err := os.Stat(path); err == nil {
			if tex, err := gdk.NewTextureFromFilename(path); err == nil {
				picture.SetPaintable(tex)
			}
		}
	}

	// 1. Пытаемся грузить кэш из сети
	if _, err := os.Stat(thumbPath); err == nil {
		loadThumb(thumbPath)
	} else {
		// 2. Если кэша нет, но wp.Path указывает на локальный файл на диске,
		// то используем сам файл как миниатюру
		if !strings.HasPrefix(wp.Path, "http") {
			loadThumb(wp.Path)
		}
	}

	// Для загрузки из сети:
	url := ""
	if wp.Thumbs != nil {
		url = wp.Thumbs.Small
		if url == "" {
			url = wp.Thumbs.Large
		}
		if url == "" {
			url = wp.Thumbs.Original
		}
	} else if strings.HasPrefix(wp.Path, "http") && len(wp.ID) > 2 {
		url = fmt.Sprintf("https://th.wallhaven.cc/small/%s/%s.jpg", wp.ID[:2], wp.ID)
	}

	if url != "" && picture.Paintable() == nil {
		go func(u, p string) {
			if download(u, p) {
				glib.IdleAdd(func() bool {
					loadThumb(p)
					return false
				})
			}
		}(url, thumbPath)
	}

	wpCopy := wp
	heart.ConnectClicked(func() {
		if toggleFav(&wpCopy) {
			heart.SetLabel("❤️")
		} else {
			heart.SetLabel("🤍")
		}
		if onFav != nil {
			onFav()
		}
	})

	click := gtk.NewGestureClick()
	click.SetButton(1)
	click.ConnectReleased(func(n int, x, y float64) {
		spinner.SetVisible(true)
		spinner.Start()
		picture.AddCSSClass("loading")

		go func() {
			a.HandleWallpaperSelection(wpCopy)

			glib.IdleAdd(func() bool {
				spinner.Stop()
				spinner.SetVisible(false)
				picture.RemoveCSSClass("loading")
				return false
			})
		}()
	})
	overlay.AddController(click)

	rclick := gtk.NewGestureClick()
	rclick.SetButton(3)
	rclick.ConnectPressed(func(n int, x, y float64) { a.ShowZoom(wpCopy, thumbPath) })
	rclick.ConnectReleased(func(n int, x, y float64) { a.HideZoom() })
	overlay.AddController(rclick)

	return tile
}
