package client

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Coding-for-Machine/cfm/internal/tunnel"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type TunnelClient struct {
	serverURL  string
	localPort  int
	authToken  string
	conn       *websocket.Conn
	publicURL  string
	subdomain  string
	clientID   string
	connected  bool
	logger     *logrus.Logger
	mu         sync.RWMutex
	stopChan   chan struct{}
	httpClient *http.Client
}

func NewTunnelClient(serverURL string, localPort int, authToken string, logger *logrus.Logger) *TunnelClient {
	return &TunnelClient{
		serverURL:  serverURL,
		localPort:  localPort,
		authToken:  authToken,
		logger:     logger,
		connected:  false,
		stopChan:   make(chan struct{}),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (tc *TunnelClient) Connect() error {
	u, err := url.Parse(tc.serverURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %v", err)
	}

	tc.logger.Debugf("Connecting to %s...", tc.serverURL)

	// Connect to WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("websocket connection failed: %v", err)
	}

	tc.conn = conn
	tc.logger.Info("✅ WebSocket connected")

	// Send registration
	if err := tc.register(); err != nil {
		conn.Close()
		return fmt.Errorf("registration failed: %v", err)
	}

	tc.connected = true

	// Start message handler
	go tc.handleMessages()

	// Start ping routine
	go tc.pingRoutine()

	return nil
}

func (tc *TunnelClient) register() error {
	registration := &tunnel.Registration{
		LocalPort: tc.localPort,
		AuthToken: tc.authToken,
	}

	msg := tunnel.NewRegistrationMessage(registration)

	if err := tc.conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to send registration: %v", err)
	}

	tc.logger.Debug("Registration sent, waiting for response...")

	// Wait for registration response
	var respMsg tunnel.Message
	if err := tc.conn.ReadJSON(&respMsg); err != nil {
		return fmt.Errorf("failed to read registration response: %v", err)
	}

	if respMsg.Type == tunnel.TypeError {
		if errorData, ok := respMsg.Data["error"].(map[string]interface{}); ok {
			return fmt.Errorf("server error: %s", errorData["message"])
		}
		return fmt.Errorf("unknown server error")
	}

	if respMsg.Type != tunnel.TypeRegistered {
		return fmt.Errorf("unexpected response type: %s", respMsg.Type)
	}

	// Parse registration response
	if regData, ok := respMsg.Data["registration"].(map[string]interface{}); ok {
		tc.clientID = regData["client_id"].(string)
		tc.subdomain = regData["subdomain"].(string)
		tc.publicURL = regData["public_url"].(string)

		tc.logger.Infof("🎉 Tunnel established!")
		tc.logger.Infof("🌍 Public URL: %s", tc.publicURL)
		tc.logger.Infof("📡 Forwarding: %s -> localhost:%d", tc.publicURL, tc.localPort)
		tc.logger.Info("🔄 Tunnel is active and ready to receive requests!")
	} else {
		return fmt.Errorf("invalid registration response format")
	}

	return nil
}

func (tc *TunnelClient) handleMessages() {
	defer func() {
		tc.logger.Info("Message handler stopped")
		tc.connected = false
	}()

	for {
		select {
		case <-tc.stopChan:
			return
		default:
			var msg tunnel.Message
			if err := tc.conn.ReadJSON(&msg); err != nil {
				if !tc.isClosing() {
					tc.logger.Errorf("Read error: %v", err)
				}
				return
			}

			tc.handleMessage(&msg)
		}
	}
}

func (tc *TunnelClient) handleMessage(msg *tunnel.Message) {
	switch msg.Type {
	case tunnel.TypeHTTPRequest:
		go tc.handleHTTPRequest(msg)

	case tunnel.TypePing:
		tc.handlePing(msg)

	case tunnel.TypeError:
		tc.handleError(msg)

	case tunnel.TypeDisconnect:
		tc.logger.Info("Server requested disconnect")
		tc.Disconnect()

	default:
		tc.logger.Warnf("Unknown message type: %s", msg.Type)
	}
}

func (tc *TunnelClient) handleHTTPRequest(msg *tunnel.Message) {
	// Parse HTTP request
	reqData, ok := msg.Data["request"].(map[string]interface{})
	if !ok {
		tc.logger.Error("Invalid HTTP request format")
		return
	}

	requestID := reqData["id"].(string)
	method := reqData["method"].(string)
	path := reqData["path"].(string)
	body := ""
	if b, ok := reqData["body"].(string); ok {
		body = b
	}

	tc.logger.Infof("🔄 Handling request: %s %s (ID: %s)", method, path, requestID)

	// Prepare local URL
	localURL := fmt.Sprintf("http://localhost:%d%s", tc.localPort, path)

	// Create HTTP request to local server
	req, err := http.NewRequest(method, localURL, strings.NewReader(body))
	if err != nil {
		tc.sendErrorResponse(requestID, 500, fmt.Sprintf("Failed to create request: %v", err))
		return
	}

	// Set headers
	if headersData, ok := reqData["headers"].(map[string]interface{}); ok {
		for key, value := range headersData {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	// Make request to local server
	resp, err := tc.httpClient.Do(req)
	if err != nil {
		tc.logger.Errorf("Local request failed: %v", err)
		tc.sendErrorResponse(requestID, 502, fmt.Sprintf("Local server error: %v", err))
		return
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		tc.logger.Errorf("Failed to read response body: %v", err)
		tc.sendErrorResponse(requestID, 502, "Failed to read response")
		return
	}

	// Prepare response headers
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Send response back to server
	httpResp := &tunnel.HTTPResponse{
		ID:         requestID,
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       string(respBody),
	}

	respMsg := tunnel.NewHTTPResponseMessage(httpResp)
	if err := tc.conn.WriteJSON(respMsg); err != nil {
		tc.logger.Errorf("Failed to send response: %v", err)
		return
	}

	tc.logger.Infof("✅ Response sent: %d %s (ID: %s)", resp.StatusCode, http.StatusText(resp.StatusCode), requestID)
}

func (tc *TunnelClient) sendErrorResponse(requestID string, statusCode int, message string) {
	httpResp := &tunnel.HTTPResponse{
		ID:         requestID,
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       message,
	}

	respMsg := tunnel.NewHTTPResponseMessage(httpResp)
	tc.conn.WriteJSON(respMsg)
}

func (tc *TunnelClient) handlePing(msg *tunnel.Message) {
	// Send pong response
	pongMsg := tunnel.NewMessage(tunnel.TypePong)
	pongMsg.ID = msg.ID
	tc.conn.WriteJSON(pongMsg)
}

func (tc *TunnelClient) handleError(msg *tunnel.Message) {
	if errorData, ok := msg.Data["error"].(map[string]interface{}); ok {
		tc.logger.Errorf("Server error: %s", errorData["message"])
		if details, ok := errorData["details"].(string); ok && details != "" {
			tc.logger.Errorf("Error details: %s", details)
		}
	}
}

func (tc *TunnelClient) pingRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-tc.stopChan:
			return
		case <-ticker.C:
			if tc.connected {
				pingMsg := tunnel.NewMessage(tunnel.TypePing)
				if err := tc.conn.WriteJSON(pingMsg); err != nil {
					tc.logger.Errorf("Failed to send ping: %v", err)
				}
			}
		}
	}
}

func (tc *TunnelClient) Disconnect() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if !tc.connected {
		return
	}

	tc.logger.Info("🔌 Disconnecting...")

	// Send disconnect message
	disconnectMsg := tunnel.NewMessage(tunnel.TypeDisconnect)
	tc.conn.WriteJSON(disconnectMsg)

	// Stop routines
	close(tc.stopChan)

	// Close connection
	tc.conn.Close()
	tc.connected = false

	tc.logger.Info("❌ Disconnected from server")
}

func (tc *TunnelClient) isClosing() bool {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return !tc.connected
}

func (tc *TunnelClient) IsConnected() bool {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.connected
}

func (tc *TunnelClient) GetPublicURL() string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.publicURL
}
