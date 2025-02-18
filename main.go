package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
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
	if *configFile == "" {
		log.Printf("Reading config file: %s", *configFile)
		cfg, err = config.ReadFile(*configFile)
	} else {
		log.Printf("Config file not set, parsing config from environment")
		cfg, err = config.ParseEnv()
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Authorization method set to: %s", cfg.Auth.Method)

	log.Printf("Starting hetzner-dnsapi-proxy, listening on %s", cfg.ListenAddr)
	if err := runServer(cfg.ListenAddr, app.New(cfg)); err != nil {
		log.Fatal("Error running server:", err)
	}
}

func runServer(listenAddr string, handler http.Handler) error {
	const (
		readHeaderTimeout = 10
		shutdownTimeout   = 5
	)

	s := &http.Server{
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout * time.Second,
	}

	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down hetzner-dnsapi-proxy")

	c, cancel := context.WithTimeout(context.Background(), shutdownTimeout*time.Second)
	defer cancel()

	return s.Shutdown(c)
}
