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
	Type       string          `json:"type"`
	Line       *LogLine        `json:"line,omitempty"`
	Fields     []string        `json:"fields,omitempty"`
	Source     *SourceState    `json:"source,omitempty"`
	TimeRange  *TimeRange      `json:"timeRange,omitempty"`
	Error      *QueryError     `json:"error,omitempty"`
	Resources  []AzureResource `json:"resources,omitempty"`
	AzureError string          `json:"errorMessage,omitempty"`
	Loading    *LoadingState   `json:"loading,omitempty"`
	Total      *int            `json:"total,omitempty"`
}

type clientCommand struct {
	Type       string     `json:"type"`
	Query      string     `json:"query,omitempty"`
	Source     string     `json:"source,omitempty"`
	TimeRange  *TimeRange `json:"timeRange,omitempty"`
	ResourceID string     `json:"resourceId,omitempty"`
}

type CommandHandler func(*Client, clientCommand)

type Hub struct {
	mu            sync.RWMutex
	clients       map[*Client]struct{}
	handleCommand CommandHandler
	upgrader      websocket.Upgrader
}

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

func NewHub(handleCommand CommandHandler) *Hub {
	return &Hub{
		clients:       make(map[*Client]struct{}),
		handleCommand: handleCommand,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, snapshot ServerSnapshot) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 4096),
		done: make(chan struct{}),
	}
	h.add(client)

	go client.writePump()
	if !client.sendSnapshot(snapshot) {
		return
	}
	client.readPump()
}

func (h *Hub) BroadcastLine(line LogLine) {
	h.broadcastJSON(wsMessage{Type: "line", Line: &line})
}

func (h *Hub) BroadcastFields(fields []string) {
	h.broadcastJSON(wsMessage{Type: "fields", Fields: fields})
}

func (h *Hub) BroadcastSource(source SourceState) {
	h.broadcastJSON(wsMessage{Type: "source", Source: &source})
}

func (h *Hub) BroadcastTimeRange(timeRange TimeRange) {
	h.broadcastJSON(wsMessage{Type: "timeRange", TimeRange: &timeRange})
}

func (h *Hub) BroadcastResultsReset(total int) {
	h.broadcastJSON(wsMessage{Type: "results.reset", Total: &total})
}

func (h *Hub) BroadcastResultsStats(total int) {
	h.broadcastJSON(wsMessage{Type: "results.stats", Total: &total})
}

func (h *Hub) BroadcastQueryError(queryError QueryError) {
	h.broadcastJSON(wsMessage{Type: "query.error", Error: &queryError})
}

func (h *Hub) BroadcastAzureResources(resources []AzureResource, errorMessage string) {
	h.broadcastJSON(wsMessage{Type: "azure.resources", Resources: resources, AzureError: errorMessage})
}

func (h *Hub) BroadcastLoading(loading LoadingState) {
	h.broadcastJSON(wsMessage{Type: "loading", Loading: &loading})
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

func (h *Hub) broadcastJSON(message wsMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("marshal websocket message failed: %v", err)
		return
	}
	h.broadcast(data)
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

func (c *Client) sendSnapshot(snapshot ServerSnapshot) bool {
	if !c.sendJSON(wsMessage{Type: "source", Source: &snapshot.Source}) {
		return false
	}
	if !c.sendJSON(wsMessage{Type: "timeRange", TimeRange: &snapshot.TimeRange}) {
		return false
	}
	if snapshot.Loading.Active {
		if !c.sendJSON(wsMessage{Type: "loading", Loading: &snapshot.Loading}) {
			return false
		}
	}
	if len(snapshot.AzureResources) > 0 || snapshot.AzureError != "" {
		if !c.sendJSON(wsMessage{Type: "azure.resources", Resources: snapshot.AzureResources, AzureError: snapshot.AzureError}) {
			return false
		}
	}
	if !c.sendJSON(wsMessage{Type: "fields", Fields: snapshot.Fields}) {
		return false
	}
	if !c.sendJSON(wsMessage{Type: "results.reset", Total: &snapshot.Total}) {
		return false
	}
	for i := range snapshot.Lines {
		if !c.sendJSON(wsMessage{Type: "line", Line: &snapshot.Lines[i]}) {
			return false
		}
	}
	return true
}

func (c *Client) sendJSON(message wsMessage) bool {
	msg, err := json.Marshal(message)
	if err != nil {
		log.Printf("marshal websocket message failed: %v", err)
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

	c.conn.SetReadLimit(64 * 1024)
	_ = c.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var command clientCommand
		if err := json.Unmarshal(data, &command); err != nil {
			log.Printf("invalid websocket command: %v", err)
			continue
		}
		if c.hub.handleCommand != nil {
			c.hub.handleCommand(c, command)
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
