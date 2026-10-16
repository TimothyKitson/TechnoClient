//go:build !windows

package main

func runWindow(url string) {
	openInApp(url)
	select {}
}
