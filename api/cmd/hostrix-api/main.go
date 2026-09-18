package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hostrix/hostrix/api/internal/config"
	"github.com/hostrix/hostrix/api/internal/database"
	"github.com/hostrix/hostrix/api/internal/httpapi"
	"github.com/hostrix/hostrix/api/internal/nodes"
	"github.com/hostrix/hostrix/api/internal/users"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err := database.SeedTemplates(db, cfg.TemplatesDir); err != nil {
		log.Fatalf("seed templates: %v", err)
	}

	if err := users.EnsureBootstrapAdmin(db, cfg); err != nil {
		log.Fatalf("bootstrap admin: %v", err)
	}

	server := httpapi.NewServer(cfg, db)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := nodes.MarkStaleOffline(db, 45*time.Second); err != nil {
				log.Printf("mark stale nodes: %v", err)
			}
		}
	}()

	go func() {
		log.Printf("hostrix-api listening on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("http: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down")
	_ = server.Close()
}
