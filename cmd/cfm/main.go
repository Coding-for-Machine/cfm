// cmd/jprq/main.go - CLI Entry Point
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configFile string
	token      string
	subdomain  string
	verbose    bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "jprq",
		Short: "JPRQ Clone - Expose local servers to the internet",
		Long: `JPRQ Clone is a tunneling service that allows you to expose 
your local servers to the internet securely.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file path")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "authentication token")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose output")

	// Commands
	rootCmd.AddCommand(httpCommand())
	rootCmd.AddCommand(tcpCommand())
	rootCmd.AddCommand(authCommand())
	rootCmd.AddCommand(statusCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func httpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "http [port]",
		Short: "Expose HTTP server",
		Long:  "Expose your local HTTP server to the internet",
		Args:  cobra.ExactArgs(1),
		RunE:  runHTTPTunnel,
	}

	cmd.Flags().StringVar(&subdomain, "subdomain", "", "custom subdomain")
	return cmd
}

func tcpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tcp [port]",
		Short: "Expose TCP server",
		Long:  "Expose your local TCP server to the internet",
		Args:  cobra.ExactArgs(1),
		RunE:  runTCPTunnel,
	}

	cmd.Flags().StringVar(&subdomain, "subdomain", "", "custom subdomain")
	return cmd
}

func authCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "auth [token]",
		Short: "Set authentication token",
		Args:  cobra.ExactArgs(1),
		RunE:  setAuthToken,
	}
}

func statusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show tunnel status",
		RunE:  showStatus,
	}
}
