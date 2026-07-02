# GoYT — YouTube Music CLI Player in Go

GoYT is a fast, keyboard-driven Terminal User Interface (TUI) music player for YouTube Music, written in Go. It is inspired by `involvex/youtube-music-cli` but compiled natively into a single lightweight binary, removing Node.js/Bun runtime overhead and skipping custom extensions or track downloads.

## Features
* **Sleek TUI Layout:** Built using Charm's Bubble Tea, Lip Gloss, and Bubbles libraries.
* **Efficient Player Control:** Offloads streaming playback to a background `mpv` daemon and communicates via JSON IPC.
* **Streaming-Only Queue:** Built-in play queues with track skip, volume control, and shuffle.
* **Low Memory Footprint:** Runs in ~15MB of RAM (compared to 100MB+ for Electron or TS wrappers).

---

## Pre-requisites

GoYT offloads video stream resolution and playback to **`mpv`** and **`yt-dlp`**. You must have both installed and present in your system's `PATH`.

### Linux (Ubuntu/Debian)
```bash
sudo apt update
sudo apt install mpv yt-dlp
```

### Linux (Arch)
```bash
sudo pacman -S mpv yt-dlp
```

### macOS (Homebrew)
```bash
brew install mpv yt-dlp
```

---

## Installation & Running

### 1. Build from Source
From the project workspace root, build the executable:
```bash
go build -o goyt ./cmd/goyt/...
```

### 2. Run the App
```bash
./goyt
```

---

## Keyboard Controls

| Key | Action |
| :--- | :--- |
| `Tab` | Toggle focus between Sidebar navigation and active workspace |
| `Up` / `Down` (or `j` / `k`) | Navigate sidebar tabs, search results, or the play queue |
| `Enter` | Open sidebar tab, focus search bar, open a playlist, or play a track |
| `a` | Add the selected track to the queue, or add ALL tracks in a playlist to the queue |
| `Esc` / `Backspace` | Blur search box, or go back to the playlist list from the playlist details |
| `Space` | Toggle Play / Pause |
| `n` | Skip to Next track |
| `p` | Skip to Previous track |
| `[` | Decrease volume (by 5%) |
| `]` | Increase volume (by 5%) |
| `Left` / `Right` | Seek backward / forward 10 seconds |
| `q` or `Ctrl+C` | Quit player |

---

## Project Structure
* [cmd/goyt/main.go](file:///home/lazerko/Projects/goyt/cmd/goyt/main.go): Core entry point, initializing wrappers, spawning `mpv`, and executing the Bubble Tea UI loops.
* [pkg/player/player.go](file:///home/lazerko/Projects/goyt/pkg/player/player.go): Headless `mpv` manager and socket-based JSON IPC driver.
* [pkg/ytmusic/ytmusic.go](file:///home/lazerko/Projects/goyt/pkg/ytmusic/ytmusic.go): API service using `wslyyy/youtube-go` to perform searches, query suggestions, and parse nested InnerTube responses.
* [pkg/queue/queue.go](file:///home/lazerko/Projects/goyt/pkg/queue/queue.go): Queue controller supporting next/prev navigation, index seeking, and random shuffling.
* [pkg/tui/tui.go](file:///home/lazerko/Projects/goyt/pkg/tui/tui.go): TUI styling and input handler managing layout blocks and reactive events.
