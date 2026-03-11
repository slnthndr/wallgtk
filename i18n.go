// --- START OF FILE i18n.go ---
package main

import (
	"os"
	"path/filepath"
)

type Language string

const (
	LangRU Language = "ru"
	LangEN Language = "en"
)

var (
	CurrentLang Language = LangRU
	langFile    string
)

// InitI18n вызывается после инициализации директорий
func InitI18n() {
	langFile = filepath.Join(cacheDir, "lang.txt")
	if data, err := os.ReadFile(langFile); err == nil {
		if string(data) == string(LangEN) {
			CurrentLang = LangEN
		}
	}
}

func ToggleLang() {
	if CurrentLang == LangRU {
		CurrentLang = LangEN
	} else {
		CurrentLang = LangRU
	}
	os.WriteFile(langFile, []byte(CurrentLang), 0644)
}

func Tr(key string) string {
	if val, ok := dict[CurrentLang][key]; ok {
		return val
	}
	return key // Возвращаем ключ, если перевод не найден
}

var dict = map[Language]map[string]string{
	LangRU: {
		"tab_browse":         "Просмотр",
		"tab_favs":           "Избранное",
		"monitor_lbl":        "Монитор:",
		"search_placeholder": "Поиск...",
		"loading":            "Загрузка...",
		"no_tags":            "Нет тегов",
		"wallpaper_id":       "Обои #",
		
		// Мониторы
		"mon_all": "Все",
		"mon_pri": "Основной монитор",
		"mon_sec": "Второй монитор",

		// Сортировка
		"sort_date": "Сначала новые",
		"sort_rel":  "Релевантные",
		"sort_rnd":  "Случайные",
		"sort_view": "По просмотрам",
		"sort_fav":  "По популярности",
		"sort_top":  "Топ лист",

		// Разрешения и соотношения
		"ratio_any_l": "Широкие (Любые)",
		"ratio_any_p": "Вертикальные (Любые)",
		"res_2k":      ">= 2K (1440p)",
		"res_fhd":     ">= Full HD (1080p)",
		"res_4k":      ">= 4K (2160p)",
	},
	LangEN: {
		"tab_browse":         "Browse",
		"tab_favs":           "Favorites",
		"monitor_lbl":        "Monitor:",
		"search_placeholder": "Search...",
		"loading":            "Loading...",
		"no_tags":            "No tags",
		"wallpaper_id":       "Wallpaper #",

		// Monitors
		"mon_all": "All",
		"mon_pri": "Primary Monitor",
		"mon_sec": "Secondary Monitor",

		// Sorting
		"sort_date": "Date Added",
		"sort_rel":  "Relevance",
		"sort_rnd":  "Random",
		"sort_view": "Views",
		"sort_fav":  "Favorites",
		"sort_top":  "Toplist",

		// Ratios & Resolutions
		"ratio_any_l": "Landscape (Any)",
		"ratio_any_p": "Portrait (Any)",
		"res_2k":      ">= 2K (1440p)",
		"res_fhd":     ">= Full HD (1080p)",
		"res_4k":      ">= 4K (2160p)",
	},
}
