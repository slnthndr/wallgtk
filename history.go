package main

import (
	"encoding/json"
	"os"
	"time"
)

const maxHistoryItems = 60

func loadHistory() {
	historyMu.Lock()
	defer historyMu.Unlock()

	if historyFile == "" {
		return
	}
	data, err := os.ReadFile(historyFile)
	if err != nil {
		return
	}
	var items []HistoryEntry
	if json.Unmarshal(data, &items) == nil {
		historyItems = items
	}
}

func saveHistory() {
	historyMu.RLock()
	data, err := json.MarshalIndent(historyItems, "", "  ")
	historyMu.RUnlock()
	if err != nil {
		return
	}
	if historyFile == "" {
		return
	}
	if err := writeFileAtomic(historyFile, data, 0644); err != nil {
		logf("[HISTORY] save failed: %v", err)
	}
}

func recordHistory(wp Wallpaper, monitor string) {
	historyMu.Lock()
	historyItems = append([]HistoryEntry{{
		Wallpaper: wp,
		Monitor:   monitor,
		AppliedAt: time.Now(),
	}}, historyItems...)
	if len(historyItems) > maxHistoryItems {
		historyItems = historyItems[:maxHistoryItems]
	}
	historyMu.Unlock()
	go saveHistory()
}

func listHistory() []HistoryEntry {
	historyMu.RLock()
	defer historyMu.RUnlock()

	items := make([]HistoryEntry, len(historyItems))
	copy(items, historyItems)
	return items
}

func clearHistory() {
	historyMu.Lock()
	historyItems = nil
	historyMu.Unlock()
	go saveHistory()
}
