package events

import (
	"log"
	"sync"
)

// subscriberBufferCap is the per-subscriber buffered-channel capacity.
//
// 64 is enough to absorb a normal burst (e.g. round.scored fan-out followed by
// round.started for the next round) without blocking the publisher. A
// subscriber that falls more than 64 events behind is almost certainly a
// broken/stalled connection — the publisher drops further events for that
// subscriber and logs a warning rather than blocking the rest of the room.
const subscriberBufferCap = 64

// roomChannelSet holds the set of subscriber channels for a single room.
//
// The slice is guarded by its own RWMutex so concurrent Publishes to the same
// room can fan out under an RLock without contending with each other; only
// Subscribe/Unsubscribe mutations need the write lock.
type roomChannelSet struct {
	mu  sync.RWMutex
	chs []chan Event
}

// PerRoomPubsub is the in-process fan-out used to drive SSE streams.
//
// It implements the Publisher interface from publisher.go. Subscribers are
// identified by an opaque channel; each Subscribe returns an unsubscribe
// closure that removes the channel and (if the room is then empty) drops the
// room entry from the outer map.
//
// Locking strategy:
//   - rooms is guarded by mu (RWMutex). Publish takes RLock; Subscribe and
//     Unsubscribe take Lock.
//   - Each room's subscriber slice is guarded by its own RWMutex, taken under
//     the outer RLock during Publish so fan-out doesn't block other rooms.
//
// The implementation deliberately favours throughput at publish time over
// latency at subscribe time — Subscribe/Unsubscribe are rare relative to
// Publish.
type PerRoomPubsub struct {
	mu    sync.RWMutex
	rooms map[string]*roomChannelSet
}

// NewPubsub constructs an empty PerRoomPubsub ready to accept Subscribe and
// Publish calls.
func NewPubsub() *PerRoomPubsub {
	return &PerRoomPubsub{
		rooms: make(map[string]*roomChannelSet),
	}
}

// Subscribe registers a new subscriber for roomID and returns the receive
// channel together with an idempotent unsubscribe closure. The returned
// channel has capacity subscriberBufferCap and is closed by the unsubscribe
// closure on its first invocation.
//
// Callers MUST invoke the unsubscribe closure (typically via defer) when they
// are done — failing to do so will leak the channel for the lifetime of the
// pubsub.
func (p *PerRoomPubsub) Subscribe(roomID string) (<-chan Event, func()) {
	ch := make(chan Event, subscriberBufferCap)

	p.mu.Lock()
	rcs, ok := p.rooms[roomID]
	if !ok {
		rcs = &roomChannelSet{}
		p.rooms[roomID] = rcs
	}
	rcs.mu.Lock()
	rcs.chs = append(rcs.chs, ch)
	rcs.mu.Unlock()
	p.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			p.unsubscribe(roomID, ch)
		})
	}
	return ch, unsubscribe
}

// unsubscribe removes ch from the room's subscriber slice, closes ch, and
// (if the room has no subscribers left) drops the room entry from the map.
func (p *PerRoomPubsub) unsubscribe(roomID string, ch chan Event) {
	p.mu.Lock()
	defer p.mu.Unlock()

	rcs, ok := p.rooms[roomID]
	if !ok {
		// Should not happen with a single-shot closure, but be defensive
		// — closing the channel is still required for callers ranging on it.
		close(ch)
		return
	}

	rcs.mu.Lock()
	for i, c := range rcs.chs {
		if c == ch {
			// Order does not matter; swap-with-last is O(1) and avoids a
			// full slice copy.
			last := len(rcs.chs) - 1
			rcs.chs[i] = rcs.chs[last]
			rcs.chs[last] = nil
			rcs.chs = rcs.chs[:last]
			break
		}
	}
	empty := len(rcs.chs) == 0
	rcs.mu.Unlock()

	close(ch)

	if empty {
		delete(p.rooms, roomID)
	}
}

// Publish fans ev out to every current subscriber of roomID. Delivery is
// non-blocking: if a subscriber's buffer is full the event is dropped for that
// subscriber (and a warning logged) rather than blocking the publisher.
//
// Publishing to a room with zero subscribers is a no-op — events are
// best-effort and there is no replay buffer.
//
// We hold the room's RLock for the entire fan-out. Unsubscribe takes the
// room's write lock before closing the channel, which means that for the
// duration of Publish no channel can be concurrently closed beneath us — this
// rules out the "send on closed channel" panic without serialising publishes
// against each other. The lock-hold window is bounded because every send uses
// select-default and therefore never blocks.
func (p *PerRoomPubsub) Publish(roomID string, ev Event) {
	p.mu.RLock()
	rcs, ok := p.rooms[roomID]
	p.mu.RUnlock()
	if !ok {
		return
	}

	rcs.mu.RLock()
	defer rcs.mu.RUnlock()
	for _, ch := range rcs.chs {
		select {
		case ch <- ev:
		default:
			log.Printf("events: dropped %q for slow subscriber in room %s (buffer full)", ev.Name, roomID)
		}
	}
}

// RoomIDs returns a snapshot of the room IDs currently tracked by the pubsub.
// Order is unspecified. Primarily useful for diagnostics + tests.
func (p *PerRoomPubsub) RoomIDs() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]string, 0, len(p.rooms))
	for k := range p.rooms {
		out = append(out, k)
	}
	return out
}

// Compile-time check that PerRoomPubsub satisfies the Publisher interface.
var _ Publisher = (*PerRoomPubsub)(nil)
