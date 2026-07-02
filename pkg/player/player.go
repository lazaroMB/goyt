package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Event represents an event or response from mpv.
type Event struct {
	Event  string      `json:"event"`
	Name   string      `json:"name"`
	Data   interface{} `json:"data"`
	Error  string      `json:"error"`
	Reason string      `json:"reason"`
}

// Command represents a JSON IPC command sent to mpv.
type Command struct {
	Command []interface{} `json:"command"`
}

// Player controls a headless mpv process.
type Player struct {
	cmd     *exec.Cmd
	ipcPath string
	conn    net.Conn
	mu      sync.Mutex
	running bool

	// Channels for communication
	eventsChan chan Event
	closeChan  chan struct{}
}

// NewPlayer creates a new Player.
func NewPlayer() *Player {
	// Generate a unique socket file in /tmp
	ipcPath := filepath.Join(os.TempDir(), fmt.Sprintf("goyt-mpv-%d.sock", time.Now().UnixNano()))
	return &Player{
		ipcPath:    ipcPath,
		eventsChan: make(chan Event, 100),
		closeChan:  make(chan struct{}),
	}
}

// Start spawns the mpv process and connects to its IPC socket.
func (p *Player) Start() error {
	p.mu.Lock()

	// Ensure mpv exists in the PATH
	if _, err := exec.LookPath("mpv"); err != nil {
		p.mu.Unlock()
		return fmt.Errorf("mpv executable not found in PATH: %w", err)
	}

	// Prepare arguments
	args := []string{
		"--no-video",
		"--idle=yes",
		"--really-quiet",
		"--msg-level=all=error",
		fmt.Sprintf("--input-ipc-server=%s", p.ipcPath),
	}

	p.cmd = exec.Command("mpv", args...)

	// Start the process
	if err := p.cmd.Start(); err != nil {
		p.mu.Unlock()
		return fmt.Errorf("failed to start mpv process: %w", err)
	}
	p.running = true
	p.mu.Unlock()

	// Wait for the IPC socket to be created (retry connection a few times)
	var conn net.Conn
	var err error
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		conn, err = net.Dial("unix", p.ipcPath)
		if err == nil {
			break
		}
	}
	if err != nil {
		p.Stop()
		return fmt.Errorf("failed to connect to mpv IPC socket: %w", err)
	}

	p.mu.Lock()
	p.conn = conn
	p.mu.Unlock()

	// Start listener goroutine
	go p.listenEvents()

	// Observe volume and time position changes
	p.observeProperty("time-pos")
	p.observeProperty("duration")
	p.observeProperty("pause")
	p.observeProperty("volume")

	return nil
}

// Stop terminates the mpv process and cleans up.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}
	p.running = false

	close(p.closeChan)

	if p.conn != nil {
		p.conn.Close()
	}

	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
		p.cmd.Wait()
	}

	// Remove the temporary socket file
	if _, err := os.Stat(p.ipcPath); err == nil {
		os.Remove(p.ipcPath)
	}
}

// LoadFile loads a media file or stream URL.
func (p *Player) LoadFile(url string) error {
	return p.sendCommand("loadfile", url)
}

// SetPause toggles pause state.
func (p *Player) SetPause(paused bool) error {
	return p.sendCommand("set_property", "pause", paused)
}

// SetVolume sets the player volume (0 to 100).
func (p *Player) SetVolume(volume int) error {
	return p.sendCommand("set_property", "volume", volume)
}

// Seek seeks to a relative offset in seconds.
func (p *Player) Seek(seconds float64) error {
	return p.sendCommand("seek", seconds, "relative")
}

// Events returns the read-only events channel.
func (p *Player) Events() <-chan Event {
	return p.eventsChan
}

// sendCommand writes a command to the IPC socket.
func (p *Player) sendCommand(args ...interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return fmt.Errorf("not connected to mpv IPC")
	}

	cmd := Command{Command: args}
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	_, err = p.conn.Write(append(data, '\n'))
	return err
}

func (p *Player) observeProperty(prop string) {
	// command: ["observe_property", id, property_name]
	// we just use a generic ID for observation or can let it be mapped
	_ = p.sendCommand("observe_property", 1, prop)
}

// listenEvents reads lines from the IPC connection.
func (p *Player) listenEvents() {
	scanner := bufio.NewScanner(p.conn)
	for scanner.Scan() {
		line := scanner.Bytes()
		var ev Event
		if err := json.Unmarshal(line, &ev); err == nil {
			select {
			case p.eventsChan <- ev:
			default:
				// Buffer full, drop event
			}
		}
	}
}
