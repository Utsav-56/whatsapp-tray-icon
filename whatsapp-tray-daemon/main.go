package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/emersion/go-autostart"
	"github.com/getlantern/systray"
)

//go:embed icons/notification/notify-*.png
var iconFs embed.FS

//go:embed extension
var extensionFs embed.FS

func SetIcon(count int) {
	if count > 9 {
		count = 10 // Assuming 10 maps to your notify-9plus.png logically
	}

	imgPath := filepath.Join("icons/notification", fmt.Sprintf("notify-%d.png", count))
	iconBytes, err := iconFs.ReadFile(imgPath)
	if err == nil {
		systray.SetIcon(iconBytes)
	}
}

func main() {
	handleAutoStart()
	registerExtension()

	go startWebServer()
	systray.Run(onReady, onExit)
}

func startWebServer() {
	http.HandleFunc("/set_count", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		countStr := r.URL.Query().Get("count")
		count, err := strconv.Atoi(countStr)
		if err != nil {
			http.Error(w, "Invalid count", http.StatusUnprocessableEntity)
			return
		}

		SetIcon(count)
		fmt.Fprintf(w, "[Daemon] Count set to %d", count)
	})

	fmt.Println("Starting local event server on :63845")
	http.ListenAndServe(":63845", nil)
}

func onReady() {
	SetIcon(0)
	systray.SetTitle("WhatsApp Tray")
}

func onExit() {}

func registerExtension() {
	homeDir, _ := os.UserHomeDir()
	extensionPath := filepath.Join(homeDir, ".local", "share", "whatsapp-tray-daemon", "extension")

	// 3xtract the embedded extension recursively using fs.WalkDir
	err := fs.WalkDir(extensionFs, "extension", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Figure out where this file should go locally
		relPath, _ := filepath.Rel("extension", path)
		targetPath := filepath.Join(extensionPath, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// Read from binary and write to disk
		data, err := extensionFs.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, 0644)
	})

	if err != nil {
		fmt.Println("Failed to extract extension:", err)
		return
	}

	fmt.Println("Extension registered at:", extensionPath)

	// register the extension in browser config files
	injectBrowserFlags(extensionPath)
}

func injectBrowserFlags(extensionDirAbs string) {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config")
	knownBrowsers := []string{"chromium", "chrome", "vivaldi", "brave", "edge"}

	targetLine := fmt.Sprintf("--load-extension=%s", extensionDirAbs)

	for _, browser := range knownBrowsers {
		confFile := filepath.Join(configDir, fmt.Sprintf("%s-flags.conf", browser))

		// Read existing content if the file exists
		var content []byte
		if _, err := os.Stat(confFile); err == nil {
			content, _ = os.ReadFile(confFile)
		}

		// If the flag isn't already there, append it
		if !strings.Contains(string(content), targetLine) {
			f, err := os.OpenFile(confFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString(targetLine + "\n")
				f.Close()
				fmt.Println("Injected flag into:", confFile)
			}
		}
	}
}

func handleAutoStart() {
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting executable path: %v\n", err)
		os.Exit(1)
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		fmt.Printf("Error resolving absolute path: %v\n", err)
		os.Exit(1)
	}

	appAutostart := &autostart.App{
		Name:        "WhatsAppTrayDaemon",
		DisplayName: "WhatsApp Tray Monitor",
		Exec:        []string{execPath},
	}

	registerFlag := flag.Bool("register", false, "Register application to run at startup")
	unregisterFlag := flag.Bool("unregister", false, "Remove application from startup")
	checkFlag := flag.Bool("check", false, "Check if application is registered at startup")

	flag.Parse()

	if *registerFlag {
		if err := appAutostart.Enable(); err != nil {
			fmt.Printf("Failed to register startup app: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Successfully registered to launch at startup!")
		os.Exit(0)
	}

	if *unregisterFlag {
		if err := appAutostart.Disable(); err != nil {
			fmt.Printf("Failed to unregister startup app: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Successfully removed from startup apps.")
		os.Exit(0)
	}

	if *checkFlag {
		if appAutostart.IsEnabled() {
			fmt.Println("Status: Registered (Enabled)")
		} else {
			fmt.Println("Status: Not Registered (Disabled)")
		}
		os.Exit(0)
	}
}
