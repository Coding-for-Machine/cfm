package shared

import (
	"os"
	"time"
)

// ServerConfig
type ServerConfig struct {
	Host           string        `json:"host"`
	Port           int           `json:"port"`
	Domain         string        `json:"domain"`
	TLSCertPath    string        `json:"tls_cert_path"`
	TLSKeyPath     string        `json:"tls_key_path"`
	ReadTimeout    time.Duration `json:"read_timeout"`
	WriteTimeout   time.Duration `json:"write_timeout"`
	MaxConnections int           `json:"max_connections"`
}

// ClientConfig
type ClientConfig struct {
	ServerUrl     string        `json:"server_url"`
	AuthToken     string        `json:"auth_token"`
	RetryInterval time.Duration `json:"retry_interval"`
	MaxRetries    int           `json:"max_retries"`
}

// TunnelConfig
type TunnelConfig struct {
	ID           string `json:"id"`
	Subdomain    string `json:"subdomain"`
	LocalPort    int    `json:"local_port"`
	Protocol     string `json:"protocol"`
	CustomDomain string `json:"custom_domain,omitempty"`
}

// GetDefaultServerConfig
func GetDefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:           "0.0.0.0",
		Port:           8080,
		Domain:         "cfm.app",
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxConnections: 30_000,
	}
}

// GetDefaultClientConfig
func GetDefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		ServerUrl:     "https://cfg.app",
		RetryInterval: 5 * time.Second,
		MaxRetries:    5,
	}
}

// LoadServerConfig
func LoadServerConfig() *ServerConfig {
	config := GetDefaultServerConfig()
	host := os.Getenv("CMF_HOST")
	domain := os.Getenv("CMF_DOMAIN")
	if host != "" {
		config.Host = host
	}
	if domain != "" {
		config.Domain = domain
	}
	return config
}

// LoadClientConfig
func LoadClientConfig() *ClientConfig {
	config := GetDefaultClientConfig()
	serverURL := os.Getenv("CFM_SERVER_URL")
	token := os.Getenv("CFG_AUTH_TOKEN")
	if serverURL != "" {
		config.ServerUrl = serverURL
	}
	if token != "" {
		config.AuthToken = token
	}
	return config
}
