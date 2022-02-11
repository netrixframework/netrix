package util

import (
	"os"
	"os/signal"
	"syscall"
)

// Term returns a channel which receives a message when there is an interrupt
func Term() chan os.Signal {
	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, os.Interrupt, syscall.SIGTERM)
	return termCh
}
