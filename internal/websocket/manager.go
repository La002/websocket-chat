package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/La002/websocket-chat/internal/pubsub"
	"github.com/rs/zerolog/log"

	"github.com/La002/websocket-chat/internal/auth"
	"github.com/gorilla/websocket"
)

var (
	websocketUpgrader = websocket.Upgrader{
		CheckOrigin:     checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Manager struct {
	sync.RWMutex
	redis         *pubsub.RedisPubSub
	otps          auth.RetentionMap
	handlers      map[string]EventHandler
	clientRoomMap map[*Client]*Room
	roomList      map[string]*Room
}

func NewManager(ctx context.Context, redis *pubsub.RedisPubSub) *Manager {
	m := &Manager{
		handlers:      make(map[string]EventHandler),
		otps:          auth.NewRetentionMap(ctx, 5*time.Second),
		clientRoomMap: make(map[*Client]*Room),
		roomList:      make(map[string]*Room),
		redis:         redis,
	}

	m.setupEventHandlers()
	m.setupRooms(ctx)
	return m
}

func (m *Manager) setupEventHandlers() {
	m.handlers[EventSendMessage] = SendMessage
	m.handlers[EventChangeRoom] = ChangeRoomHandler
}

func (m *Manager) setupRooms(ctx context.Context) {
	m.roomList = make(map[string]*Room)
	for i := 0; i < 10; i++ {
		roomName := fmt.Sprintf("%d", i)
		var room = NewRoom(ctx, roomName, m.redis)
		m.roomList[roomName] = room
		go room.Run(ctx)
	}
}

func ChangeRoomHandler(event Event, c *Client) error {
	var changeRoomEvent ChangeRoomEvent

	if err := json.Unmarshal(event.Payload, &changeRoomEvent); err != nil {
		return fmt.Errorf("Bad Payload in request: %v", err)
	}
	m := c.manager

	name := changeRoomEvent.Name
	if _, ok := m.roomList[name]; !ok {
		return fmt.Errorf("This room does not exist. Total rooms : %s", len(m.roomList))
	}

	oldRoom := m.clientRoomMap[c]
	oldRoom.PullClient(c)

	newRoom := m.roomList[name]
	m.clientRoomMap[c] = newRoom
	newRoom.SendClient(c)

	c.chatroom = name
	return nil
}

func SendMessage(event Event, c *Client) error {
	m := c.manager
	chatRoom := m.clientRoomMap[c]

	chatRoom.BroadCast(event)

	return nil
}

func (m *Manager) routeEvent(event Event, c *Client) error {
	if handler, ok := m.handlers[event.Type]; ok {
		if err := handler(event, c); err != nil {
			return err
		}
		return nil
	} else {
		return errors.New("there is no such event type")
	}
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	otp := r.URL.Query().Get("otp")
	if otp == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !m.otps.VerifyOTP(otp) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	log.Debug().Msg("new connection")
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to upgrade websocket connection")
		return
	}

	client := NewClient(conn, m)
	m.addClient(client)

	// Start client processes
	go client.readMessages()
	go client.writeMessages()
}

func (m *Manager) LoginHandler(w http.ResponseWriter, r *http.Request) {
	type userLoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var req userLoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Username == "lak" && req.Password == "123" {
		type response struct {
			OTP string `json:"otp"`
		}

		otp := m.otps.NewOTP()

		resp := response{
			OTP: otp.Key,
		}

		data, err := json.Marshal(resp)
		if err != nil {
			log.Error().Err(err).
				Str("username", req.Username).
				Str("otp", otp.Key).
				Msg("failed to marshal login response")
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
}

func (m *Manager) addClient(client *Client) {
	m.Lock()

	defer m.Unlock()
	defaultRoom := m.roomList["0"]
	m.clientRoomMap[client] = defaultRoom
	defaultRoom.SendClient(client)
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if room, ok := m.clientRoomMap[client]; ok {
		room.PullClient(client)
		client.connection.Close()
		delete(m.clientRoomMap, client)
		log.Debug().Str("chatroom", client.chatroom).Msg("client removed")
	}
}

// Shutdown gracefully closes all client connections
func (m *Manager) Shutdown() {
	log.Info().Int("client_count", len(m.clientRoomMap)).Msg("shutting down manager")

	m.Lock()
	clientsToClose := make([]*Client, 0, len(m.clientRoomMap))
	for client := range m.clientRoomMap {
		clientsToClose = append(clientsToClose, client)
	}
	m.Unlock()

	for _, client := range clientsToClose {
		close(client.egress)
	}

	time.Sleep(100 * time.Millisecond)

	log.Info().Msg("manager shutdown complete")
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	switch origin {
	case "http://localhost:8080":
		return true
	default:
		return false
	}
}
