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

	configPath := flag.String("config", "serveg.toml", "path to config file")
	flag.Parse()

	conf, err := server.ParseOrCreateConfig(*configPath)
	if err != nil {
		logger.Error("can't parse config: ", err)
		return 1
	}

	srv := server.Server{Conf: conf}
	if err := srv.StartGoroutine(); err != nil {
		logger.Error(err)
		return 1
	}

	// got SIGINT/SIGTERM
	<-ctx.Done()
	stop() // restore default behaviour: a second Ctrl+C kills immediately
	logger.Info("shutting down")
	// waits for server stop
	srv.Stop()

	return 0
}
