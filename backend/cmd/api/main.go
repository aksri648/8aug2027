package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/saas-agent-platform/backend/internal/agents/shared"
	"github.com/saas-agent-platform/backend/internal/api"
	"github.com/saas-agent-platform/backend/internal/queue"
	"github.com/saas-agent-platform/backend/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	appStore := store.NewStore()
	daytonaClient := shared.NewDaytonaClient()
	jobQueue := queue.NewRedisQueue(redisAddr)

	server := api.NewServer(appStore, daytonaClient, jobQueue)

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      server.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown channel
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		log.Printf("⚡ SaaS Agentic Development Platform Backend listening on :%s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Fatal HTTP server error: %v", err)
		}
	}()

	<-stop
	log.Println("🛑 Gracefully shutting down API server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("⚠️ HTTP server forced shutdown error: %v", err)
	} else {
		log.Println("✅ API server stopped cleanly.")
	}
}
