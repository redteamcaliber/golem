package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/tkerber/golem/golem"
)

// exitCode contains the exit code that golem should exit with.
var exitCode = 0

// handleSignals handles os signals for golem.
func handleSignals(g *golem.Golem) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		<-c
		exitCode = 1
		g.Close()
	}()
}