package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-autostart"
	"github.com/getlantern/systray"
)

//go:embed icons/notification/notify-*.png
var iconFs embed.FS

//go:embed extension
var extensionFs embed.FS
var appAutostart *autostart.App

type SingletonPayload struct {
	StartTimestamp int    `json:"start_timestamp"`
	PID            int    `json:"pid"`
	ExecPath       string `json:"exec_path"`
}

type AppFlags struct {
	register   bool
	unregister bool
	check      bool
	verbose    bool
}

type AppConfig struct {
	port                  int
	embeddedExtensionHash string
	currentExtensionHash  string
	currentExtensionPath  string
	execPath              string
	flags                 AppFlags
	processInfo           SingletonPayload
}

var config AppConfig

// printv acts like fmt.Printf but respects the verbose flag
func printv(format string, a ...interface{}) {
	if config.flags.verbose {
		fmt.Printf(format, a...)
	}
}

func init() {

	execPath, err := os.Executable()
	if err == nil {
		if resolvedPath, err := filepath.EvalSymlinks(execPath); err == nil {
			execPath = resolvedPath
		}
		if absPath, err := filepath.Abs(execPath); err == nil {
			execPath = absPath
		}
	} else {
		fmt.Printf("Error getting executable path: %v\n", err)
		os.Exit(1)
	}

	config = AppConfig{
		port:                 63845,
		execPath:             execPath,
		currentExtensionPath: getLocalExtensionPath(),
	}

	config.embeddedExtensionHash = getEmbeddedExtensionHash()
	config.currentExtensionHash = calculateLocalExtensionHash()

	config.processInfo = SingletonPayload{
		StartTimestamp: int(time.Now().Unix()),
		PID:            os.Getpid(),
		ExecPath:       execPath,
	}

	flag.BoolVar(&config.flags.register, "register", false, "Register application to run at startup")
	flag.BoolVar(&config.flags.unregister, "unregister", false, "Remove application from startup")
	flag.BoolVar(&config.flags.check, "check", false, "Check if application is registered at startup")
	flag.BoolVar(&config.flags.verbose, "v", false, "Enable verbose logging")
	flag.BoolVar(&config.flags.verbose, "verbose", false, "Enable verbose logging")
	flag.Parse()

	// 4. Autostart Setup
	appAutostart = &autostart.App{
		Name:        "WhatsAppTrayDaemon",
		DisplayName: "WhatsApp Tray Monitor",
		Exec:        []string{execPath},
	}
}

func main() {

	handleFlags()
	enforceSingleton()

	printv("[Init] Daemon starting up (PID: %d)...\n", config.processInfo.PID)

	if config.embeddedExtensionHash == config.currentExtensionHash && config.embeddedExtensionHash != "" {
		printv("[Init] Extension checksums match (%s). Integrity OK.\n", config.embeddedExtensionHash[:8])
	} else {
		printv("[Init] Extension missing or mismatched. Rebuilding...\n")
		registerExtension()
	}

	go startWebServer()
	systray.Run(onReady, onExit)
}

func handleFlags() {
	if config.flags.register {
		registerAsAutostart()
		os.Exit(0)
	}

	if config.flags.unregister {
		unregisterAsAutostart()
		os.Exit(0)
	}

	if config.flags.check {
		isRegisteredAsAutostart()
		os.Exit(0)
	}
}

func enforceSingleton() {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/ping/check_singleton", config.port))

	// If connection refused, port is free.
	if err != nil {
		printv("[Singleton] Port is free. Binding application.\n")
		return
	}
	defer resp.Body.Close()

	var existing SingletonPayload
	err = json.NewDecoder(resp.Body).Decode(&existing)
	if err != nil {
		fmt.Printf("Port %d is occupied by an unknown service. Exiting...\n", config.port)
		os.Exit(1)
	}

	fmt.Printf("Application is already running since %s with PID %d.\n",
		time.Unix(int64(existing.StartTimestamp), 0).Format(time.RFC1123),
		existing.PID,
	)

	if existing.ExecPath != config.execPath {
		fmt.Printf("\n[WARNING] Path mismatch detected!\n")
		fmt.Printf("Running instance path : %s\n", existing.ExecPath)
		fmt.Printf("Current invoked path  : %s\n", config.execPath)
	}

	os.Exit(1)
}

func isRegisteredAsAutostart() bool {
	if appAutostart.IsEnabled() {
		fmt.Println("Status: Registered (Enabled)")
		return true
	} else {
		fmt.Println("Status: Not Registered (Disabled)")
		return false
	}
}

func registerAsAutostart() {
	if err := appAutostart.Enable(); err != nil {
		fmt.Printf("Failed to register startup app: %v\n", err)
	}
	fmt.Println("Successfully registered to launch at startup!")

	// Force an extension registration during the --register process as well
	registerExtension()
}

func unregisterAsAutostart() {
	if err := appAutostart.Disable(); err != nil {
		fmt.Printf("Failed to unregister startup app: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Successfully removed from startup apps.")
}

func getEmbeddedExtensionHash() string {
	h := sha256.New()
	fs.WalkDir(extensionFs, "extension", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			data, _ := extensionFs.ReadFile(path)
			h.Write(data)
		}
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))
}

func getLocalExtensionPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".local", "share", "whatsapp-tray-daemon", "extension")
}

func calculateLocalExtensionHash() string {
	h := sha256.New()
	err := fs.WalkDir(extensionFs, "extension", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			relPath, _ := filepath.Rel("extension", path)
			localFile := filepath.Join(config.currentExtensionPath, relPath)
			data, readErr := os.ReadFile(localFile)
			if readErr != nil {
				return readErr
			}
			h.Write(data)
		}
		return nil
	})
	if err != nil {
		return "" // Triggers a mismatch
	}
	return hex.EncodeToString(h.Sum(nil))
}

func registerExtension() {
	printv("[Action] Rebuilding local extension directory...\n")
	os.RemoveAll(config.currentExtensionPath)

	err := fs.WalkDir(extensionFs, "extension", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel("extension", path)
		targetPath := filepath.Join(config.currentExtensionPath, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

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

	printv("[Action] Injecting flags into browsers...\n")
	injectBrowserFlags(config.currentExtensionPath)
}

func injectBrowserFlags(extensionDirAbs string) {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config")
	knownBrowsers := []string{"chromium", "chrome", "vivaldi", "brave", "edge"}

	targetLine := fmt.Sprintf("--load-extension=%s", extensionDirAbs)

	for _, browser := range knownBrowsers {
		confFile := filepath.Join(configDir, fmt.Sprintf("%s-flags.conf", browser))

		var content []byte
		if _, err := os.Stat(confFile); err == nil {
			content, _ = os.ReadFile(confFile)
		}

		if !strings.Contains(string(content), targetLine) {
			f, err := os.OpenFile(confFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString(targetLine + "\n")
				f.Close()
				printv("  -> Updated %s\n", confFile)
			}
		}
	}
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
		printv("[Event] WhatsApp unread count changed to: %d\n", count)
		fmt.Fprintf(w, "[Daemon] Count set to %d", count)
	})

	http.HandleFunc("/ping/check_singleton", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		respBytes, _ := json.Marshal(config.processInfo)
		w.Write(respBytes)
	})

	printv("[Daemon] Listening for WhatsApp triggers on :%d\n", config.port)
	http.ListenAndServe(fmt.Sprintf(":%d", config.port), nil)
}

func SetIcon(count int) {
	if count > 9 {
		count = 10
	}
	imgPath := filepath.Join("icons/notification", fmt.Sprintf("notify-%d.png", count))
	iconBytes, err := iconFs.ReadFile(imgPath)
	if err == nil {
		systray.SetIcon(iconBytes)
	}
}

func onReady() {
	SetIcon(0)
	systray.SetTitle("WhatsApp Tray")
}

func onExit() {
	printv("[Shutdown] Closing daemon gracefully.\n")
}
