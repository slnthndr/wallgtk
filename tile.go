package main

import (
	"path/filepath"
	"os"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (a *App) CreateTile(wp Wallpaper, onFav func()) *gtk.Box {
	tile := gtk.NewBox(gtk.OrientationVertical, 0)
	tile.SetHAlign(gtk.AlignCenter)
	tile.SetVAlign(gtk.AlignCenter)

	monIdx := a.MonDD.Selected()
	isPort := false

	if monIdx == 2 {
		isPort = true 
	} else if monIdx == 1 {
		isPort = false 
	} else {
		isPort = isPortrait(wp.Resolution)
	}

	tw, th := LandW, LandH
	if isPort {
		tw, th = PortW, PortH
		tile.AddCSSClass("tile-p")
	} else {
		tile.AddCSSClass("tile-l")
	}

	// Жёстко фиксируем размер контейнера карточки
	tile.SetSizeRequest(tw, th)
	tile.SetHExpand(false)
	tile.SetVExpand(false)

	overlay := gtk.NewOverlay()
	// Это закруглит края и не даст картинке вылезти за них
	overlay.SetOverflow(gtk.OverflowHidden)
	overlay.AddCSSClass("tile-clip") 

	// === ВОЗВРАЩАЕМ РАСПОРКУ ===
	// Именно она делает сетку красивой и строгой, заставляя элементы быть 198x352
	strut := gtk.NewDrawingArea()
	strut.SetSizeRequest(tw, th)
	overlay.SetChild(strut)

	// === КАРТИНКА ПОВЕРХ РАСПОРКИ ===
	picture := gtk.NewPicture()
	picture.SetContentFit(gtk.ContentFitCover) // Кроп с сохранением пропорций
	picture.SetCanShrink(true)                 // Позволяет огромным текстурам ужиматься
	
	// ВАЖНО: заставляет картинку принять ровно форму распорки 198x352, а не свои размеры
	picture.SetHAlign(gtk.AlignFill)
	picture.SetVAlign(gtk.AlignFill)
	
	overlay.AddOverlay(picture)

	// Лайк
	heart := gtk.NewButton()
	heart.AddCSSClass("heart-btn")
	heart.SetVAlign(gtk.AlignEnd)
	heart.SetHAlign(gtk.AlignEnd)
	heart.SetMarginEnd(6)
	heart.SetMarginBottom(6)

	favMutex.RLock()
	_, isFav := favorites[wp.ID]
	favMutex.RUnlock()
	if isFav { heart.SetLabel("❤️") } else { heart.SetLabel("🤍") }
	overlay.AddOverlay(heart)

	// Разрешение
	if wp.Resolution != "" {
		lbl := gtk.NewLabel(wp.Resolution)
		lbl.AddCSSClass("res-label")
		lbl.SetHAlign(gtk.AlignStart)
		lbl.SetVAlign(gtk.AlignEnd)
		lbl.SetMarginBottom(6)
		lbl.SetMarginStart(6)
		overlay.AddOverlay(lbl)
	}

	// Индикатор загрузки
	spinner := gtk.NewSpinner()
	spinner.SetHAlign(gtk.AlignCenter)
	spinner.SetVAlign(gtk.AlignCenter)
	spinner.SetVisible(false)
	overlay.AddOverlay(spinner)

	tile.Append(overlay)

	actualPort := isPortrait(wp.Resolution)
	suffix := "_thumb.jpg"
	if actualPort { suffix = "_thumb_p.jpg" }
	thumbPath := filepath.Join(cacheDir, wp.ID+suffix)

	loadThumb := func() {
		if _, err := os.Stat(thumbPath); err == nil {
			if tex, err := gdk.NewTextureFromFilename(thumbPath); err == nil {
				picture.SetPaintable(tex)
			}
		}
	}
	loadThumb()

	if wp.Thumbs != nil {
		url := wp.Thumbs.Large
		if url == "" { url = wp.Thumbs.Original }
		if url == "" { url = wp.Thumbs.Small }
		if url != "" && picture.Paintable() == nil {
			go func(u, p string) {
				if download(u, p) {
					glib.IdleAdd(func() bool {
						loadThumb()
						return false
					})
				}
			}(url, thumbPath)
		}
	}

	wpCopy := wp
	heart.ConnectClicked(func() {
		if toggleFav(&wpCopy) {
			heart.SetLabel("❤️")
		} else {
			heart.SetLabel("🤍")
		}
		if onFav != nil { onFav() }
	})

	click := gtk.NewGestureClick()
	click.SetButton(1)
	click.ConnectReleased(func(n int, x, y float64) {
		mon := a.GetSelectedMonitor()
		spinner.SetVisible(true)
		spinner.Start()
		picture.AddCSSClass("loading")
		go func() {
			p := filepath.Join(cacheDir, wpCopy.ID+".jpg")
			if download(wpCopy.Path, p) {
				setWallpaper(p, mon)
			}
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
