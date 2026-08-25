package main

import (
	"os"
	"path/filepath"
	"testing"
)

// backendByName достаёт зарегистрированный бэкенд, чтобы тесты не зависели
// от его позиции в списке.
func backendByName(t *testing.T, name string) *wallpaperBackend {
	t.Helper()
	for _, b := range wallpaperBackends {
		if b.Name == name {
			return b
		}
	}
	t.Fatalf("бэкенд %q не зарегистрирован", name)
	return nil
}

// Omarchy должен стоять выше swww: иначе обои уедут мимо состояния темы.
func TestOmarchyOutranksSwww(t *testing.T) {
	omarchy, swww := -1, -1
	for i, b := range wallpaperBackends {
		switch b.Name {
		case "omarchy":
			omarchy = i
		case "swww":
			swww = i
		}
	}
	if omarchy < 0 || swww < 0 {
		t.Fatalf("нужны оба бэкенда, получили omarchy=%d swww=%d", omarchy, swww)
	}
	if omarchy > swww {
		t.Errorf("omarchy на позиции %d, swww на %d — omarchy должен быть раньше", omarchy, swww)
	}
	if backendByName(t, "omarchy").PerOutput {
		t.Error("у omarchy не должно быть PerOutput: фон один на весь рабочий стол")
	}
}

// Запрос на разные обои по мониторам обязан обойти omarchy и уйти к swww/awww,
// иначе один из мониторов остался бы со старым фоном.
func TestPerOutputRequestPrefersSwww(t *testing.T) {
	saveActive := activeBackend
	defer func() { activeBackend = saveActive }()

	omarchy := backendByName(t, "omarchy")
	swww := backendByName(t, "swww")
	if !swww.Detect() {
		t.Skip("swww/awww не установлен")
	}
	activeBackend = omarchy

	targets := []wallpaperTarget{
		{Key: "mon_pri", Output: "DP-1", Path: "/tmp/a.jpg"},
		{Key: "mon_sec", Output: "DP-2", Path: "/tmp/b.jpg"},
	}
	got := candidateBackends(targets)
	if len(got) == 0 || got[0] != swww {
		name := "(пусто)"
		if len(got) > 0 {
			name = got[0].Name
		}
		t.Errorf("первым кандидатом ожидался swww, получили %s", name)
	}

	// Одна цель — omarchy остаётся основным.
	got = candidateBackends(targets[:1])
	if len(got) == 0 || got[0] != omarchy {
		name := "(пусто)"
		if len(got) > 0 {
			name = got[0].Name
		}
		t.Errorf("для одной цели ожидался omarchy, получили %s", name)
	}
}

// Активный бэкенд упал — запрос обязан дойти до запасного.
func TestFallbackAfterFailure(t *testing.T) {
	saveActive, saveList := activeBackend, wallpaperBackends
	defer func() { activeBackend, wallpaperBackends = saveActive, saveList }()

	triedBroken, triedSpare := false, false
	broken := &wallpaperBackend{
		Name:   "broken",
		Detect: func() bool { return true },
		Apply: func([]wallpaperTarget) error {
			triedBroken = true
			return os.ErrPermission
		},
	}
	spare := &wallpaperBackend{
		Name:   "spare",
		Detect: func() bool { return true },
		Apply: func([]wallpaperTarget) error {
			triedSpare = true
			return nil
		},
	}
	wallpaperBackends = []*wallpaperBackend{broken, spare}
	activeBackend = broken

	if !setWallpaperPair(map[string]string{"mon_all": "/tmp/a.jpg"}) {
		t.Fatal("ожидалось, что запасной бэкенд справится")
	}
	if !triedBroken || !triedSpare {
		t.Errorf("ожидались попытки обоих: broken=%v spare=%v", triedBroken, triedSpare)
	}
}

// Живая проверка против настоящей Omarchy. Переставляет те обои, что уже стоят,
// поэтому картинка на экране не меняется, а весь путь проверяется по-настоящему.
func TestApplyOmarchyLive(t *testing.T) {
	if !detectOmarchy() {
		t.Skip("omarchy-theme-bg-set / omarchy-shell недоступны")
	}

	link := filepath.Join(os.Getenv("HOME"), ".local/state/omarchy/current/background")
	current, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Skipf("нет текущих обоев для повторной установки: %v", err)
	}

	if name := detectWallpaperBackend(); name != "omarchy" {
		t.Errorf("detectWallpaperBackend() = %q, ожидалось %q", name, "omarchy")
	}

	if err := applyOmarchy([]wallpaperTarget{{Key: "mon_all", Path: current}}); err != nil {
		t.Fatalf("applyOmarchy не смог переставить текущие обои %q: %v", current, err)
	}

	after, err := filepath.EvalSymlinks(link)
	if err != nil || after != current {
		t.Errorf("обои стали %q (ошибка %v), ожидались неизменные %q", after, err, current)
	}
}
