package utils

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Spinner initializes the progress indicator.
type Spinner struct {
	mu         *sync.RWMutex
	delay      time.Duration
	writer     io.Writer
	message    string
	lastOutput string
	hideCursor bool
	stopChan   chan struct{}
}

// NewSpinner instantiates a new progress indicator.
func NewSpinner(msg string, d time.Duration, hideCursor bool) *Spinner {
	return &Spinner{
		mu:         &sync.RWMutex{},
		delay:      d,
		writer:     os.Stderr,
		message:    msg,
		hideCursor: hideCursor,
		stopChan:   make(chan struct{}, 1),
	}
}

// Start starts the progress indicator.
func (s *Spinner) Start() {
	if s.hideCursor && runtime.GOOS != "windows" {
		// hides the cursor
		fmt.Fprintf(s.writer, "\033[?25l")
	}

	go func() {
		for {
			for _, r := range `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` {
				select {
				case <-s.stopChan:
					return
				default:
					s.mu.Lock()
					s.erase(s.lastOutput) // clear the last line

					output := fmt.Sprintf("\r%s%s %c%s", s.message, SuccessColor, r, DefaultColor)
					fmt.Fprintf(s.writer, output)
					s.lastOutput = output

					s.mu.Unlock()
					time.Sleep(s.delay)
				}
			}
		}
	}()
}

// Stop stops the progress indicator.
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hideCursor && runtime.GOOS != "windows" {
		// makes the cursor visible
		fmt.Fprint(s.writer, "\033[?25h")
	}
	s.erase(s.lastOutput)
	s.stopChan <- struct{}{}
}

// erase deletes the last terminal line.
// Caller must hold the the locker.
func (s *Spinner) erase(lastOutput string) {
	n := utf8.RuneCountInString(lastOutput)
	if runtime.GOOS == "windows" {
		clearString := "\r" + strings.Repeat(" ", n) + "\r"
		fmt.Fprint(s.writer, clearString)
		return
	}
	for _, c := range []string{"\b", "\127", "\b", "\033[K"} { // "\033[K" for macOS Terminal
		fmt.Fprint(s.writer, strings.Repeat(c, n))
	}
	fmt.Fprintf(s.writer, "\r\033[K") // erases to end of line
}
