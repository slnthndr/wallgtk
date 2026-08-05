package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"path/filepath"
)

type detectedMonitor struct {
	Name     string
	Primary  bool
	Portrait bool
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func detectWallpaperBackend() string {
	func detectWallpaperBackend() string {
	if commandExists("awww") && commandExists("awww-daemon") {
		return "awww"
	}
	if commandExists("gsettings") {
		return "gnome"
	}
	return ""
	}
}

func setWallpaper(path, monitor string) bool {
	if monitor == "mon_all" {
		return setWallpaperPair(map[string]string{
			"mon_pri": path,
			"mon_sec": path,
		})
	}
	return setWallpaperPair(map[string]string{monitor: path})
}

func setWallpaperPair(paths map[string]string) bool {
	if backendName == "" {
		backendName = detectWallpaperBackend()
	}

	switch backendName {
	case "awww":
		exec.Command("awww-daemon").Start()
		time.Sleep(100 * time.Millisecond)

		ok := false
		for monitor, path := range paths {
			args := []string{"img", path, "--transition-type", "grow", "--transition-fps", "120", "--transition-duration", "1"}
			if out, exists := MonitorOutputs[monitor]; exists && out != "" {
				args = append(args, "--outputs", out)
			}
			if err := exec.Command("awww", args...).Run(); err == nil {
				ok = true
			}
		}
		return ok
		case "gnome":
		var path string
		for _, p := range paths {
			path = p
			break
		}
		if path == "" {
			return false
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return false
		}
		uri := "file://" + abs
		if err := exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", uri).Run(); err != nil {
			return false
		}
		exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", uri).Run()
		exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-options", "scaled").Run()
		return true
	default:
		return false
	}
}

func detectMonitorOutputs() {
	if outputs, ok := detectHyprctlMonitorOutputs(); ok {
		assignMonitorOutputs(outputs)
		return
	}
	if outputs, ok := detectXrandrMonitorOutputs(); ok {
		assignMonitorOutputs(outputs)
		return
	}
	if outputs, ok := detectAwwwMonitorOutputs(); ok {
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

func detectAwwwMonitorOutputs() ([]detectedMonitor, bool) {
	if !commandExists("awww") {
		return nil, false
	}
	out, err := exec.Command("awww", "query").Output()
	if err != nil {
		return nil, false
	}
	var names []detectedMonitor
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := strings.TrimSuffix(fields[0], ":")
		if name != "" {
			names = append(names, detectedMonitor{Name: name, Primary: len(names) == 0})
		}
	}
	return names, len(names) > 0
}

func detectXrandrMonitorOutputs() ([]detectedMonitor, bool) {
	if !commandExists("xrandr") {
		return nil, false
	}
	out, err := exec.Command("xrandr", "--query").Output()
	if err != nil {
		return nil, false
	}
	var monitors []detectedMonitor
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, " connected") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			monitor := detectedMonitor{
				Name:    fields[0],
				Primary: strings.Contains(line, " connected primary "),
			}
			for _, field := range fields[1:] {
				if !strings.Contains(field, "x") || !strings.Contains(field, "+") {
					continue
				}
				res := strings.SplitN(field, "+", 2)[0]
				parts := strings.SplitN(res, "x", 2)
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
	}
	return monitors, len(monitors) > 0
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
