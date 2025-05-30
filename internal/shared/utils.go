package shared

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"time"
)

var Logger = log.New(log.Writer(), "[CFM]", log.LstdFlags|log.Lshortfile)

// GenerateRandomString
func GenerateRandomString(length int) string {
	bytes_ := make([]byte, length/2)
	if _, err := rand.Read(bytes_); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes_)[:length]
}

// GenerateSubdomen
func GenerateSubdomen() string {
	adjectives := []string{
		"quick", "bright", "smart", "fast", "cool", "warm", "fresh",
		"clear", "smooth", "sharp", "strong", "light", "dark", "soft",
	}

	nouns := []string{
		"tiger", "eagle", "wolf", "bear", "lion", "shark", "falcon",
		"dragon", "phoenix", "thunder", "storm", "wind", "fire", "ice",
	}

	adj := adjectives[time.Now().UnixNano()%int64(len(adjectives))]
	noun := nouns[time.Now().UnixNano()%int64(len(nouns))]

	return fmt.Sprintf("%s-%s-%s", adj, noun, GenerateRandomString(4))
}

func IsValidSubdomain(subdomain string) bool {
	matched, err := regexp.MatchString(`^[a-z0-9-]+$`, subdomain)
	if err != nil {
		return false
	}
	if !matched {
		return false
	}
	if strings.HasPrefix(subdomain, "-") || strings.HasSuffix(subdomain, "-") {
		return false
	}
	if len(subdomain) < 3 || len(subdomain) > 63 {
		return false
	}
	return true
}

// IsValidPort - Port validatsiyasi
func IsValidPort(port int) bool {
	return port > 0 && port <= 65535
}

// IsLocalPortAvailable - Local port mavjudligini tekshirish
func IsLocalPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// GetLocalIP - Local IP manzilini olish
func GetLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// FormatBytes - Byte'larni formatlash
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// SafeClose - Xavfsiz yopish
func SafeClose(closer interface{ Close() error }) {
	if closer != nil {
		if err := closer.Close(); err != nil {
			Logger.Printf("Close error: %v", err)
		}
	}
}
