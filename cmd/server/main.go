// cmd/server/main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/Coding-for-Machine/cfm/internal/server"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func main() {
	// Command line flags
	var (
		port     = flag.Int("port", 8080, "Server port")
		domain   = flag.String("domain", "localhost", "Base domain")
		useHTTPS = flag.Bool("https", false, "Use HTTPS")
		logLevel = flag.String("log", "info", "Log level (debug, info, warn, error)")
		// configFile = flag.String("config", "", "Config file path")
	)
	flag.Parse()

	// Setup logging
	logger := logrus.New()
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Create tunnel manager
	manager := server.NewTunnelManager(*domain, *useHTTPS)

	// Setup routes
	r := mux.NewRouter()

	// WebSocket endpoint for tunnels
	r.HandleFunc("/tunnel", manager.HandleWebSocket)

	// Dashboard
	r.HandleFunc("/dashboard", serveDashboard).Methods("GET")
	r.HandleFunc("/api/tunnels", manager.HandleAPITunnels).Methods("GET")

	// Static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","version":"1.0.0"}`)
	}).Methods("GET")

	// Default handler for subdomains (HTTP proxy)
	r.PathPrefix("/").HandlerFunc(manager.HandleHTTPProxy)

	// Server info
	protocol := "HTTP"
	if *useHTTPS {
		protocol = "HTTPS"
	}

	logger.Infof("🚀 JPRQ Clone Server starting...")
	logger.Infof("📡 Protocol: %s", protocol)
	logger.Infof("🌐 Domain: %s", *domain)
	logger.Infof("🔌 Port: %d", *port)
	logger.Infof("📊 Dashboard: http://localhost:%d/dashboard", *port)
	logger.Infof("🔗 WebSocket: ws://localhost:%d/tunnel", *port)

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	if *useHTTPS {
		// HTTPS setup (Let's Encrypt yoki manual certificates)
		log.Fatal(http.ListenAndServeTLS(addr, "cert.pem", "key.pem", r))
	} else {
		log.Fatal(http.ListenAndServe(addr, r))
	}
}

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>JPRQ Clone - Dashboard</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container { 
            max-width: 1200px; 
            margin: 0 auto; 
            background: white;
            border-radius: 10px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(45deg, #2196F3, #21CBF3);
            color: white;
            padding: 30px;
            text-align: center;
        }
        .header h1 { font-size: 2.5em; margin-bottom: 10px; }
        .header p { opacity: 0.9; font-size: 1.1em; }
        .content { padding: 30px; }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .stat-card {
            background: linear-gradient(45deg, #f093fb 0%, #f5576c 100%);
            color: white;
            padding: 20px;
            border-radius: 10px;
            text-align: center;
        }
        .stat-card h3 { font-size: 2em; margin-bottom: 5px; }
        .tunnels-table {
            background: #f8f9fa;
            border-radius: 10px;
            overflow: hidden;
        }
        .table-header {
            background: #343a40;
            color: white;
            padding: 15px;
            font-weight: bold;
        }
        .tunnel-item {
            padding: 15px;
            border-bottom: 1px solid #dee2e6;
            display: grid;
            grid-template-columns: auto 150px 100px 120px;
            align-items: center;
            gap: 15px;
        }
        .tunnel-item:last-child { border-bottom: none; }
        .tunnel-url { 
            color: #007bff; 
            text-decoration: none;
            font-weight: 500;
        }
        .tunnel-url:hover { text-decoration: underline; }
        .status-online { 
            background: #28a745; 
            color: white; 
            padding: 5px 10px; 
            border-radius: 20px;
            font-size: 0.8em;
        }
        .loading { text-align: center; padding: 40px; }
        .empty { text-align: center; padding: 40px; color: #6c757d; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 JPRQ Clone</h1>
            <p>Tunnel Management Dashboard</p>
        </div>
        
        <div class="content">
            <div class="stats">
                <div class="stat-card">
                    <h3 id="total-tunnels">0</h3>
                    <p>Active Tunnels</p>
                </div>
                <div class="stat-card">
                    <h3 id="total-requests">0</h3>
                    <p>Total Requests</p>
                </div>
                <div class="stat-card">
                    <h3 id="server-uptime">0m</h3>
                    <p>Uptime</p>
                </div>
            </div>
            
            <div class="tunnels-table">
                <div class="table-header">
                    Active Tunnels
                </div>
                <div id="tunnels-list" class="loading">
                    Loading tunnels...
                </div>
            </div>
        </div>
    </div>

    <script>
        const startTime = Date.now();

        function updateUptime() {
            const minutes = Math.floor((Date.now() - startTime) / 1000 / 60);
            document.getElementById('server-uptime').textContent = minutes + 'm';
        }

        async function loadTunnels() {
            try {
                const response = await fetch('/api/tunnels');
                const data = await response.json();

                // Support multiple response shapes
                let tunnels = [];
                if (Array.isArray(data)) {
                    tunnels = data;
                } else if (data && Array.isArray(data.tunnels)) {
                    tunnels = data.tunnels;
                } else if (data && Array.isArray(data.items)) {
                    tunnels = data.items;
                } else {
                    // fallback: if it's an object with numeric keys
                    tunnels = [];
                }

                const container = document.getElementById('tunnels-list');
                document.getElementById('total-tunnels').textContent = tunnels.length || 0;

                if (!tunnels || tunnels.length === 0) {
                    container.innerHTML = '<div class="empty">No active tunnels</div>';
                    return;
                }

                container.innerHTML = tunnels.map(function(tunnel) {
                    const publicUrl = tunnel.public_url || tunnel.publicUrl || tunnel.url || '#';
                    const localPort = tunnel.local_port || tunnel.localPort || '-';
                    const subdomain = tunnel.subdomain || (publicUrl.split ? publicUrl.split('.')[0] : '-');
                    const connectedAt = tunnel.connected_at || tunnel.connectedAt || null;
                    const timeText = connectedAt ? new Date(connectedAt).toLocaleString() : '-';

                    return '<div class="tunnel-item">' +
                        '<div>' +
                            '<a href="' + publicUrl + '" target="_blank" class="tunnel-url">' + publicUrl + '</a>' +
                            '<br><small>→ localhost:' + localPort + '</small>' +
                        '</div>' +
                        '<div>' + subdomain + '</div>' +
                        '<div><span class="status-online">Online</span></div>' +
                        '<div>' + timeText + '</div>' +
                    '</div>';
                }).join('');
            } catch (error) {
                console.error('Failed to load tunnels:', error);
                document.getElementById('tunnels-list').innerHTML = '<div class="empty">Failed to load tunnels</div>';
            }
        }

        // Auto refresh
        setInterval(loadTunnels, 5000);
        setInterval(updateUptime, 1000);

        // Initial load
        loadTunnels();
        updateUptime();
    </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
