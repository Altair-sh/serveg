package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"serveg/internal/log"
	"serveg/internal/server"
	"syscall"
)

func main() {
	os.Exit(run())
}

// main with int return value
func run() int {
	defer log.Defer()
	log.StartGoroutine()
	logger := log.LoggerContext{Name: "main"}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	configPath := *flag.String("config", "serveg.toml", "path to config file")
	flag.Parse()

	conf, err := server.ParseOrCreateConfig(configPath)
	if err != nil {
		logger.Error(err)
		return 1
	}

	errCh := make(chan error, 1)
	go func() { errCh <- server.Serve(conf) }()
	logger.Info("server started")

	select {
	// server failed to start
	case err := <-errCh:
		logger.Error(err)
		return 1
	// got SIGINT/SIGTERM
	case <-ctx.Done():
		stop() // restore default behaviour: a second Ctrl+C kills immediately
		logger.Info("shutting down")
	}

	return 0
}
