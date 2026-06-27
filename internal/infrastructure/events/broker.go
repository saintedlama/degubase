// Package events provides a in-process pub/sub broker for row mutations.
package events

import (
	"sync"

	"github.com/saintedlama/degubase/internal/models"
)

type RowEventType string

const (
	RowEventCreated RowEventType = "row.created"
	RowEventUpdated RowEventType = "row.updated"
	RowEventDeleted RowEventType = "row.deleted"
)

// RowEvent carries a row mutation for SSE consumers and automation triggers.
type RowEvent struct {
	Type            RowEventType `json:"type"`
	Row             *models.Row  `json:"row,omitempty"`
	ID              int64        `json:"id,omitempty"`
	TableID         int64        `json:"table_id,omitempty"`
	CommandSourceID string       `json:"command_source_id,omitempty"`
	CommandID       string       `json:"command_id,omitempty"`
	ExecutionID     int64        `json:"execution_id,omitempty"`
}

// Broker fans out row events to per-table subscribers.
type Broker struct {
	mu      sync.RWMutex
	subs    map[int64][]chan RowEvent
	allSubs []chan RowEvent
}

func NewBroker() *Broker {
	return &Broker{subs: make(map[int64][]chan RowEvent)}
}

// SubscribeAll returns a channel that receives every event across all tables.
func (b *Broker) SubscribeAll() chan RowEvent {
	ch := make(chan RowEvent, 64)
	b.mu.Lock()
	b.allSubs = append(b.allSubs, ch)
	b.mu.Unlock()
	return ch
}

// Subscribe returns a channel that receives events for the given table.
func (b *Broker) Subscribe(tableID int64) chan RowEvent {
	ch := make(chan RowEvent, 16)
	b.mu.Lock()
	b.subs[tableID] = append(b.subs[tableID], ch)
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscription and closes its channel.
func (b *Broker) Unsubscribe(tableID int64, ch chan RowEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	list := b.subs[tableID]
	for i, c := range list {
		if c == ch {
			b.subs[tableID] = append(list[:i], list[i+1:]...)
			close(ch)
			return
		}
	}
}

// Publish fans out an event to all subscribers for the given table.
func (b *Broker) Publish(tableID int64, ev RowEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	ev.TableID = tableID
	for _, ch := range b.subs[tableID] {
		select {
		case ch <- ev:
		default:
		}
	}
	for _, ch := range b.allSubs {
		select {
		case ch <- ev:
		default:
		}
	}
}
