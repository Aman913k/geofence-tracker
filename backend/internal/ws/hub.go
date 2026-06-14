package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	sendBufferSize = 32
)

// Hub maintains the set of connected alert clients and broadcasts messages to
// all of them. It is safe for concurrent use and managed via its Run goroutine.
type Hub struct {
	register   chan *client
	unregister chan *client
	broadcast  chan []byte

	mu      sync.RWMutex
	clients map[*client]struct{}
}

type client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// NewHub constructs an empty hub. Call Run to start its event loop.
func NewHub() *Hub {
	return &Hub{
		register:   make(chan *client),
		unregister: make(chan *client),
		broadcast:  make(chan []byte, 64),
		clients:    make(map[*client]struct{}),
	}
}

// Run processes register/unregister/broadcast events until ctx is cancelled.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.mu.Lock()
			for c := range h.clients {
				close(c.send)
				c.conn.Close()
				delete(h.clients, c)
			}
			h.mu.Unlock()
			return
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					// Slow client: drop it rather than blocking the hub.
					close(c.send)
					delete(h.clients, c)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast marshals v and queues it for delivery to every connected client.
// It never blocks the caller (the HTTP request path) for more than a buffered send.
func (h *Hub) Broadcast(v any) {
	msg, err := json.Marshal(v)
	if err != nil {
		log.Printf("ws: marshal broadcast: %v", err)
		return
	}
	select {
	case h.broadcast <- msg:
	default:
		log.Printf("ws: broadcast buffer full, dropping message")
	}
}

// ClientCount returns the number of currently connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Alerts are broadcast-only and public; allow any origin so the deployed
	// frontend (different host) can subscribe.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWS upgrades an HTTP request to a WebSocket connection and registers it.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: upgrade failed: %v", err)
		return
	}
	c := &client{hub: h, conn: conn, send: make(chan []byte, sendBufferSize)}
	h.register <- c

	go c.writePump()
	go c.readPump()
}

// readPump drains inbound frames (we expect none) and keeps the connection
// alive via pong handlers; it unregisters the client on close.
func (c *client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

// writePump fans queued messages out to the socket and sends periodic pings.
func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
