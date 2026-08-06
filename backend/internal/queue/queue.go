package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/saas-agent-platform/backend/internal/models"
)

// JobQueue defines the Redis-backed async job queue interface
type JobQueue interface {
	EnqueueJob(ctx context.Context, job *models.Job) error
	PublishEvent(ctx context.Context, projectID string, event *models.WSEvent) error
	SubscribeEvents(ctx context.Context, projectID string) (<-chan *models.WSEvent, error)
	EnqueueDLQ(ctx context.Context, job *models.Job, errReason string) error
}

// RedisQueue implements JobQueue with go-redis client and fallback in-memory channel
type RedisQueue struct {
	mu          sync.RWMutex
	client      *redis.Client
	subscribers map[string][]chan *models.WSEvent
	jobs        chan *models.Job
	redisAddr   string
	isRealRedis bool
}

func NewRedisQueue(redisAddr string) *RedisQueue {
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	rq := &RedisQueue{
		client:      rdb,
		subscribers: make(map[string][]chan *models.WSEvent),
		jobs:        make(chan *models.Job, 1000),
		redisAddr:   redisAddr,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️ Redis server ping failed at %s: %v. Running in high-performance memory queue mode.", redisAddr, err)
		rq.isRealRedis = false
	} else {
		log.Printf("🔌 Connected to Redis Server at %s (Pub/Sub & Task Queue Active)", redisAddr)
		rq.isRealRedis = true
	}

	return rq
}

func (r *RedisQueue) EnqueueJob(ctx context.Context, job *models.Job) error {
	if r.isRealRedis {
		payloadBytes, err := json.Marshal(job)
		if err == nil {
			if err := r.client.LPush(ctx, "queue:agent_jobs", payloadBytes).Err(); err == nil {
				log.Printf("📥 [Redis LPUSH %s] Enqueued job %s (type: %s)", r.redisAddr, job.ID, job.Type)
				return nil
			}
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	select {
	case r.jobs <- job:
		log.Printf("📥 [Memory Queue] Enqueued job %s (type: %s)", job.ID, job.Type)
		return nil
	default:
		return fmt.Errorf("queue capacity exhausted")
	}
}

func (r *RedisQueue) EnqueueDLQ(ctx context.Context, job *models.Job, errReason string) error {
	log.Printf("🚨 [DLQ %s] Job %s (type: %s) failed: %s", r.redisAddr, job.ID, job.Type, errReason)
	if r.isRealRedis {
		dlqPayload := map[string]interface{}{
			"job":        job,
			"failed_at":  time.Now(),
			"err_reason": errReason,
		}
		b, _ := json.Marshal(dlqPayload)
		r.client.LPush(ctx, "dlq:agent_jobs", b)
	}
	return nil
}

func (r *RedisQueue) PublishEvent(ctx context.Context, projectID string, event *models.WSEvent) error {
	if r.isRealRedis {
		eventBytes, err := json.Marshal(event)
		if err == nil {
			channelName := fmt.Sprintf("events:project:%s", projectID)
			r.client.Publish(ctx, channelName, eventBytes)
		}
	}

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
