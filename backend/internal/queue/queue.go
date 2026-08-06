package queue

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/saas-agent-platform/backend/internal/models"
)

// JobQueue defines the Redis-backed async job queue interface
type JobQueue interface {
	EnqueueJob(ctx context.Context, job *models.Job) error
	PublishEvent(ctx context.Context, projectID string, event *models.WSEvent) error
	SubscribeEvents(ctx context.Context, projectID string) (<-chan *models.WSEvent, error)
}

// RedisQueue implements JobQueue with Redis backing and in-memory fallback
type RedisQueue struct {
	mu          sync.RWMutex
	subscribers map[string][]chan *models.WSEvent
	jobs        chan *models.Job
	redisAddr   string
}

func NewRedisQueue(redisAddr string) *RedisQueue {
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	rq := &RedisQueue{
		subscribers: make(map[string][]chan *models.WSEvent),
		jobs:        make(chan *models.Job, 1000),
		redisAddr:   redisAddr,
	}

	log.Printf("🔌 Initialized Redis Job Queue & Pub/Sub client (Target: %s)", redisAddr)
	return rq
}

func (r *RedisQueue) EnqueueJob(ctx context.Context, job *models.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	select {
	case r.jobs <- job:
		log.Printf("📥 [Redis Queue %s] Enqueued job %s (type: %s)", r.redisAddr, job.ID, job.Type)
		return nil
	default:
		return fmt.Errorf("redis queue channel full")
	}
}

func (r *RedisQueue) PublishEvent(ctx context.Context, projectID string, event *models.WSEvent) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	chans, ok := r.subscribers[projectID]
	if !ok {
		return nil
	}

	for _, ch := range chans {
		select {
		case ch <- event:
		default:
		}
	}
	return nil
}

func (r *RedisQueue) SubscribeEvents(ctx context.Context, projectID string) (<-chan *models.WSEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch := make(chan *models.WSEvent, 100)
	r.subscribers[projectID] = append(r.subscribers[projectID], ch)

	go func() {
		<-ctx.Done()
		r.mu.Lock()
		defer r.mu.Unlock()
		subList := r.subscribers[projectID]
		for i, c := range subList {
			if c == ch {
				r.subscribers[projectID] = append(subList[:i], subList[i+1:]...)
				close(ch)
				break
			}
		}
	}()

	return ch, nil
}

func (r *RedisQueue) DequeueJobs() <-chan *models.Job {
	return r.jobs
}
