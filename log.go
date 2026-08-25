package main

import (
	"fmt"
	"os"
)

// verbose включается флагом -v или переменной окружения WALLGTK_DEBUG.
var verbose = os.Getenv("WALLGTK_DEBUG") != ""

func logf(format string, args ...any) {
	if !verbose {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
