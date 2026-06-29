package main

import (
	"context"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/charmbracelet/fang"

	"github.com/AhmedAburady/rcm-go/internal/cmd"
	"github.com/AhmedAburady/rcm-go/internal/ssh"
)

func resolveVersion() string {
	if cmd.Version != "dev" {
		return cmd.Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func main() {
	// Tear down pooled SSH connections on an interrupt or SIGTERM (systemd,
	// `docker stop`, `kill`). A second signal still hard-kills the process.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		ssh.CloseAll()
		os.Exit(130)
	}()

	os.Exit(run())
}

// run executes the CLI and returns the process exit code. It is split from main
// so cleanup runs on every path — os.Exit does not honor deferred functions.
func run() int {
	defer ssh.CloseAll()
	// Resolve once so both fang's --version flag and the `version` subcommand
	// report the same value (the tag for go-install/release builds via BuildInfo).
	cmd.Version = resolveVersion()
	if err := fang.Execute(
		context.Background(),
		cmd.Root(),
		fang.WithVersion(cmd.Version),
	); err != nil {
		return 1
	}
	return 0
}
