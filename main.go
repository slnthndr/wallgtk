package main

import (
	"os"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func main() {
	initDirs()

	// Отключаем Vulkan, чтобы NVIDIA-драйвер перестал спамить в консоль
	os.Setenv("GSK_RENDERER", "gl")

	app := gtk.NewApplication("org.wallgtk.app", 0)
	app.ConnectActivate(func() {
		setupCSS()
		wallApp := NewApp(app)
		wallApp.Show()
	})
	app.Run(os.Args)
}

func setupCSS() {
	css := `
		.tile-l { min-width: 384px; min-height: 216px; }
		.tile-p { min-width: 198px; min-height: 352px; }
		.tile-clip { border-radius: 8px; background-color: rgba(30, 30, 40, 0.5); } /* ВАЖНО для правильного кропа! */
		.heart-btn { background: rgba(20,20,30,0.65); color: #ff4d4d; border-radius: 50%; min-width: 34px; min-height: 34px; font-size: 18px; padding: 0; border: none; }
		.heart-btn:hover { background: rgba(255,80,80,0.9); color: white; }
		.res-label { background: rgba(0,0,0,0.6); color: white; border-radius: 8px; padding: 2px 8px; font-size: 11px; }
		.loading { opacity: 0.45; }
		flowboxchild { padding: 0; margin: 0; border: none; }
		picture { border-radius: 8px; }

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

