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
	writeFileAtomic(langFile, []byte(CurrentLang), 0644)
}

func Tr(key string) string {
	if val, ok := dict[CurrentLang][key]; ok {
		return val
	}
	return key // Возвращаем ключ, если перевод не найден
}

var dict = map[Language]map[string]string{
	LangRU: {
		"tab_browse":            "Просмотр",
		"tab_favs":              "Избранное",
		"tab_history":           "История",
		"monitor_lbl":           "Монитор",
		"search_placeholder":    "Поиск...",
		"loading":               "Загрузка...",
		"no_tags":               "Нет тегов",
		"wallpaper_id":          "Обои #",
		"more_filters":          "Дополнительно",
		"drop_hint":             "Можно перетащить локальные изображения в окно для импорта в библиотеку.",
		"drop_success":          "Изображения импортированы в локальную библиотеку.",
		"drop_invalid":          "Не удалось импортировать перетаскиваемые файлы.",
		"download_failed":       "Не удалось скачать изображение.",
		"local_missing":         "Локальный файл не найден.",
		"backend_missing":       "Не удалось применить обои: нет подходящего backend'а. Запустите `wallgtk -list-backends`.",
		"nsfw_requires_api":     "NSFW через Wallhaven API требует ключ. Укажите `WALLHAVEN_API` или `APIKey` в config.go.",
		"nsfw_all_requires_api": "Фильтр \"Все\" переключён на SFW+Sketchy: NSFW через API требует ключ.",
		"wallpaper_applied":     "Обои применены.",
		"pair_applied":          "Пара обоев применена на два монитора.",
		"pair_mode_wait_first":  "Pairing mode: выберите первый wallpaper.",
		"pair_mode_wait_second": "Pairing mode: выберите второй wallpaper для второго монитора.",
		"no_candidates":         "Нет подходящих изображений для действия.",
		"clear_history":         "Очистить историю",
		"history_cleared":       "История очищена.",
		"preview_crop":          "Превью: Crop",
		"preview_fit":           "Превью: Fit",
		"preview_fill":          "Превью: Fill",
		"preview_center":        "Превью: Center",

		// Мониторы
		"mon_all":        "Все",
		"mon_pri":        "Основной монитор",
		"mon_sec":        "Второй монитор",
		"mode_single":    "Одиночный",
		"mode_pair":      "Pairing",
		"purity_sfw":     "SFW",
		"purity_sketchy": "Sketchy",
		"purity_nsfw":    "NSFW",
		"purity_all":     "Все",

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
		"tab_browse":            "Browse",
		"tab_favs":              "Favorites",
		"tab_history":           "History",
		"monitor_lbl":           "Monitor",
		"search_placeholder":    "Search...",
		"loading":               "Loading...",
		"no_tags":               "No tags",
		"wallpaper_id":          "Wallpaper #",
		"more_filters":          "More",
		"drop_hint":             "Drag local images into the window to import them into the library.",
		"drop_success":          "Images were imported into the local library.",
		"drop_invalid":          "Failed to import dropped files.",
		"download_failed":       "Failed to download the image.",
		"local_missing":         "Local file is missing.",
		"backend_missing":       "Could not apply the wallpaper: no usable backend. Run `wallgtk -list-backends`.",
		"nsfw_requires_api":     "NSFW via the Wallhaven API requires an API key. Set `WALLHAVEN_API` or `APIKey` in config.go.",
		"nsfw_all_requires_api": "The \"All\" filter was downgraded to SFW+Sketchy: NSFW via the API requires a key.",
		"wallpaper_applied":     "Wallpaper applied.",
		"pair_applied":          "Wallpaper pair applied to two monitors.",
		"pair_mode_wait_first":  "Pairing mode: choose the first wallpaper.",
		"pair_mode_wait_second": "Pairing mode: choose the second wallpaper.",
		"no_candidates":         "No matching wallpapers available.",
		"clear_history":         "Clear History",
		"history_cleared":       "History cleared.",
		"preview_crop":          "Preview: Crop",
		"preview_fit":           "Preview: Fit",
		"preview_fill":          "Preview: Fill",
		"preview_center":        "Preview: Center",

		// Monitors
		"mon_all":        "All",
		"mon_pri":        "Primary Monitor",
		"mon_sec":        "Secondary Monitor",
		"mode_single":    "Single",
		"mode_pair":      "Pairing",
		"purity_sfw":     "SFW",
		"purity_sketchy": "Sketchy",
		"purity_nsfw":    "NSFW",
		"purity_all":     "All",

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
