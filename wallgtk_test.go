package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestIsPortrait(t *testing.T) {
	cases := map[string]bool{
		"1080x1920": true,
		"1920x1080": false,
		"1080x1080": false,
		"":          false,
		"garbage":   false,
		"1920x":     false,
	}
	for res, want := range cases {
		if got := isPortrait(res); got != want {
			t.Errorf("isPortrait(%q) = %v, want %v", res, got, want)
		}
	}
}

func TestPurityNeedsAPIKey(t *testing.T) {
	cases := map[string]bool{
		"100": false, // sfw
		"010": false, // sketchy
		"001": true,  // nsfw
		"111": true,  // all
		"11":  false, // мусор не должен паниковать
		"":    false,
	}
	for purity, want := range cases {
		if got := purityNeedsAPIKey(purity); got != want {
			t.Errorf("purityNeedsAPIKey(%q) = %v, want %v", purity, got, want)
		}
	}
}

func TestMonitorPrefersPortraitUsesDetection(t *testing.T) {
	saved := MonitorEntries
	defer func() { MonitorEntries = saved }()

	MonitorEntries = []MonitorEntry{
		{Key: "mon_all"},
		{Key: "mon_pri", Output: "DP-1", Portrait: false},
		{Key: "mon_sec", Output: "DP-2", Portrait: false},
		{Key: "output:HDMI-1", Output: "HDMI-1", Portrait: true},
	}

	// Раньше mon_sec безусловно считался вертикальным.
	if monitorPrefersPortrait("mon_sec") {
		t.Error("mon_sec must follow detected orientation, not a hardcoded guess")
	}
	if !monitorPrefersPortrait("output:HDMI-1") {
		t.Error("detected portrait output reported as landscape")
	}
	if monitorPrefersPortrait("unknown") {
		t.Error("unknown monitor must default to landscape")
	}
}

func TestAssignMonitorOutputs(t *testing.T) {
	savedOutputs, savedEntries := MonitorOutputs, MonitorEntries
	defer func() { MonitorOutputs, MonitorEntries = savedOutputs, savedEntries }()

	assignMonitorOutputs([]detectedMonitor{
		{Name: "DP-1", Portrait: false},
		{Name: "DP-2", Primary: true, Portrait: true},
		{Name: "HDMI-1"},
	})

	if MonitorOutputs["mon_pri"] != "DP-2" {
		t.Errorf("primary = %q, want DP-2", MonitorOutputs["mon_pri"])
	}
	if MonitorOutputs["mon_sec"] != "DP-1" {
		t.Errorf("secondary = %q, want DP-1", MonitorOutputs["mon_sec"])
	}
	if MonitorOutputs["output:HDMI-1"] != "HDMI-1" {
		t.Error("third monitor missing from MonitorOutputs")
	}
	if !monitorPrefersPortrait("mon_pri") {
		t.Error("portrait flag lost for primary monitor")
	}
	if len(MonitorEntries) != 4 {
		t.Errorf("MonitorEntries = %d, want 4 (all + 3)", len(MonitorEntries))
	}
}

func TestAssignMonitorOutputsEmpty(t *testing.T) {
	savedOutputs, savedEntries := MonitorOutputs, MonitorEntries
	defer func() { MonitorOutputs, MonitorEntries = savedOutputs, savedEntries }()

	assignMonitorOutputs(nil)
	if len(MonitorOutputs) != 0 {
		t.Errorf("MonitorOutputs = %v, want empty", MonitorOutputs)
	}
	if len(MonitorEntries) != 1 || MonitorEntries[0].Key != "mon_all" {
		t.Errorf("MonitorEntries = %v, want only mon_all", MonitorEntries)
	}
}

func TestSortedKeysIsDeterministic(t *testing.T) {
	paths := map[string]string{
		"output:HDMI-1": "c.jpg",
		"mon_sec":       "b.jpg",
		"mon_pri":       "a.jpg",
	}
	want := []string{"mon_pri", "mon_sec", "output:HDMI-1"}
	// Порядок обхода мапы в Go случаен — прогоняем несколько раз.
	for i := 0; i < 20; i++ {
		got := sortedKeys(paths)
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("sortedKeys = %v, want %v", got, want)
			}
		}
	}
}

func TestMatchesLocalQuery(t *testing.T) {
	wp := Wallpaper{ID: "abc123", Path: "/home/u/Wallpapers/forest.jpg", Resolution: "1920x1080", Source: "library"}
	for _, q := range []string{"", "  ", "FOREST", "abc", "1920", "library"} {
		if !matchesLocalQuery(wp, q) {
			t.Errorf("query %q should match", q)
		}
	}
	if matchesLocalQuery(wp, "desert") {
		t.Error("query \"desert\" should not match")
	}
}

func TestIsSupportedImage(t *testing.T) {
	for _, name := range []string{"a.jpg", "a.JPEG", "b.png", "c.webp"} {
		if !isSupportedImage(name) {
			t.Errorf("%q should be supported", name)
		}
	}
	for _, name := range []string{"a.txt", "a.mp4", "noext"} {
		if isSupportedImage(name) {
			t.Errorf("%q should not be supported", name)
		}
	}
}

func TestIsCacheArtifact(t *testing.T) {
	for _, name := range []string{"x_thumb.jpg", "x.png", "x.jpg.tmp"} {
		if !isCacheArtifact(name) {
			t.Errorf("%q should be prunable", name)
		}
	}
	// Служебные файлы обрезка кэша трогать не должна.
	for _, name := range []string{"history.json", "lang.txt", "favorites.json"} {
		if isCacheArtifact(name) {
			t.Errorf("%q must never be pruned", name)
		}
	}
}

func TestStringsHasHTTP(t *testing.T) {
	if !stringsHasHTTP("https://w.cc/a.jpg") || !stringsHasHTTP("http://w.cc/a.jpg") {
		t.Error("http(s) URLs not recognised")
	}
	if stringsHasHTTP("/home/u/a.jpg") || stringsHasHTTP("ht") {
		t.Error("local path treated as URL")
	}
}

// Регрессия: раньше все ожидающие, кроме первого, получали false
// из закрытого канала, и уже скачанный файл считался неудачей.
func TestConcurrentDownloadSharesResult(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "shared.jpg")
	if err := os.WriteFile(dest, []byte("already here"), 0644); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	results := make([]bool, 8)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = download("http://127.0.0.1:1/never", dest)
		}(i)
	}
	wg.Wait()

	for i, ok := range results {
		if !ok {
			t.Fatalf("waiter %d got false for an existing file", i)
		}
	}
}

func TestWriteFileAtomicLeavesNoTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")
	if err := writeFileAtomic(path, []byte(`[]`), 0644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "[]" {
		t.Fatalf("content = %q, err = %v", data, err)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file %q left behind", e.Name())
		}
	}
}

func TestCopyFileAtomic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.jpg")
	if err := os.WriteFile(src, []byte("pixels"), 0644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "nested", "dst.jpg")
	if !copyFile(src, dst) {
		t.Fatal("copyFile reported failure")
	}
	if data, err := os.ReadFile(dst); err != nil || string(data) != "pixels" {
		t.Fatalf("content = %q, err = %v", data, err)
	}
	if copyFile(filepath.Join(dir, "missing.jpg"), dst) {
		t.Error("copyFile should fail for a missing source")
	}
}

func TestParseXrandrQuery(t *testing.T) {
	out := `Screen 0: minimum 320 x 200, current 3000 x 1920
DP-2 connected primary 1080x1920+0+0 (normal left inverted right x axis y axis) 340mm x 190mm
   1080x1920     60.00*+
DP-1 connected 1920x1080+1080+0 (normal left inverted right x axis y axis) 600mm x 340mm
HDMI-1 disconnected (normal left inverted right x axis y axis)
`
	monitors := parseXrandrQuery(out)
	if len(monitors) != 2 {
		t.Fatalf("got %d monitors, want 2 (disconnected must be skipped)", len(monitors))
	}
	if monitors[0].Name != "DP-2" || !monitors[0].Primary || !monitors[0].Portrait {
		t.Errorf("DP-2 parsed as %+v, want primary portrait", monitors[0])
	}
	if monitors[1].Name != "DP-1" || monitors[1].Primary || monitors[1].Portrait {
		t.Errorf("DP-1 parsed as %+v, want secondary landscape", monitors[1])
	}
}

func TestParseSwwwQuery(t *testing.T) {
	out := "DP-1: 1920x1080, scale: 1, currently displaying: image: /a.jpg\nDP-2: 1080x1920, scale: 1\n\n"
	monitors := parseSwwwQuery(out)
	if len(monitors) != 2 {
		t.Fatalf("got %d monitors, want 2", len(monitors))
	}
	if monitors[0].Name != "DP-1" || !monitors[0].Primary {
		t.Errorf("first monitor = %+v, want DP-1 primary", monitors[0])
	}
	if monitors[1].Name != "DP-2" || monitors[1].Primary {
		t.Errorf("second monitor = %+v, want DP-2 non-primary", monitors[1])
	}
	if len(parseSwwwQuery("\n  \n")) != 0 {
		t.Error("blank output should yield no monitors")
	}
}

func TestParseSwayOutputs(t *testing.T) {
	data := []byte(`[
	  {"name":"DP-1","active":true,"focused":false,"rect":{"width":1920,"height":1080}},
	  {"name":"DP-2","active":true,"focused":true,"rect":{"width":1080,"height":1920}},
	  {"name":"HDMI-A-1","active":false,"focused":false,"rect":{"width":0,"height":0}}
	]`)
	monitors := parseSwayOutputs(data)
	if len(monitors) != 2 {
		t.Fatalf("got %d monitors, want 2 (inactive must be skipped)", len(monitors))
	}
	if monitors[0].Name != "DP-1" || monitors[0].Portrait || monitors[0].Primary {
		t.Errorf("DP-1 parsed as %+v", monitors[0])
	}
	if !monitors[1].Primary || !monitors[1].Portrait {
		t.Errorf("DP-2 parsed as %+v, want focused portrait", monitors[1])
	}
	if parseSwayOutputs([]byte("not json")) != nil {
		t.Error("garbage input should yield no monitors")
	}
}

func withMonitors(t *testing.T, entries []MonitorEntry, outputs map[string]string) {
	t.Helper()
	savedEntries, savedOutputs := MonitorEntries, MonitorOutputs
	t.Cleanup(func() { MonitorEntries, MonitorOutputs = savedEntries, savedOutputs })
	MonitorEntries, MonitorOutputs = entries, outputs
}

func testMonitors(t *testing.T) {
	t.Helper()
	withMonitors(t,
		[]MonitorEntry{
			{Key: "mon_all"},
			{Key: "mon_pri", Output: "DP-1"},
			{Key: "mon_sec", Output: "DP-2"},
			{Key: "output:HDMI-1", Output: "HDMI-1"},
		},
		map[string]string{"mon_pri": "DP-1", "mon_sec": "DP-2", "output:HDMI-1": "HDMI-1"},
	)
}

func TestBuildTargetsResolvesOutputs(t *testing.T) {
	testMonitors(t)

	targets := buildTargets(map[string]string{"mon_sec": "b.jpg", "mon_pri": "a.jpg"})
	if len(targets) != 2 {
		t.Fatalf("got %d targets, want 2", len(targets))
	}
	if targets[0].Key != "mon_pri" || targets[0].Output != "DP-1" || targets[0].Path != "a.jpg" {
		t.Errorf("first target = %+v", targets[0])
	}
	if targets[1].Output != "DP-2" {
		t.Errorf("second target = %+v", targets[1])
	}

	// mon_all не имеет имени выхода: бэкенды трактуют пустую строку как «все».
	all := buildTargets(map[string]string{"mon_all": "a.jpg"})
	if len(all) != 1 || all[0].Output != "" {
		t.Errorf("mon_all target = %+v, want empty output", all)
	}
}

func TestMonitorIndex(t *testing.T) {
	testMonitors(t)

	for output, want := range map[string]int{"DP-1": 0, "DP-2": 1, "HDMI-1": 2, "VGA-1": -1} {
		if got := monitorIndex(output); got != want {
			t.Errorf("monitorIndex(%q) = %d, want %d", output, got, want)
		}
	}
}

func TestPathsInMonitorOrder(t *testing.T) {
	testMonitors(t)

	// Оба монитора заданы явно.
	got := pathsInMonitorOrder([]wallpaperTarget{
		{Key: "mon_pri", Output: "DP-1", Path: "a.jpg"},
		{Key: "mon_sec", Output: "DP-2", Path: "b.jpg"},
	})
	want := []string{"a.jpg", "b.jpg", "a.jpg"} // HDMI-1 добирает запасной путь
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}

	// Одна цель на все мониторы — один аргумент, feh растянет его сам.
	single := pathsInMonitorOrder([]wallpaperTarget{{Key: "mon_all", Path: "a.jpg"}})
	if len(single) != 1 || single[0] != "a.jpg" {
		t.Errorf("got %v, want [a.jpg]", single)
	}

	if pathsInMonitorOrder(nil) != nil {
		t.Error("no targets should yield no paths")
	}
}

func TestSelectWallpaperBackendHonoursOverride(t *testing.T) {
	saved := backendOverride
	t.Cleanup(func() { backendOverride = saved })
	t.Setenv("WALLGTK_BACKEND", "")

	backendOverride = "hyprpaper"
	// Переопределение не должно зависеть от того, установлен ли бинарь.
	if b := selectWallpaperBackend(); b == nil || b.Name != "hyprpaper" {
		t.Fatalf("override ignored, got %v", b)
	}

	backendOverride = "NiTrOgEn"
	if b := selectWallpaperBackend(); b == nil || b.Name != "nitrogen" {
		t.Error("override should be case-insensitive")
	}
}

func TestBackendRegistryIsWellFormed(t *testing.T) {
	seen := make(map[string]bool)
	for _, b := range wallpaperBackends {
		if b.Name == "" {
			t.Error("backend without a name")
		}
		if seen[b.Name] {
			t.Errorf("duplicate backend name %q breaks -backend selection", b.Name)
		}
		seen[b.Name] = true
		if b.Detect == nil || b.Apply == nil {
			t.Errorf("backend %q is missing Detect or Apply", b.Name)
		}
	}
}

func TestSessionDetection(t *testing.T) {
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	t.Setenv("WAYLAND_DISPLAY", "wayland-1")
	t.Setenv("DISPLAY", ":0")
	if !isWayland() {
		t.Error("wayland session not detected")
	}
	// DISPLAY под Wayland — это XWayland, X11-утилиты обои там не поменяют.
	if isX11() {
		t.Error("XWayland must not be treated as an X11 session")
	}

	t.Setenv("XDG_SESSION_TYPE", "x11")
	t.Setenv("WAYLAND_DISPLAY", "")
	if isWayland() || !isX11() {
		t.Error("x11 session not detected")
	}

	t.Setenv("XDG_CURRENT_DESKTOP", "ubuntu:GNOME")
	if !desktopIs("gnome") || desktopIs("kde") {
		t.Error("desktopIs failed on a composite XDG_CURRENT_DESKTOP")
	}
}

func TestSwwwSkippedOnGnome(t *testing.T) {
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	t.Setenv("WAYLAND_DISPLAY", "wayland-1")
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	t.Setenv("XDG_SESSION_DESKTOP", "")
	t.Setenv("DESKTOP_SESSION", "")
	// Mutter не поддерживает wlr-layer-shell, обоев от swww там не увидеть.
	if detectSwww() {
		t.Error("swww must not be selected under GNOME")
	}
}
