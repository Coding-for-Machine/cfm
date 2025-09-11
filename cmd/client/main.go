// cmd/client/main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/Coding-for-Machine/cfm/internal/client"
	"github.com/sirupsen/logrus"
)

func main() {
	// Command line flags
	var (
		serverURL = flag.String("server", "ws://localhost:8080/tunnel", "Server URL")
		localPort = flag.Int("port", 0, "Local port to expose")
		// subdomain  = flag.String("subdomain", "", "Custom subdomain (if supported)")
		authToken = flag.String("token", "", "Authentication token")
		logLevel  = flag.String("log", "info", "Log level (debug, info, warn, error)")
		// configFile = flag.String("config", "", "Config file path")
	)
	flag.Parse()

	// Check if port is provided as argument
	if *localPort == 0 {
		if len(flag.Args()) > 0 {
			if p, err := strconv.Atoi(flag.Args()[0]); err == nil {
				*localPort = p
			}
		}
	}

	if *localPort == 0 {
		fmt.Println("🚀 JPRQ Clone - Client")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  jprq-client [options] <local-port>")
		fmt.Println("  jprq-client -port=3000")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -server   Server URL (default: ws://localhost:8080/tunnel)")
		fmt.Println("  -port     Local port to expose")
		fmt.Println("  -token    Authentication token")
		fmt.Println("  -log      Log level (debug, info, warn, error)")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  jprq-client 3000")
		fmt.Println("  jprq-client -server=ws://tunnel.example.com/tunnel 8080")
		os.Exit(1)
	}

	// Setup logging
	logger := logrus.New()
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Create tunnel client
	tunnelClient := client.NewTunnelClient(*serverURL, *localPort, *authToken, logger)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Connect to server
	logger.Infof("🔗 Connecting to server: %s", *serverURL)
	logger.Infof("📡 Exposing localhost:%d", *localPort)

	if err := tunnelClient.Connect(); err != nil {
		log.Fatalf("❌ Connection failed: %v", err)
	}

	// Wait for shutdown signal
	<-sigChan
	logger.Info("🛑 Shutting down...")

	tunnelClient.Disconnect()
	logger.Info("👋 Goodbye!")
}
