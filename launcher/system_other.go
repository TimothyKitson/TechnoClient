//go:build !windows

package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func totalRAMMB() uint64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			kb, _ := strconv.ParseUint(fields[1], 10, 64)
			return kb / 1024
		}
	}
	return 0
}

func dataDir() string {
	if x := os.Getenv("VOID_DATA_DIR"); x != "" {
		return x
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "VoidClient")
}

var openFolder = func(path string) error {
	if err := exec.Command("xdg-open", path).Start(); err == nil {
		return nil
	}
	return exec.Command("open", path).Start()
}

func minecraftDir() string {
	if x := os.Getenv("VOID_MC_DIR"); x != "" {
		return x
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".minecraft")
}

func openInApp(url string) {
	// On non-Windows (dev), just print; a browser can be opened manually.
	if exec.Command("xdg-open", url).Start() != nil {
		// ignore
	}
}

func openMinecraftLauncher() error { return nil }

func launcherRunning() bool { return false }

func showError(msg string) {}
