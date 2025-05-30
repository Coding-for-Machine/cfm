// internal/client/http_tunnel.go - HTTP Tunnel Client
package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yourusername/jprq-clone/internal/shared"
	"github.com/yourusername/jprq-clone/pkg/protocol"
	"github.com/yourusername/jprq-clone/pkg/websocket"
)

type HTTPTunnelClient struct {
	config     *shared.Config
	localPort  int
	subdomain  string
	token      string
	connection *websocket.Connection
	httpClient *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewHTTPTunnelClient(cfg *shared.Config, port int, subdomain, token string) *HTTPTunnelClient {
	ctx, cancel := context.WithCancel(context.Background())

	return &HTTPTunnelClient{
		config:    cfg,
		localPort: port,
		subdomain: subdomain,
		token:     token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

func (c *HTTPTunnelClient) Start() error {
	// Server'ga WebSocket connection
	if err := c.connectToServer(); err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}

	// Tunnel register qilish
	if err := c.registerTunnel(); err != nil {
		return fmt.Errorf("failed to register tunnel: %v", err)
	}

	log.Printf("✅ HTTP tunnel started successfully!")
	log.Printf("📡 Public URL: http://%s.%s", c.subdomain, c.config.Client.ServerHost)
	log.Printf("🔗 Local server: http://localhost:%d", c.localPort)

	// Message handling
	go c.handleMessages()

	// Keep alive
	<-c.ctx.Done()
	return nil
}

func (c *HTTPTunnelClient) Stop() {
	log.Println("🛑 Stopping HTTP tunnel...")
	c.cancel()
	if c.connection != nil {
		c.connection.Close()
	}
}

func (c *HTTPTunnelClient) connectToServer() error {
	wsURL := fmt.Sprintf("ws://%s/ws?token=%s", c.config.Client.ServerHost, c.token)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}

	// WebSocket connection wrapper yaratish
	c.connection = websocket.NewConnectionFromConn(conn)
	c.connection.Start()

	return nil
}

func (c *HTTPTunnelClient) registerTunnel() error {
	registerMsg := &protocol.TunnelRegisterMessage{
		BaseMessage: protocol.BaseMessage{
			Type:      protocol.MsgTypeTunnelRegister,
			Timestamp: time.Now(),
		},
		Subdomain: c.subdomain,
		LocalPort: c.localPort,
		Protocol:  "http",
	}

	if err := c.connection.SendMessage(registerMsg); err != nil {
		return err
	}

	// Confirmation kutish
	select {
	case msg := <-c.connection.ReceiveMessage():
		if confirmMsg, ok := msg.(*protocol.TunnelConfirmMessage); ok {
			if confirmMsg.Success {
				c.subdomain = confirmMsg.Subdomain
				return nil
			}
			return fmt.Errorf("tunnel registration failed: %s", confirmMsg.Error)
		}
		return fmt.Errorf("unexpected message type")
	case <-time.After(10 * time.Second):
		return fmt.Errorf("tunnel registration timeout")
	}
}

func (c *HTTPTunnelClient) handleMessages() {
	for {
		select {
		case msg := <-c.connection.ReceiveMessage():
			c.processMessage(msg)
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *HTTPTunnelClient) processMessage(msg protocol.Message) {
	switch m := msg.(type) {
	case *protocol.HTTPRequestMessage:
		go c.handleHTTPRequest(m)
	case *protocol.HeartbeatMessage:
		c.handleHeartbeat(m)
	default:
		log.Printf("Unknown message type: %T", msg)
	}
}

func (c *HTTPTunnelClient) handleHTTPRequest(req *protocol.HTTPRequestMessage) {
	// Local server'ga request yuborish
	localURL := fmt.Sprintf("http://localhost:%d%s", c.localPort, req.Path)
	if req.Query != "" {
		localURL += "?" + req.Query
	}

	httpReq, err := http.NewRequest(req.Method, localURL, nil)
	if err != nil {
		c.sendErrorResponse(req.RequestID, 500, "Failed to create request")
		return
	}

	// Headers qo'shish
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Body qo'shish
	if len(req.Body) > 0 {
		httpReq.Body = io.NopCloser(bytes.NewReader(req.Body))
	}

	// Request yuborish
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.sendErrorResponse(req.RequestID, 502, "Local server unreachable")
		return
	}
	defer resp.Body.Close()

	// Response body o'qish
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.sendErrorResponse(req.RequestID, 500, "Failed to read response")
		return
	}

	// Headers convert qilish
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Response yuborish
	responseMsg := &protocol.HTTPResponseMessage{
		BaseMessage: protocol.BaseMessage{
			Type:      protocol.MsgTypeHTTPResponse,
			Timestamp: time.Now(),
		},
		RequestID:  req.RequestID,
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       body,
	}

	if err := c.connection.SendMessage(responseMsg); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

func (c *HTTPTunnelClient) sendErrorResponse(requestID string, statusCode int, message string) {
	errorResp := &protocol.HTTPResponseMessage{
		BaseMessage: protocol.BaseMessage{
			Type:      protocol.MsgTypeHTTPResponse,
			Timestamp: time.Now(),
		},
		RequestID:  requestID,
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       []byte(message),
	}

	c.connection.SendMessage(errorResp)
}

func (c *HTTPTunnelClient) handleHeartbeat(msg *protocol.HeartbeatMessage) {
	if msg.Ping {
		pong := &protocol.HeartbeatMessage{
			BaseMessage: protocol.BaseMessage{
				Type:      protocol.MsgTypeHeartbeat,
				Timestamp: time.Now(),
			},
			Ping: false,
		}
		c.connection.SendMessage(pong)
	}
}
