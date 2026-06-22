Here is a complete, production-ready README file. It has been heavily structured to be welcoming and easy to understand for everyday users, while moving the heavy technical details into a dedicated "For Developers" section at the bottom.

---

# WhatsApp Tray Watcher

A lightweight, privacy-respecting system tray utility that brings native unread message badges back to your WhatsApp Web app.

If you use WhatsApp Web as an app (PWA) on Linux (like on Hyprland, GNOME, or KDE), you've likely noticed that closing the window means you lose track of your unread messages. **WhatsApp Tray Watcher** fixes this. It places a clean, real-time unread counter directly in your system tray, so you always know when you have a message waiting, even if your browser is hidden.

---

## Features

* **Real-time Tray Icon:** Instantly updates to show your exact unread message count (1-9, or 9+).
* **Fully Automated Setup:** One command registers the app to start when your computer boots and automatically configures your browser.
* **Zero Jargon, Zero Hassle:** No need to manually install browser extensions or move image files around. Everything is bundled inside one single file.
* **Smart Auto-Healing:** If the background extension accidentally gets deleted, the app detects it and rebuilds itself instantly on the next boot.
* **Lightweight:** Uses virtually zero CPU and RAM. It only wakes up when your unread count actually changes.

---

## Privacy & Security First

Your privacy is our absolute highest priority. This application is designed to be **100% local and offline**.

* **No Server Connections:** This app does **not** connect to the internet, and we do not have servers.
* **No Chat Reading:** The extension is physically incapable of reading your messages. It strictly looks at the text in the tab title (e.g., `(3) WhatsApp`) to extract the number.
* **No Account Access:** We do not need your passwords, QR codes, or session tokens.
* **Open Source:** You can read every single line of code to verify these claims.

---

## How to Use It (For Normal Users)

### 1. Installation & Setup

You do not need to install anything complicated.

1. Download the `whatsapp-tray-daemon` file.
2. Open your terminal in the folder where you downloaded it.
3. Run this single command to install it:
```bash
./whatsapp-tray-daemon --register

```

**That's it!** The app is now running in your system tray, and it has configured itself to launch automatically every time you turn on your computer.

### 2. Supported Browsers

The auto-setup currently works out-of-the-box for:

* Google Chrome
* Chromium
* Brave
* Vivaldi
* Microsoft Edge

### ⚠️ What to Expect: The Browser Warning

When you open your browser after installing this app, you might see a small popup in the corner saying: **"Disable developer mode extensions."**

**This is completely normal and safe.** Because this app installs the WhatsApp extension locally on your machine (rather than making you download it from the Google Web Store), Chromium browsers show this default security warning.

* **What to do:** Simply click the "X" to close the warning. You do not need to disable anything.

### 3. Uninstalling

If you ever want to remove the app and stop it from starting with your computer, simply run:

```bash
./whatsapp-tray-daemon --unregister

```

---

## 💻 For Developers: Technical Architecture

Under the hood, WhatsApp Tray Watcher is a heavily optimized, self-contained Go binary communicating with a locally injected Chromium extension via a Singleton HTTP instance.

### Architecture Overview

1. **The Event Trigger (Chromium Extension):** A strictly scoped content script (`web.whatsapp.com/*`) uses a `MutationObserver` to watch the DOM title. When the title changes, it pushes the parsed integer to a local HTTP endpoint.
2. **The Tray Daemon (Go):** A native Go application utilizing `systray` to bind to XDG AppIndicators. It runs a lightweight, zero-allocation HTTP listener to receive updates.

### Technical Highlights

* **`//go:embed` Asset Bundling:** The 11 tray icons and the entire browser extension source code are compiled directly into the binary. There are no external asset folders to manage or lose.
* **Self-Healing Checksums (SHA256):** On boot, the daemon hashes the embedded extension and compares it against the local `~/.local/share/whatsapp-tray-daemon/extension` directory. If the local folder is missing or modified, the daemon instantly rebuilds it from memory.
* **Singleton HTTP Lock:** To prevent ghost processes, the daemon checks for an existing instance on boot by pinging a dedicated `/ping` route. It validates a specific `SingletonPayload` (verifying PID and ExecPath) to ensure port collisions don't crash the system.
* **Automated Browser Flag Injection:** The `--register` flag dynamically locates XDG config directories and injects `--load-extension=/path/to/extension` directly into standard browser `.conf` files (e.g., `~/.config/brave-flags.conf`).

### CLI Commands

```bash
# Register to OS Autostart and inject browser flags
./whatsapp-tray-daemon --register

# Remove from OS Autostart and exit
./whatsapp-tray-daemon --unregister

# Check current autostart status
./whatsapp-tray-daemon --check

# Run with verbose logging for debugging
./whatsapp-tray-daemon -v

```

### Building from Source

Prerequisites: Go 1.16+ (for `//go:embed` support).

```bash
# Clone the repository
git clone https://github.com/Utsav-56/whatsapp-tray-icon.git.git
cd whatsapp-tray-icon

# Build the standalone binary
go build -o whatsapp-tray-daemon

# Run with verbose logging to test
./whatsapp-tray-daemon -v

```