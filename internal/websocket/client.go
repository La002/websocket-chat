package websocket

import (
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gorilla/websocket"
)

var (
	pongWait     = 10 * time.Second
	pingInterval = (pongWait * 9) / 10
)

type Client struct {
	connection *websocket.Conn
	manager    *Manager
	chatroom   string
	egress     chan Event
}

func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		egress:     make(chan Event, 50),
	}
}

func (c *Client) readMessages() {
	defer func() {
		//cleanup connection
		c.manager.removeClient(c)
	}()

	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.
			Err(err).
			Str("chatroom", c.chatroom).
			Msg("failed to set read-deadline")
		return
	}

	c.connection.SetReadLimit(512)

	c.connection.SetPongHandler(c.pongHandler)

	for {
		_, payload, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Str("chatroom", c.chatroom).Msg("error reading message")
			}
			break
		}

		var request Event

		if err := json.Unmarshal(payload, &request); err != nil {
			log.Error().Err(err).Str("chatroom", c.chatroom).Msg("error in marshalling event")
			break
		}

		if err := c.manager.routeEvent(request, c); err != nil {
			log.Error().Err(err).Str("chatroom", c.chatroom).Msg("error in handling message")
		}
	}
}

func (c *Client) writeMessages() {
	ticker := time.NewTicker(pingInterval)

	defer func() {
		c.manager.removeClient(c)
		ticker.Stop()
	}()

	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				c.connection.SetWriteDeadline(time.Now().Add(time.Second))
				if err := c.connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
					log.Error().Err(err).Str("chatroom", c.chatroom).Msg("connection closed")
				}
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Error().Err(err).Str("chatroom", c.chatroom).Msg("error in marshalling")
				continue
			}

			c.connection.SetWriteDeadline(time.Now().Add(time.Second))
			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Error().Err(err).Str("chatroom", c.chatroom).Msg("failed to send message")
				return
			}

			log.Debug().Msg("message sent")

		case <-ticker.C:
			log.Debug().Msg("ping")

			c.connection.SetWriteDeadline(time.Now().Add(time.Second))
			// Send ping to client (browser)
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				log.Error().Err(err).Str("chatroom", c.chatroom).Msg("error in writemsg")
				return
			}

		}

	}
}

func (c *Client) pongHandler(pongMsg string) error {
	log.Debug().Msg("pong")
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}

func (c *Client) CloseConnection() error {
	return c.connection.Close()
}
