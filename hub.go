package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type wsMessage struct {
	Type   string   `json:"type"`
	Line   *LogLine `json:"line,omitempty"`
	Fields []string `json:"fields,omitempty"`
}

type Hub struct {
	mu       sync.RWMutex
	clients  map[*Client]struct{}
	upgrader websocket.Upgrader
}

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, fields []string, lines []LogLine) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 1024),
		done: make(chan struct{}),
	}
	h.add(client)

	go client.writePump()
	if !client.sendFields(fields) {
		return
	}
	for i := range lines {
		if !client.sendLine(lines[i]) {
			return
		}
	}
	client.readPump()
}

func (h *Hub) BroadcastLine(line LogLine) {
	msg, err := json.Marshal(wsMessage{Type: "line", Line: &line})
	if err != nil {
		log.Printf("marshal line message failed: %v", err)
		return
	}
	h.broadcast(msg)
}

func (h *Hub) BroadcastFields(fields []string) {
	msg, err := json.Marshal(wsMessage{Type: "fields", Fields: fields})
	if err != nil {
		log.Printf("marshal fields message failed: %v", err)
		return
	}
	h.broadcast(msg)
}

func (h *Hub) add(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = struct{}{}
}

func (h *Hub) remove(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		client.close()
	}
}

func (c *Client) close() {
	c.closeOnce.Do(func() {
		close(c.done)
	})
}

func (h *Hub) broadcast(msg []byte) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case <-client.done:
		case client.send <- msg:
		default:
			h.remove(client)
			_ = client.conn.Close()
		}
	}
}

func (c *Client) sendFields(fields []string) bool {
	msg, err := json.Marshal(wsMessage{Type: "fields", Fields: fields})
	if err != nil {
		log.Printf("marshal initial fields failed: %v", err)
		return true
	}
	return c.sendMessage(msg)
}

func (c *Client) sendLine(line LogLine) bool {
	msg, err := json.Marshal(wsMessage{Type: "line", Line: &line})
	if err != nil {
		log.Printf("marshal initial line failed: %v", err)
		return true
	}
	return c.sendMessage(msg)
}

func (c *Client) sendMessage(msg []byte) bool {
	select {
	case c.send <- msg:
		return true
	case <-c.done:
		return false
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.remove(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(1024)
	_ = c.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
		return nil
	})

	for {
		if _, _, err := c.conn.NextReader(); err != nil {
			return
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.hub.remove(c)
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.done:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
			return
		}
	}
}
