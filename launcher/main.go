package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

//go:embed web
var webFiles embed.FS

func main() {
	store, err := LoadStore()
	if err != nil {
		fatal("Could not open the Void Launcher data folder:\n%v", err)
	}

	mux := http.NewServeMux()
	registerAPI(mux, store)

	web, err := fs.Sub(webFiles, "web")
	if err != nil {
		fatal("%v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(web)))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fatal("Could not start the launcher server:\n%v", err)
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/", ln.Addr().(*net.TCPAddr).Port)

	srv := &http.Server{Handler: logRequests(mux)}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Println(err)
		}
	}()

	fmt.Println("Void Launcher running at", url)
	runWindow(url)
}

func logRequests(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func fatal(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Println(msg)
	showError(msg)
	os.Exit(1)
}
