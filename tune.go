package tune

//go:generate go run channels_gen.go

import (
	"bufio"
	"errors"
	"log"
	"os/exec"
	"strings"
	"sync"
)

// Core represents the central manager of application activity.
type Core struct {
	Config Config
	Events chan Event
	mu     sync.RWMutex
	cmd    *exec.Cmd
}

// Config represents the core configuration parameters.
type Config struct {
	Addr      string
	ListenKey string
	PublicDir string
}

// Channel represents a channel on a station.
type Channel struct {
	Name     string
	Playlist string
}

// Event represents a track change event.
type Event struct {
	Station string `json:"station,omitempty"`
	Channel string `json:"channel,omitempty"`
	Track   string `json:"track,omitempty"`
}

// ErrNotFound represents a channel not found error.
var ErrNotFound = errors.New("tune: channel not found")

// titlePrefix represents the mpv title prefix.
const titlePrefix = " icy-title: "

// NewCore returns a new *Core.
func NewCore(config Config) (*Core, error) {
	c := &Core{
		Config: config,
		Events: make(chan Event),
	}
	return c, nil
}

// Play stops the current channel, if any, and plays
// the channel identified by id.
func (c *Core) Play(station string, id int) error {
	s, ok := Channels[station]
	if !ok {
		return ErrNotFound
	}
	channel, ok := s[id]
	if !ok {
		return ErrNotFound
	}
	err := c.Stop()
	if err != nil {
		return err
	}
	cmd := exec.Command("mpv", "--no-video", c.channelURL(channel))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	err = cmd.Start()
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.cmd = cmd
	c.mu.Unlock()
	go func(channel *Channel) {
		var line string
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line = scanner.Text()
			if !strings.HasPrefix(line, titlePrefix) {
				continue
			}
			c.Events <- Event{
				Station: station,
				Channel: channel.Name,
				Track:   line[len(titlePrefix):],
			}
		}
		err = scanner.Err()
		if err != nil {
			log.Println(err)
		}
	}(channel)
	go func() {
		err := cmd.Wait()
		if err != nil {
			_, ok := err.(*exec.ExitError)
			if !ok {
				log.Println(err)
			}
		}
	}()
	return nil
}

// Stop stops the current channel, if any.
func (c *Core) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cmd == nil {
		return nil
	}
	err := c.cmd.Process.Kill()
	c.cmd = nil
	return err
}

func (c *Core) channelURL(channel *Channel) string {
	return channel.Playlist + "?" + c.Config.ListenKey
}
