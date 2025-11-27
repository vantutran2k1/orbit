package socket

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/vantutran2k1/orbit/internal/adapter/handler/middleware"
	"github.com/vantutran2k1/orbit/internal/adapter/storage/redis"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type TunnelMessage struct {
	JobID   string         `json:"job_id"`
	Payload map[string]any `json:"payload"`
}

type Hub struct {
	clients map[uuid.UUID]*websocket.Conn
	mu      sync.RWMutex
}

var GlobalHub = &Hub{
	clients: make(map[uuid.UUID]*websocket.Conn),
}

func (h *Hub) HandleConnection(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	h.register(tenantID, conn)
	defer h.unregister(tenantID)

	go h.subscribeToRedis(tenantID, conn)

	log.Printf("agent connected for tenant: %s", tenantID)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (h *Hub) subscribeToRedis(tenantID uuid.UUID, conn *websocket.Conn) {
	ctx := context.Background()

	channel := "orbit:tunnel:" + tenantID.String()
	pubsub := redis.RDB.Subscribe(ctx, channel)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
			log.Printf("error writing to ws: %v", err)
			return
		}
	}
}

func (h *Hub) register(tenantID uuid.UUID, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[tenantID] = conn
	log.Printf("after register tenant: %d", len(h.clients))
}

func (h *Hub) unregister(tenantID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conn, ok := h.clients[tenantID]; ok {
		conn.Close()
		delete(h.clients, tenantID)
	}
	log.Printf("agent disconnected for tenant: %s", tenantID)
}
