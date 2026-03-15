package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/La002/websocket-chat/internal/pubsub"
	"github.com/rs/zerolog/log"
)

type Room struct {
	name string
	sync.RWMutex
	join    chan *Client
	leave   chan *Client
	clients map[*Client]bool
	pubsub  *pubsub.RedisPubSub
	ctx     context.Context
}

func NewRoom(ctx context.Context, name string, redis *pubsub.RedisPubSub) *Room {
	return &Room{
		name:    name,
		join:    make(chan *Client),
		leave:   make(chan *Client),
		clients: make(map[*Client]bool),
		pubsub:  redis,
		ctx:     ctx,
	}
}

func (r *Room) Run(ctx context.Context) {
	r.pubsub.Subscribe(ctx, r.name, r.handleRedisMessage)
	for {
		select {
		case c, ok := <-r.join:
			if !ok {
				// Do what here?
			}

			r.addClient(ctx, c)

		case c, ok := <-r.leave:
			if !ok {

			}

			r.removeClient(c)
		}
	}
}

func (r *Room) handleRedisMessage(message []byte) {
	var event Event
	json.Unmarshal(message, &event)

	r.RLock()
	defer r.RUnlock()
	for client := range r.clients {
		select {
		case client.egress <- event:
		default:
			// slow consumer
		}
	}
}

func (r *Room) addClient(ctx context.Context, c *Client) {
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
	var chatEvent SendMessageEvent
	if err := json.Unmarshal(event.Payload, &chatEvent); err != nil {
		log.Err(err).Msg("failed to unmarshal chat event")
		return
	}

	broadcastMessage := NewMessageEvent{
		Sent: time.Now(),
		SendMessageEvent: SendMessageEvent{
			Message: chatEvent.Message,
			From:    chatEvent.From,
		},
	}

	payload, err := json.Marshal(broadcastMessage)
	if err != nil {
		log.Err(err).Msg("failed to marshal broadcast message")
		return
	}

	outgoingEvent := Event{
		Type:    EventNewMessage,
		Payload: payload,
	}

	data, _ := json.Marshal(outgoingEvent)
	r.pubsub.Publish(r.ctx, r.name, data)
}
