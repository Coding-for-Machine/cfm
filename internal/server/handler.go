// internal/server/handler.go - HTTP Request Handler
package server

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Coding-for-Machine/cfm/internal/shared"
	"github.com/Coding-for-Machine/cfm/pkg/protocol"
	"github.com/Coding-for-Machine/cfm/pkg/websocket"
	"github.com/valyala/fasthttp"
)

type HTTPHandler struct {
	tunnelManager *TunnelManager
	authManager   *AuthManager
	config        *shared.Config
}

func NewHTTPHandler(tm *TunnelManager, am *AuthManager, cfg *shared.Config) *HTTPHandler {
	return &HTTPHandler{
		tunnelManager: tm,
		authManager:   am,
		config:        cfg,
	}
}

// Asosiy HTTP request handler
func (h *HTTPHandler) HandleHTTP(ctx *fasthttp.RequestCtx) {
	host := string(ctx.Request.Host())

	// Subdomain'ni ajratib olish
	subdomain := h.extractSubdomain(host)

	if subdomain == "" {
		h.handleMainDomain(ctx)
		return
	}

	// Tunnel topish
	tunnel := h.tunnelManager.GetTunnelBySubdomain(subdomain)
	if tunnel == nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("Tunnel not found")
		return
	}

	// Request'ni tunnel orqali yuborish
	h.forwardRequestToTunnel(ctx, tunnel)
}

// WebSocket connection handler
func (h *HTTPHandler) HandleWebSocket(ctx *fasthttp.RequestCtx) {
	token := string(ctx.QueryArgs().Peek("token"))
	if token == "" {
		ctx.SetStatusCode(fasthttp.StatusUnauthorized)
		ctx.SetBodyString("Token required")
		return
	}

	user := h.authManager.ValidateToken(token)
	if user == nil {
		ctx.SetStatusCode(fasthttp.StatusUnauthorized)
		ctx.SetBodyString("Invalid token")
		return
	}

	conn, err := websocket.NewConnection(ctx)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString("Failed to upgrade to WebSocket")
		return
	}

	// Client'ni tunnel sifatida ro'yxatga olish
	h.handleTunnelConnection(conn, user)
}

// API endpoints
func (h *HTTPHandler) HandleAPI(ctx *fasthttp.RequestCtx) {
	path := string(ctx.PATH())
	method := string(ctx.Method())

	switch {
	case path == "/api/auth" && method == "POST":
		h.handleAuth(ctx)
	case path == "/api/tunnels" && method == "GET":
		h.handleGetTunnels(ctx)
	case path == "/api/tunnels" && method == "POST":
		h.handleCreateTunnel(ctx)
	case strings.HasPrefix(path, "/api/tunnels/") && method == "DELETE":
		h.handleDeleteTunnel(ctx)
	default:
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("API endpoint not found")
	}
}

func (h *HTTPHandler) extractSubdomain(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) >= 3 && parts[len(parts)-2] == "localhost" {
		return parts[0]
	}
	return ""
}

func (h *HTTPHandler) handleMainDomain(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/html")
	ctx.SetBodyString(`
    <!DOCTYPE html>
    <html>
    <head><title>JPRQ Clone</title></head>
    <body>
        <h1>JPRQ Clone - Tunneling Service</h1>
        <p>Local server'ni internetga ochish uchun JPRQ Clone'dan foydalaning!</p>
        <h3>Ishlatish:</h3>
        <pre>jprq http 3000</pre>
    </body>
    </html>
    `)
}

func (h *HTTPHandler) forwardRequestToTunnel(ctx *fasthttp.RequestCtx, tunnel *Tunnel) {
	if !tunnel.IsConnected() {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		ctx.SetBodyString("Tunnel is not connected")
		return
	}

	// Request'ni protocol message'ga convert qilish
	reqMsg := &protocol.HTTPRequestMessage{
		BaseMessage: protocol.BaseMessage{
			Type:      protocol.MsgTypeHTTPRequest,
			Timestamp: time.Now(),
		},
		Method:  string(ctx.Method()),
		Path:    string(ctx.Path()),
		Headers: h.convertHeaders(ctx),
		Body:    ctx.PostBody(),
		Query:   string(ctx.QueryArgs().QueryString()),
	}

	// Tunnel'ga request yuborish
	responseMsg, err := tunnel.SendRequestAndWaitResponse(reqMsg, 30*time.Second)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusGatewayTimeout)
		ctx.SetBodyString("Tunnel timeout")
		return
	}

	// Response'ni client'ga qaytarish
	h.writeResponseToContext(ctx, responseMsg)
}

func (h *HTTPHandler) convertHeaders(ctx *fasthttp.RequestCtx) map[string]string {
	headers := make(map[string]string)
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})
	return headers
}

func (h *HTTPHandler) writeResponseToContext(ctx *fasthttp.RequestCtx, resp *protocol.HTTPResponseMessage) {
	ctx.SetStatusCode(resp.StatusCode)

	for key, value := range resp.Headers {
		ctx.Response.Header.Set(key, value)
	}

	ctx.SetBody(resp.Body)
}

func (h *HTTPHandler) handleTunnelConnection(conn *websocket.Connection, user *User) {
	tunnel := NewTunnel(user.ID, conn)

	conn.OnMessage(func(msg protocol.Message) {
		tunnel.HandleMessage(msg)
	})

	conn.OnClose(func() {
		h.tunnelManager.RemoveTunnel(tunnel.ID)
	})

	h.tunnelManager.AddTunnel(tunnel)
	conn.Start()
}

// API handlers
func (h *HTTPHandler) handleAuth(ctx *fasthttp.RequestCtx) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString("Invalid JSON")
		return
	}

	user := h.authManager.Authenticate(req.Username, req.Password)
	if user == nil {
		ctx.SetStatusCode(fasthttp.StatusUnauthorized)
		ctx.SetBodyString("Invalid credentials")
		return
	}

	token := h.authManager.GenerateToken(user)

	response := map[string]interface{}{
		"token": token,
		"user":  user,
	}

	data, _ := json.Marshal(response)
	ctx.SetContentType("application/json")
	ctx.SetBody(data)
}

func (h *HTTPHandler) handleGetTunnels(ctx *fasthttp.RequestCtx) {
	token := string(ctx.Request.Header.Peek("Authorization"))
	token = strings.TrimPrefix(token, "Bearer ")

	user := h.authManager.ValidateToken(token)
	if user == nil {
		ctx.SetStatusCode(fasthttp.StatusUnauthorized)
		return
	}

	tunnels := h.tunnelManager.GetUserTunnels(user.ID)
	data, _ := json.Marshal(tunnels)
	ctx.SetContentType("application/json")
	ctx.SetBody(data)
}

func (h *HTTPHandler) handleCreateTunnel(ctx *fasthttp.RequestCtx) {
	// Implementation for creating new tunnel
	ctx.SetStatusCode(fasthttp.StatusNotImplemented)
	ctx.SetBodyString("Not implemented yet")
}

func (h *HTTPHandler) handleDeleteTunnel(ctx *fasthttp.RequestCtx) {
	// Implementation for deleting tunnel
	ctx.SetStatusCode(fasthttp.StatusNotImplemented)
	ctx.SetBodyString("Not implemented yet")
}
