package websocket

import (
	"context"
	"fmt"
	"time"

	"sync"

	"github.com/Coding-for-Machine/cfm/pkg/protocol"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024 * 1024 // 1MB
)

// Connection WebSocket ulanish uchun abstraksiya
type Connection struct {
	conn        *websocket.Conn
	send        chan []byte
	receive     chan protocol.Message
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	isConnected bool
	onMessage   func(protocol.Message)
	onClose     func()
}

var upgrader = websocket.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true
	},
}

func NewConnection(ctx *fasthttp.RequestCtx) (*Connection, error) {
	var connPtr *websocket.Conn
	err := upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		connPtr = conn
	})
	if err != nil {
		return nil, err
	}

	ctxx, cancel := context.WithCancel(context.Background())
	c := &Connection{
		conn:        connPtr,
		send:        make(chan []byte, 256),
		receive:     make(chan protocol.Message, 256),
		ctx:         ctxx,
		cancel:      cancel,
		isConnected: true,
	}

	connPtr.SetReadLimit(maxMessageSize)
	connPtr.SetReadDeadline(time.Now().Add(pongWait))
	connPtr.SetPongHandler(func(string) error {
		connPtr.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	return c, nil
}

func (c *Connection) Start() {
	go c.readPump()
	go c.writePump()
	go c.pingPump()
}

func (c *Connection) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

func (c *Connection) SendMessage(msg protocol.Message) error {
	if !c.IsConnected() {
		return ErrConnectionClosed
	}
	data, err := protocol.MarshalMessage(msg)
	if err != nil {
		return err
	}
	select {
	case c.send <- data:
		return nil
	case <-c.ctx.Done():
		return ErrConnectionClosed
	case <-time.After(writeWait):
		return ErrWriteTimeout
	}
}

func (c *Connection) ReceiveMessage() <-chan protocol.Message {
	return c.receive
}

func (c *Connection) OnMessage(handler func(protocol.Message)) {
	c.onMessage = handler
}

func (c *Connection) OnClose(handler func()) {
	c.onClose = handler
}

func (c *Connection) Close() error {
	c.mu.Lock()
	if !c.isConnected {
		c.mu.Unlock()
		return nil
	}
	c.isConnected = false
	c.mu.Unlock()

	c.cancel()

	c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	return c.conn.Close()
}

func (c *Connection) readPump() {
	defer func() {
		c.Close()
		if c.onClose != nil {
			c.onClose()
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, messageData, err := c.conn.ReadMessage()
			if err != nil {
				return
			}
			msg, err := protocol.UnmarshalMessage(messageData)
			if err != nil {
				continue
			}
			if c.onMessage != nil {
				c.onMessage(msg)
			}
			select {
			case c.receive <- msg:
			case <-c.ctx.Done():
				return
			default:
			}
		}
	}
}

func (c *Connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Connection) pingPump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			heartbeat := &protocol.HeartbeatMessage{
				BaseMessage: protocol.BaseMessage{
					Type:      protocol.MsgTypeHeartbeat,
					Timestamp: time.Now(),
				},
				Ping: true,
			}
			if err := c.SendMessage(heartbeat); err != nil {
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

var (
	ErrConnectionClosed = fmt.Errorf("connection closed")
	ErrWriteTimeout     = fmt.Errorf("write timeout")
)
