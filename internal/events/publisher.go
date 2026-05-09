// Package events defines the cross-package interface for room-scoped event
// fan-out. The concrete pubsub implementation (and the SSE handler that
// subscribes to it) lives in a sibling file owned by the events agent — this
// file is the contract every other package programs against.
package events

// Event is one named payload broadcast to subscribers of a single room.
//
// Name follows the dotted convention used in SPEC.md §5 (e.g. "round.started").
// Data is a value the SSE handler can JSON-encode; concrete types live near
// each producer rather than being centralised here.
type Event struct {
	Name string
	Data interface{}
}

// Publisher is the only surface other packages depend on. The concrete
// PerRoomPubsub implementation will satisfy this; tests and standalone tooling
// can use NoopPublisher instead.
//
// Implementations MUST be safe for concurrent use from many goroutines.
// Implementations MUST NOT block the caller waiting for slow subscribers —
// fan-out is best-effort with bounded buffers per subscriber.
type Publisher interface {
	Publish(roomID string, ev Event)
}

// NoopPublisher is a stand-in for tests and the bootstrap server start before
// the events agent wires in PerRoomPubsub. Calls are silently dropped.
type NoopPublisher struct{}

// Publish implements Publisher; it does nothing.
func (NoopPublisher) Publish(string, Event) {}
