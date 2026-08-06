# SaaS Agentic Development Platform — Setup & Configuration Guide

This guide provides step-by-step setup instructions, environment variables configuration, and launch commands to run the complete platform (Go API Gateway, GORM Persistent Store, Redis Queue Worker Fleet, Daytona Sandboxes, Azure Cloud Compute Engine, and React 19 Frontend).

---

## 1. Prerequisites

Ensure the following tools and services are installed on your host system:

- **Go**: `v1.22.0` or higher (`go version`)
- **Node.js**: `v18.0.0` or higher & `npm` (`node -v`)
- **Redis Server**: Running locally or accessible remotely (`redis-cli ping` returning `PONG`)
- **PostgreSQL Database** *(Optional)*: Fallback SQLite disk persistence (`saas_platform.db`) is automatically used if PostgreSQL DSN is not provided.
- **Git**: Installed and configured (`git --version`)

---

## 2. Environment Variables Setup

### Backend Environment Variables (`backend/.env` or export in shell)

Create or export the following environment variables:

```bash
# Server Port Configuration
PORT=8080

# Database Persistence Configuration
# (Leave blank to use SQLite persistent database file at saas_platform.db)
DATABASE_URL=postgres://postgres:password@localhost:5432/saas_agent_db?sslmode=disable
POSTGRES_ADDR=localhost:5432

# Redis Queue & Pub/Sub Configuration
REDIS_ADDR=localhost:6379

# LLM Provider API Keys
ANTHROPIC_API_KEY=sk-ant-api03-...
OPENAI_API_KEY=sk-proj-...
DEEPSEEK_API_KEY=sk-ds-...

# Custom OpenAI-Compatible Provider Setup (vLLM / Ollama / LocalAI / LM Studio)
CUSTOM_OPENAI_BASE_URL=http://localhost:8000/v1
CUSTOM_OPENAI_API_KEY=sk-custom-...
CUSTOM_OPENAI_MODEL=deepseek-ai/DeepSeek-R1

# Daytona Cloud Sandbox Credentials
DAYTONA_SERVER_URL=https://app.daytona.io/api
DAYTONA_API_KEY=daytona_pat_...

# Azure Cloud Infrastructure Credentials
AZURE_SUBSCRIPTION_ID=00000000-0000-0000-0000-000000000000
AZURE_TENANT_ID=tenant-id-...
AZURE_CLIENT_ID=client-id-...
AZURE_CLIENT_SECRET=secret-...
AZURE_BEARER_TOKEN=eyJ0eXAi...
```

---

## 3. Installation & Database Setup

### Step A: Clone Repository & Dependencies

```bash
# Clone the repository
git clone https://github.com/aksri648/8aug2027.git
cd 8aug2027

# Install Backend Go Dependencies
cd backend
go mod download
go build ./...

# Install Frontend Node Dependencies
cd ../frontend
npm install
```

### Step B: Database Schema Migrations *(Optional for PostgreSQL)*

If using PostgreSQL, execute the SQL migration script:

```bash
psql -d saas_agent_db -f backend/migrations/000001_init_schema.up.sql
```

*Note: The backend GORM store automatically runs `db.AutoMigrate(...)` on startup for both SQLite and PostgreSQL.*

---

## 4. Running the Backend Services

Open two separate terminal windows (or background jobs) in the `backend/` directory:

### Terminal 1: Launch Backend API Server Gateway

```bash
cd backend
# Option A: Run directly via Go compiler
go run ./cmd/api

# Option B: Compile & run production binary
go build -o api-server ./cmd/api
./api-server
```

### Terminal 2: Launch Async Redis Queue Worker Fleet

```bash
cd backend
# Option A: Run directly via Go compiler
go run ./cmd/worker

# Option B: Compile & run production binary
go build -o worker-server ./cmd/worker
./worker-server
```

---

## 5. Running the Frontend Application

Open a terminal window in the `frontend/` directory:

```bash
cd frontend

# Development Server (Listening on http://localhost:3000/)
npm run dev

# Production Build Verification
npm run build
```

---

## 6. System Verification & Health Checks

Verify all components are active:

```bash
# 1. Test Backend API Projects Endpoint
curl -s http://localhost:8080/api/v1/projects

# 2. Test Prometheus Metrics Endpoint
curl -s http://localhost:8080/metrics

# 3. Test Provider Connection Endpoint
curl -s -X POST http://localhost:8080/api/v1/providers/test \
  -H "Content-Type: application/json" \
  -d '{"base_url":"http://localhost:8000/v1"}'

# 4. Access Web UI in browser
open http://localhost:3000/
```

---

## 7. Platform Settings & Configuration Modal

Once the Web UI is open:
1. Click the **Settings** icon (bottom left profile area or top bar wrench).
2. Configure **LLM Providers** (Anthropic, OpenAI, Custom vLLM/Ollama Base URL).
3. Click **"Test Connection"** to verify API connectivity.
4. Click **"Discover Models"** to auto-populate model dropdowns.
5. Save configuration.
