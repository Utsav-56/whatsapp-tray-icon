package main

import (
	"embed"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/getlantern/systray"
)

//go:embed icons/notification/notify-*.png
var iconFs embed.FS
var iconDir string = "icons/notification"

func SetIcon(count int) {
	if count > 9 {
		count = 10
	}

	imgPath := filepath.Join(iconDir, fmt.Sprintf("notify-%d.png", count))
	iconBytes, err := iconFs.ReadFile(imgPath)
	if err == nil {
		systray.SetIcon(iconBytes)
	}
}

func main() {
	go startWebServer()
	systray.Run(onReady, onExit)
}

func startWebServer() {
	http.HandleFunc("/set_count", func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Private-Network", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		fmt.Println("Received request:", r.URL.String())

		countStr := r.URL.Query().Get("count")
		count, err := strconv.Atoi(countStr)
		if err != nil {
			http.Error(w, "Invalid count", http.StatusUnprocessableEntity)
			return
		}
		SetIcon(count)

		println("Count updated to", count)
		fmt.Fprintf(w, "[Daemon] Count set to %d", count)
	})

	println("Starting server in")
	http.ListenAndServe(":63845", nil)
}

func onReady() {
	SetIcon(0)
	systray.SetTitle("Whatsapp tray")
	systray.SetTooltip("This is a tooltip")
}

func onExit() {
	println("Exitting")
}



// 




