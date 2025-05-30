// cmd/server/main.go - Server entry point
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourusername/jprq-clone/internal/server"
	"github.com/yourusername/jprq-clone/internal/shared"
)

func main() {
	// Config yuklash
	config, err := shared.LoadConfigFromEnv()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Server yaratish
	srv := server.NewServer(config)

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("🛑 Shutting down server...")
		if err := srv.Stop(); err != nil {
			log.Printf("Error stopping server: %v", err)
		}
		os.Exit(0)
	}()

	// Server boshlash
	log.Fatal(srv.Start())
}
