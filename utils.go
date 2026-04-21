package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func copyFile(src, dst string) bool {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return false
	}
	in, err := os.Open(src)
	if err != nil {
		return false
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return false
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return false
	}
	return out.Close() == nil
}

func extractPalette(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}

	type colorCount struct {
		hex   string
		count int
	}

	counts := make(map[string]int)
	bounds := img.Bounds()
	stepX := max(1, bounds.Dx()/32)
	stepY := max(1, bounds.Dy()/32)
	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r, g, b, _ := img.At(x, y).RGBA()
			hex := fmt.Sprintf("#%02x%02x%02x", uint8(r>>8)&0xF0, uint8(g>>8)&0xF0, uint8(b>>8)&0xF0)
			counts[hex]++
		}
	}

	var ordered []colorCount
	for hex, count := range counts {
		ordered = append(ordered, colorCount{hex: hex, count: count})
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].count > ordered[j].count
	})

	var palette []string
	for _, item := range ordered {
		palette = append(palette, item.hex)
		if len(palette) == 4 {
			break
		}
	}
	return palette
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
