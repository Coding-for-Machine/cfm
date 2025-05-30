// cmd/jprq/commands.go - CLI Command implementations
package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/yourusername/jprq-clone/internal/client"
	"github.com/yourusername/jprq-clone/internal/shared"
)

func runHTTPTunnel(cmd *cobra.Command, args []string) error {
	port, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid port: %s", args[0])
	}

	// Config yukash
	config, err := loadConfig()
	if err != nil {
		return err
	}

	// Token olish
	authToken, err := getAuthToken()
	if err != nil {
		return err
	}

	// Subdomain generate qilish
	if subdomain == "" {
		subdomain = shared.GenerateRandomString(8)
	}

	// HTTP tunnel client yaratish
	client := client.NewHTTPTunnelClient(config, port, subdomain, authToken)

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\n⏹️  Shutting down...")
		client.Stop()
		os.Exit(0)
	}()

	// Tunnel boshlash
	return client.Start()
}

func runTCPTunnel(cmd *cobra.Command, args []string) error {
	// TCP tunnel implementation (keyingi darsda)
	return fmt.Errorf("TCP tunnel not implemented yet")
}

func setAuthToken(cmd *cobra.Command, args []string) error {
	token := args[0]

	// Token'ni saqlash (file yoki environment variable)
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	jprqDir := filepath.Join(configDir, "jprq")
	if err := os.MkdirAll(jprqDir, 0755); err != nil {
		return err
	}

	tokenFile := filepath.Join(jprqDir, "token")
	if err := os.WriteFile(tokenFile, []byte(token), 0600); err != nil {
		return err
	}

	fmt.Println("✅ Authentication token saved successfully!")
	return nil
}

func showStatus(cmd *cobra.Command, args []string) error {
	// Status check implementation
	fmt.Println("🔍 Checking tunnel status...")
	fmt.Println("⚠️  Status check not implemented yet")
	return nil
}

func loadConfig() (*shared.Config, error) {
	if configFile != "" {
		return shared.LoadConfig(configFile)
	}

	// Default config
	return &shared.Config{
		Client: shared.ClientConfig{
			ServerHost: "localhost:8080",
			Timeout:    30,
		},
	}, nil
}

func getAuthToken() (string, error) {
	// CLI argument'dan
	if token != "" {
		return token, nil
	}

	// Environment variable'dan
	if envToken := os.Getenv("JPRQ_TOKEN"); envToken != "" {
		return envToken, nil
	}

	// File'dan
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	tokenFile := filepath.Join(configDir, "jprq", "token")
	if data, err := os.ReadFile(tokenFile); err == nil {
		return string(data), nil
	}

	return "", fmt.Errorf("authentication token not found. Use 'jprq auth <token>' or set JPRQ_TOKEN environment variable")
}
