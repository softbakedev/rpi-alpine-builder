package internal

import (
	"fmt"
	"sync"
	"time"
)

type Spinner struct {
	mu           sync.Mutex
	running      bool
	wg           sync.WaitGroup
	index        int
	interval     time.Duration
	spinnerChars []rune
}

// NewSpinner initializes a Spinner with default values.
func NewSpinner() *Spinner {
	return &Spinner{
		interval:     150 * time.Millisecond,
		spinnerChars: []rune{'|', '/', '-', '\\'},
		index:        0,
	}
}

// Start spins up a goroutine to animate the spinner.
// If it's already running, Start() does nothing.
func (s *Spinner) Start() {
	s.mu.Lock()
	// If it's already running, don't start another goroutine.
	if s.running {
		s.mu.Unlock()
		return
	}
	// Mark the spinner as running
	s.running = true
	s.mu.Unlock()

	// Add one worker to the wait group
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			s.mu.Lock()
			// Check if we should keep running
			if !s.running {
				s.mu.Unlock()
				return
			}
			// Print one frame of the spinner
			fmt.Printf("\r%c", s.spinnerChars[s.index])
			s.index = (s.index + 1) % len(s.spinnerChars)
			s.mu.Unlock()

			// Sleep to control the speed of the spinner
			time.Sleep(s.interval)
		}
	}()
}

// Stop tells the spinner goroutine to stop and waits for it to finish.
// It also clears the spinner from the console.
func (s *Spinner) Stop() {
	s.mu.Lock()
	// If it's not running, nothing to do
	if !s.running {
		s.mu.Unlock()
		return
	}
	// Mark the spinner as not running
	s.running = false
	s.mu.Unlock()

	// Wait until the spinner goroutine exits
	s.wg.Wait()

	// Clear the console line where the spinner was shown
	fmt.Printf("\r\033[K")
}
