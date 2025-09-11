package tunnel

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Subdomain generator
func GenerateSubdomain(length int) (string, error) {
	if length <= 0 {
		length = 8
	}

	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)

	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}

	return string(result), nil
}

// Request ID generator
func GenerateRequestID() string {
	timestamp := time.Now().UnixNano()
	randomPart, _ := GenerateSubdomain(6)
	return fmt.Sprintf("%d-%s", timestamp, randomPart)
}

// Extract subdomain from host
func ExtractSubdomain(host, baseDomain string) string {
	// example.com dan subdomain.example.com -> subdomain
	if !strings.Contains(host, baseDomain) {
		return ""
	}

	parts := strings.Split(host, ".")
	if len(parts) <= 2 {
		return ""
	}

	return parts[0]
}

// Validate subdomain
func IsValidSubdomain(subdomain string) bool {
	if len(subdomain) < 3 || len(subdomain) > 20 {
		return false
	}

	// Faqat harflar va raqamlar
	for _, r := range subdomain {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return false
		}
	}

	return true
}

// Format public URL
func FormatPublicURL(subdomain, baseDomain string, useHTTPS bool) string {
	protocol := "http"
	if useHTTPS {
		protocol = "https"
	}

	return fmt.Sprintf("%s://%s.%s", protocol, subdomain, baseDomain)
}

// Time helper
func GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// Check if port is valid
func IsValidPort(port int) bool {
	return port > 0 && port <= 65535
}
