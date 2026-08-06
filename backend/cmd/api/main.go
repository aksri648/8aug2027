package main

import (
	"log"
	"net/http"
	"os"

	"github.com/saas-agent-platform/backend/internal/agents/shared"
	"github.com/saas-agent-platform/backend/internal/api"
	"github.com/saas-agent-platform/backend/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	appStore := store.NewStore()
	daytonaClient := shared.NewDaytonaClient()
	server := api.NewServer(appStore, daytonaClient)

	log.Printf("⚡ SaaS Agentic Development Platform Backend listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, server.Router()))
}
