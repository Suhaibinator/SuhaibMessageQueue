package embedded_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Suhaibinator/SuhaibMessageQueue/embedded"
)

// This example uses a Queue as a durable outbox: events are published
// locally, forwarded in bounded batches, and deleted once acknowledged.
func Example() {
	dir, err := os.MkdirTemp("", "smq-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	ctx := context.Background()
	q, err := embedded.Open(filepath.Join(dir, "outbox.db"), embedded.Options{})
	if err != nil {
		log.Fatal(err)
	}
	defer q.Close()

	if err := q.CreateTopic(ctx, "events"); err != nil {
		log.Fatal(err)
	}
	if _, err := q.PublishBatch(ctx, "events", [][]byte{
		[]byte("joined"), []byte("message"), []byte("left"),
	}); err != nil {
		log.Fatal(err)
	}

	// Forward from the earliest retained offset, two events at a time.
	next, err := q.EarliestOffset(ctx, "events")
	if err != nil {
		log.Fatal(err)
	}
	for {
		batch, err := q.Read(ctx, "events", next, embedded.ReadOptions{Limit: 2, MaxBytes: 1 << 20})
		if err != nil {
			log.Fatal(err)
		}
		if len(batch) == 0 {
			break
		}
		for _, msg := range batch {
			fmt.Printf("send %d %s\n", msg.Offset, msg.Data)
		}
		acked := batch[len(batch)-1].Offset // after the upstream acknowledges
		if err := q.DeleteUntilOffset(ctx, "events", acked); err != nil {
			log.Fatal(err)
		}
		next = acked + 1
	}

	stats, err := q.Stats(ctx, "events")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("pending:", stats.Count)
	// Output:
	// send 1 joined
	// send 2 message
	// send 3 left
	// pending: 0
}
