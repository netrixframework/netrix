package testlib

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/netrixframework/netrix/config"
	"github.com/netrixframework/netrix/log"
)

func TestTestingServer(t *testing.T) {
	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, os.Interrupt, syscall.SIGTERM)

	srv, err := NewTestingServer(&config.Config{
		APIServerAddr: "192.168.0.2:7074",
	}, nil, []*TestCase{NewTestCase("dummy", 2*time.Second, NewStateMachine(), NewFilterSet())})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}

	go func() {
		oscall := <-termCh
		log.Info(fmt.Sprintf("Received syscall: %s", oscall.String()))
		srv.Stop()
	}()
	srv.Start()
}
