package tunnel

import (
	"encoding/json"
	"time"
)

// Message types
type MessageType string

const (
	TypeRegister     MessageType = "register"      // Client ro'yxatdan o'tish
	TypeRegistered   MessageType = "registered"    // Muvaffaqiyatli ro'yxat
	TypeHTTPRequest  MessageType = "http_request"  // HTTP so'rovi
	TypeHTTPResponse MessageType = "http_response" // HTTP javobi
	TypePing         MessageType = "ping"          // Connection check
	TypePong         MessageType = "pong"          // Ping javobi
	TypeError        MessageType = "error"         // Xatolik
	TypeDisconnect   MessageType = "disconnect"    // Uzilish
)

// Message structure
type Message struct {
	Type      MessageType            `json:"type"`
	ID        string                 `json:"id,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// HTTP request data
type HTTPRequest struct {
	ID      string            `json:"id"`
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	Host    string            `json:"host"`
}

// HTTP response data
type HTTPResponse struct {
	ID         string            `json:"id"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// Registration data
type Registration struct {
	ClientID  string `json:"client_id"`
	LocalPort int    `json:"local_port"`
	Subdomain string `json:"subdomain,omitempty"`  // Server tomonidan beriladi
	PublicURL string `json:"public_url,omitempty"` // Server tomonidan beriladi
	AuthToken string `json:"auth_token,omitempty"`
}

// Error data
type ErrorData struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Helper methods
func NewMessage(msgType MessageType) *Message {
	return &Message{
		Type:      msgType,
		Timestamp: time.Now().Unix(),
		Data:      make(map[string]interface{}),
	}
}

func (m *Message) SetData(key string, value interface{}) {
	if m.Data == nil {
		m.Data = make(map[string]interface{})
	}
	m.Data[key] = value
}

func (m *Message) GetData(key string) (interface{}, bool) {
	if m.Data == nil {
		return nil, false
	}
	value, exists := m.Data[key]
	return value, exists
}

func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func (m *Message) FromJSON(data []byte) error {
	return json.Unmarshal(data, m)
}

// Specialized message creators
func NewHTTPRequestMessage(req *HTTPRequest) *Message {
	msg := NewMessage(TypeHTTPRequest)
	msg.ID = req.ID
	msg.Data["request"] = req
	return msg
}

func NewHTTPResponseMessage(resp *HTTPResponse) *Message {
	msg := NewMessage(TypeHTTPResponse)
	msg.ID = resp.ID
	msg.Data["response"] = resp
	return msg
}

func NewRegistrationMessage(reg *Registration) *Message {
	msg := NewMessage(TypeRegister)
	msg.Data["registration"] = reg
	return msg
}

func NewErrorMessage(code int, message, details string) *Message {
	msg := NewMessage(TypeError)
	msg.Data["error"] = &ErrorData{
		Code:    code,
		Message: message,
		Details: details,
	}
	return msg
}
