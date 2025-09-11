package server

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Coding-for-Machine/cfm/internal/tunnel"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Client connection info
type ClientConnection struct {
	ID           string
	Conn         *websocket.Conn
	LocalPort    int
	Subdomain    string
	PublicURL    string
	LastPing     time.Time
	RequestChan  chan *tunnel.HTTPRequest
	ResponseChan chan *tunnel.HTTPResponse
	Connected    bool
}

// Tunnel Manager
type TunnelManager struct {
	clients    map[string]*ClientConnection // clientID -> connection
	subdomains map[string]string            // subdomain -> clientID
	baseDomain string
	useHTTPS   bool
	mu         sync.RWMutex
	logger     *logrus.Logger
	upgrader   websocket.Upgrader
}

func NewTunnelManager(baseDomain string, useHTTPS bool) *TunnelManager {
	return &TunnelManager{
		clients:    make(map[string]*ClientConnection),
		subdomains: make(map[string]string),
		baseDomain: baseDomain,
		useHTTPS:   useHTTPS,
		logger:     logrus.New(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Production'da buni cheklash kerak
			},
			HandshakeTimeout: 10 * time.Second,
		},
	}
}

// WebSocket connection handler
func (tm *TunnelManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := tm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		tm.logger.Errorf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	tm.logger.Info("New WebSocket connection")
	tm.handleClientConnection(conn)
}

func (tm *TunnelManager) handleClientConnection(conn *websocket.Conn) {
	// First message should be registration
	var msg tunnel.Message
	if err := conn.ReadJSON(&msg); err != nil {
		tm.logger.Errorf("Failed to read registration: %v", err)
		return
	}

	if msg.Type != tunnel.TypeRegister {
		tm.sendError(conn, 400, "First message must be registration", "")
		return
	}

	// Parse registration data
	regData, ok := msg.Data["registration"].(map[string]interface{})
	if !ok {
		tm.sendError(conn, 400, "Invalid registration data", "")
		return
	}

	localPort, ok := regData["local_port"].(float64) // JSON numbers are float64
	if !ok || !tunnel.IsValidPort(int(localPort)) {
		tm.sendError(conn, 400, "Invalid local port", "")
		return
	}

	// Generate subdomain
	subdomain, err := tunnel.GenerateSubdomain(8)
	if err != nil {
		tm.sendError(conn, 500, "Failed to generate subdomain", err.Error())
		return
	}

	// Create client connection
	clientID := tunnel.GenerateRequestID()
	publicURL := tunnel.FormatPublicURL(subdomain, tm.baseDomain, tm.useHTTPS)

	client := &ClientConnection{
		ID:           clientID,
		Conn:         conn,
		LocalPort:    int(localPort),
		Subdomain:    subdomain,
		PublicURL:    publicURL,
		LastPing:     time.Now(),
		RequestChan:  make(chan *tunnel.HTTPRequest, 100),
		ResponseChan: make(chan *tunnel.HTTPResponse, 100),
		Connected:    true,
	}

	// Register client
	tm.mu.Lock()
	tm.clients[clientID] = client
	tm.subdomains[subdomain] = clientID
	tm.mu.Unlock()

	tm.logger.Infof("Client registered: %s -> %s", clientID, publicURL)

	// Send registration confirmation
	regResp := &tunnel.Registration{
		ClientID:  clientID,
		LocalPort: client.LocalPort,
		Subdomain: subdomain,
		PublicURL: publicURL,
	}

	respMsg := tunnel.NewMessage(tunnel.TypeRegistered)
	respMsg.Data["registration"] = regResp

	if err := conn.WriteJSON(respMsg); err != nil {
		tm.logger.Errorf("Failed to send registration response: %v", err)
		tm.removeClient(clientID)
		return
	}

	// Start handling this client
	tm.handleClientMessages(client)
}

func (tm *TunnelManager) handleClientMessages(client *ClientConnection) {
	defer tm.removeClient(client.ID)

	// Start response handler
	go tm.handleResponses(client)

	for {
		var msg tunnel.Message
		if err := client.Conn.ReadJSON(&msg); err != nil {
			tm.logger.Errorf("Read error from client %s: %v", client.ID, err)
			break
		}

		switch msg.Type {
		case tunnel.TypeHTTPResponse:
			if respData, ok := msg.Data["response"].(map[string]interface{}); ok {
				response := &tunnel.HTTPResponse{
					ID:         respData["id"].(string),
					StatusCode: int(respData["status_code"].(float64)),
					Body:       respData["body"].(string),
				}

				// Extract headers
				if headersData, ok := respData["headers"].(map[string]interface{}); ok {
					response.Headers = make(map[string]string)
					for k, v := range headersData {
						response.Headers[k] = v.(string)
					}
				}

				select {
				case client.ResponseChan <- response:
				case <-time.After(5 * time.Second):
					tm.logger.Warnf("Response channel full for client %s", client.ID)
				}
			}

		case tunnel.TypePing:
			client.LastPing = time.Now()
			pongMsg := tunnel.NewMessage(tunnel.TypePong)
			client.Conn.WriteJSON(pongMsg)

		case tunnel.TypeDisconnect:
			tm.logger.Infof("Client %s disconnecting", client.ID)
			return

		default:
			tm.logger.Warnf("Unknown message type from client %s: %s", client.ID, msg.Type)
		}
	}
}

func (tm *TunnelManager) handleResponses(client *ClientConnection) {
	for response := range client.ResponseChan {
		// Bu yerda HTTP response'larni handle qilamiz
		// Keyingi bosqichda to'liq implement qilamiz
		tm.logger.Debugf("Received response for request %s: %d", response.ID, response.StatusCode)
	}
}

func (tm *TunnelManager) sendError(conn *websocket.Conn, code int, message, details string) {
	errorMsg := tunnel.NewErrorMessage(code, message, details)
	conn.WriteJSON(errorMsg)
}

func (tm *TunnelManager) removeClient(clientID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if client, exists := tm.clients[clientID]; exists {
		client.Connected = false
		close(client.RequestChan)
		close(client.ResponseChan)
		delete(tm.subdomains, client.Subdomain)
		delete(tm.clients, clientID)

		tm.logger.Infof("Client removed: %s", clientID)
	}
}

// HTTP proxy handler
func (tm *TunnelManager) HandleHTTPProxy(w http.ResponseWriter, r *http.Request) {
	// Extract subdomain
	host := r.Host
	subdomain := tunnel.ExtractSubdomain(host, tm.baseDomain)

	if subdomain == "" {
		http.Error(w, "Invalid subdomain", http.StatusBadRequest)
		return
	}

	// Find client
	tm.mu.RLock()
	clientID, exists := tm.subdomains[subdomain]
	tm.mu.RUnlock()

	if !exists {
		http.Error(w, "Tunnel not found", http.StatusNotFound)
		return
	}

	tm.mu.RLock()
	client, exists := tm.clients[clientID]
	tm.mu.RUnlock()

	if !exists || !client.Connected {
		http.Error(w, "Client not connected", http.StatusServiceUnavailable)
		return
	}

	tm.logger.Infof("Proxying request: %s %s -> %s", r.Method, r.URL.Path, subdomain)

	// Bu yerda HTTP proxy logic implement qilamiz
	// Keyingi bosqichda...
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
    <h1>🚀 JPRQ Clone - Active Tunnel!</h1>
    <p><strong>Subdomain:</strong> %s</p>
    <p><strong>Method:</strong> %s</p>
    <p><strong>Path:</strong> %s</p>
    <p><strong>Client ID:</strong> %s</p>
    <hr>
    <p>Tunnel is active and receiving requests!</p>
    `, subdomain, r.Method, r.URL.Path, clientID)
}

// Get active tunnels
func (tm *TunnelManager) GetActiveTunnels() map[string]*ClientConnection {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make(map[string]*ClientConnection)
	for id, client := range tm.clients {
		result[id] = client
	}

	return result
}
