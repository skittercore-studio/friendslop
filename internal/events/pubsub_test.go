package events

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// recv pulls one event from ch with a generous timeout so a flake doesn't hang
// the suite. Using a long-ish timeout because under -race goroutine scheduling
// can be slow.
func recv(t *testing.T, ch <-chan Event) (Event, bool) {
	t.Helper()
	select {
	case ev, ok := <-ch:
		return ev, ok
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for event")
		return Event{}, false
	}
}

func TestPubsub_SingleSubscriberReceives(t *testing.T) {
	p := NewPubsub()
	ch, unsub := p.Subscribe("room-A")
	defer unsub()

	want := Event{Name: "round.started", Data: map[string]any{"round_number": 1}}
	p.Publish("room-A", want)

	got, ok := recv(t, ch)
	if !ok {
		t.Fatal("channel closed before event")
	}
	if got.Name != want.Name {
		t.Fatalf("got name=%q, want %q", got.Name, want.Name)
	}
}

func TestPubsub_FanOutMultipleSubscribers(t *testing.T) {
	p := NewPubsub()
	const n = 5
	chs := make([]<-chan Event, n)
	unsubs := make([]func(), n)
	for i := 0; i < n; i++ {
		chs[i], unsubs[i] = p.Subscribe("room-fan")
	}
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	ev := Event{Name: "player.joined", Data: map[string]any{"player_id": "p1"}}
	p.Publish("room-fan", ev)

	for i, ch := range chs {
		got, ok := recv(t, ch)
		if !ok {
			t.Fatalf("subscriber %d: channel closed", i)
		}
		if got.Name != ev.Name {
			t.Fatalf("subscriber %d: got %q want %q", i, got.Name, ev.Name)
		}
	}
}

func TestPubsub_SlowSubscriberDoesNotBlockOthers(t *testing.T) {
	p := NewPubsub()

	// "Slow" subscriber: never reads. We'll fill its buffer + overflow.
	slow, unsubSlow := p.Subscribe("room-S")
	defer unsubSlow()
	// "Fast" subscriber: drains promptly.
	fast, unsubFast := p.Subscribe("room-S")
	defer unsubFast()

	// Drain fast in the background so its buffer can't fill.
	fastReceived := make(chan int, 1)
	done := make(chan struct{})
	go func() {
		count := 0
		for {
			select {
			case <-fast:
				count++
				if count == 5 {
					fastReceived <- count
				}
			case <-done:
				return
			}
		}
	}()
	defer close(done)

	// Stuff well past the 64-event buffer cap so the slow subscriber drops.
	// If the publisher were blocking on slow, fast would never see all 5
	// new events arrive after the buffer is full.
	const overflow = subscriberBufferCap + 32
	for i := 0; i < overflow; i++ {
		p.Publish("room-S", Event{Name: "spam"})
	}
	for i := 0; i < 5; i++ {
		p.Publish("room-S", Event{Name: "post-overflow"})
	}

	select {
	case got := <-fastReceived:
		if got < 5 {
			t.Fatalf("fast subscriber only saw %d events; publisher must have blocked", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("fast subscriber starved — slow subscriber blocked publisher")
	}

	// The slow channel's buffer should be full (cap items waiting); we never
	// expect more than cap to be queued. Drain a few to confirm.
	drained := 0
drainLoop:
	for {
		select {
		case <-slow:
			drained++
		default:
			break drainLoop
		}
	}
	if drained == 0 {
		t.Fatal("slow channel had no buffered events; expected ~cap")
	}
	if drained > subscriberBufferCap {
		t.Fatalf("slow channel had %d buffered, expected <= cap=%d", drained, subscriberBufferCap)
	}
}

func TestPubsub_UnsubscribeClosesChannelAndPublishDoesNotPanic(t *testing.T) {
	p := NewPubsub()
	ch, unsub := p.Subscribe("room-U")

	unsub()
	// Channel should be closed.
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected closed channel, got value")
		}
	case <-time.After(time.Second):
		t.Fatal("expected closed channel to be readable immediately")
	}

	// Idempotent unsubscribe — second call must not panic.
	unsub()

	// Publishing post-unsubscribe must not panic and should be a no-op.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("publish after unsubscribe panicked: %v", r)
		}
	}()
	p.Publish("room-U", Event{Name: "ghost"})

	// The room should have been dropped from the inner map.
	p.mu.RLock()
	_, exists := p.rooms["room-U"]
	p.mu.RUnlock()
	if exists {
		t.Fatal("room entry should have been dropped after last unsubscribe")
	}
}

func TestPubsub_RoomsAreIsolated(t *testing.T) {
	p := NewPubsub()
	chA, unsubA := p.Subscribe("room-A")
	defer unsubA()
	chB, unsubB := p.Subscribe("room-B")
	defer unsubB()

	p.Publish("room-A", Event{Name: "for-A"})

	// B must not receive A's event.
	select {
	case ev := <-chB:
		t.Fatalf("room-B leaked event from room-A: %+v", ev)
	case <-time.After(50 * time.Millisecond):
		// expected — B is silent
	}

	// A receives normally.
	got, ok := recv(t, chA)
	if !ok {
		t.Fatal("room-A channel closed")
	}
	if got.Name != "for-A" {
		t.Fatalf("got %q want for-A", got.Name)
	}

	// Now publish to B and confirm A stays quiet.
	p.Publish("room-B", Event{Name: "for-B"})
	got, ok = recv(t, chB)
	if !ok {
		t.Fatal("room-B channel closed")
	}
	if got.Name != "for-B" {
		t.Fatalf("got %q want for-B", got.Name)
	}
	select {
	case ev := <-chA:
		t.Fatalf("room-A leaked event from room-B: %+v", ev)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestPubsub_RaceConcurrentPubSub(t *testing.T) {
	// Run with -race to catch data races. We exercise:
	//   - 100 publishers fanning to the same room concurrently
	//   - 100 subscribers churning subscribe/unsubscribe concurrently
	// No deadlock, no panic, and every active subscriber's drain goroutine
	// must complete within the test budget.
	p := NewPubsub()

	const (
		nPub = 100
		nSub = 100
	)

	stop := make(chan struct{})

	var pubWG sync.WaitGroup
	var pubs int64
	for i := 0; i < nPub; i++ {
		pubWG.Add(1)
		go func() {
			defer pubWG.Done()
			for {
				select {
				case <-stop:
					return
				default:
					p.Publish("hot-room", Event{Name: "tick"})
					atomic.AddInt64(&pubs, 1)
				}
			}
		}()
	}

	var subWG sync.WaitGroup
	for i := 0; i < nSub; i++ {
		subWG.Add(1)
		go func() {
			defer subWG.Done()
			for j := 0; j < 10; j++ {
				ch, unsub := p.Subscribe("hot-room")
				// Drain a handful of events, then unsubscribe.
				deadline := time.After(50 * time.Millisecond)
			drain:
				for {
					select {
					case <-ch:
					case <-deadline:
						break drain
					}
				}
				unsub()
			}
		}()
	}

	// Let the storm run briefly.
	time.Sleep(300 * time.Millisecond)
	close(stop)

	// All goroutines must wind down promptly.
	doneSubs := make(chan struct{})
	go func() { subWG.Wait(); close(doneSubs) }()
	select {
	case <-doneSubs:
	case <-time.After(5 * time.Second):
		t.Fatal("subscribers did not finish — possible deadlock")
	}

	donePubs := make(chan struct{})
	go func() { pubWG.Wait(); close(donePubs) }()
	select {
	case <-donePubs:
	case <-time.After(5 * time.Second):
		t.Fatal("publishers did not finish — possible deadlock")
	}

	if atomic.LoadInt64(&pubs) == 0 {
		t.Fatal("no publishes recorded — race test was a no-op")
	}

	// After all unsubscribes the room map should be empty.
	p.mu.RLock()
	leftover := len(p.rooms)
	p.mu.RUnlock()
	if leftover != 0 {
		t.Fatalf("expected empty rooms map, got %d entries", leftover)
	}
}
