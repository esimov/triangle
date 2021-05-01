package utils

import (
	"fmt"
	"time"
)

const (
	ErrorColor   = "\x1b[0;31m"
	SuccessColor = "\x1b[0;32m"
	DefaultColor = "\x1b[0m"
)

// Spinner initializes the process indicator.
type Spinner struct {
	stopChan chan struct{}
}

// NewSpinner instantiates a new Spinner struct.
func NewSpinner() *Spinner {
	return &Spinner{}
}

// Start starts the process indicator.
func (s *Spinner) Start(message string) {
	s.stopChan = make(chan struct{}, 1)

	go func() {
		for {
			for _, r := range `-\|/` {
				select {
				case <-s.stopChan:
					return
				default:
					fmt.Printf("\r%s%s %c%s", message, SuccessColor, r, DefaultColor)
					time.Sleep(time.Millisecond * 100)
				}
			}
		}
	}()
}

// Stop stops the process indicator.
func (s *Spinner) Stop() {
	s.stopChan <- struct{}{}
}
