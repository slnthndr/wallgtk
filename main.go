package main

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func main() {
	initDirs()

	// ВАЖНО: Отключаем Vulkan, чтобы NVIDIA-драйвер перестал спамить в консоль
	os.Setenv("GSK_RENDERER", "gl")

	app := gtk.NewApplication("org.wallgtk.app", 0)
	app.ConnectActivate(func() {
		activate(app)
	})
	app.Run(os.Args)
}

func activate(app *gtk.Application) {
	win := gtk.NewApplicationWindow(app)
	win.SetTitle("WallGTK")
	win.SetDefaultSize(1200, 800)

	var (
		page      = 1
		lastPage  = 999
		loading   = false
		query     = ""
		sorting   = "date_added"
		reqRatio  = ""
		reqRes    = ""
		seen      = make(map[string]bool)
		stateLock sync.Mutex
	)

	header := gtk.NewHeaderBar()

	monDD := gtk.NewDropDownFromStrings(MonitorNames)
	monDD.SetSelected(1) // 1 = Основной монитор

	monBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	monBox.Append(gtk.NewLabel("Монитор:"))
	monBox.Append(monDD)
	header.PackStart(monBox)

	searchEntry := gtk.NewSearchEntry()
	searchEntry.SetPlaceholderText("Поиск...")
	searchEntry.SetHExpand(true)
	header.PackStart(searchEntry)

	sortDD := gtk.NewDropDownFromStrings(SortOptions)
	sortDD.SetSelected(0)
	header.PackEnd(sortDD)

	resDD := gtk.NewDropDownFromStrings(LandscapeResNames)
	header.PackEnd(resDD)

	ratioDD := gtk.NewDropDownFromStrings(LandscapeRatiosNames)
	header.PackEnd(ratioDD)

	win.SetTitlebar(header)

	browseFlow := gtk.NewFlowBox()
	browseFlow.SetVAlign(gtk.AlignStart)
	browseFlow.SetHAlign(gtk.AlignCenter)
	browseFlow.SetSelectionMode(gtk.SelectionNone)
	browseFlow.SetHomogeneous(false)
	browseFlow.SetColumnSpacing(uint(TileSpacing))
	browseFlow.SetRowSpacing(uint(TileSpacing))
	browseFlow.SetMarginTop(TileSpacing)
	browseFlow.SetMarginBottom(TileSpacing)
	browseFlow.SetMinChildrenPerLine(1)
	browseFlow.SetMaxChildrenPerLine(30)

	browseScroll := gtk.NewScrolledWindow()
	browseScroll.SetVExpand(true)
	browseScroll.SetHExpand(true)
	browseScroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	browseScroll.SetChild(browseFlow)

	favFlow := gtk.NewFlowBox()
	favFlow.SetVAlign(gtk.AlignStart)
	favFlow.SetHAlign(gtk.AlignCenter)
	favFlow.SetSelectionMode(gtk.SelectionNone)
	favFlow.SetHomogeneous(false)
	favFlow.SetColumnSpacing(uint(TileSpacing))
	favFlow.SetRowSpacing(uint(TileSpacing))
	favFlow.SetMarginTop(TileSpacing)
	favFlow.SetMarginBottom(TileSpacing)
	favFlow.SetMinChildrenPerLine(1)
	favFlow.SetMaxChildrenPerLine(30)

	favScroll := gtk.NewScrolledWindow()
	favScroll.SetVExpand(true)
	favScroll.SetHExpand(true)
	favScroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	favScroll.SetChild(favFlow)

	stack := gtk.NewStack()
	stack.SetTransitionType(gtk.StackTransitionTypeSlideLeftRight)
	stack.AddTitled(browseScroll, "browse", "Просмотр")
	stack.AddTitled(favScroll, "favs", "Избранное")

	switcher := gtk.NewStackSwitcher()
	switcher.SetStack(stack)
	switcher.SetHAlign(gtk.AlignCenter)
	switcher.SetMarginTop(6)
	switcher.SetMarginBottom(6)

	body := gtk.NewBox(gtk.OrientationVertical, 0)
	body.Append(switcher)
	body.Append(stack)

	// === ОВЕРЛЕЙ ДЛЯ ЗУМА ===
	mainOverlay := gtk.NewOverlay()
	mainOverlay.SetChild(body)

	peekBox := gtk.NewBox(gtk.OrientationVertical, 0)
	peekBox.AddCSSClass("zoom-bg")
	peekBox.SetVisible(false)

	peekPic := gtk.NewPicture()
	peekPic.SetContentFit(gtk.ContentFitContain)
	peekPic.SetVExpand(true)
	peekPic.SetHExpand(true)

	peekTagsBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	peekTagsBox.SetHAlign(gtk.AlignCenter)
	peekTagsBox.SetVAlign(gtk.AlignEnd)
	peekTagsBox.SetMarginBottom(30)
	peekTagsBox.AddCSSClass("tags-overlay")

	innerPeekOverlay := gtk.NewOverlay()
	innerPeekOverlay.SetChild(peekPic)
	innerPeekOverlay.AddOverlay(peekTagsBox)
	peekBox.Append(innerPeekOverlay)

	// Делаем ВЕСЬ оверлей зума голограммой, сквозь которую проходят клики мыши.
	// Это 100% гарантирует, что жест удержания не прервется системой.
	peekBox.SetCanTarget(false)
	innerPeekOverlay.SetCanTarget(false)
	peekPic.SetCanTarget(false)
	peekTagsBox.SetCanTarget(false)

	mainOverlay.AddOverlay(peekBox)
	win.SetChild(mainOverlay)

	getMon := func() string {
		idx := monDD.Selected()
		if int(idx) < len(MonitorNames) {
			return MonitorNames[idx]
		}
		return "Все"
	}

	makeTile := func(wp Wallpaper, onFav func()) *gtk.Box {
		tile := gtk.NewBox(gtk.OrientationVertical, 0)
		tile.SetHAlign(gtk.AlignCenter)
		tile.SetVAlign(gtk.AlignCenter)

		monIdx := monDD.Selected()
		isPort := false

		// СТРОГАЯ ЛОГИКА ОТОБРАЖЕНИЯ ФОРМАТА:
		if monIdx == 2 {
			isPort = true // Второй монитор = всегда показываем как вертикальные
		} else if monIdx == 1 {
			isPort = false // Основной = всегда показываем как горизонтальные
		} else {
			isPort = isPortrait(wp.Resolution) // Все = зависит от оригинального разрешения
		}

		tw, th := LandscapeTileW, LandscapeTileH
		if isPort {
			tw, th = PortraitTileW, PortraitTileH
			tile.AddCSSClass("tile-p")
		} else {
			tile.AddCSSClass("tile-l")
		}

		tile.SetSizeRequest(tw, th)
		tile.SetHExpand(false)
		tile.SetVExpand(false)

		overlay := gtk.NewOverlay()

		picture := gtk.NewPicture()
		picture.SetContentFit(gtk.ContentFitCover)
		picture.SetSizeRequest(tw, th)
		picture.SetHExpand(false)
		picture.SetVExpand(false)

		overlay.SetChild(picture)

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

		if wp.Resolution != "" {
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

		// Выбираем какую миниатюру грузить (горизонтальную или вертикальную обрезку)
		actualPort := isPortrait(wp.Resolution)
		suffix := "_thumb.jpg"
		if actualPort {
			suffix = "_thumb_p.jpg"
		}
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
			if url == "" {
				url = wp.Thumbs.Original
			}
			if url == "" {
				url = wp.Thumbs.Small
			}
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
			if onFav != nil {
				onFav()
			}
		})

		// Левый клик (Установить обои)
		click := gtk.NewGestureClick()
		click.SetButton(1)
		click.ConnectReleased(func(n int, x, y float64) {
			mon := getMon()
			spinner.SetVisible(true)
			spinner.Start()
			tile.AddCSSClass("loading")
			go func() {
				p := filepath.Join(cacheDir, wpCopy.ID+".jpg")
				if download(wpCopy.Path, p) {
					setWallpaper(p, mon)
				}
				glib.IdleAdd(func() bool {
					spinner.Stop()
					spinner.SetVisible(false)
					tile.RemoveCSSClass("loading")
					return false
				})
			}()
		})
		overlay.AddController(click)

		// Удержание правой кнопки (ЗУМ И ТЕГИ)
		rclick := gtk.NewGestureClick()
		rclick.SetButton(3)

		var currentPeekID string

		rclick.ConnectPressed(func(n int, x, y float64) {
			currentPeekID = wpCopy.ID
			peekBox.SetVisible(true)

			for child := peekTagsBox.FirstChild(); child != nil; child = peekTagsBox.FirstChild() {
				peekTagsBox.Remove(child)
			}
			loadingLbl := gtk.NewLabel("Загрузка...")
			loadingLbl.AddCSSClass("tag-lbl")
			peekTagsBox.Append(loadingLbl)

			if tex, err := gdk.NewTextureFromFilename(thumbPath); err == nil {
				peekPic.SetPaintable(tex)
			}

			go func(id string, path string) {
				p := filepath.Join(cacheDir, id+".jpg")
				if download(path, p) {
					glib.IdleAdd(func() bool {
						if currentPeekID == id {
							if tex, err := gdk.NewTextureFromFilename(p); err == nil {
								peekPic.SetPaintable(tex)
							}
						}
						return false
					})
				}
			}(wpCopy.ID, wpCopy.Path)

			go func(id string) {
				tags := fetchTags(id)
				glib.IdleAdd(func() bool {
					if currentPeekID == id {
						for child := peekTagsBox.FirstChild(); child != nil; child = peekTagsBox.FirstChild() {
							peekTagsBox.Remove(child)
						}

						titleLbl := gtk.NewLabel("Обои #" + id)
						titleLbl.AddCSSClass("tag-title")
						peekTagsBox.Append(titleLbl)

						if len(tags) == 0 {
							noTags := gtk.NewLabel("Нет тегов")
							noTags.AddCSSClass("tag-lbl")
							peekTagsBox.Append(noTags)
						} else {
							for i, t := range tags {
								if i >= 5 {
									break
								}
								lbl := gtk.NewLabel("#" + t)
								lbl.AddCSSClass("tag-lbl")
								peekTagsBox.Append(lbl)
							}
						}
					}
					return false
				})
			}(wpCopy.ID)
		})

		hidePeek := func() {
			peekBox.SetVisible(false)
			currentPeekID = ""
		}

		rclick.ConnectReleased(func(n int, x, y float64) { hidePeek() })
		rclick.ConnectStopped(hidePeek)

		overlay.AddController(rclick)

		return tile
	}

	refreshFavs := func() {
		for child := favFlow.FirstChild(); child != nil; child = favFlow.FirstChild() {
			favFlow.Remove(child)
		}

		idx := monDD.Selected()
		showLandscape := (idx == 0 || idx == 1)
		showPortrait := (idx == 0 || idx == 2)

		favMutex.RLock()
		for _, wp := range favorites {
			isPort := isPortrait(wp.Resolution)
			if isPort && !showPortrait {
				continue
			}
			if !isPort && !showLandscape {
				continue
			}
			tile := makeTile(wp, nil)
			favFlow.Append(tile)
		}
		favMutex.RUnlock()
	}

	onFavChanged := func() {
		if stack.VisibleChildName() == "favs" {
			refreshFavs()
		}
	}

	var loadMore func()
	loadMore = func() {
		stateLock.Lock()
		if loading || page > lastPage {
			stateLock.Unlock()
			return
		}
		loading = true
		p := page
		q := query
		s := sorting
		r := reqRatio
		res := reqRes
		stateLock.Unlock()

		go func() {
			data, lp := fetchPage(q, s, r, res, p)

			glib.IdleAdd(func() bool {
				stateLock.Lock()
				lastPage = lp
				for _, wp := range data {
					if !seen[wp.ID] {
						seen[wp.ID] = true
						tile := makeTile(wp, onFavChanged)
						browseFlow.Append(tile)
					}
				}
				page++
				loading = false
				stateLock.Unlock()
				return false
			})
		}()
	}

	reload := func() {
		for child := browseFlow.FirstChild(); child != nil; child = browseFlow.FirstChild() {
			browseFlow.Remove(child)
		}
		stateLock.Lock()
		page = 1
		lastPage = 999
		loading = false
		seen = make(map[string]bool)

		query = searchEntry.Text()

		if idx := sortDD.Selected(); int(idx) < len(SortOptions) {
			sorting = SortOptions[idx]
		}

		isPort := monDD.Selected() == 2
		rIdx := ratioDD.Selected()
		resIdx := resDD.Selected()

		if isPort {
			reqRatio = PortraitRatios[0]
			reqRes = PortraitRes[0]
			if int(rIdx) < len(PortraitRatios) { reqRatio = PortraitRatios[rIdx] }
			if int(resIdx) < len(PortraitRes) { reqRes = PortraitRes[resIdx] }
		} else {
			reqRatio = LandscapeRatios[0]
			reqRes = LandscapeRes[0]
			if int(rIdx) < len(LandscapeRatios) { reqRatio = LandscapeRatios[rIdx] }
			if int(resIdx) < len(LandscapeRes) { reqRes = LandscapeRes[resIdx] }
		}

		stateLock.Unlock()
		loadMore()
	}

	reload()

	vadj := browseScroll.VAdjustment()
	vadj.ConnectValueChanged(func() {
		v := vadj.Value()
		ps := vadj.PageSize()
		u := vadj.Upper()
		if u > 0 && ps > 0 && (v+ps) >= (u-ScrollThreshold) {
			loadMore()
		}
	})

	lastMon := monDD.Selected()
	lastRatio := ratioDD.Selected()
	lastRes := resDD.Selected()
	lastTab := stack.VisibleChildName()
	startupTicks := 0

	glib.TimeoutAdd(100, func() bool {
		startupTicks++

		if startupTicks%5 == 0 {
			va := browseScroll.VAdjustment()
			upper := va.Upper()
			pageSize := va.PageSize()

			stateLock.Lock()
			canLoad := !loading && page <= lastPage
			stateLock.Unlock()

			if canLoad && upper > 0 && pageSize > 0 && upper <= pageSize {
				loadMore()
			}
		}

		if monDD.Selected() != lastMon {
			lastMon = monDD.Selected()

			if lastMon == 2 {
				ratioDD.SetModel(gtk.NewStringList(PortraitRatiosNames))
				resDD.SetModel(gtk.NewStringList(PortraitResNames))
			} else {
				ratioDD.SetModel(gtk.NewStringList(LandscapeRatiosNames))
				resDD.SetModel(gtk.NewStringList(LandscapeResNames))
			}

			ratioDD.SetSelected(0)
			resDD.SetSelected(0)
			lastRatio = 0
			lastRes = 0

			reload()
			refreshFavs()
		}

		if ratioDD.Selected() != lastRatio || resDD.Selected() != lastRes {
			lastRatio = ratioDD.Selected()
			lastRes = resDD.Selected()
			reload()
		}

		if stack.VisibleChildName() != lastTab {
			lastTab = stack.VisibleChildName()
			if lastTab == "favs" {
				refreshFavs()
			}
		}

		return true
	})

	searchEntry.ConnectActivate(func() {
		reload()
	})

	css := `
		.tile-l {
			min-width: 380px;
			min-height: 214px;
		}
		.tile-p {
			min-width: 180px;
			min-height: 320px;
		}
		.heart-btn {
			background: rgba(20,20,30,0.65);
			color: #ff4d4d;
			border-radius: 50%;
			min-width: 34px;
			min-height: 34px;
			font-size: 18px;
			padding: 0;
			border: none;
		}
		.heart-btn:hover {
			background: rgba(255,80,80,0.9);
			color: white;
		}
		.res-label {
			background: rgba(0,0,0,0.6);
			color: white;
			border-radius: 8px;
			padding: 2px 8px;
			font-size: 11px;
		}
		.loading { opacity: 0.45; }
		flowboxchild { padding: 0; margin: 0; }
		picture { border-radius: 12px; }

		.zoom-bg { background-color: rgba(10, 10, 15, 0.95); }
		.tags-overlay {
			background: rgba(0,0,0,0.7);
			border-radius: 12px;
			padding: 8px 14px;
		}
		.tag-title {
			color: #ff6b6b;
			font-weight: bold;
			font-size: 14px;
			margin-right: 12px;
		}
		.tag-lbl {
			background: rgba(50,50,70,0.8);
			color: white;
			border-radius: 8px;
			padding: 4px 10px;
			font-size: 13px;
		}
	`
	provider := gtk.NewCSSProvider()
	provider.LoadFromString(css)
	gtk.StyleContextAddProviderForDisplay(
		gdk.DisplayGetDefault(),
		provider,
		gtk.STYLE_PROVIDER_PRIORITY_APPLICATION,
	)

	win.Show()
}
