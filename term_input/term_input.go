// Package term_input provides routines for reading stdin without needing
// newlines as well as catching the interrupt signal.
package term_input

import (
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
)

var (
	inputQueue chan byte
	interrupt  atomic.Bool
	signals    chan os.Signal
)

func Begin() {
	inputQueue = make(chan byte)
	go func() {
		// This just runs until the process terminates.
		for {
			input := make([]byte, 1)
			os.Stdin.Read(input)
			inputQueue <- input[0]
		}
	}()
	signals = make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range signals {
			interrupt.Store(true)
		}
	}()
}

func Stop() {
	signal.Stop(signals)
	close(signals)
	close(inputQueue)
}

// Read returns one byte of input. If an interrupt signal was received 0
// is returned.
func Read() byte {
	if interrupt.Swap(false) {
		return 0
	}
	return <-inputQueue
}
