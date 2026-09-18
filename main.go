package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/app"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
)

func main() {
	configFile := flag.String("c", "", "Path to config file")
	flag.Parse()

	var (
		cfg *config.Config
		err error
	)
	if *configFile != "" {
		log.Printf("Reading config file: %s", *configFile)
		cfg, err = config.ReadFile(*configFile)
	} else {
		log.Printf("Config file not set, parsing config from environment")
		cfg, err = config.ParseEnv()
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Enabled endpoints: %s", strings.Join(cfg.Endpoints.Enabled(), ", "))
	log.Printf("Authorization method set to: %s", cfg.Auth.Method)
	log.Printf("Starting hetzner-dnsapi-proxy, listening on %s", cfg.ListenAddr)
	if err := runServer(cfg.ListenAddr, app.New(cfg)); err != nil {
		log.Fatalf("Error running server: %v", err)
	}
}

func runServer(listenAddr string, handler http.Handler) error {
	const (
		readHeaderTimeout = 10
		readTimeout       = 30
		writeTimeout      = 30
		idleTimeout       = 120
		shutdownTimeout   = 5
	)

	s := &http.Server{
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout * time.Second,
		ReadTimeout:       readTimeout * time.Second,
		WriteTimeout:      writeTimeout * time.Second,
		IdleTimeout:       idleTimeout * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	log.Printf("Shutting down hetzner-dnsapi-proxy: %v", context.Cause(ctx))

	c, cancel := context.WithTimeout(context.Background(), shutdownTimeout*time.Second)
	defer cancel()

	return s.Shutdown(c)
}
