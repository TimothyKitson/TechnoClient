package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	silent := flag.Bool("silent", false, "install without a window")
	minecraftDir := flag.String("minecraft-dir", "", "override the .minecraft folder")
	gameDir := flag.String("game-dir", "", "override the Void Client game folder")
	logPath := flag.String("log", "", "write progress to this file")
	flag.Parse()

	p := defaultPlan()
	if *minecraftDir != "" {
		p.MinecraftDir = *minecraftDir
	}
	if *gameDir != "" {
		p.GameDir = *gameDir
	}

	if !*silent {
		runWindow(p)
		return
	}

	out := os.Stdout
	if *logPath != "" {
		file, err := os.Create(*logPath)
		if err != nil {
			os.Exit(2)
		}
		defer file.Close()
		out = file
	}
	fmt.Fprintf(out, "RAM %d MB, heap %d MB\n", p.TotalRAMMB, heapMB(p.TotalRAMMB))
	err := install(p, func(fraction float64, status string) {
		fmt.Fprintf(out, "%3.0f%% %s\n", fraction*100, status)
	})
	if err != nil {
		fmt.Fprintln(out, "ERROR", err)
		if cause := errors.Unwrap(err); cause != nil {
			fmt.Fprintln(out, "CAUSE", cause)
		}
		out.Close()
		os.Exit(1)
	}
	fmt.Fprintln(out, "DONE")
}
