package main

import (
	"fmt"
	"path/filepath"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func (a *App) BuildZoomOverlay() *gtk.Box {
	a.PeekBox = gtk.NewBox(gtk.OrientationVertical, 0)
	a.PeekBox.AddCSSClass("zoom-bg")
	a.PeekBox.SetVisible(false)

	a.PeekPic = gtk.NewPicture()
	a.PeekPic.SetContentFit(gtk.ContentFitContain) // Тут важно сохранить Contain для превью!
	a.PeekPic.SetVExpand(true)
	a.PeekPic.SetHExpand(true)

	tagsBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	tagsBox.SetHAlign(gtk.AlignCenter)
	tagsBox.SetVAlign(gtk.AlignEnd)
	tagsBox.SetMarginBottom(30)
	tagsBox.AddCSSClass("tags-overlay")
	tagsBox.SetSizeRequest(600, 40)

	a.PeekTitleLbl = gtk.NewLabel("")
	a.PeekTitleLbl.AddCSSClass("tag-title")
	tagsBox.Append(a.PeekTitleLbl)

	for i := 0; i < 5; i++ {
		lbl := gtk.NewLabel("")
		lbl.AddCSSClass("tag-lbl")
		lbl.SetVisible(false)
		tagsBox.Append(lbl)
		a.PeekTagLbls = append(a.PeekTagLbls, lbl)
	}

	overlay := gtk.NewOverlay()
	overlay.SetChild(a.PeekPic)
	overlay.AddOverlay(tagsBox)
	a.PeekBox.Append(overlay)

	// Блокируем перехват кликов внутри виджетов зума
	a.PeekBox.SetCanTarget(false)
	overlay.SetCanTarget(false)
	a.PeekPic.SetCanTarget(false)
	tagsBox.SetCanTarget(false)
	a.PeekTitleLbl.SetCanTarget(false)
	for _, l := range a.PeekTagLbls {
		l.SetCanTarget(false)
	}

	return a.PeekBox
}

func (a *App) ShowZoom(wp Wallpaper, thumbPath string) {
	a.CurrentPeekID = wp.ID
	fmt.Printf("[\033[35mЗУМ\033[0m] Открываем ID: %s\n", wp.ID)
	
	a.PeekBox.SetVisible(true)
	a.PeekTitleLbl.SetText("Загрузка...")
	for _, lbl := range a.PeekTagLbls {
		lbl.SetVisible(false)
	}

	// Загружаем мыльную миниатюру сразу
	if tex, err := gdk.NewTextureFromFilename(thumbPath); err == nil {
		a.PeekPic.SetPaintable(tex)
	}

	// Асинхронно тянем большую
	go func(id, path string) {
		p := filepath.Join(cacheDir, id+".jpg")
		if download(path, p) {
			glib.IdleAdd(func() bool {
				if a.CurrentPeekID == id {
					if tex, err := gdk.NewTextureFromFilename(p); err == nil {
						a.PeekPic.SetPaintable(tex) // Больше не сбивает фокус!
					}
				}
				return false
			})
		}
	}(wp.ID, wp.Path)

	// Тянем теги
	go func(id string) {
		tags := fetchTags(id)
		glib.IdleAdd(func() bool {
			if a.CurrentPeekID == id {
				a.PeekTitleLbl.SetText("Обои #" + id)
				if len(tags) == 0 {
					a.PeekTagLbls[0].SetText("Нет тегов")
					a.PeekTagLbls[0].SetVisible(true)
				} else {
					for i, t := range tags {
						if i >= 5 { break }
						a.PeekTagLbls[i].SetText("#" + t)
						a.PeekTagLbls[i].SetVisible(true)
					}
				}
			}
			return false
		})
	}(wp.ID)
}

func (a *App) HideZoom() {
	if a.CurrentPeekID != "" {
		fmt.Printf("[\033[35mЗУМ\033[0m] Закрыт\n")
	}
	a.PeekBox.SetVisible(false)
	a.CurrentPeekID = ""
}
