package protocol

import (
	"encoding/json"
	"time"
)

// MessageType - Xabar turlari
type MessageType string

const (
	// Client -> Server
	MsgTypeAuth         MessageType = "auth"
	MsgTypeTunnelCreate MessageType = "tunnel_create"
	MsgTypeTunnelClose  MessageType = "tunnel_close"
	MsgTypeHeartbeat    MessageType = "heartbeat"
	MsgTypeData         MessageType = "data"

	// Server -> Client
	MsgTypeAuthSuccess MessageType = "auth_success"
	MsgTypeAuthFailed  MessageType = "auth_failed"
	MsgTypeTunnelReady MessageType = "tunnel_ready"
	MsgTypeTunnelError MessageType = "tunnel_error"
	MsgTypeRequest     MessageType = "request"
	MsgTypeResponse    MessageType = "response"
	MsgTypeError       MessageType = "error"
)

// BaseMessage - Asosiy xabar strukturasi
type BaseMessage struct {
	Type      MessageType `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	ID        string      `json:"id,omitempty"`
}

// AuthMessage - Authentication xabari
type AuthMessage struct {
	BaseMessage
	Token string `json:"token"`
}

// HTTP Request/Response messages
type HTTPRequestMessage struct {
	BaseMessage
	RequestID string            `json:"request_id"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Headers   map[string]string `json:"headers"`
	Body      []byte            `json:"body"`
	Query     string            `json:"query"`
}

type HTTPResponseMessage struct {
	BaseMessage
	RequestID  string            `json:"request_id"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

// Tunnel management messages
type TunnelRegisterMessage struct {
	BaseMessage
	Subdomain string `json:"subdomain"`
	LocalPort int    `json:"local_port"`
	Protocol  string `json:"protocol"`
}

type TunnelConfirmMessage struct {
	BaseMessage
	Success   bool   `json:"success"`
	Subdomain string `json:"subdomain"`
	PublicURL string `json:"public_url"`
	Error     string `json:"error,omitempty"`
}

// TunnelCreateMessage - Tunnel yaratish xabari
type TunnelCreateMessage struct {
	BaseMessage
	Protocol     string `json:"protocol"` // http, tcp
	LocalPort    int    `json:"local_port"`
	Subdomain    string `json:"subdomain,omitempty"`
	CustomDomain string `json:"custom_domain,omitempty"`
}

// TunnelReadyMessage - Tunnel tayyor xabari
type TunnelReadyMessage struct {
	BaseMessage
	TunnelID  string `json:"tunnel_id"`
	PublicURL string `json:"public_url"`
	Subdomain string `json:"subdomain"`
}

// TunnelErrorMessage - Tunnel xatosi xabari
type TunnelErrorMessage struct {
	BaseMessage
	Error   string `json:"error"`
	Code    int    `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// RequestMessage - HTTP request xabari
type RequestMessage struct {
	BaseMessage
	TunnelID string            `json:"tunnel_id"`
	Method   string            `json:"method"`
	Path     string            `json:"path"`
	Headers  map[string]string `json:"headers"`
	Body     []byte            `json:"body,omitempty"`
	ClientIP string            `json:"client_ip"`
}

// ResponseMessage - HTTP response xabari
type ResponseMessage struct {
	BaseMessage
	RequestID  string            `json:"request_id"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body,omitempty"`
}

// DataMessage - Raw data xabari (TCP uchun)
type DataMessage struct {
	BaseMessage
	TunnelID     string `json:"tunnel_id"`
	ConnectionID string `json:"connection_id"`
	Data         []byte `json:"data"`
	IsClose      bool   `json:"is_close,omitempty"`
}

// ErrorMessage - Umumiy xato xabari
type ErrorMessage struct {
	BaseMessage
	Error   string `json:"error"`
	Code    int    `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// HeartbeatMessage - Heartbeat xabari
type HeartbeatMessage struct {
	BaseMessage
	Ping bool `json:"ping,omitempty"`
	Pong bool `json:"pong,omitempty"`
}

// Message - Umumiy message interface
type Message interface {
	GetType() MessageType
	GetID() string
}

// Implement Message interface
func (m *BaseMessage) GetType() MessageType { return m.Type }
func (m *BaseMessage) GetID() string        { return m.ID }

// MarshalMessage - Message'ni JSON'ga aylantirish
func MarshalMessage(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

// UnmarshalMessage - JSON'dan message'ga aylantirish
func UnmarshalMessage(data []byte) (Message, error) {
	var base BaseMessage
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, err
	}

	switch base.Type {
	case MsgTypeAuth:
		var msg AuthMessage
		err := json.Unmarshal(data, &msg)
		return &msg, err

	case MsgTypeTunnelCreate:
		var msg TunnelCreateMessage
		err := json.Unmarshal(data, &msg)
		return &msg, err

	case MsgTypeTunnelReady:
		var msg TunnelReadyMessage
		err := json.Unmarshal(data, &msg)
		return &msg, err

	case MsgTypeTunnelError:
		var msg TunnelErrorMessage
		err := json.Unmarshal(data, &msg)
		return &msg, err

	case MsgTypeRequest:
		var msg RequestMessage
		err := json.Unmarshal(data, &msg)
		return &msg, err

	case MsgTypeResponse:
		var msg ResponseMessage
		err := json.Unmarshal(data, &msg)
		return &msg, err

	case MsgTypeData:
		var msg DataMessage
		err := json.Unmarshal(data, &msg)
		return &msg, err

	case MsgTypeError:
		var msg ErrorMessage
		err := json.Unmarshal(data, &msg)
		return &msg, err

	case MsgTypeHeartbeat:
		var msg HeartbeatMessage
		err := json.Unmarshal(data, &msg)
		return &msg, err

	default:
		return &base, nil
	}
}
