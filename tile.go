package main

import (
	"fmt"
	"path/filepath"

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
		isPort = monitorPrefersPortrait(a.currentMonitorFilter())
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
	picture.SetSizeRequest(tw, th)
	picture.SetContentFit(a.previewContentFit(isPort))
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

	if wp.Resolution != "" && stringsHasHTTP(wp.Path) {
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

	// Распаковка идёт в фоне, главный поток только ставит готовую текстуру.
	loadThumb := func(path string) {
		if !fileExists(path) {
			return
		}
		loadTextureAsync(path, func(tex *gdk.Texture) {
			picture.SetPaintable(tex)
		})
	}

	// 1. Пытаемся грузить кэш из сети
	if fileExists(thumbPath) {
		loadThumb(thumbPath)
	} else if !stringsHasHTTP(wp.Path) {
		// 2. Если кэша нет, но wp.Path указывает на локальный файл на диске,
		// то используем сам файл как миниатюру
		loadThumb(wp.Path)
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
	} else if stringsHasHTTP(wp.Path) && len(wp.ID) > 2 {
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

func (a *App) previewContentFit(isPortraitTile bool) gtk.ContentFit {
	if !isPortraitTile || a.PreviewDD == nil {
		return gtk.ContentFitCover
	}

	switch a.PreviewDD.Selected() {
	case 1:
		return gtk.ContentFitContain
	case 2:
		return gtk.ContentFitFill
	case 3:
		return gtk.ContentFitScaleDown
	default:
		return gtk.ContentFitCover
	}
}
