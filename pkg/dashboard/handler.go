// pkg/dashboard/handler.go
package dashboard

import (
	"encoding/json"
	"html/template"

	"github.com/Coding-for-Machine/cfm/pkg/auth"
	"github.com/Coding-for-Machine/cfm/pkg/tunnel"
	"github.com/valyala/fasthttp"
)

type Dashboard struct {
	tunnelManager *tunnel.Manager
	authService   *auth.Service
	templates     *template.Template
}

func NewDashboard(tm *tunnel.Manager, authSvc *auth.Service) *Dashboard {
	tmpl := template.Must(template.ParseGlob("web/templates/*.html"))

	return &Dashboard{
		tunnelManager: tm,
		authService:   authSvc,
		templates:     tmpl,
	}
}

func (d *Dashboard) Handler(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())

	switch {
	case path == "/":
		d.handleHome(ctx)
	case path == "/api/tunnels":
		d.handleAPITunnels(ctx)
	case path == "/api/stats":
		d.handleAPIStats(ctx)
	case path == "/login":
		d.handleLogin(ctx)
	case path == "/dashboard":
		d.handleDashboard(ctx)
	default:
		d.handleStatic(ctx)
	}
}

func (d *Dashboard) handleHome(ctx *fasthttp.RequestCtx) {
	data := struct {
		Title         string
		ActiveTunnels int
		TotalRequests int64
	}{
		Title:         "JPRQ - Instant tunnels to localhost",
		ActiveTunnels: d.tunnelManager.GetActiveTunnelCount(),
		TotalRequests: d.tunnelManager.GetTotalRequestCount(),
	}

	ctx.SetContentType("text/html")
	d.templates.ExecuteTemplate(ctx, "home.html", data)
}

func (d *Dashboard) handleAPITunnels(ctx *fasthttp.RequestCtx) {
	token := string(ctx.Request.Header.Peek("Authorization"))
	if token == "" {
		ctx.SetStatusCode(fasthttp.StatusUnauthorized)
		return
	}

	user, err := d.authService.ValidateToken(token)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusUnauthorized)
		return
	}

	tunnels := d.tunnelManager.GetUserTunnels(user.ID)

	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(map[string]interface{}{
		"tunnels": tunnels,
		"count":   len(tunnels),
	})
}

func (d *Dashboard) handleDashboard(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/html")
	d.templates.ExecuteTemplate(ctx, "dashboard.html", nil)
}
