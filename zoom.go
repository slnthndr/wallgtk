package main

import (
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
	a.PeekPic.SetContentFit(gtk.ContentFitContain)
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
	logf("[ZOOM] open %s", wp.ID)

	a.PeekPic.SetPaintable(nil)
	a.PeekBox.SetVisible(true)
	a.PeekTitleLbl.SetText(Tr("loading"))
	for _, lbl := range a.PeekTagLbls {
		lbl.SetVisible(false)
	}

	id := wp.ID
	showFull := func(path string) {
		loadTextureAsync(path, func(tex *gdk.Texture) {
			if a.CurrentPeekID == id {
				a.PeekPic.SetPaintable(tex)
			}
		})
	}

	loadTextureAsync(thumbPath, func(tex *gdk.Texture) {
		// Полноразмерная картинка могла успеть приехать раньше миниатюры.
		if a.CurrentPeekID == id && a.PeekPic.Paintable() == nil {
			a.PeekPic.SetPaintable(tex)
		}
	})

	if !stringsHasHTTP(wp.Path) {
		showFull(wp.Path)
	} else {
		go func(path string) {
			p := filepath.Join(cacheDir, id+".jpg")
			if download(path, p) {
				glib.IdleAdd(func() bool {
					showFull(p)
					return false
				})
			}
		}(wp.Path)
	}

	go func(id string) {
		tags := fetchTags(id)
		glib.IdleAdd(func() bool {
			if a.CurrentPeekID == id {
				a.PeekTitleLbl.SetText(Tr("wallpaper_id") + id)
				if len(tags) == 0 {
					a.PeekTagLbls[0].SetText(Tr("no_tags"))
					a.PeekTagLbls[0].SetVisible(true)
				} else {
					for i, t := range tags {
						if i >= 5 {
							break
						}
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
		logf("[ZOOM] close")
	}
	a.PeekBox.SetVisible(false)
	a.CurrentPeekID = ""
}
