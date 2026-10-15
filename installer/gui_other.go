//go:build !windows

package main

import (
	"fmt"
	"os"
)

func runWindow(p plan) {
	fmt.Println("The installer window is Windows only. Run with -silent.")
	os.Exit(2)
}
