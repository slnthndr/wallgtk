package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// wallpaperTarget описывает одно назначение: какой картинкой закрыть какой выход.
// Пустой Output означает «все мониторы».
type wallpaperTarget struct {
	Key    string
	Output string
	Path   string
}

// wallpaperBackend — способ применить обои. Detect обязан проверять не только
// наличие бинаря, но и пригодность сессии: swww бесполезен под GNOME, а
// hyprpaper — вне Hyprland.
type wallpaperBackend struct {
	Name string
	// PerOutput: бэкенд умеет разные обои на разных мониторах.
	PerOutput bool
	Detect    func() bool
	Apply     func(targets []wallpaperTarget) error
}

// Порядок важен: побеждает первый подошедший. Сверху — точные, привязанные к
// конкретному композитору/DE; снизу — универсальные запасные варианты.
var wallpaperBackends = []*wallpaperBackend{
	{Name: "swww", PerOutput: true, Detect: detectSwww, Apply: applySwww},
	{Name: "hyprpaper", PerOutput: true, Detect: detectHyprpaper, Apply: applyHyprpaper},
	{Name: "sway", PerOutput: true, Detect: detectSway, Apply: applySway},
	{Name: "plasma", Detect: detectPlasma, Apply: applyPlasma},
	{Name: "gnome", Detect: detectGnome, Apply: applyGnome},
	{Name: "cinnamon", Detect: detectCinnamon, Apply: applyCinnamon},
	{Name: "mate", Detect: detectMate, Apply: applyMate},
	{Name: "xfce", PerOutput: true, Detect: detectXfce, Apply: applyXfce},
	{Name: "xwallpaper", PerOutput: true, Detect: detectXwallpaper, Apply: applyXwallpaper},
	{Name: "feh", PerOutput: true, Detect: detectFeh, Apply: applyFeh},
	{Name: "nitrogen", PerOutput: true, Detect: detectNitrogen, Apply: applyNitrogen},
	// Последний шанс: схема GNOME есть, но окружение опознать не удалось.
	{Name: "gsettings", Detect: detectGsettingsFallback, Apply: applyGnome},
}

var activeBackend *wallpaperBackend

// selectWallpaperBackend выбирает бэкенд, учитывая ручное переопределение
// через -backend или WALLGTK_BACKEND.
func selectWallpaperBackend() *wallpaperBackend {
	if forced := strings.TrimSpace(os.Getenv("WALLGTK_BACKEND")); forced != "" && backendOverride == "" {
		backendOverride = forced
	}
	if backendOverride != "" {
		for _, b := range wallpaperBackends {
			if strings.EqualFold(b.Name, backendOverride) {
				logf("[BACKEND] forced: %s", b.Name)
				return b
			}
		}
		logf("[BACKEND] unknown backend %q, falling back to autodetection", backendOverride)
	}
	for _, b := range wallpaperBackends {
		if b.Detect() {
			logf("[BACKEND] detected: %s (per-output: %v)", b.Name, b.PerOutput)
			return b
		}
	}
	logf("[BACKEND] nothing detected")
	return nil
}

// availableBackends перечисляет всё, что подошло бы — для -list-backends.
func availableBackends() []string {
	var names []string
	for _, b := range wallpaperBackends {
		if b.Detect() {
			names = append(names, b.Name)
		}
	}
	return names
}

// --- определение сессии ---------------------------------------------------

func envSet(name string) bool {
	return strings.TrimSpace(os.Getenv(name)) != ""
}

func isWayland() bool {
	return envSet("WAYLAND_DISPLAY") || strings.EqualFold(os.Getenv("XDG_SESSION_TYPE"), "wayland")
}

// isX11 намеренно ложно под Wayland: DISPLAY там выставлен XWayland,
// и X11-утилиты обои не поменяют.
func isX11() bool {
	if isWayland() {
		return false
	}
	return envSet("DISPLAY") || strings.EqualFold(os.Getenv("XDG_SESSION_TYPE"), "x11")
}

func desktopIs(names ...string) bool {
	current := strings.ToLower(strings.Join([]string{
		os.Getenv("XDG_CURRENT_DESKTOP"),
		os.Getenv("XDG_SESSION_DESKTOP"),
		os.Getenv("DESKTOP_SESSION"),
	}, ":"))
	for _, name := range names {
		if strings.Contains(current, strings.ToLower(name)) {
			return true
		}
	}
	return false
}

var (
	schemaOnce sync.Once
	gsSchemas  map[string]bool
)

func gsettingsSchemaExists(schema string) bool {
	if !commandExists("gsettings") {
		return false
	}
	schemaOnce.Do(func() {
		gsSchemas = make(map[string]bool)
		out, err := exec.Command("gsettings", "list-schemas").Output()
		if err != nil {
			return
		}
		for _, line := range strings.Split(string(out), "\n") {
			if s := strings.TrimSpace(line); s != "" {
				gsSchemas[s] = true
			}
		}
	})
	return gsSchemas[schema]
}

// --- swww / awww ----------------------------------------------------------

func detectSwww() bool {
	if _, _, ok := swwwBinaries(); !ok {
		return false
	}
	// Mutter и KWin рисуют фон сами; layer-shell-обои под ними не видны
	// или перекрыты собственным фоном DE.
	return isWayland() && !desktopIs("gnome", "kde", "plasma")
}

func applySwww(targets []wallpaperTarget) error {
	bin, daemon, ok := swwwBinaries()
	if !ok {
		return fmt.Errorf("swww not available")
	}
	ensureSwwwDaemon(bin, daemon)

	var firstErr error
	applied := 0
	for _, t := range targets {
		args := []string{"img", t.Path, "--transition-type", "grow", "--transition-fps", "120", "--transition-duration", "1"}
		if t.Output != "" {
			args = append(args, "--outputs", t.Output)
		}
		if err := run(bin, args...); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		applied++
	}
	if applied == 0 {
		return firstErr
	}
	return nil
}

// --- hyprpaper ------------------------------------------------------------

func detectHyprpaper() bool {
	return commandExists("hyprpaper") && commandExists("hyprctl") && envSet("HYPRLAND_INSTANCE_SIGNATURE")
}

func applyHyprpaper(targets []wallpaperTarget) error {
	// В IPC hyprpaper монитор и путь разделяются запятой, экранирования нет.
	for _, t := range targets {
		if strings.Contains(t.Path, ",") {
			return fmt.Errorf("hyprpaper cannot handle a comma in %q", t.Path)
		}
	}
	if err := ensureHyprpaper(); err != nil {
		return err
	}

	seen := make(map[string]bool)
	for _, t := range targets {
		if seen[t.Path] {
			continue
		}
		seen[t.Path] = true
		if err := hyprpaperIPC("preload", t.Path); err != nil {
			return err
		}
	}
	for _, t := range targets {
		// Пустое имя монитора у hyprpaper означает «все выходы».
		if err := hyprpaperIPC("wallpaper", t.Output+","+t.Path); err != nil {
			return err
		}
	}
	// Иначе hyprpaper держит в памяти все когда-либо показанные картинки.
	if err := hyprpaperIPC("unload", "unused"); err != nil {
		logf("[BACKEND] hyprpaper unload unused: %v", err)
	}
	return nil
}

func ensureHyprpaper() error {
	if hyprpaperIPC("listloaded") == nil {
		return nil
	}
	cmd := exec.Command("hyprpaper")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot start hyprpaper: %w", err)
	}
	go cmd.Wait()

	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		if hyprpaperIPC("listloaded") == nil {
			return nil
		}
	}
	return fmt.Errorf("hyprpaper did not become ready")
}

func hyprpaperIPC(args ...string) error {
	out, err := exec.Command("hyprctl", append([]string{"hyprpaper"}, args...)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("hyprctl hyprpaper %s: %w", strings.Join(args, " "), err)
	}
	// hyprctl рапортует об ошибках нулевым кодом возврата и текстом в stdout.
	text := strings.TrimSpace(string(out))
	if strings.Contains(strings.ToLower(text), "invalid") || strings.HasPrefix(text, "Couldn't") {
		return fmt.Errorf("hyprctl hyprpaper %s: %s", strings.Join(args, " "), text)
	}
	return nil
}

// --- sway -----------------------------------------------------------------

func detectSway() bool {
	return commandExists("swaymsg") && envSet("SWAYSOCK")
}

func applySway(targets []wallpaperTarget) error {
	var firstErr error
	applied := 0
	for _, t := range targets {
		output := t.Output
		if output == "" {
			output = "*"
		}
		if err := run("swaymsg", "output", output, "bg", t.Path, "fill"); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		applied++
	}
	if applied == 0 {
		return firstErr
	}
	return nil
}

// --- KDE Plasma -----------------------------------------------------------

func detectPlasma() bool {
	return commandExists("plasma-apply-wallpaperimage") && desktopIs("kde", "plasma")
}

func applyPlasma(targets []wallpaperTarget) error {
	// plasma-apply-wallpaperimage ставит один фон на все экраны.
	abs, err := firstAbsPath(targets)
	if err != nil {
		return err
	}
	return run("plasma-apply-wallpaperimage", abs)
}

// --- семейство gsettings --------------------------------------------------

type gsettingsScheme struct {
	schema     string
	pathKey    string
	darkKey    string
	optionsKey string
	optionsVal string
	asURI      bool
}

func applyGsettings(scheme gsettingsScheme, targets []wallpaperTarget) error {
	abs, err := firstAbsPath(targets)
	if err != nil {
		return err
	}
	value := abs
	if scheme.asURI {
		value = "file://" + abs
	}
	if err := run("gsettings", "set", scheme.schema, scheme.pathKey, value); err != nil {
		return err
	}
	if scheme.darkKey != "" {
		if err := run("gsettings", "set", scheme.schema, scheme.darkKey, value); err != nil {
			logf("[BACKEND] %s %s: %v", scheme.schema, scheme.darkKey, err)
		}
	}
	if scheme.optionsKey != "" {
		if err := run("gsettings", "set", scheme.schema, scheme.optionsKey, scheme.optionsVal); err != nil {
			logf("[BACKEND] %s %s: %v", scheme.schema, scheme.optionsKey, err)
		}
	}
	return nil
}

var gnomeScheme = gsettingsScheme{
	schema:     "org.gnome.desktop.background",
	pathKey:    "picture-uri",
	darkKey:    "picture-uri-dark",
	optionsKey: "picture-options",
	optionsVal: "zoom",
	asURI:      true,
}

func detectGnome() bool {
	return gsettingsSchemaExists(gnomeScheme.schema) &&
		desktopIs("gnome", "unity", "pantheon", "budgie", "pop")
}

func applyGnome(targets []wallpaperTarget) error {
	return applyGsettings(gnomeScheme, targets)
}

func detectGsettingsFallback() bool {
	return gsettingsSchemaExists(gnomeScheme.schema)
}

var cinnamonScheme = gsettingsScheme{
	schema:     "org.cinnamon.desktop.background",
	pathKey:    "picture-uri",
	optionsKey: "picture-options",
	optionsVal: "zoom",
	asURI:      true,
}

func detectCinnamon() bool {
	return gsettingsSchemaExists(cinnamonScheme.schema) && desktopIs("cinnamon", "x-cinnamon")
}

func applyCinnamon(targets []wallpaperTarget) error {
	return applyGsettings(cinnamonScheme, targets)
}

var mateScheme = gsettingsScheme{
	schema:     "org.mate.background",
	pathKey:    "picture-filename",
	optionsKey: "picture-options",
	optionsVal: "zoom",
}

func detectMate() bool {
	return gsettingsSchemaExists(mateScheme.schema) && desktopIs("mate")
}

func applyMate(targets []wallpaperTarget) error {
	return applyGsettings(mateScheme, targets)
}

// --- XFCE -----------------------------------------------------------------

func detectXfce() bool {
	return commandExists("xfconf-query") && desktopIs("xfce")
}

func applyXfce(targets []wallpaperTarget) error {
	out, err := exec.Command("xfconf-query", "-c", "xfce4-desktop", "-l").Output()
	if err != nil {
		return fmt.Errorf("xfconf-query -l: %w", err)
	}

	var props []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, "/last-image") {
			props = append(props, line)
		}
	}
	if len(props) == 0 {
		return fmt.Errorf("no xfce4-desktop backdrop properties found")
	}

	applied := 0
	for _, t := range targets {
		for _, prop := range props {
			// Свойства выглядят как /backdrop/screen0/monitorDP-1/workspace0/last-image.
			if t.Output != "" && !strings.Contains(prop, "/monitor"+t.Output+"/") {
				continue
			}
			if err := run("xfconf-query", "-c", "xfce4-desktop", "-p", prop, "-s", t.Path); err != nil {
				logf("[BACKEND] xfce %s: %v", prop, err)
				continue
			}
			// 5 = zoomed; свойства может не быть, поэтому создаём через -n.
			style := strings.TrimSuffix(prop, "last-image") + "image-style"
			run("xfconf-query", "-c", "xfce4-desktop", "-p", style, "-n", "-t", "int", "-s", "5")
			applied++
		}
	}
	if applied == 0 {
		return fmt.Errorf("no matching xfce backdrop property")
	}
	return nil
}

// --- X11 ------------------------------------------------------------------

func detectXwallpaper() bool { return isX11() && commandExists("xwallpaper") }

func applyXwallpaper(targets []wallpaperTarget) error {
	var args []string
	for _, t := range targets {
		if t.Output != "" {
			args = append(args, "--output", t.Output)
		}
		args = append(args, "--zoom", t.Path)
	}
	if len(args) == 0 {
		return fmt.Errorf("nothing to apply")
	}
	return run("xwallpaper", args...)
}

func detectFeh() bool { return isX11() && commandExists("feh") }

func applyFeh(targets []wallpaperTarget) error {
	// feh раздаёт аргументы по экранам позиционно, имён выходов он не знает.
	paths := pathsInMonitorOrder(targets)
	if len(paths) == 0 {
		return fmt.Errorf("nothing to apply")
	}
	return run("feh", append([]string{"--no-fehbg", "--bg-fill"}, paths...)...)
}

func detectNitrogen() bool { return isX11() && commandExists("nitrogen") }

func applyNitrogen(targets []wallpaperTarget) error {
	var firstErr error
	applied := 0
	for _, t := range targets {
		args := []string{"--set-zoom-fill"}
		if t.Output != "" {
			if head := monitorIndex(t.Output); head >= 0 {
				args = append(args, fmt.Sprintf("--head=%d", head))
			}
		}
		args = append(args, t.Path, "--save")
		if err := run("nitrogen", args...); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		applied++
	}
	if applied == 0 {
		return firstErr
	}
	return nil
}

// --- общие помощники ------------------------------------------------------

func run(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		if text := strings.TrimSpace(string(out)); text != "" {
			return fmt.Errorf("%s: %w: %s", name, err, text)
		}
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func firstAbsPath(targets []wallpaperTarget) (string, error) {
	if len(targets) == 0 {
		return "", fmt.Errorf("nothing to apply")
	}
	return filepath.Abs(targets[0].Path)
}

// monitorIndex возвращает позицию выхода среди обнаруженных мониторов —
// её ждут инструменты, адресующие экраны номером, а не именем.
func monitorIndex(output string) int {
	index := 0
	for _, entry := range MonitorEntries {
		if entry.Key == "mon_all" {
			continue
		}
		if entry.Output == output {
			return index
		}
		index++
	}
	return -1
}

// pathsInMonitorOrder раскладывает пути по порядку мониторов; неуказанные
// экраны получают общую картинку, чтобы feh не оставил их чёрными.
func pathsInMonitorOrder(targets []wallpaperTarget) []string {
	byOutput := make(map[string]string)
	fallback := ""
	for _, t := range targets {
		if t.Output == "" {
			if fallback == "" {
				fallback = t.Path
			}
			continue
		}
		byOutput[t.Output] = t.Path
	}
	if fallback == "" && len(targets) > 0 {
		fallback = targets[0].Path
	}
	if len(byOutput) == 0 {
		if fallback == "" {
			return nil
		}
		return []string{fallback}
	}

	var paths []string
	for _, entry := range MonitorEntries {
		if entry.Key == "mon_all" || entry.Output == "" {
			continue
		}
		if p, ok := byOutput[entry.Output]; ok {
			paths = append(paths, p)
		} else {
			paths = append(paths, fallback)
		}
	}
	if len(paths) == 0 {
		return []string{fallback}
	}
	return paths
}
