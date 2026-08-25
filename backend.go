package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type detectedMonitor struct {
	Name     string
	Primary  bool
	Portrait bool
}

var (
	backendOnce   sync.Once
	swwwBin       string
	swwwDaemonBin string
)

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// swwwBinaries ищет swww или его форк awww (Omarchy). CLI у них совместимый,
// поэтому достаточно запомнить, какой из них установлен.
func swwwBinaries() (string, string, bool) {
	backendOnce.Do(func() {
		for _, name := range []string{"swww", "awww"} {
			if commandExists(name) && commandExists(name+"-daemon") {
				swwwBin, swwwDaemonBin = name, name+"-daemon"
				return
			}
		}
	})
	return swwwBin, swwwDaemonBin, swwwBin != ""
}

// detectWallpaperBackend выбирает бэкенд один раз и запоминает его.
func detectWallpaperBackend() string {
	activeBackend = selectWallpaperBackend()
	if activeBackend == nil {
		return ""
	}
	return activeBackend.Name
}

// ensureSwwwDaemon поднимает демон, только если он ещё не отвечает, и ждёт его
// готовности вместо фиксированной паузы.
func ensureSwwwDaemon(bin, daemon string) {
	if exec.Command(bin, "query").Run() == nil {
		return
	}
	cmd := exec.Command(daemon)
	if err := cmd.Start(); err != nil {
		logf("[BACKEND] cannot start %s: %v", daemon, err)
		return
	}
	// Демон переживает нас; освобождаем ресурсы процесса, чтобы не копить зомби.
	go cmd.Wait()

	for i := 0; i < 20; i++ {
		if exec.Command(bin, "query").Run() == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	logf("[BACKEND] %s did not become ready", daemon)
}

func setWallpaperPair(paths map[string]string) bool {
	if activeBackend == nil {
		backendName = detectWallpaperBackend()
	}
	if activeBackend == nil {
		return false
	}

	targets := buildTargets(paths)
	if len(targets) == 0 {
		return false
	}

	for _, backend := range candidateBackends(targets) {
		use := targets
		// Бэкенды без поддержки отдельных выходов ставят один фон на всё;
		// берём первую цель в стабильном порядке, а не случайную.
		if !backend.PerOutput && len(use) > 1 {
			logf("[BACKEND] %s cannot address monitors separately, applying %s only",
				backend.Name, use[0].Key)
			use = use[:1]
		}
		if err := backend.Apply(use); err != nil {
			logf("[BACKEND] %s failed: %v", backend.Name, err)
			continue
		}
		if backend != activeBackend {
			logf("[BACKEND] applied via fallback %s", backend.Name)
		}
		return true
	}
	return false
}

// candidateBackends перечисляет бэкенды для конкретного запроса в порядке
// предпочтения: активный первым, остальные подошедшие — следом, как запасные
// на случай ошибки. Единственное исключение — запрос на разные обои по
// мониторам, когда активный бэкенд раздельные выходы не умеет: тогда вперёд
// выходит первый умеющий, иначе часть мониторов молча осталась бы без обоев.
// Так omarchy остаётся основным, а swww/awww подхватывает и per-output, и сбои.
func candidateBackends(targets []wallpaperTarget) []*wallpaperBackend {
	var ordered []*wallpaperBackend
	add := func(b *wallpaperBackend) {
		for _, seen := range ordered {
			if seen == b {
				return
			}
		}
		ordered = append(ordered, b)
	}

	if len(targets) > 1 && !activeBackend.PerOutput {
		for _, b := range wallpaperBackends {
			if b.PerOutput && b.Detect() {
				add(b)
				break
			}
		}
	}
	add(activeBackend)
	for _, b := range wallpaperBackends {
		if b.Detect() {
			add(b)
		}
	}
	return ordered
}

func buildTargets(paths map[string]string) []wallpaperTarget {
	targets := make([]wallpaperTarget, 0, len(paths))
	for _, key := range sortedKeys(paths) {
		// Для mon_all имени выхода нет — бэкенды трактуют это как «все мониторы».
		targets = append(targets, wallpaperTarget{
			Key:    key,
			Output: MonitorOutputs[key],
			Path:   paths[key],
		})
	}
	return targets
}

// sortedKeys даёт стабильный порядок обхода: mon_pri раньше mon_sec,
// остальные — по алфавиту.
func sortedKeys(paths map[string]string) []string {
	keys := make([]string, 0, len(paths))
	for k := range paths {
		keys = append(keys, k)
	}
	rank := func(key string) int {
		switch key {
		case "mon_all":
			return 0
		case "mon_pri":
			return 1
		case "mon_sec":
			return 2
		default:
			return 3
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if ri, rj := rank(keys[i]), rank(keys[j]); ri != rj {
			return ri < rj
		}
		return keys[i] < keys[j]
	})
	return keys
}

func detectMonitorOutputs() {
	if outputs, ok := detectHyprctlMonitorOutputs(); ok {
		assignMonitorOutputs(outputs)
		return
	}
	if outputs, ok := detectSwayMonitorOutputs(); ok {
		assignMonitorOutputs(outputs)
		return
	}
	if outputs, ok := detectXrandrMonitorOutputs(); ok {
		assignMonitorOutputs(outputs)
		return
	}
	if outputs, ok := detectSwwwMonitorOutputs(); ok {
		assignMonitorOutputs(outputs)
		return
	}
	assignMonitorOutputs(nil)
}

func assignMonitorOutputs(monitors []detectedMonitor) {
	MonitorOutputs = make(map[string]string)
	MonitorEntries = []MonitorEntry{{Key: "mon_all"}}
	if len(monitors) == 0 {
		return
	}

	primaryIndex := 0
	for i, monitor := range monitors {
		if monitor.Primary {
			primaryIndex = i
			break
		}
	}

	primary := monitors[primaryIndex]
	MonitorOutputs["mon_pri"] = primary.Name
	MonitorEntries = append(MonitorEntries, MonitorEntry{
		Key:      "mon_pri",
		Output:   primary.Name,
		Portrait: primary.Portrait,
	})

	secondaryAdded := false
	for i, monitor := range monitors {
		if i == primaryIndex {
			continue
		}
		if !secondaryAdded {
			MonitorOutputs["mon_sec"] = monitor.Name
			MonitorEntries = append(MonitorEntries, MonitorEntry{
				Key:      "mon_sec",
				Output:   monitor.Name,
				Portrait: monitor.Portrait,
			})
			secondaryAdded = true
			continue
		}
		MonitorOutputs["output:"+monitor.Name] = monitor.Name
		MonitorEntries = append(MonitorEntries, MonitorEntry{
			Key:      "output:" + monitor.Name,
			Output:   monitor.Name,
			Portrait: monitor.Portrait,
		})
	}
}

func detectHyprctlMonitorOutputs() ([]detectedMonitor, bool) {
	if !commandExists("hyprctl") {
		return nil, false
	}
	out, err := exec.Command("hyprctl", "monitors", "-j").Output()
	if err != nil {
		return nil, false
	}

	var monitors []struct {
		Name    string `json:"name"`
		Focused bool   `json:"focused"`
		Width   int    `json:"width"`
		Height  int    `json:"height"`
	}
	if err := json.Unmarshal(out, &monitors); err != nil {
		return nil, false
	}

	var ordered []detectedMonitor
	for _, mon := range monitors {
		if mon.Name == "" {
			continue
		}
		ordered = append(ordered, detectedMonitor{
			Name:     mon.Name,
			Primary:  mon.Focused,
			Portrait: mon.Height > mon.Width,
		})
	}
	return ordered, len(ordered) > 0
}

func detectSwayMonitorOutputs() ([]detectedMonitor, bool) {
	if !detectSway() {
		return nil, false
	}
	out, err := exec.Command("swaymsg", "-t", "get_outputs", "-r").Output()
	if err != nil {
		return nil, false
	}
	monitors := parseSwayOutputs(out)
	return monitors, len(monitors) > 0
}

func parseSwayOutputs(data []byte) []detectedMonitor {
	var outputs []struct {
		Name    string `json:"name"`
		Active  bool   `json:"active"`
		Focused bool   `json:"focused"`
		Rect    struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"rect"`
	}
	if err := json.Unmarshal(data, &outputs); err != nil {
		return nil
	}

	var monitors []detectedMonitor
	for _, o := range outputs {
		if o.Name == "" || !o.Active {
			continue
		}
		monitors = append(monitors, detectedMonitor{
			Name:     o.Name,
			Primary:  o.Focused,
			Portrait: o.Rect.Height > o.Rect.Width,
		})
	}
	return monitors
}

func detectSwwwMonitorOutputs() ([]detectedMonitor, bool) {
	bin, _, ok := swwwBinaries()
	if !ok {
		return nil, false
	}
	out, err := exec.Command(bin, "query").Output()
	if err != nil {
		return nil, false
	}
	monitors := parseSwwwQuery(string(out))
	return monitors, len(monitors) > 0
}

func parseSwwwQuery(out string) []detectedMonitor {
	var names []detectedMonitor
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			continue
		}
		name := strings.TrimSuffix(fields[0], ":")
		if name == "" {
			continue
		}
		names = append(names, detectedMonitor{Name: name, Primary: len(names) == 0})
	}
	return names
}

func detectXrandrMonitorOutputs() ([]detectedMonitor, bool) {
	if !commandExists("xrandr") {
		return nil, false
	}
	out, err := exec.Command("xrandr", "--query").Output()
	if err != nil {
		return nil, false
	}
	monitors := parseXrandrQuery(string(out))
	return monitors, len(monitors) > 0
}

func parseXrandrQuery(out string) []detectedMonitor {
	var monitors []detectedMonitor
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, " connected") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		monitor := detectedMonitor{
			Name:    fields[0],
			Primary: strings.Contains(line, " connected primary "),
		}
		for _, field := range fields[1:] {
			// Ищем геометрию вида 1920x1080+0+0.
			if !strings.Contains(field, "x") || !strings.Contains(field, "+") {
				continue
			}
			parts := strings.SplitN(strings.SplitN(field, "+", 2)[0], "x", 2)
			if len(parts) != 2 {
				continue
			}
			w, wErr := strconv.Atoi(parts[0])
			h, hErr := strconv.Atoi(parts[1])
			if wErr == nil && hErr == nil {
				monitor.Portrait = h > w
			}
			break
		}
		monitors = append(monitors, monitor)
	}
	return monitors
}

func monitorMenuLabel(entry MonitorEntry) string {
	switch entry.Key {
	case "mon_all":
		return Tr("mon_all")
	case "mon_pri":
		return fmt.Sprintf("%s (%s)", Tr("mon_pri"), entry.Output)
	case "mon_sec":
		return fmt.Sprintf("%s (%s)", Tr("mon_sec"), entry.Output)
	default:
		index := 3
		for _, monitor := range MonitorEntries {
			if strings.HasPrefix(monitor.Key, "output:") {
				if monitor.Key == entry.Key {
					break
				}
				index++
			}
		}
		if CurrentLang == LangRU {
			return fmt.Sprintf("Монитор %d (%s)", index, entry.Output)
		}
		return fmt.Sprintf("Monitor %d (%s)", index, entry.Output)
	}
}

func monitorShortLabel(entry MonitorEntry) string {
	switch entry.Key {
	case "mon_all":
		return Tr("mon_all")
	case "mon_pri":
		if CurrentLang == LangRU {
			return "Основной"
		}
		return "Primary"
	case "mon_sec":
		if CurrentLang == LangRU {
			return "Второй"
		}
		return "Secondary"
	default:
		index := 3
		for _, monitor := range MonitorEntries {
			if strings.HasPrefix(monitor.Key, "output:") {
				if monitor.Key == entry.Key {
					break
				}
				index++
			}
		}
		if CurrentLang == LangRU {
			return fmt.Sprintf("Монитор %d", index)
		}
		return fmt.Sprintf("Monitor %d", index)
	}
}
