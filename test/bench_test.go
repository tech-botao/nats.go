package test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats"
)

func BenchmarkPublishSpeed(b *testing.B) {
	b.StopTimer()
	s := RunDefaultServer()
	defer s.Shutdown()
	nc := NewDefaultConnection(b)
	defer nc.Close()
	b.StartTimer()

	msg := []byte("Hello World")

	for i := 0; i < b.N; i++ {
		if err := nc.Publish("foo", msg); err != nil {
			b.Fatalf("Error in benchmark during Publish: %v\n", err)
		}
	}
	// Make sure they are all processed.
	nc.Flush()
	b.StopTimer()
}

func BenchmarkPubSubSpeed(b *testing.B) {
	b.StopTimer()
	s := RunDefaultServer()
	defer s.Shutdown()
	nc := NewDefaultConnection(b)
	defer nc.Close()

	ch := make(chan bool)

	nc.Opts.AsyncErrorCB = func(nc *nats.Conn, s *nats.Subscription, err error) {
		b.Fatalf("Error : %v\n", err)
	}

	received := int32(0)

	nc.Subscribe("foo", func(m *nats.Msg) {
		atomic.AddInt32(&received, 1)
		if atomic.LoadInt32(&received) >= int32(b.N) {
			ch <- true
		}
	})

	msg := []byte("Hello World")

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		if err := nc.Publish("foo", msg); err != nil {
			b.Fatalf("Error in benchmark during Publish: %v\n", err)
		}
		// Don't overrun ourselves and be a slow consumer, server will cut us off
		if int32(i)-atomic.LoadInt32(&received) > 8192 {
			time.Sleep(100)
		}
	}

	// Make sure they are all processed.
	err := WaitTime(ch, 10*time.Second)
	if err != nil {
		b.Fatal("Timed out waiting for messages")
	} else if atomic.LoadInt32(&received) != int32(b.N) {
		b.Fatal(nc.LastError())
	}
	b.StopTimer()
}
