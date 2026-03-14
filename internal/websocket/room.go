package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Room struct {
	name string
	sync.RWMutex
	join      chan *Client
	leave     chan *Client
	broadcast chan Event
	clients   map[*Client]bool
}

func NewRoom(name string) *Room {
	return &Room{
		name:      name,
		join:      make(chan *Client),
		leave:     make(chan *Client),
		clients:   make(map[*Client]bool),
		broadcast: make(chan Event, 100),
	}
}

func (r *Room) Run(ctx context.Context) {
	for {
		select {
		case c, ok := <-r.join:
			if !ok {
				// Do what here?
			}

			r.addClient(c)

		case c, ok := <-r.leave:
			if !ok {

			}

			r.removeClient(c)

		// does event contain the sender or no? What type of message are we expecting?
		case event, ok := <-r.broadcast:
			if !ok {

			}
			err := r.broadcastMessage(event)
			if err != nil {
				log.Err(err).Msg("Failed to broadcast")
			} else {
				log.Debug().Msg("Broadcast done")
			}
		}
	}
}

func (r *Room) addClient(c *Client) {
	r.Lock()
	defer r.Unlock()

	r.clients[c] = true
}

func (r *Room) removeClient(c *Client) {
	r.Lock()
	defer r.Unlock()

	//c.CloseConnection()
	delete(r.clients, c)
}

func (r *Room) broadcastMessage(event Event) error {
	var chatEvent SendMessageEvent

	if err := json.Unmarshal(event.Payload, &chatEvent); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}

	var broadcastMessage NewMessageEvent

	broadcastMessage.Sent = time.Now()
	broadcastMessage.Message = chatEvent.Message
	broadcastMessage.From = chatEvent.From

	data, err := json.Marshal(broadcastMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal the broad message: %v", err)
	}

	outgoingEvent := Event{
		Payload: data,
		Type:    EventNewMessage,
	}

	for client := range r.clients {
		select {
		case client.egress <- outgoingEvent:
		default:
			log.Warn().Msg("slow consumer, disconnecting")
			close(client.egress)
			delete(r.clients, client)
		}

	}

	return nil
}

func (r *Room) SendClient(c *Client) {
	r.join <- c
}

func (r *Room) PullClient(c *Client) {
	r.leave <- c
}

func (r *Room) BroadCast(event Event) {
	select {
	case r.broadcast <- event:
	default:
		log.Warn().Msg("queue full, dropping message")

	}
}
