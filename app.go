package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	glibv2 "github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type App struct {
	Win           *gtk.ApplicationWindow
	Header        *gtk.HeaderBar
	BrowseFlow    *gtk.FlowBox
	FavFlow       *gtk.FlowBox
	HistoryFlow   *gtk.FlowBox
	Stack         *gtk.Stack
	BrowseScroll  *gtk.ScrolledWindow
	FavScroll     *gtk.ScrolledWindow
	HistoryScroll *gtk.ScrolledWindow

	MonBtn       *gtk.MenuButton
	MonBtnLabel  *gtk.Label
	MonMenuBox   *gtk.Box
	ModeDD       *gtk.DropDown
	PurityDD     *gtk.DropDown
	RatioDD      *gtk.DropDown
	ResDD        *gtk.DropDown
	SortDD       *gtk.DropDown
	SearchEntry  *gtk.SearchEntry
	LangBtn      *gtk.Button
	ClearHistBtn *gtk.Button
	StatusLbl    *gtk.Label
	OverflowBtn  *gtk.MenuButton
	OverflowBox  *gtk.Box
	SelectedMon  string

	PeekBox       *gtk.Box
	PeekPic       *gtk.Picture
	PeekTitleLbl  *gtk.Label
	PeekTagLbls   []*gtk.Label
	CurrentPeekID string

	Page      int
	LastPage  int
	Loading   bool
	Query     string
	Sorting   string
	ReqRatio  string
	ReqRes    string
	ReqPurity string
	Seen      map[string]bool
	Lock      sync.Mutex

	PairingSelection *Wallpaper
	favsLoaded       bool
	historyLoaded    bool
	statusSeq        int
}

func NewApp(app *gtk.Application) *App {
	a := &App{
		Win:      gtk.NewApplicationWindow(app),
		Page:     1,
		LastPage: 999,
		Seen:     make(map[string]bool),
	}
	a.Win.SetTitle("WallGTK")
	a.Win.SetDefaultSize(1280, 860)
	a.BuildUI()
	a.SetupEvents()
	return a
}

func (a *App) getLocalizedList(keys []string) []string {
	var list []string
	for _, k := range keys {
		list = append(list, Tr(k))
	}
	return list
}

func (a *App) BuildUI() {
	a.Header = gtk.NewHeaderBar()

	a.MonMenuBox = gtk.NewBox(gtk.OrientationVertical, 4)
	a.MonMenuBox.SetMarginTop(8)
	a.MonMenuBox.SetMarginBottom(8)
	a.MonMenuBox.SetMarginStart(8)
	a.MonMenuBox.SetMarginEnd(8)
	monPopover := gtk.NewPopover()
	monPopover.SetChild(a.MonMenuBox)

	a.MonBtn = gtk.NewMenuButton()
	a.MonBtn.SetPopover(monPopover)
	a.MonBtnLabel = gtk.NewLabel("")
	a.MonBtn.SetChild(a.MonBtnLabel)
	a.Header.PackStart(a.MonBtn)

	a.SearchEntry = gtk.NewSearchEntry()
	a.SearchEntry.SetPlaceholderText(Tr("search_placeholder"))
	a.SearchEntry.SetHExpand(true)
	a.Header.SetTitleWidget(a.SearchEntry)

	a.LangBtn = gtk.NewButtonWithLabel(string(CurrentLang))
	a.LangBtn.ConnectClicked(func() {
		ToggleLang()
		a.LangBtn.SetLabel(string(CurrentLang))
		a.OverflowBtn.Popdown()
		a.UpdateLanguageUI()
	})

	a.ModeDD = gtk.NewDropDownFromStrings(a.getLocalizedList(ModeKeys))
	a.PurityDD = gtk.NewDropDownFromStrings(a.getLocalizedList(PurityKeys))
	a.SortDD = gtk.NewDropDownFromStrings(a.getLocalizedList(SortKeys))
	a.ResDD = gtk.NewDropDownFromStrings(a.getLocalizedList(LandscapeResKeys))
	a.RatioDD = gtk.NewDropDownFromStrings(a.getLocalizedList(LandscapeRatiosKeys))
	a.ClearHistBtn = gtk.NewButtonWithLabel(Tr("clear_history"))
	a.ClearHistBtn.ConnectClicked(func() {
		clearHistory()
		a.historyLoaded = false
		if a.Stack.VisibleChildName() == "history" {
			a.RefreshHistory()
			a.historyLoaded = true
		}
		a.updateClearHistoryState()
		a.OverflowBtn.Popdown()
		a.updateStatus(Tr("history_cleared"))
	})

	a.OverflowBox = gtk.NewBox(gtk.OrientationVertical, 6)
	a.OverflowBox.SetMarginTop(8)
	a.OverflowBox.SetMarginBottom(8)
	a.OverflowBox.SetMarginStart(8)
	a.OverflowBox.SetMarginEnd(8)

	overflowPopover := gtk.NewPopover()
	overflowPopover.SetChild(a.OverflowBox)

	a.OverflowBtn = gtk.NewMenuButton()
	a.OverflowBtn.SetIconName("pan-down-symbolic")
	a.OverflowBtn.SetTooltipText(Tr("more_filters"))
	a.OverflowBtn.SetPopover(overflowPopover)
	a.OverflowBtn.SetVisible(true)

	a.SelectedMon = defaultMonitorKey()
	a.rebuildMonitorMenu()
	a.updateMonitorButtonLabel()
	a.Header.PackEnd(a.OverflowBtn)

	a.Win.SetTitlebar(a.Header)

	a.BrowseFlow = gtk.NewFlowBox()
	a.FavFlow = gtk.NewFlowBox()
	a.HistoryFlow = gtk.NewFlowBox()
	a.configureFlowBox(a.BrowseFlow)
	a.configureFlowBox(a.FavFlow)
	a.configureFlowBox(a.HistoryFlow)

	a.BrowseScroll = newScroll(a.BrowseFlow)
	a.FavScroll = newScroll(a.FavFlow)
	a.HistoryScroll = newScroll(a.HistoryFlow)

	a.Stack = gtk.NewStack()
	a.Stack.SetTransitionType(gtk.StackTransitionTypeCrossfade)
	a.Stack.AddTitled(a.BrowseScroll, "browse", Tr("tab_browse"))
	a.Stack.AddTitled(a.FavScroll, "favs", Tr("tab_favs"))
	a.Stack.AddTitled(a.HistoryScroll, "history", Tr("tab_history"))
	a.rebuildOverflowMenu()

	switcher := gtk.NewStackSwitcher()
	switcher.SetStack(a.Stack)
	switcher.SetHAlign(gtk.AlignCenter)
	switcher.SetVAlign(gtk.AlignStart)
	switcher.SetMarginTop(14)
	switcher.AddCSSClass("floating-switcher")

	a.StatusLbl = gtk.NewLabel("")
	a.StatusLbl.SetHAlign(gtk.AlignStart)
	a.StatusLbl.SetVAlign(gtk.AlignEnd)
	a.StatusLbl.SetMarginStart(14)
	a.StatusLbl.SetMarginBottom(14)
	a.StatusLbl.AddCSSClass("status-toast")
	a.StatusLbl.SetVisible(false)

	mainOverlay := gtk.NewOverlay()
	mainOverlay.SetChild(a.Stack)
	mainOverlay.AddOverlay(switcher)
	mainOverlay.AddOverlay(a.StatusLbl)
	mainOverlay.AddOverlay(a.BuildZoomOverlay())
	a.Win.SetChild(mainOverlay)
	a.installDropTarget()
}

func newScroll(child gtk.Widgetter) *gtk.ScrolledWindow {
	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	scroll.SetHExpand(true)
	scroll.SetChild(child)
	return scroll
}

func (a *App) UpdateLanguageUI() {
	a.SearchEntry.SetPlaceholderText(Tr("search_placeholder"))
	a.OverflowBtn.SetTooltipText(Tr("more_filters"))
	a.LangBtn.SetLabel(string(CurrentLang))
	a.ClearHistBtn.SetLabel(Tr("clear_history"))
	a.rebuildMonitorMenu()
	a.updateMonitorButtonLabel()

	updatePageTitle := func(widget gtk.Widgetter, title string) {
		if page := a.Stack.Page(widget); page != nil {
			page.SetTitle(title)
		}
	}
	updatePageTitle(a.BrowseScroll, Tr("tab_browse"))
	updatePageTitle(a.FavScroll, Tr("tab_favs"))
	updatePageTitle(a.HistoryScroll, Tr("tab_history"))

	updateDD := func(dd *gtk.DropDown, keys []string) {
		idx := dd.Selected()
		dd.SetModel(gtk.NewStringList(a.getLocalizedList(keys)))
		dd.SetSelected(idx)
	}

	updateDD(a.ModeDD, ModeKeys)
	updateDD(a.PurityDD, PurityKeys)
	updateDD(a.SortDD, SortKeys)

	if a.selectedMonitorIsPortrait() {
		updateDD(a.RatioDD, PortraitRatiosKeys)
		updateDD(a.ResDD, PortraitResKeys)
	} else {
		updateDD(a.RatioDD, LandscapeRatiosKeys)
		updateDD(a.ResDD, LandscapeResKeys)
	}
	a.rebuildOverflowMenu()
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
	lastRatio := a.RatioDD.Selected()
	lastRes := a.ResDD.Selected()
	lastPurity := a.PurityDD.Selected()
	lastSort := a.SortDD.Selected()
	lastMode := a.ModeDD.Selected()
	startupTicks := 0
	lastVisible := ""

	glibv2.TimeoutAdd(120, func() bool {
		startupTicks++
		if startupTicks%5 == 0 {
			a.Lock.Lock()
			canLoad := !a.Loading && a.Page <= a.LastPage
			a.Lock.Unlock()

			if canLoad && a.Stack.VisibleChildName() == "browse" {
				vadj := a.BrowseScroll.VAdjustment()
				upper := vadj.Upper()
				pageSize := vadj.PageSize()
				value := vadj.Value()

				if a.BrowseFlow.FirstChild() == nil ||
					(upper > 0 && pageSize > 0 && upper <= pageSize+1200) ||
					(upper > 0 && (value+pageSize) >= (upper-800)) {
					a.LoadMore()
				}
			}
		}

		if a.RatioDD.Selected() != lastRatio || a.ResDD.Selected() != lastRes || a.PurityDD.Selected() != lastPurity || a.SortDD.Selected() != lastSort {
			lastRatio = a.RatioDD.Selected()
			lastRes = a.ResDD.Selected()
			lastPurity = a.PurityDD.Selected()
			lastSort = a.SortDD.Selected()
			a.OverflowBtn.Popdown()
			a.Reload()
		}

		if a.ModeDD.Selected() != lastMode {
			lastMode = a.ModeDD.Selected()
			a.PairingSelection = nil
			if lastMode == 1 {
				a.updateStatus(Tr("pair_mode_wait_first"))
			} else {
				a.updateStatus(Tr("drop_hint"))
			}
		}

		visible := a.Stack.VisibleChildName()
		if visible != lastVisible {
			lastVisible = visible
			a.refreshVisibleLibraryView()
			a.updateClearHistoryState()
			a.rebuildOverflowMenu()
		}

		return true
	})

	a.SearchEntry.ConnectActivate(func() {
		if a.Stack.VisibleChildName() == "browse" {
			a.Reload()
			return
		}
		a.invalidateVisibleLibraryView()
		a.refreshVisibleLibraryView()
	})
}

func (a *App) Reload() {
	clearFlow(a.BrowseFlow)

	a.Lock.Lock()
	a.Page = 1
	a.LastPage = 999
	a.Loading = false
	a.Seen = make(map[string]bool)
	a.Query = a.SearchEntry.Text()
	if idx := a.SortDD.Selected(); int(idx) < len(SortOptions) {
		a.Sorting = SortOptions[idx]
	}
	if idx := a.PurityDD.Selected(); int(idx) < len(PurityOptions) {
		a.ReqPurity = PurityOptions[idx]
	} else {
		a.ReqPurity = PurityOptions[0]
	}
	if getWallhavenAPIKey() == "" && purityNeedsAPIKey(a.ReqPurity) {
		if a.ReqPurity == "111" {
			a.ReqPurity = "110"
			a.updateStatus(Tr("nsfw_all_requires_api"))
		} else {
			a.updateStatus(Tr("nsfw_requires_api"))
		}
	}

	isPort := a.selectedMonitorIsPortrait()
	rIdx := a.RatioDD.Selected()
	resIdx := a.ResDD.Selected()
	if isPort {
		a.ReqRatio, a.ReqRes = PortraitRatios[0], PortraitRes[0]
		if int(rIdx) < len(PortraitRatios) {
			a.ReqRatio = PortraitRatios[rIdx]
		}
		if int(resIdx) < len(PortraitRes) {
			a.ReqRes = PortraitRes[resIdx]
		}
	} else {
		a.ReqRatio, a.ReqRes = LandscapeRatios[0], LandscapeRes[0]
		if int(rIdx) < len(LandscapeRatios) {
			a.ReqRatio = LandscapeRatios[rIdx]
		}
		if int(resIdx) < len(LandscapeRes) {
			a.ReqRes = LandscapeRes[resIdx]
		}
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
	p, q, s, r, res, purity := a.Page, a.Query, a.Sorting, a.ReqRatio, a.ReqRes, a.ReqPurity
	a.Lock.Unlock()

	go func() {
		data, lp := fetchPage(q, s, r, res, purity, p)
		glibv2.IdleAdd(func() bool {
			a.Lock.Lock()
			a.LastPage = lp
			for _, wp := range data {
				if !a.Seen[wp.ID] {
					a.Seen[wp.ID] = true
					tile := a.CreateTile(wp, a.RefreshLibraryViews)
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

func (a *App) RefreshLibraryViews() {
	a.favsLoaded = false
	a.refreshVisibleLibraryView()
}

func (a *App) invalidateVisibleLibraryView() {
	switch a.Stack.VisibleChildName() {
	case "favs":
		a.favsLoaded = false
	case "history":
		a.historyLoaded = false
	}
}

func (a *App) refreshVisibleLibraryView() {
	switch a.Stack.VisibleChildName() {
	case "favs":
		if !a.favsLoaded {
			a.RefreshFavs()
			a.favsLoaded = true
		}
	case "history":
		if !a.historyLoaded {
			a.RefreshHistory()
			a.historyLoaded = true
		}
	}
}

func (a *App) currentMonitorFilter() string {
	if a.SelectedMon == "" {
		return "mon_all"
	}
	return a.SelectedMon
}

func (a *App) selectedMonitorIsPortrait() bool {
	return monitorPrefersPortrait(a.currentMonitorFilter())
}

func monitorPrefersPortrait(key string) bool {
	if key == "mon_sec" {
		return true
	}
	for _, entry := range MonitorEntries {
		if entry.Key == key {
			return entry.Portrait
		}
	}
	return false
}

func (a *App) selectMonitor(key string) {
	if key == "" {
		key = "mon_all"
	}
	if a.SelectedMon == key {
		return
	}
	a.SelectedMon = key
	if a.selectedMonitorIsPortrait() {
		a.RatioDD.SetModel(gtk.NewStringList(a.getLocalizedList(PortraitRatiosKeys)))
		a.ResDD.SetModel(gtk.NewStringList(a.getLocalizedList(PortraitResKeys)))
	} else {
		a.RatioDD.SetModel(gtk.NewStringList(a.getLocalizedList(LandscapeRatiosKeys)))
		a.ResDD.SetModel(gtk.NewStringList(a.getLocalizedList(LandscapeResKeys)))
	}
	a.RatioDD.SetSelected(0)
	a.ResDD.SetSelected(0)
	a.updateMonitorButtonLabel()
	a.favsLoaded = false
	a.historyLoaded = false
	a.Reload()
	a.refreshVisibleLibraryView()
}

func (a *App) rebuildMonitorMenu() {
	clearBox(a.MonMenuBox)
	for _, entry := range MonitorEntries {
		entry := entry
		btn := gtk.NewButtonWithLabel(monitorMenuLabel(entry))
		btn.SetHAlign(gtk.AlignFill)
		btn.ConnectClicked(func() {
			a.selectMonitor(entry.Key)
			a.MonBtn.Popdown()
		})
		a.MonMenuBox.Append(btn)
	}
	modeBtn := gtk.NewButtonWithLabel(a.pairingMenuLabel())
	modeBtn.SetHAlign(gtk.AlignFill)
	modeBtn.ConnectClicked(func() {
		if a.ModeDD.Selected() == 1 {
			a.ModeDD.SetSelected(0)
		} else {
			a.ModeDD.SetSelected(1)
		}
		a.rebuildMonitorMenu()
		a.MonBtn.Popdown()
	})
	a.MonMenuBox.Append(modeBtn)
}

func (a *App) updateMonitorButtonLabel() {
	current := a.currentMonitorFilter()
	for _, entry := range MonitorEntries {
		if entry.Key == current {
			a.MonBtnLabel.SetText(monitorShortLabel(entry))
			return
		}
	}
	a.MonBtnLabel.SetText(Tr("mon_all"))
}

func defaultMonitorKey() string {
	if len(MonitorEntries) > 1 {
		return MonitorEntries[1].Key
	}
	return "mon_all"
}

func (a *App) updateClearHistoryState() {
	hasHistory := len(listHistory()) > 0
	a.ClearHistBtn.SetSensitive(hasHistory)
	a.ClearHistBtn.SetVisible(a.Stack.VisibleChildName() == "history")
}

func (a *App) RefreshFavs() {
	clearFlow(a.FavFlow)
	favMutex.RLock()
	var items []Wallpaper
	for _, wp := range favorites {
		items = append(items, wp)
	}
	favMutex.RUnlock()
	for _, wp := range filterByMonitor(filterByQuery(items, a.SearchEntry.Text()), a.currentMonitorFilter()) {
		a.FavFlow.Append(a.CreateTile(wp, a.RefreshLibraryViews))
	}
}

func (a *App) RefreshHistory() {
	clearFlow(a.HistoryFlow)
	var items []Wallpaper
	for _, entry := range listHistory() {
		wp := entry.Wallpaper
		if wp.Source == "" {
			wp.Source = entry.Monitor
		}
		items = append(items, wp)
	}
	for _, wp := range filterByMonitor(filterByQuery(items, a.SearchEntry.Text()), a.currentMonitorFilter()) {
		a.HistoryFlow.Append(a.CreateTile(wp, nil))
	}
	a.updateClearHistoryState()
}

func filterByMonitor(items []Wallpaper, mon string) []Wallpaper {
	if mon == "mon_all" {
		return items
	}
	var filtered []Wallpaper
	portrait := monitorPrefersPortrait(mon)
	for _, wp := range items {
		if isPortrait(wp.Resolution) != portrait {
			continue
		}
		filtered = append(filtered, wp)
	}
	return filtered
}

func filterByQuery(items []Wallpaper, query string) []Wallpaper {
	var filtered []Wallpaper
	for _, wp := range items {
		if matchesLocalQuery(wp, query) {
			filtered = append(filtered, wp)
		}
	}
	return filtered
}

func clearFlow(flow *gtk.FlowBox) {
	for child := flow.FirstChild(); child != nil; child = flow.FirstChild() {
		flow.Remove(child)
	}
}

func (a *App) GetSelectedMonitor() string {
	return a.currentMonitorFilter()
}

func (a *App) PairingModeEnabled() bool {
	return a.ModeDD.Selected() == 1
}

func (a *App) HandleWallpaperSelection(wp Wallpaper) {
	if !a.PairingModeEnabled() {
		a.ApplySingleWallpaper(wp)
		return
	}

	if a.PairingSelection == nil {
		clone := wp
		a.PairingSelection = &clone
		a.updateStatus(Tr("pair_mode_wait_second"))
		return
	}

	first := *a.PairingSelection
	a.PairingSelection = nil
	a.ApplyPairWallpaper(first, wp)
}

func (a *App) ApplySingleWallpaper(wp Wallpaper) {
	a.applyResolvedWallpaper(map[string]Wallpaper{
		a.GetSelectedMonitor(): wp,
	}, a.GetSelectedMonitor())
}

func (a *App) ApplyPairWallpaper(primary, secondary Wallpaper) {
	targets := map[string]Wallpaper{
		"mon_pri": primary,
		"mon_sec": secondary,
	}
	a.applyResolvedWallpaper(targets, "pair")
}

func (a *App) applyResolvedWallpaper(targets map[string]Wallpaper, historyMonitor string) {
	go func() {
		resolved := make(map[string]string)
		for monitor, wp := range targets {
			finalPath := wp.Path
			if stringsHasHTTP(finalPath) {
				p := filepath.Join(cacheDir, wp.ID+"_"+monitor+".jpg")
				if !download(wp.Path, p) {
					glibv2.IdleAdd(func() bool {
						a.updateStatus(Tr("download_failed"))
						return false
					})
					return
				}
				finalPath = p
			} else if !fileExists(finalPath) {
				glibv2.IdleAdd(func() bool {
					a.updateStatus(Tr("local_missing"))
					return false
				})
				return
			}
			resolved[monitor] = finalPath
		}

		if !setWallpaperPair(resolved) {
			glibv2.IdleAdd(func() bool {
				a.updateStatus(Tr("backend_missing"))
				return false
			})
			return
		}

		for _, wp := range targets {
			recordHistory(wp, historyMonitor)
		}
		glibv2.IdleAdd(func() bool {
			a.historyLoaded = false
			if a.Stack.VisibleChildName() == "history" {
				a.RefreshHistory()
				a.historyLoaded = true
			}
			if historyMonitor == "pair" {
				a.updateStatus(Tr("pair_applied"))
			} else {
				a.updateStatus(Tr("wallpaper_applied"))
			}
			return false
		})
	}()
}

func (a *App) installDropTarget() {
	builder := gdk.NewContentFormatsBuilder()
	builder.AddMIMEType("text/uri-list")
	target := gtk.NewDropTargetAsync(builder.ToFormats(), gdk.ActionCopy)
	target.ConnectAccept(func(drop gdk.Dropper) bool {
		return true
	})
	target.ConnectDrop(func(drop gdk.Dropper, x, y float64) bool {
		baseDrop := gdk.BaseDrop(drop)
		baseDrop.ReadAsync(context.Background(), []string{"text/uri-list"}, 0, func(res gio.AsyncResulter) {
			mimeType, stream, err := baseDrop.ReadFinish(res)
			if err != nil || mimeType != "text/uri-list" || stream == nil {
				baseDrop.Finish(gdk.ActionCopy)
				glibv2.IdleAdd(func() bool {
					a.updateStatus(Tr("drop_invalid"))
					return false
				})
				return
			}

			go func() {
				data := readAllInputStream(stream)
				imported := importDroppedURIList(string(data))
				glibv2.IdleAdd(func() bool {
					baseDrop.Finish(gdk.ActionCopy)
					if len(imported) == 0 {
						a.updateStatus(Tr("drop_invalid"))
						return false
					}
					a.favsLoaded = false
					a.historyLoaded = false
					a.updateStatus(Tr("drop_success"))
					return false
				})
			}()
		})
		return true
	})
	a.Win.AddController(target)
}

func (a *App) updateStatus(text string) {
	a.statusSeq++
	seq := a.statusSeq
	a.StatusLbl.SetText(text)
	a.StatusLbl.SetVisible(text != "")
	if text == "" {
		return
	}
	go func() {
		time.Sleep(5 * time.Second)
		glibv2.IdleAdd(func() bool {
			if a.statusSeq == seq {
				a.StatusLbl.SetText("")
				a.StatusLbl.SetVisible(false)
			}
			return false
		})
	}()
}

func (a *App) Show() {
	a.Win.Show()
	a.updateClearHistoryState()
	glibv2.IdleAdd(func() bool {
		a.Reload()
		return false
	})
}

func stringsHasHTTP(path string) bool {
	return len(path) >= 4 && path[:4] == "http"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readAllInputStream(stream gio.InputStreamer) []byte {
	input := gio.BaseInputStream(stream)
	defer input.Close(context.Background())

	var out bytes.Buffer
	buf := make([]byte, 32*1024)
	for {
		n, err := input.Read(context.Background(), buf)
		if n > 0 {
			out.Write(buf[:n])
		}
		if err != nil || n == 0 {
			break
		}
	}
	return out.Bytes()
}

func (a *App) rebuildOverflowMenu() {
	clearBox(a.OverflowBox)
	for _, widget := range []gtk.Widgetter{
		a.RatioDD,
		a.ResDD,
		a.SortDD,
		a.PurityDD,
		a.LangBtn,
	} {
		a.OverflowBox.Append(widget)
	}
	if a.Stack != nil && a.Stack.VisibleChildName() == "history" {
		a.OverflowBox.Append(a.ClearHistBtn)
	}
}

func (a *App) pairingMenuLabel() string {
	if CurrentLang == LangRU {
		if a.ModeDD.Selected() == 1 {
			return "Pairing: Вкл"
		}
		return "Pairing: Выкл"
	}
	if a.ModeDD.Selected() == 1 {
		return "Pairing: On"
	}
	return "Pairing: Off"
}

func clearBox(box *gtk.Box) {
	for child := box.FirstChild(); child != nil; child = box.FirstChild() {
		box.Remove(child)
	}
}
