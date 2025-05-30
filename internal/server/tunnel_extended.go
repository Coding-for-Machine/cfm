// internal/server/tunnel_extended.go - Tunnel Request/Response handling
package server

import (
	"context"
	"fmt"
	"time"

	"github.com/Coding-for-Machine/cfm/pkg/protocol"
)

// Tunnel'ga request/response qo'shimcha funksiyalar
type PendingRequest struct {
	ID       string
	Message  *protocol.HTTPRequestMessage
	Response chan *protocol.HTTPResponseMessage
	Timeout  time.Duration
	Created  time.Time
}

// Tunnel struct'ga qo'shimcha fieldlar
func (t *Tunnel) init() {
	t.pendingRequests = make(map[string]*PendingRequest)
}

func (t *Tunnel) SendRequestAndWaitResponse(req *protocol.HTTPRequestMessage, timeout time.Duration) (*protocol.HTTPResponseMessage, error) {
	requestID := shared.GenerateID()
	req.RequestID = requestID

	pending := &PendingRequest{
		ID:       requestID,
		Message:  req,
		Response: make(chan *protocol.HTTPResponseMessage, 1),
		Timeout:  timeout,
		Created:  time.Now(),
	}

	t.mu.Lock()
	t.pendingRequests[requestID] = pending
	t.mu.Unlock()

	// Request yuborish
	if err := t.connection.SendMessage(req); err != nil {
		t.mu.Lock()
		delete(t.pendingRequests, requestID)
		t.mu.Unlock()
		return nil, err
	}

	// Response kutish
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case response := <-pending.Response:
		t.mu.Lock()
		delete(t.pendingRequests, requestID)
		t.mu.Unlock()
		return response, nil
	case <-ctx.Done():
		t.mu.Lock()
		delete(t.pendingRequests, requestID)
		t.mu.Unlock()
		return nil, fmt.Errorf("request timeout")
	}
}

func (t *Tunnel) HandleMessage(msg protocol.Message) {
	switch m := msg.(type) {
	case *protocol.HTTPResponseMessage:
		t.handleHTTPResponse(m)
	case *protocol.TunnelRegisterMessage:
		t.handleTunnelRegister(m)
	case *protocol.HeartbeatMessage:
		t.handleHeartbeat(m)
	}
}

func (t *Tunnel) handleHTTPResponse(resp *protocol.HTTPResponseMessage) {
	t.mu.RLock()
	pending, exists := t.pendingRequests[resp.RequestID]
	t.mu.RUnlock()

	if !exists {
		return
	}

	select {
	case pending.Response <- resp:
	default:
		// Channel to'lgan bo'lsa, ignore qilish
	}
}

func (t *Tunnel) handleTunnelRegister(msg *protocol.TunnelRegisterMessage) {
	t.mu.Lock()
	t.subdomain = msg.Subdomain
	t.localPort = msg.LocalPort
	t.protocol = msg.Protocol
	t.mu.Unlock()

	// Confirmation yuborish
	confirm := &protocol.TunnelConfirmMessage{
		BaseMessage: protocol.BaseMessage{
			Type:      protocol.MsgTypeTunnelConfirm,
			Timestamp: time.Now(),
		},
		Success:   true,
		Subdomain: msg.Subdomain,
		PublicURL: fmt.Sprintf("http://%s.localhost:8080", msg.Subdomain),
	}

	t.connection.SendMessage(confirm)
}

func (t *Tunnel) handleHeartbeat(msg *protocol.HeartbeatMessage) {
	t.mu.Lock()
	t.lastHeartbeat = time.Now()
	t.mu.Unlock()

	// Pong yuborish
	pong := &protocol.HeartbeatMessage{
		BaseMessage: protocol.BaseMessage{
			Type:      protocol.MsgTypeHeartbeat,
			Timestamp: time.Now(),
		},
		Ping: false,
	}

	t.connection.SendMessage(pong)
}

// Health check
func (t *Tunnel) IsHealthy() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.connection.IsConnected() {
		return false
	}

	// 2 daqiqadan ko'p heartbeat kelmagan bo'lsa
	return time.Since(t.lastHeartbeat) < 2*time.Minute
}

// Cleanup expired requests
func (t *Tunnel) cleanupExpiredRequests() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for id, req := range t.pendingRequests {
		if now.Sub(req.Created) > req.Timeout {
			close(req.Response)
			delete(t.pendingRequests, id)
		}
	}
}
