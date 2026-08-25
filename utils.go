package main

import (
	"io"
	"os"
	"path/filepath"
)

// copyFile пишет во временный файл и переименовывает его, чтобы наблюдатель
// никогда не увидел наполовину скопированную картинку.
func copyFile(src, dst string) bool {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return false
	}
	in, err := os.Open(src)
	if err != nil {
		return false
	}
	defer in.Close()

	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return false
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(tmp)
		return false
	}
	if os.Rename(tmp, dst) != nil {
		os.Remove(tmp)
		return false
	}
	return true
}

// writeFileAtomic сохраняет служебный JSON без риска обрезать его при падении.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
