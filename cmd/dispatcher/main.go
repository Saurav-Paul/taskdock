// The taskdock dispatcher: a host-side daemon that turns "assign a ticket
// to claude" into a running Claude Code session in a tmux window.
//
// Build:  go build -o bin/dispatcher ./cmd/dispatcher
// Run:    ./bin/dispatcher -config dispatcher.json
// Pause:  touch ~/.taskdock-dispatcher-pause
package main

import (
	"flag"
	"log"

	"github.com/Saurav-Paul/taskdock/internal/dispatcher"
)

func main() {
	configPath := flag.String("config", "dispatcher.json", "path to dispatcher config")
	dryRun := flag.Bool("dry-run", false, "log launch decisions without starting sessions")
	flag.Parse()

	cfg, err := dispatcher.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	d := dispatcher.New(cfg, *dryRun)
	log.Fatal(d.Run())
}
