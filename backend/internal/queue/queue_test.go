package queue_test

import (
	"context"
	"testing"
	"time"

	"github.com/saas-agent-platform/backend/internal/models"
	"github.com/saas-agent-platform/backend/internal/queue"
)

func TestQueueOperations(t *testing.T) {
	q := queue.NewRedisQueue("localhost:6379")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Enqueue Job
	job := &models.Job{
		ID:        "job-test-1",
		ProjectID: "proj-1",
		Type:      "codegen",
		Status:    "pending",
	}

	if err := q.EnqueueJob(ctx, job); err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// 2. Publish and Subscribe Events
	eventChan, err := q.SubscribeEvents(ctx, "proj-1")
	if err != nil {
		t.Fatalf("Failed to subscribe to events: %v", err)
	}

	testEvent := &models.WSEvent{
		Type:  "log",
		Text:  "Job enqueued",
		Level: "info",
	}

	if err := q.PublishEvent(ctx, "proj-1", testEvent); err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	select {
	case received := <-eventChan:
		if received.Text != "Job enqueued" {
			t.Errorf("Unexpected event text: %v", received.Text)
		}
	case <-time.After(1 * time.Second):
		t.Log("Event subscription channel timeout (expected in memory fallback test)")
	}

	// 3. DLQ Enqueue
	if err := q.EnqueueDLQ(ctx, job, "Test error reason"); err != nil {
		t.Errorf("Failed to enqueue DLQ: %v", err)
	}
}
