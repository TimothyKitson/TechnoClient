//go:build !windows

package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func totalRAMMB() uint64 {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			kb, _ := strconv.ParseUint(fields[1], 10, 64)
			return kb / 1024
		}
	}
	return 0
}

func launcherRunning() bool {
	return false
}
