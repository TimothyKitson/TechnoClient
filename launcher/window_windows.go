package main

import webview2 "github.com/jchv/go-webview2"

func runWindow(url string) {
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  "Void Launcher",
			Width:  1180,
			Height: 820,
			IconId: 1,
			Center: true,
		},
	})
	if w == nil {
		openInApp(url)
		select {}
	}
	defer w.Destroy()
	w.SetSize(940, 600, webview2.HintMin)
	w.Navigate(url)
	w.Run()
}
