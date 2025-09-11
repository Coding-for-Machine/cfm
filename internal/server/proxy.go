// internal/server/proxy.go
package server

import (
	"io"
	"net/http"
	"time"

	"github.com/Coding-for-Machine/cfm/internal/tunnel"
)

// HTTP Proxy handler with full request/response cycle
func (tm *TunnelManager) HandleHTTPProxyFull(w http.ResponseWriter, r *http.Request) {
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

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Prepare headers
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Create HTTP request
	requestID := tunnel.GenerateRequestID()
	httpReq := &tunnel.HTTPRequest{
		ID:      requestID,
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: headers,
		Body:    string(body),
		Host:    host,
	}

	// Add query parameters
	if r.URL.RawQuery != "" {
		httpReq.Path += "?" + r.URL.RawQuery
	}

	tm.logger.Infof("Proxying request: %s %s -> %s (ID: %s)",
		r.Method, httpReq.Path, subdomain, requestID)

	// Send request to client
	msg := tunnel.NewHTTPRequestMessage(httpReq)
	if err := client.Conn.WriteJSON(msg); err != nil {
		tm.logger.Errorf("Failed to send request to client: %v", err)
		http.Error(w, "Failed to forward request", http.StatusInternalServerError)
		return
	}

	// Wait for response with timeout
	select {
	case response := <-client.ResponseChan:
		if response.ID == requestID {
			tm.writeHTTPResponse(w, response)
			tm.logger.Infof("Response sent for request %s: %d", requestID, response.StatusCode)
		} else {
			tm.logger.Warnf("Response ID mismatch: expected %s, got %s", requestID, response.ID)
			http.Error(w, "Response ID mismatch", http.StatusInternalServerError)
		}
	case <-time.After(30 * time.Second):
		tm.logger.Warnf("Request timeout for %s", requestID)
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
	}
}

func (tm *TunnelManager) writeHTTPResponse(w http.ResponseWriter, response *tunnel.HTTPResponse) {
	// Set headers
	for key, value := range response.Headers {
		w.Header().Set(key, value)
	}

	// Set status code
	w.WriteHeader(response.StatusCode)

	// Write body
	if response.Body != "" {
		w.Write([]byte(response.Body))
	}
}

// API handler for dashboard
func (tm *TunnelManager) HandleAPITunnels(w http.ResponseWriter, r *http.Request) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	type TunnelInfo struct {
		ID          string    `json:"id"`
		Subdomain   string    `json:"subdomain"`
		PublicURL   string    `json:"public_url"`
		LocalPort   int       `json:"local_port"`
		ConnectedAt time.Time `json:"connected_at"`
		LastPing    time.Time `json:"last_ping"`
	}

	var tunnels []TunnelInfo
	for _, client := range tm.clients {
		if client.Connected {
			tunnels = append(tunnels, TunnelInfo{
				ID:          client.ID,
				Subdomain:   client.Subdomain,
				PublicURL:   client.PublicURL,
				LocalPort:   client.LocalPort,
				ConnectedAt: time.Unix(0, 0), // Bu yerda actual connection time kerak
				LastPing:    client.LastPing,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Simple JSON response
	w.Write([]byte("["))
	for i, tunnel := range tunnels {
		if i > 0 {
			w.Write([]byte(","))
		}
		w.Write([]byte(`{`))
		w.Write([]byte(`"id":"` + tunnel.ID + `",`))
		w.Write([]byte(`"subdomain":"` + tunnel.Subdomain + `",`))
		w.Write([]byte(`"public_url":"` + tunnel.PublicURL + `",`))
		w.Write([]byte(`"local_port":` + string(rune(tunnel.LocalPort)) + `,`))
		w.Write([]byte(`"connected_at":"` + tunnel.ConnectedAt.Format(time.RFC3339) + `"`))
		w.Write([]byte(`}`))
	}
	w.Write([]byte("]"))
}
