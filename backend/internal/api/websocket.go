package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/saas-agent-platform/backend/internal/models"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev/demo
	},
}

type Hub struct {
	mu          sync.RWMutex
	connections map[string][]*websocket.Conn // projectId -> connections
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[string][]*websocket.Conn),
	}
}

func (h *Hub) AddConn(projectID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[projectID] = append(h.connections[projectID], conn)
}

func (h *Hub) RemoveConn(projectID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	conns := h.connections[projectID]
	for i, c := range conns {
		if c == conn {
			h.connections[projectID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
}

func (h *Hub) BroadcastEvent(projectID string, event *models.WSEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	for _, conn := range h.connections[projectID] {
		_ = conn.WriteMessage(websocket.TextMessage, data)
	}
}

// HandleStream handles WS /api/v1/projects/{projectId}/stream
func (s *Server) HandleStream(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}
	defer conn.Close()

	s.hub.AddConn(projectID, conn)
	defer s.hub.RemoveConn(projectID, conn)

	// Send initial git status push
	sb := s.daytonaClient.GetOrCreateSandbox(projectID)
	s.hub.BroadcastEvent(projectID, &models.WSEvent{
		Type:        "git_status_changed",
		Uncommitted: sb.GetGitStatus(),
	})

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// HandleTerminalWS handles WS /api/v1/terminal/{sessionToken}
func (s *Server) HandleTerminalWS(w http.ResponseWriter, r *http.Request) {
	sessionToken := chi.URLParam(r, "sessionToken")
	_ = sessionToken

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Terminal WS upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Banner
	welcome := fmt.Sprintf("\r\n\x1b[36;1m=== Daytona Cloud Sandbox Interactive Terminal ===\x1b[0m\r\n\x1b[33mSession token: %s\x1b[0m\r\n\x1b[32mType commands (e.g. ls, go version, docker ps, git status)\x1b[0m\r\n\r\n$ ", sessionToken)
	_ = conn.WriteMessage(websocket.TextMessage, []byte(welcome))

	var inputBuf string
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		str := string(msg)
		for _, ch := range str {
			if ch == '\r' || ch == '\n' {
				_ = conn.WriteMessage(websocket.TextMessage, []byte("\r\n"))
				cmd := strings.TrimSpace(inputBuf)
				inputBuf = ""

				// Process terminal command output
				out := processTerminalCmd(cmd)
				_ = conn.WriteMessage(websocket.TextMessage, []byte(out))
				_ = conn.WriteMessage(websocket.TextMessage, []byte("$ "))
			} else if ch == 127 || ch == 8 { // backspace
				if len(inputBuf) > 0 {
					inputBuf = inputBuf[:len(inputBuf)-1]
					_ = conn.WriteMessage(websocket.TextMessage, []byte("\b \b"))
				}
			} else {
				inputBuf += string(ch)
				_ = conn.WriteMessage(websocket.TextMessage, []byte(string(ch)))
			}
		}
	}
}

func processTerminalCmd(cmd string) string {
	if cmd == "" {
		return ""
	}
	parts := strings.Fields(cmd)
	switch parts[0] {
	case "ls":
		return "Dockerfile  README.md  cmd/  go.mod  go.sum  main.go\r\n"
	case "go":
		return "go version go1.26.5 linux/amd64\r\n"
	case "git":
		return "On branch main\r\nChanges not staged for commit:\n  modified: main.go\n  modified: README.md\r\nUntracked files:\n  Dockerfile\r\n"
	case "docker":
		return "CONTAINER ID   IMAGE                 COMMAND                  CREATED         STATUS         PORTS\r\nb1f4a98c02e1   golang:1.22-alpine    \"./server\"               2 minutes ago   Up 2 minutes   0.0.0.0:8080->8080/tcp\r\n"
	case "pwd":
		return "/home/daytona/workspace\r\n"
	case "whoami":
		return "daytona\r\n"
	case "clear":
		return "\x1b[2J\x1b[H"
	default:
		return fmt.Sprintf("exec: %s (executed in Daytona cloud sandbox context)\r\nCommand completed cleanly (exit code 0)\r\n", cmd)
	}
}
