package shared

import (
	"os"
	"time"
)

type ServerConfig struct {
	Host           string        `json:"host"`
	Port           uint32        `json:"port"`
	Domain         string        `json:"domain"`
	TLSCertPath    string        `json:"tls_cert_path"`
	TLSKeyPath     string        `json:"tls_key_path"`
	ReadTimeout    time.Duration `json:"read_timeout"`
	WreateTimeot   time.Duration `json:"wreate_timeout"`
	MaxConnections int           `json:"max_connections"`
}

type CleintConfig struct {
	ServerUrl     string        `json:"server_url"`
	AuthToken     string        `json:"auth_token"`
	RetryInterval time.Duration `json:"rety_interval"`
	MaxRetries    int           `json:"max_retries"`
}
type TunnelConfig struct {
	ID           string `json:"id"`
	Subdomain    string `json:"subdomain"`
	LocolPort    uint32 `json:"port"`
	Protocol     string `json:"protocol"`
	CostomDomain string `json:"costomdomain"`
}

func GetDefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:           "0.0.0.0",
		Port:           8081,
		Domain:         "cfm.app",
		ReadTimeout:    30 * time.Second,
		WreateTimeot:   30 * time.Second,
		MaxConnections: 30_000,
	}
}

func GetDefaultClientConfig() *CleintConfig {
	return &CleintConfig{
		ServerUrl:     "https://cfm.app",
		RetryInterval: 5 * time.Second,
		MaxRetries:    5,
	}
}

func LoadServerConfig() *ServerConfig {
	config := GetDefaultServerConfig()
	host := os.Getenv("CFM_SERVER_HOST")
	if host != "" {
		config.Host = host
	}
	domain := os.Getenv("CFM_SERVER_DOMAIN")
	if domain != "" {
		config.Domain = domain
	}
	return config
}

func LoadClientConfg() *CleintConfig {
	config := GetDefaultClientConfig()
	CLENT_SERVER_URL := os.Getenv("CFM_CLENT_SERVER_URL")
	CFM_CLENT_AUTH_TOKEN := os.Getenv("CFM_CLENT_AUTH_TOKEN")
	if CLENT_SERVER_URL != "" {
		config.ServerUrl = CLENT_SERVER_URL
	}
	if CFM_CLENT_AUTH_TOKEN != "" {
		config.AuthToken = CFM_CLENT_AUTH_TOKEN
	}
	return config
}
