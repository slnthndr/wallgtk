package main

import (
	"sync"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

type App struct {
	Win          *gtk.ApplicationWindow
	BrowseFlow   *gtk.FlowBox
	FavFlow      *gtk.FlowBox
	Stack        *gtk.Stack
	
	// ВАЖНО: Сохраняем доступ к скроллу
	BrowseScroll *gtk.ScrolledWindow 

	MonDD       *gtk.DropDown
	RatioDD     *gtk.DropDown
	ResDD       *gtk.DropDown
	SortDD      *gtk.DropDown
	SearchEntry *gtk.SearchEntry

	// Zoom State
	PeekBox       *gtk.Box
	PeekPic       *gtk.Picture
	PeekTitleLbl  *gtk.Label
	PeekTagLbls   []*gtk.Label
	CurrentPeekID string

	// Pagination State
	Page     int
	LastPage int
	Loading  bool
	Query    string
	Sorting  string
	ReqRatio string
	ReqRes   string
	Seen     map[string]bool
	Lock     sync.Mutex
}

func NewApp(app *gtk.Application) *App {
	a := &App{
		Win:      gtk.NewApplicationWindow(app),
		Page:     1,
		LastPage: 999,
		Seen:     make(map[string]bool),
	}
	a.Win.SetTitle("WallGTK")
	a.Win.SetDefaultSize(1200, 800)
	a.BuildUI()
	a.SetupEvents()
	return a
}

func (a *App) BuildUI() {
	header := gtk.NewHeaderBar()

	a.MonDD = gtk.NewDropDownFromStrings(MonitorNames)
	a.MonDD.SetSelected(1)
	
	monBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	monBox.Append(gtk.NewLabel("Монитор:"))
	monBox.Append(a.MonDD)
	header.PackStart(monBox)

	a.SearchEntry = gtk.NewSearchEntry()
	a.SearchEntry.SetPlaceholderText("Поиск...")
	a.SearchEntry.SetHExpand(true)
	header.PackStart(a.SearchEntry)

	a.SortDD = gtk.NewDropDownFromStrings(SortOptions)
	a.SortDD.SetSelected(0)
	header.PackEnd(a.SortDD)

	a.ResDD = gtk.NewDropDownFromStrings(LandscapeResNames)
	header.PackEnd(a.ResDD)

	a.RatioDD = gtk.NewDropDownFromStrings(LandscapeRatiosNames)
	header.PackEnd(a.RatioDD)

	a.Win.SetTitlebar(header)

	a.BrowseFlow = gtk.NewFlowBox()
	a.configureFlowBox(a.BrowseFlow)

	a.BrowseScroll = gtk.NewScrolledWindow() // Сохранено в структуру
	a.BrowseScroll.SetVExpand(true)
	a.BrowseScroll.SetHExpand(true)
	a.BrowseScroll.SetChild(a.BrowseFlow)

	a.FavFlow = gtk.NewFlowBox()
	a.configureFlowBox(a.FavFlow)

	favScroll := gtk.NewScrolledWindow()
	favScroll.SetVExpand(true)
	favScroll.SetHExpand(true)
	favScroll.SetChild(a.FavFlow)

	a.Stack = gtk.NewStack()
	a.Stack.SetTransitionType(gtk.StackTransitionTypeSlideLeftRight)
	a.Stack.AddTitled(a.BrowseScroll, "browse", "Просмотр")
	a.Stack.AddTitled(favScroll, "favs", "Избранное")

	switcher := gtk.NewStackSwitcher()
	switcher.SetStack(a.Stack)
	switcher.SetHAlign(gtk.AlignCenter)
	switcher.SetMarginTop(6)
	switcher.SetMarginBottom(6)

	body := gtk.NewBox(gtk.OrientationVertical, 0)
	body.Append(switcher)
	body.Append(a.Stack)

	mainOverlay := gtk.NewOverlay()
	mainOverlay.SetChild(body)
	mainOverlay.AddOverlay(a.BuildZoomOverlay())

	a.Win.SetChild(mainOverlay)
}

func (a *App) configureFlowBox(flow *gtk.FlowBox) {
	flow.SetVAlign(gtk.AlignStart)
	flow.SetHAlign(gtk.AlignCenter)
	flow.SetSelectionMode(gtk.SelectionNone)
	flow.SetHomogeneous(false)
	flow.SetColumnSpacing(uint(Gap))
	flow.SetRowSpacing(uint(Gap))
	flow.SetMarginTop(Gap)
	flow.SetMarginBottom(Gap)
	flow.SetMinChildrenPerLine(1)
	flow.SetMaxChildrenPerLine(30)
}

func (a *App) SetupEvents() {
	lastMon := a.MonDD.Selected()
	lastRatio := a.RatioDD.Selected()
	lastRes := a.ResDD.Selected()
	lastTab := a.Stack.VisibleChildName()
	startupTicks := 0

	glib.TimeoutAdd(100, func() bool {
		startupTicks++
		if startupTicks%5 == 0 {
			a.Lock.Lock()
			canLoad := !a.Loading && a.Page <= a.LastPage
			a.Lock.Unlock()

			if canLoad && a.Stack.VisibleChildName() == "browse" {
				// === УМНАЯ ПРОВЕРКА РАСТЯГИВАНИЯ ОКНА ===
				vadj := a.BrowseScroll.VAdjustment()
				upper := vadj.Upper()
				pageSize := vadj.PageSize()
				value := vadj.Value()

				// Три сценария:
				// 1. Совсем пусто (FirstChild == nil)
				// 2. Размер контента меньше или равен размеру окна (upper <= pageSize + запас) - актуально при растягивании окна!
				// 3. Мы докрутили скролл почти до конца экрана
				if a.BrowseFlow.FirstChild() == nil || 
				   (upper > 0 && pageSize > 0 && upper <= pageSize+1200) ||
				   (upper > 0 && (value+pageSize) >= (upper-800)) {
					a.LoadMore()
				}
			}
		}

		if a.MonDD.Selected() != lastMon {
			lastMon = a.MonDD.Selected()
			if lastMon == 2 {
				a.RatioDD.SetModel(gtk.NewStringList(PortraitRatiosNames))
				a.ResDD.SetModel(gtk.NewStringList(PortraitResNames))
			} else {
				a.RatioDD.SetModel(gtk.NewStringList(LandscapeRatiosNames))
				a.ResDD.SetModel(gtk.NewStringList(LandscapeResNames))
			}
			a.RatioDD.SetSelected(0)
			a.ResDD.SetSelected(0)
			lastRatio, lastRes = 0, 0
			a.Reload()
			a.RefreshFavs()
		}

		if a.RatioDD.Selected() != lastRatio || a.ResDD.Selected() != lastRes {
			lastRatio = a.RatioDD.Selected()
			lastRes = a.ResDD.Selected()
			a.Reload()
		}

		if a.Stack.VisibleChildName() != lastTab {
			lastTab = a.Stack.VisibleChildName()
			if lastTab == "favs" {
				a.RefreshFavs()
			}
		}
		return true
	})

	a.SearchEntry.ConnectActivate(func() {
		a.Reload()
	})
}

func (a *App) Reload() {
	for child := a.BrowseFlow.FirstChild(); child != nil; child = a.BrowseFlow.FirstChild() {
		a.BrowseFlow.Remove(child)
	}

	a.Lock.Lock()
	a.Page = 1
	a.LastPage = 999
	a.Loading = false
	a.Seen = make(map[string]bool)

	a.Query = a.SearchEntry.Text()
	if idx := a.SortDD.Selected(); int(idx) < len(SortOptions) {
		a.Sorting = SortOptions[idx]
	}

	isPort := a.MonDD.Selected() == 2
	rIdx := a.RatioDD.Selected()
	resIdx := a.ResDD.Selected()

	if isPort {
		a.ReqRatio, a.ReqRes = PortraitRatios[0], PortraitRes[0]
		if int(rIdx) < len(PortraitRatios) { a.ReqRatio = PortraitRatios[rIdx] }
		if int(resIdx) < len(PortraitRes) { a.ReqRes = PortraitRes[resIdx] }
	} else {
		a.ReqRatio, a.ReqRes = LandscapeRatios[0], LandscapeRes[0]
		if int(rIdx) < len(LandscapeRatios) { a.ReqRatio = LandscapeRatios[rIdx] }
		if int(resIdx) < len(LandscapeRes) { a.ReqRes = LandscapeRes[resIdx] }
	}
	a.Lock.Unlock()

	a.LoadMore()
}

func (a *App) LoadMore() {
	a.Lock.Lock()
	if a.Loading || a.Page > a.LastPage {
		a.Lock.Unlock()
		return
	}
	a.Loading = true
	p, q, s, r, res := a.Page, a.Query, a.Sorting, a.ReqRatio, a.ReqRes
	a.Lock.Unlock()

	go func() {
		data, lp := fetchPage(q, s, r, res, p)
		glib.IdleAdd(func() bool {
			a.Lock.Lock()
			a.LastPage = lp
			for _, wp := range data {
				if !a.Seen[wp.ID] {
					a.Seen[wp.ID] = true
					tile := a.CreateTile(wp, a.RefreshFavsListener)
					a.BrowseFlow.Append(tile)
				}
			}
			a.Page++
			a.Loading = false
			a.Lock.Unlock()
			return false
		})
	}()
}

func (a *App) RefreshFavsListener() {
	if a.Stack.VisibleChildName() == "favs" {
		a.RefreshFavs()
	}
}

func (a *App) RefreshFavs() {
	for child := a.FavFlow.FirstChild(); child != nil; child = a.FavFlow.FirstChild() {
		a.FavFlow.Remove(child)
	}

	idx := a.MonDD.Selected()
	showLandscape := (idx == 0 || idx == 1)
	showPortrait := (idx == 0 || idx == 2)

	favMutex.RLock()
	for _, wp := range favorites {
		isPort := isPortrait(wp.Resolution)
		if isPort && !showPortrait { continue }
		if !isPort && !showLandscape { continue }
		tile := a.CreateTile(wp, nil)
		a.FavFlow.Append(tile)
	}
	favMutex.RUnlock()
}

func (a *App) GetSelectedMonitor() string {
	idx := a.MonDD.Selected()
	if int(idx) < len(MonitorNames) { return MonitorNames[idx] }
	return "Все"
}

func (a *App) Show() {
	a.Win.Show()
	a.Reload()
}
