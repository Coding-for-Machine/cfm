// pkg/tunnel/tcp.go
package tunnel

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/Coding-for-Machine/cfm/pkg/protocol"
	"github.com/fasthttp/websocket"
)

type TCPTunnel struct {
	ID         string
	LocalPort  int
	RemotePort int
	conn       *websocket.Conn
	listener   net.Listener
	mu         sync.RWMutex
	closed     bool
}

func NewTCPTunnel(id string, localPort, remotePort int, conn *websocket.Conn) *TCPTunnel {
	return &TCPTunnel{
		ID:         id,
		LocalPort:  localPort,
		RemotePort: remotePort,
		conn:       conn,
	}
}

func (t *TCPTunnel) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", t.RemotePort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", t.RemotePort, err)
	}

	t.listener = listener
	fmt.Printf("🚀 TCP Tunnel active: localhost:%d -> %s:%d\n",
		t.LocalPort, "tunnel.jprq.io", t.RemotePort)

	go t.handleConnections(ctx)
	return nil
}

func (t *TCPTunnel) handleConnections(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := t.listener.Accept()
			if err != nil {
				if !t.isClosed() {
					fmt.Printf("Accept error: %v\n", err)
				}
				return
			}

			go t.handleConnection(conn)
		}
	}
}

func (t *TCPTunnel) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Create connection request
	connReq := &protocol.ConnectionRequest{
		TunnelID:   t.ID,
		Protocol:   "tcp",
		LocalPort:  t.LocalPort,
		RemoteAddr: conn.RemoteAddr().String(),
	}

	// Send to server
	if err := t.conn.WriteJSON(connReq); err != nil {
		fmt.Printf("Failed to send connection request: %v\n", err)
		return
	}

	// Proxy data bidirectionally
	go t.proxyData(conn)
}

func (t *TCPTunnel) proxyData(conn net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	// Local -> Remote
	go func() {
		defer wg.Done()
		buffer := make([]byte, 32*1024)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				if err != io.EOF {
					fmt.Printf("Read error: %v\n", err)
				}
				break
			}

			data := &protocol.TunnelData{
				TunnelID: t.ID,
				Data:     buffer[:n],
				Type:     "tcp_data",
			}

			if err := t.conn.WriteJSON(data); err != nil {
				fmt.Printf("Write to tunnel error: %v\n", err)
				break
			}
		}
	}()

	// Remote -> Local (handled via WebSocket messages)
	go func() {
		defer wg.Done()
		// This will be handled by message receiver
	}()

	wg.Wait()
}

func (t *TCPTunnel) isClosed() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.closed
}

func (t *TCPTunnel) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true
	if t.listener != nil {
		return t.listener.Close()
	}
	return nil
}
