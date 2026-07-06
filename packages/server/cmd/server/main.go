// Command server is the ai-reviewer backend: the single process that owns all
// state and is consumed by every client (TUI, SPA, Telegram bot).
package main

import (
	"log"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/config"
)

func main() {
	cfg := config.Load()

	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("ai-reviewer: %v", err)
	}
	defer db.Close()

	log.Printf("ai-reviewer: database ready at %s (migrations applied)", cfg.DBPath)
}
