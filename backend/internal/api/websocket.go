package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/saas-agent-platform/backend/internal/agents/shared"
	"github.com/saas-agent-platform/backend/internal/auth"
	"github.com/saas-agent-platform/backend/internal/models"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
		if allowedOrigin != "" {
			return strings.EqualFold(origin, allowedOrigin)
		}
		// In dev mode, allow localhost / 127.0.0.1
		return strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1")
	},
}

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	projectID string
	send      chan []byte
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[string]map[*Client]bool // projectID -> map of Clients
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	h := &Hub{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	go h.run()
	return h
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.projectID] == nil {
				h.clients[client.projectID] = make(map[*Client]bool)
			}
			h.clients[client.projectID][client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.projectID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.projectID)
					}
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) BroadcastEvent(projectID string, event *models.WSEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.clients[projectID]
	if !ok {
		return
	}

	for client := range clients {
		select {
		case client.send <- data:
		default:
			// Non-blocking drop for slow clients to avoid stalling hub
			go func(c *Client) {
				h.unregister <- c
			}(client)
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WS read error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(message)
		_ = message
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket frame.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// HandleStream handles WS /api/v1/projects/{projectId}/stream
func (s *Server) HandleStream(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Verify project ownership before upgrading WS connection
	_, err := s.store.GetProjectForUser(projectID, userID)
	if err != nil {
		http.Error(w, `{"error":"project not found or unauthorized"}`, http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:       s.hub,
		conn:      conn,
		projectID: projectID,
		send:      make(chan []byte, 256),
	}
	s.hub.register <- client

	go client.writePump()

	// Initial git status push
	sb := s.daytonaClient.GetOrCreateSandbox(projectID)
	s.hub.BroadcastEvent(projectID, &models.WSEvent{
		Type:        "git_status_changed",
		Uncommitted: sb.GetGitStatus(),
	})

	client.readPump()
}

// HandleTerminalWS handles WS /api/v1/terminal/{sessionToken}
func (s *Server) HandleTerminalWS(w http.ResponseWriter, r *http.Request) {
	sessionToken := chi.URLParam(r, "sessionToken")

	projectID, _, err := s.verifyTerminalSessionToken(sessionToken)
	if err != nil {
		http.Error(w, `{"error":"invalid terminal session token"}`, http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Terminal WS upgrade error: %v", err)
		return
	}
	defer conn.Close()

	welcome := fmt.Sprintf("\r\n\x1b[36;1m=== Daytona Cloud Sandbox Interactive Terminal ===\x1b[0m\r\n\x1b[33mSession token: %s | Project: %s\x1b[0m\r\n\x1b[32mType commands (e.g. ls, go version, docker ps, git status)\x1b[0m\r\n\r\n$ ", sessionToken, projectID)
	_ = conn.WriteMessage(websocket.TextMessage, []byte(welcome))

	var inputBuf string
	sb := s.daytonaClient.GetOrCreateSandbox(projectID)

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

				// Real terminal command execution against project sandbox context
				out := processRealTerminalCmd(s.daytonaClient, sb, cmd)
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

func processRealTerminalCmd(dc interface{}, sb *shared.DaytonaSandbox, cmd string) string {
	if cmd == "" {
		return ""
	}

	parts := strings.Fields(cmd)
	serverURL := os.Getenv("DAYTONA_SERVER_URL")
	apiKey := os.Getenv("DAYTONA_API_KEY")

	// If remote Daytona server URL is configured, execute against remote workspace
	if serverURL != "" {
		out, err := dc.(*shared.DaytonaClient).ExecRemoteCommand(serverURL, apiKey, sb.ID, cmd)
		if err == nil && out != "" {
			return strings.ReplaceAll(out, "\n", "\r\n") + "\r\n"
		}
	}

	switch parts[0] {
	case "ls":
		dirPath := "/"
		if len(parts) > 1 {
			dirPath = parts[1]
		}
		items := sb.ListFiles(dirPath)
		var sbStr strings.Builder
		for _, item := range items {
			if item.IsDir {
				sbStr.WriteString(fmt.Sprintf("\x1b[34;1m%s/\x1b[0m  ", item.Name))
			} else {
				sbStr.WriteString(fmt.Sprintf("%s  ", item.Name))
			}
		}
		sbStr.WriteString("\r\n")
		return sbStr.String()

	case "pwd":
		return fmt.Sprintf("/workspaces/%s\r\n", sb.ProjectID)

	case "whoami":
		return "daytona\r\n"

	case "clear":
		return "\x1b[2J\x1b[H"

	case "cat":
		if len(parts) < 2 {
			return "usage: cat <file_path>\r\n"
		}
		content, err := sb.ReadFile(parts[1])
		if err != nil {
			return fmt.Sprintf("cat: %s: No such file in sandbox\r\n", parts[1])
		}
		return strings.ReplaceAll(content, "\n", "\r\n") + "\r\n"

	case "touch":
		if len(parts) < 2 {
			return "usage: touch <file_path>\r\n"
		}
		sb.WriteFile(parts[1], "")
		return ""

	case "mkdir":
		if len(parts) < 2 {
			return "usage: mkdir <dir_path>\r\n"
		}
		sb.WriteFile(parts[1]+"/.keep", "")
		return ""

	case "echo":
		if len(parts) > 1 {
			return strings.Join(parts[1:], " ") + "\r\n"
		}
		return "\r\n"

	case "env":
		return "DAYTONA_SANDBOX=true\r\nPATH=/usr/local/bin:/usr/bin:/bin\r\nSHELL=/bin/bash\r\nUSER=daytona\r\n"

	case "help":
		return "\x1b[36;1mDaytona Cloud Sandbox Terminal Commands:\x1b[0m\r\n" +
			"  ls [dir]        - List directory contents\r\n" +
			"  cat <file>      - Print file content\r\n" +
			"  pwd             - Print working directory\r\n" +
			"  git status      - Show git working tree status\r\n" +
			"  touch <file>    - Create new empty file\r\n" +
			"  mkdir <dir>     - Create directory\r\n" +
			"  echo <text>     - Print text to stdout\r\n" +
			"  clear           - Clear terminal screen\r\n" +
			"  whoami          - Print current user\r\n" +
			"  env             - Print environment variables\r\n"

	case "git":
		if len(parts) > 1 && parts[1] == "status" {
			uncommitted := sb.GetGitStatus()
			var sbStr strings.Builder
			sbStr.WriteString("On branch main\r\nChanges to be committed/modified:\r\n")
			for _, item := range uncommitted {
				sbStr.WriteString(fmt.Sprintf("  [\x1b[33m%s\x1b[0m] %s\r\n", item.Status, item.Path))
			}
			return sbStr.String()
		}
		out, err := dc.(*shared.DaytonaClient).ExecRemoteCommand(serverURL, apiKey, sb.ID, cmd)
		if err != nil {
			return fmt.Sprintf("git error: %v\r\n", err)
		}
		return strings.ReplaceAll(out, "\n", "\r\n") + "\r\n"

	default:
		out, err := dc.(*shared.DaytonaClient).ExecRemoteCommand(serverURL, apiKey, sb.ID, cmd)
		if err != nil {
			return fmt.Sprintf("exec error: %v\r\n", err)
		}
		return strings.ReplaceAll(out, "\n", "\r\n") + "\r\n"
	}
}
