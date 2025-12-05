package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/jackc/pgx/v5"
)

type TableEvent struct {
	Table  string          `json:"table"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type EventBroker struct {
	subs map[chan string]struct{}
	lock sync.RWMutex
}

func NewEventBroker() *EventBroker {
	return &EventBroker{
		subs: make(map[chan string]struct{}),
	}
}

func (b *EventBroker) Subscribe() chan string {
	ch := make(chan string, 10)
	b.lock.Lock()
	b.subs[ch] = struct{}{}
	b.lock.Unlock()
	return ch
}

func (b *EventBroker) Unsubscribe(ch chan string) {
	b.lock.Lock()
	delete(b.subs, ch)
	close(ch)
	b.lock.Unlock()
}

func (b *EventBroker) Publish(event string) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	for ch := range b.subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func ListenPostgres(
	ctx context.Context,
	dsn string,
	broker *EventBroker,
	tables []string,
) error {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to Postgres: %w", err)
	}

	for _, table := range tables {
		channel := table + "_events"
		_, err := conn.Exec(ctx, "LISTEN "+channel)
		if err != nil {
			return fmt.Errorf(
				"failed to listen on channel %s: %w", channel, err,
			)
		}
		log.Printf("Listening on Postgres channel: %s", channel)
	}

	go func() {
		defer conn.Close(ctx)

		for {
			notification, err := conn.WaitForNotification(ctx)
			if err != nil {
				select {
				case <-ctx.Done():
					log.Println("Stopping Postgres listener")
					return
				default:
					log.Printf(
						"Error while waiting for notification: %v", err,
					)
					continue
				}
			}

			var event TableEvent
			if err := json.Unmarshal([]byte(notification.Payload), &event); err != nil {
				log.Printf(
					"Failed to unmarshal notification payload: %v", err,
				)
				continue
			}

			broker.Publish(notification.Payload)
		}
	}()

	return nil
}
