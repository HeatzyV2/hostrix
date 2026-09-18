package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hostrix/hostrix/agent/internal/config"
	"github.com/hostrix/hostrix/agent/internal/heartbeat"
	"github.com/hostrix/hostrix/agent/internal/httpapi"
	incusmgr "github.com/hostrix/hostrix/agent/internal/incus"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	mgr := incusmgr.New(cfg.IncusSocket)
	server := httpapi.New(cfg, mgr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go heartbeat.New(cfg, mgr).Run(ctx)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("http: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	cancel()
	_ = server.Close()
}
