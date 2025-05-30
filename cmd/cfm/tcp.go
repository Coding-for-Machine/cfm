// cmd/tcp.go
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Coding-for-Machine/cfm/pkg/client"
	"github.com/spf13/cobra"
)

var tcpCmd = &cobra.Command{
	Use:   "tcp [local-port]",
	Short: "Create a TCP tunnel",
	Long:  `Create a TCP tunnel to expose your local TCP service to the internet`,
	Args:  cobra.ExactArgs(1),
	Run:   runTCPTunnel,
}

var (
	tcpRemotePort int
	tcpSubdomain  string
)

func init() {
	rootCmd.AddCommand(tcpCmd)

	tcpCmd.Flags().IntVarP(&tcpRemotePort, "remote-port", "r", 0, "Remote port (random if not specified)")
	tcpCmd.Flags().StringVarP(&tcpSubdomain, "subdomain", "s", "", "Custom subdomain")
}

func runTCPTunnel(cmd *cobra.Command, args []string) {
	localPort := args[0]

	config := &client.Config{
		ServerURL:  serverURL,
		LocalPort:  localPort,
		Protocol:   "tcp",
		RemotePort: tcpRemotePort,
		Subdomain:  tcpSubdomain,
		Token:      getAuthToken(),
	}

	client, err := client.NewClient(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create client: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\n🛑 Shutting down tunnel...")
		cancel()
	}()

	fmt.Printf("🚀 Starting TCP tunnel: localhost:%s -> tcp://tunnel.jprq.io\n", localPort)

	if err := client.StartTCPTunnel(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Tunnel failed: %v\n", err)
		os.Exit(1)
	}
}
