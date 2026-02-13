package server

import (
	"log/slog"
	"time"

	"coin-futures-websocket/internal/websocket/protocol"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Client represents a WebSocket client connection
type Client struct {
	id            string
	ajaibID       string
	cfxUserID     string
	hub           *Hub
	conn          *websocket.Conn
	send          chan []byte
	subscriptions map[string]bool
	logger        *slog.Logger

	pingInterval time.Duration
	pingTimeout  time.Duration
	writeWait    time.Duration
	readLimit    int64
}

// ClientConfig holds configuration for client connections
type ClientConfig struct {
	PingInterval time.Duration
	PingTimeout  time.Duration
	WriteWait    time.Duration
	ReadLimit    int64
	SendBuffer   int
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		PingInterval: 2 * time.Second,
		PingTimeout:  30 * time.Second,
		WriteWait:    10 * time.Second,
		ReadLimit:    512 * 1024, // 512KB
		SendBuffer:   256,
	}
}

// NewClient creates a new client instance
func NewClient(hub *Hub, conn *websocket.Conn, config *ClientConfig, ajaibID, cfxUserID string, logger *slog.Logger) *Client {
	if config == nil {
		config = DefaultClientConfig()
	}

	return &Client{
		id:            uuid.New().String(),
		ajaibID:       ajaibID,
		cfxUserID:     cfxUserID,
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, config.SendBuffer),
		subscriptions: make(map[string]bool),
		logger:        logger,
		pingInterval:  config.PingInterval,
		pingTimeout:   config.PingTimeout,
		writeWait:     config.WriteWait,
		readLimit:     config.ReadLimit,
	}
}

// ID returns client's unique identifier
func (c *Client) ID() string {
	return c.id
}

// AjaibID returns Ajaib user ID parsed from JWT at connection time
func (c *Client) AjaibID() string {
	return c.ajaibID
}

// CfxUserID returns CFX user ID resolved from Ajaib ID at connection time
func (c *Client) CfxUserID() string {
	return c.cfxUserID
}

// ReadPump pumps messages from the WebSocket connection to hub
func (c *Client) ReadPump(handler MessageHandler) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(c.readLimit)
	c.conn.SetReadDeadline(time.Now().Add(c.pingTimeout))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.pingTimeout))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Warn("unexpected close error", "client_id", c.id, "error", err)
			}
			break
		}

		c.conn.SetReadDeadline(time.Now().Add(c.pingTimeout))

		if handler != nil {
			if err := handler.HandleMessage(c, message); err != nil {
				c.logger.Error("message handler error", "client_id", c.id, "error", err)
			}
		}
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(c.pingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.logger.Warn("failed to write message", "client_id", c.id, "error", err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send sends a message to client
func (c *Client) Send(data []byte) error {
	select {
	case c.send <- data:
		return nil
	default:
		return ErrClientBufferFull
	}
}

// SendMessage sends a protocol message to client
func (c *Client) SendMessage(msg *protocol.Message) error {
	data, err := msg.Encode()
	if err != nil {
		return err
	}
	return c.Send(data)
}

// Close closes client connection
func (c *Client) Close() {
	c.hub.unregister <- c
}

// MessageHandler handles incoming messages from clients
type MessageHandler interface {
	HandleMessage(client *Client, message []byte) error
}
