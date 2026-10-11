package sqlite

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Suhaibinator/SuhaibMessageQueue/internal/queue"
)

// Each concurrent producer must receive the offset of its own message.
func TestConcurrentAddMessageReturnsOwnOffset(t *testing.T) {
	driver, err := NewDBDriver(filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatalf("NewDBDriver() error = %v", err)
	}
	defer driver.Close()
	if err := driver.CreateTopic("events"); err != nil {
		t.Fatalf("CreateTopic() error = %v", err)
	}

	const writers, perWriter = 16, 100
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				data := fmt.Sprintf("%d-%d", w, i)
				offset, err := driver.AddMessage("events", []byte(data))
				if err != nil {
					errs <- err
					return
				}
				got, err := driver.GetMessageAtOffset("events", offset)
				if err != nil {
					errs <- err
					return
				}
				if string(got) != data {
					errs <- fmt.Errorf("offset %d holds %q, returned for %q", offset, got, data)
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

func TestOperationsAfterCloseReturnErrStoreClosed(t *testing.T) {
	driver, err := NewDBDriver(":memory:")
	if err != nil {
		t.Fatalf("NewDBDriver() error = %v", err)
	}
	if err := driver.CreateTopic("events"); err != nil {
		t.Fatalf("CreateTopic() error = %v", err)
	}
	if err := driver.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	checks := map[string]error{}
	_, checks["AddMessage"] = driver.AddMessage("events", []byte("x"))
	_, checks["GetLatestOffset"] = driver.GetLatestOffset("events")
	_, checks["ReadMessages"] = driver.ReadMessages(context.Background(), "events", 1, 10, 0)
	checks["DeleteMessagesUntilOffset"] = driver.DeleteMessagesUntilOffset("events", 1)
	checks["CreateTopic"] = driver.CreateTopic("other")
	checks["DeleteTopic"] = driver.DeleteTopic("events")
	for name, err := range checks {
		if !errors.Is(err, queue.ErrStoreClosed) {
			t.Errorf("%s after Close error = %v, want ErrStoreClosed", name, err)
		}
	}
}
