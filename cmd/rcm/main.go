package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/AhmedAburady/rcm-go/internal/cmd"
	"github.com/AhmedAburady/rcm-go/internal/ssh"
)

func main() {
	// Handle Ctrl+C and other termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		ssh.CloseAll()
		os.Exit(0)
	}()

	// Clean up SSH connections when app exits normally
	defer ssh.CloseAll()

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
