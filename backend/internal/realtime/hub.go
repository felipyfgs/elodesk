package realtime

import (
	"fmt"
	"sync"

	"github.com/gofiber/contrib/websocket"

	"backend/internal/logger"
)

type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	userID  int64
	rooms   map[string]struct{}
	send    chan []byte
	mu      sync.RWMutex
	checker MembershipChecker
}

type broadcastEvent struct {
	room    string
	message []byte
}

type Hub struct {
	clients    map[*Client]struct{}
	rooms      map[string]map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan broadcastEvent
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		rooms:      make(map[string]map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan broadcastEvent, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = struct{}{}
			h.mu.Unlock()
			logger.Debug().Str("component", "realtime").Int64("userId", client.userID).Msg("client registered")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.mu.RLock()
				for room := range client.rooms {
					if roomClients, ok := h.rooms[room]; ok {
						delete(roomClients, client)
						if len(roomClients) == 0 {
							delete(h.rooms, room)
						}
					}
				}
				client.mu.RUnlock()
				close(client.send)
			}
			h.mu.Unlock()
			logger.Debug().Str("component", "realtime").Int64("userId", client.userID).Msg("client unregistered")

		case event := <-h.broadcast:
			h.mu.RLock()
			roomClients, ok := h.rooms[event.room]
			if !ok {
				h.mu.RUnlock()
				continue
			}
			for client := range roomClients {
				select {
				case client.send <- event.message:
				default:
					go func(c *Client) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) Broadcast(room string, message []byte) {
	h.broadcast <- broadcastEvent{room: room, message: message}
}

func (h *Hub) JoinRoom(client *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client.mu.Lock()
	client.rooms[room] = struct{}{}
	client.mu.Unlock()

	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*Client]struct{})
	}
	h.rooms[room][client] = struct{}{}
	logger.Debug().Str("component", "realtime").Int64("userId", client.userID).Str("room", room).Msg("client joined room")
}

func (h *Hub) LeaveRoom(client *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client.mu.Lock()
	delete(client.rooms, room)
	client.mu.Unlock()

	if roomClients, ok := h.rooms[room]; ok {
		delete(roomClients, client)
		if len(roomClients) == 0 {
			delete(h.rooms, room)
		}
	}
	logger.Debug().Str("component", "realtime").Int64("userId", client.userID).Str("room", room).Msg("client left room")
}

func AccountRoom(accountID int64) string {
	return fmt.Sprintf("account:%d", accountID)
}

func InboxRoom(inboxID int64) string {
	return fmt.Sprintf("inbox:%d", inboxID)
}

func ConversationRoom(conversationID int64) string {
	return fmt.Sprintf("conversation:%d", conversationID)
}
