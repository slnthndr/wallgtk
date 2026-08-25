package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func main() {
	listBackends := flag.Bool("list-backends", false, "показать подходящие бэкенды обоев и выйти")
	flag.BoolVar(&verbose, "v", verbose, "подробный лог в stderr")
	flag.StringVar(&backendOverride, "backend", "", "принудительно выбрать бэкенд обоев (см. -list-backends)")
	flag.Parse()

	initDirs()
	initLibraryDirs()
	InitI18n() // Инициализируем локализацию
	detectMonitorOutputs()
	backendName = detectWallpaperBackend()

	if *listBackends {
		printBackends()
		return
	}
	loadHistory()
	go pruneCache()

	// Отключаем Vulkan, чтобы NVIDIA-драйвер перестал спамить в консоль
	os.Setenv("GSK_RENDERER", "gl")

	app := gtk.NewApplication("org.wallgtk.app", 0)
	app.ConnectActivate(func() {
		setupCSS()
		wallApp := NewApp(app)
		wallApp.Show()
	})
	// Аргументы уже разобраны flag; GTK передаём только имя программы.
	app.Run(os.Args[:1])
}

func setupCSS() {
	css := `
		.tile-l { min-width: 384px; min-height: 216px; }
		.tile-p { min-width: 192px; min-height: 288px; }
		.tile-clip { border-radius: 8px; background-color: rgba(30, 30, 40, 0.5); }
		.heart-btn { background: rgba(20,20,30,0.65); color: #ff4d4d; border-radius: 50%; min-width: 34px; min-height: 34px; font-size: 18px; padding: 0; border: none; }
		.heart-btn:hover { background: rgba(255,80,80,0.9); color: white; }
		.res-label { background: rgba(0,0,0,0.6); color: white; border-radius: 8px; padding: 2px 8px; font-size: 11px; }
		.loading { opacity: 0.45; }
		flowboxchild { padding: 0; margin: 0; border: none; }
		picture { border-radius: 8px; }
		.floating-switcher {
			background: transparent;
			border: none;
			padding: 0;
			box-shadow: none;
		}
		.status-toast {
			background: rgba(14, 14, 20, 0.82);
			color: white;
			border-radius: 999px;
			padding: 6px 12px;
			box-shadow: 0 8px 20px rgba(0, 0, 0, 0.24);
		}

		.zoom-bg { background-color: rgba(10, 10, 15, 0.95); }
		.tags-overlay { background: rgba(0,0,0,0.7); border-radius: 12px; padding: 8px 14px; }
		.tag-title { color: #ff6b6b; font-weight: bold; font-size: 14px; margin-right: 12px; }
		.tag-lbl { background: rgba(50,50,70,0.8); color: white; border-radius: 8px; padding: 4px 10px; font-size: 13px; }
	`
	provider := gtk.NewCSSProvider()
	provider.LoadFromString(css)
	gtk.StyleContextAddProviderForDisplay(
		gdk.DisplayGetDefault(),
		provider,
		gtk.STYLE_PROVIDER_PRIORITY_APPLICATION,
	)
}

func printBackends() {
	if backendName == "" {
		fmt.Println("Активный бэкенд: не выбран — обои применить не получится.")
	} else {
		fmt.Println("Активный бэкенд: " + backendName)
	}

	available := availableBackends()
	if len(available) == 0 {
		fmt.Println("Автоопределение: ничего подходящего не найдено.")
	} else {
		fmt.Println("Автоопределение (в порядке приоритета): " + strings.Join(available, ", "))
	}
	fmt.Println()
	fmt.Println("Все поддерживаемые:")
	for _, b := range wallpaperBackends {
		perOutput := "один фон на все мониторы"
		if b.PerOutput {
			perOutput = "разные обои по мониторам"
		}
		fmt.Printf("  %-12s %s\n", b.Name, perOutput)
	}
	fmt.Println()
	fmt.Println("Выбрать вручную: wallgtk -backend <имя> или WALLGTK_BACKEND=<имя>")
}
