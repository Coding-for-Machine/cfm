// pkg/websocket/conn_wrapper.go - WebSocket connection wrapper (fasthttp)
package websocket

import (
	"context"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/yourusername/jprq-clone/pkg/protocol"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// Connection WebSocket ulanish uchun abstraksiya
type Connection struct {
	conn        *websocket.Conn
	send        chan []byte
	receive     chan protocol.Message
	ctx         context.Context
	cancel      context.CancelFunc
	isConnected bool

	onMessage func(protocol.Message)
	onClose   func()
}

// fasthttp websocket connection'dan yangi Connection yaratuvchi funksiya
func NewConnectionFromConn(conn *websocket.Conn) *Connection {
	ctx, cancel := context.WithCancel(context.Background())

	c := &Connection{
		conn:        conn,
		send:        make(chan []byte, 256),
		receive:     make(chan protocol.Message, 256),
		ctx:         ctx,
		cancel:      cancel,
		isConnected: true,
	}

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(appData string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	return c
}

// WebSocket connection uchun readPump (fasthttp versiyasi)
func (c *Connection) ReadPump() {
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
				continue // noto‘g‘ri formatdagi xabar
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

// WebSocket connection uchun writePump (fasthttp versiyasi)
func (c *Connection) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// kanal yopilgan bo‘lsa, ulanishni yopamiz
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

// Ulanishni yopish
func (c *Connection) Close() {
	c.cancel()
	c.isConnected = false
	close(c.send)
}
