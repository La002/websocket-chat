package pubsub

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type RedisPubSub struct {
	client *redis.Client
}

func NewRedisPubSub(addr string) (*RedisPubSub, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Info().Str("addr", addr).Msg("connected to redis")
	return &RedisPubSub{
		client: client,
	}, nil
}

func (r *RedisPubSub) Publish(ctx context.Context, room string, message []byte) error {
	channel := fmt.Sprintf("room:%s", room)
	return r.client.Publish(ctx, channel, message).Err()
}

func (r *RedisPubSub) Subscribe(ctx context.Context, room string, handler func([]byte)) {
	channel := fmt.Sprintf("room:%s", room)
	sub := r.client.Subscribe(ctx, channel)

	go func() {
		defer sub.Close()

		ch := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				handler([]byte(msg.Payload))
			}
		}
	}()

	log.Debug().Str("channel", channel).Msg("subscribed to redis channel")
}

func (r *RedisPubSub) Close() error {
	return r.client.Close()
}
