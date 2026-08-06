# Implementation Specification: Agentic SaaS Development Platform

**Purpose of this document:** This is a complete, unambiguous build specification for an AI coding assistant (human or LLM) to implement end-to-end. It intentionally over-specifies behavior, data shapes, and sequencing so that no creative interpretation is required. Follow the sections in order. Do not skip the "Build Order" section (Section 14) — it defines the sequence in which pieces must be built and wired together so the system is runnable at every milestone.

Read this entire document before writing any code.

---

## 1. Product Summary

Build a production-grade SaaS platform that lets a user chat with an AI system (visually and behaviorally cloned from the Claude.ai web chat UI) to design, build, deploy, and maintain software projects and self-hosted LLM endpoints. The user never writes infrastructure code by hand — they describe intent in chat, the backend's multi-agent system asks clarifying questions, then performs real work (writing code, provisioning Azure infrastructure, deploying containers, deploying LLMs, and fixing bugs in an existing GitHub repository) inside isolated Daytona cloud sandboxes.

The system has two halves:

1. **Frontend** — a React + Vite single-page application that visually and interactionally mirrors the Claude.ai web chat client, extended with a left sidebar of developer tooling (Skills, Projects, Terminal, File Explorer, Uncommitted Git Files, Git Remote/Push).
2. **Backend** — a Golang REST + WebSocket API that hosts a multi-agent system built on **Google's Agent Development Kit (ADK) for Go**, orchestrating one Master Agent and four specialist "slave" agents, each of which performs real infrastructure and code actions inside **Daytona** sandboxes and against **Azure**, **GitHub**, **Hugging Face**, and **NVIDIA NIM**.

---

## 2. Guiding Principles for the Builder

- **No placeholders in the critical path.** Every button, modal, and sidebar item defined below must be wired to a real backend endpoint. If a feature cannot be completed in one pass, stub the backend response but keep the exact contract described here so the frontend never needs to change later.
- **Everything long-running is asynchronous.** Any operation that takes more than ~2 seconds (codegen, docker build, VM provisioning, LLM deployment, repo cloning) must be modeled as a background job with a job ID returned immediately, progress events streamed to the frontend, and a terminal state (`succeeded`, `failed`, `cancelled`).
- **All credentials are secrets, never plaintext in the database or logs.** Azure credentials, GitHub PATs, Hugging Face tokens, and NVIDIA NIM auth tokens are written directly to a secrets vault and only referenced by an opaque secret ID elsewhere in the system.
- **One sandbox per task/session.** Every agent action that touches code or runs commands happens inside a dedicated Daytona sandbox scoped to that project/task, never on the backend host itself.
- **Match the reference UI exactly**, not approximately: dark theme, centered greeting with the user's name, rounded input composer pinned near the bottom-third of the screen (not attached to the true bottom), suggestion chips below the composer, and a slim icon+label sidebar. Section 5 specifies this pixel-level.

---

## 3. Technology Stack

| Layer | Technology | Notes |
|---|---|---|
| Frontend framework | React 18 + Vite | SPA, client-side routing with `react-router` |
| Frontend styling | Tailwind CSS | Utility classes only, dark theme by default |
| Terminal emulation | xterm.js (`@xterm/xterm` + `@xterm/addon-fit` + `@xterm/addon-attach`) | Rendered inside a modal, connected over WebSocket to a Daytona sandbox process |
| Backend language | Go (1.22+) | REST + WebSocket API |
| Backend HTTP router | `chi` or `gin` | Either is acceptable; pick one and use it consistently |
| Agent framework | Google Agent Development Kit (ADK) — Go module `google.golang.org/adk` | Master + 4 slave agents, tool-calling, session/state management |
| Optional orchestration layer | LangGraph (Python) as a sidecar service, OR pure ADK Go graph workflows | See Section 7.1 for the decision guidance |
| Sandbox / code execution | Daytona (Go SDK: `daytona/go-sdk`) | One sandbox per active task/session |
| Containerization (app deploy) | Docker Engine API via the official Go SDK (`github.com/docker/docker/client` or the newer `github.com/moby/moby/client`) | Used by the App Deployer Agent inside the sandbox |
| Cloud provider | Microsoft Azure | Compute (VMs), AKS, Load Balancer, Storage, Key Vault |
| Azure SDK | Azure SDK for Go (`azidentity`, `armcompute`, `armnetwork`, `armresources`, `armcontainerservice`, `azsecrets`) | |
| LLM self-hosting engines | vLLM (OpenAI-compatible server) and NVIDIA NIM microservices | Chosen per-deployment by the LLM Deployer Agent |
| Model/weights source | Hugging Face Hub | Model download and gated-model auth via HF token |
| Source control | GitHub REST API | PAT-based auth for the App Maintainer Agent |
| Primary database | PostgreSQL | Users, projects, agent sessions, task/job metadata |
| Cache / job queue backing | Redis | Pub/sub for job progress, queue backing store |
| Blob storage | Azure Blob Storage | Build logs, generated codebase archives, artifacts |
| Secrets management | Azure Key Vault (via `azsecrets` Go SDK) | All third-party credentials |
| Async job execution | A Go worker pool consuming from a Redis-backed queue (e.g. `asynq`) | Every agent task runs as a job |
| Observability | Structured JSON logging + Prometheus metrics + Grafana dashboards | |

---

## 4. Repository / Project Structure

Use a monorepo with two top-level applications and shared contracts:

- `/frontend` — React + Vite app.
- `/backend` — Go module containing:
  - `/backend/cmd/api` — REST/WebSocket server entrypoint.
  - `/backend/cmd/worker` — async job worker entrypoint (separate deployable binary, same module).
  - `/backend/internal/agents/master` — Master Agent (intent analysis + routing).
  - `/backend/internal/agents/appdeveloper`
  - `/backend/internal/agents/appdeployer`
  - `/backend/internal/agents/llmdeployer`
  - `/backend/internal/agents/appmaintainer`
  - `/backend/internal/agents/shared` — shared tool implementations (Daytona client wrapper, Azure clients, GitHub client, HF client, NIM client).
  - `/backend/internal/api` — HTTP handlers and WebSocket handlers.
  - `/backend/internal/store` — PostgreSQL access layer.
  - `/backend/internal/queue` — Redis-backed job queue.
  - `/backend/internal/secrets` — Key Vault wrapper.
  - `/backend/internal/models` — shared DTOs/entities.
- `/docs/api-contract.md` — the canonical REST/WebSocket contract (generate this from Section 6 below and keep it in sync).

---

## 5. Frontend Application (React + Vite)

### 5.1 Visual Design Requirements — Claude.ai Chat UI Clone

Reproduce the following visual/interaction pattern exactly:

- **Global theme:** near-black background (`#1a1a1a`–`#212121` range), off-white text, a single warm accent color (orange/coral, e.g. `#d97757`-ish) used sparingly for the logo mark, the active nav item, and primary buttons.
- **Top bar (right panel only):** left-aligned hamburger/menu icon is NOT needed since the sidebar is always visible (unlike claude.ai's collapsible version) — instead show a small "Claude ▾"-style model/workspace selector, and a user avatar circle in the top-right.
- **Empty-state chat screen:** vertically centered, roughly at 35–40% viewport height: a small radial/asterisk-style logo mark, a large greeting line ("Good afternoon, {username}") using time-of-day logic (morning/afternoon/evening based on local time), and a muted subtitle ("How can Claude help you today?" — replace "Claude" with your product's agent name).
- **Composer:** a rounded rectangle input area, NOT full width — comfortably inset with margin, containing: a "+" attachment button (bottom-left), a settings/sliders icon, a multi-line auto-growing textarea with placeholder "Message Claude...", a model/agent selector on the bottom-right, and a circular send button (up-arrow icon) that is disabled when the input is empty.
- **Suggestion chips row:** directly beneath the composer, 4 pill-shaped buttons with short task prompts (e.g. "Write a function in Go", "Explain a concept", "Debug this error", "Summarize this document") that, when clicked, populate the composer with that text.
- **Conversation view (after first message):** messages stack top-to-bottom, user messages right-aligned in a subtly different bubble shade, assistant messages left-aligned with no bubble background (plain text on the dark canvas), streamed token-by-token.
- **Left sidebar:** fixed width (~260px), icon + label rows, no icons for user avatar/plan info stacked at the very bottom (name + plan badge, e.g. "Pro Plan").

### 5.2 Layout Shell

Two-column CSS grid: fixed-width left sidebar, flexible-width right panel. The right panel is the chat UI described above. The sidebar is always visible (no collapse needed) and contains, top to bottom:

1. **Skills** (icon: sparkle/wand)
2. **Projects** (icon: folder)
3. **Terminal** (icon: terminal/`>_`)
4. **File Explorer** (icon: document/tree)
5. **Uncommitted Git Files** (icon: git-branch) — this is not just a nav item; it is a live-updating **inline list** of changed files rendered directly in the sidebar (not only a modal trigger), showing filename + change type (Added/Modified/Deleted) with colored dots (green/yellow/red).
6. **Git Remote URL** section — a labeled text input (placeholder `https://github.com/user/repo.git`) directly below the Uncommitted Git Files list.
7. **Push** button — full-width, accent-colored, directly below the Git Remote URL input.
8. User identity block pinned to the bottom of the sidebar (avatar, name, plan tier).

Each of Skills, Projects, Terminal, and File Explorer opens a **modal overlay** on click (not a page navigation). Modals share a common shell component: dark rounded panel, header with title + close (X) button, scrollable body, optional footer action bar.

### 5.3 Sidebar Item Behaviors

#### 5.3.1 Skills → Modal

Purpose: manage the reusable "skill" definitions the backend agents can draw on (instructions, playbooks, or tool configs consumed by the App Developer / App Maintainer agents when generating or fixing code).

Modal contents:
- Two tabs: **"Upload"** and **"Create manually."**
- **Upload tab:** drag-and-drop / file-picker area accepting one or more `.md` files named by convention `<skill-name>.md`. On drop, POST each file's raw content to the backend (see Section 6.2). Show per-file upload progress and a success/failure row per file.
- **Create manually tab:** a form with fields — `name` (text), `description` (textarea, one line of guidance on when this skill applies), `content` (large markdown textarea for the actual instructions). Submitting POSTs a new skill record.
- Below the tabs: a live list of all skills already registered for this user/org, each row showing name, description, source (`uploaded` or `manual`), last-updated timestamp, and a delete (trash) icon.
- Skills are **global to the account**, not scoped to a single project, since any agent in any project may reference them.

#### 5.3.2 Projects → Modal

Purpose: list and switch between the user's projects (each project maps 1:1 to a Master Agent session + its own Daytona sandbox lineage + its own uncommitted-git-files/file-tree state).

Modal contents:
- A list of project cards, each showing: project name, short description (auto-summarized from the first user prompt), status badge (`draft`, `building`, `deployed`, `error`), last-activity timestamp, and the linked git remote URL if set.
- A "+ New Project" button at the top that creates an empty project and switches the whole app (sidebar state + chat session) into that project's context.
- Clicking a project card switches the active project context (updates the right-panel chat to that project's conversation history, and updates File Explorer / Uncommitted Git Files / Git Remote URL to that project's sandbox state) and closes the modal.

#### 5.3.3 Terminal → Modal (xterm.js + Daytona)

Purpose: give the user a live shell into the currently active Daytona sandbox for the active project, showing the exact commands the agents are running plus allowing the user to type their own commands.

Modal contents:
- A large modal (near-fullscreen) containing a single `<div>` mount point for an `xterm.js` `Terminal` instance, resized responsively with the `fit` addon.
- On modal open: the frontend requests a terminal session from the backend (Section 6.3 WebSocket contract), which returns a WebSocket URL scoped to the active project's live Daytona sandbox (creating one if none is currently running for that project).
- The frontend opens that WebSocket, pipes `onData` from the terminal into outbound WebSocket messages, and writes inbound WebSocket messages (raw PTY bytes) into `term.write()`.
- On modal close, keep the WebSocket connection alive for a short grace period (do not kill the sandbox process) so re-opening the terminal resumes the same session; only after an idle timeout (configurable, default 10 minutes) does the backend tear the PTY down.
- Show a small colored status dot in the modal header: green = connected, yellow = connecting/reconnecting, red = disconnected, with an automatic reconnect-with-backoff strategy.

#### 5.3.4 File Explorer → Modal

Purpose: show the current file tree of the active project's sandbox workspace.

Modal contents:
- Left pane: collapsible tree view of directories/files (fetched from the backend's sandbox filesystem-listing endpoint, Section 6.2).
- Right pane: read-only syntax-highlighted preview of the selected file's contents (fetch on click; do not preload every file).
- A refresh icon in the header re-fetches the tree (agents may be actively writing files).
- File icons differ by extension (basic mapping is sufficient — no need for a full icon theme).

#### 5.3.5 Uncommitted Git Files (sidebar-inline list)

- Polls (or receives via WebSocket push — preferred) the git status of the active project's sandbox working directory.
- Each row: filename (path relative to repo root), a single-letter/colored badge for status (`A` green, `M` yellow, `D` red, `??` grey for untracked), and clicking a row opens a diff view (can reuse the File Explorer modal's preview pane, or a lightweight inline expand — implementer's choice, but a diff must be viewable, not just filenames).
- Empty state: muted text "No uncommitted changes."

#### 5.3.6 Git Remote URL + Push

- Text input bound to the active project's stored `git_remote_url` field; on blur or explicit "Save" affordance, PATCH the project record.
- **Push** button: disabled if `git_remote_url` is empty or there are zero uncommitted files. On click, POSTs a push job (Section 6.2) which is handled by the same git tooling the App Maintainer Agent uses (stage all → commit with a generated or user-suppliable message → push to the configured remote using the stored GitHub PAT). Show a toast/inline status (`pushing…`, `pushed`, `failed: <reason>`).
- If no GitHub PAT secret exists yet for this project when Push is clicked, open a small credential-prompt dialog (label: "GitHub Personal Access Token", password-masked input, helper text linking to GitHub's PAT documentation) before proceeding. Store the entered PAT via the secrets endpoint (Section 6.2) — never send it to any endpoint other than the dedicated secrets endpoint, and never persist it in frontend state longer than the single request lifecycle.

### 5.4 Right Panel — Chat UI

- Standard chat semantics: an ordered list of messages (`role`: `user` | `assistant` | `system-status`), each rendered per Section 5.1's styling.
- **`system-status` messages** are a special inline message type used to surface agent progress without polluting the conversational flow — e.g. "🔧 App Deployer Agent: provisioning Azure VM…", "✅ Codebase generated (14 files)", "❌ Deployment failed: insufficient quota in region eastus". Render these visually distinct (smaller, muted, icon-prefixed, no avatar).
- Follow-up questions asked by any agent (see Section 7) are rendered as normal assistant chat messages; the user answers by typing a normal chat message back. There is no special "form" UI required for follow-ups — this keeps the interaction pattern identical to claude.ai. (Optional enhancement, not required: render suggested quick-reply chips under a follow-up question when the backend flags the message with a small set of suggested answers.)
- Messages stream token-by-token from the backend over the same WebSocket connection used for job/status updates (see Section 6.3) — do not implement chat over polling.

### 5.5 Frontend State Management

- Use React Context (or a lightweight state library such as Zustand) for: active project, active chat session, sidebar-derived state (skills list, uncommitted files, file tree), and websocket connection status. Avoid prop-drilling across the modal components.
- Each modal manages its own local fetch/loading/error state but reads the "active project ID" from the shared context so all modals stay consistent when the user switches projects.

### 5.6 Frontend ↔ Backend Communication Contract

The frontend must talk to the backend using exactly the endpoints and message shapes defined in Section 6. Do not invent alternate paths. If the backend is not yet implemented for a given call, mock it behind the same interface so swapping in the real backend requires no frontend changes.

---

## 6. Backend Application (Golang)

### 6.1 Service Responsibilities

The Go backend is the single source of truth for: user/project/session persistence, secrets brokering, job orchestration, and hosting the ADK-based agent runtime. It exposes a REST API for CRUD-style operations and a WebSocket API for anything real-time (chat streaming, terminal PTY, job progress, git status push updates).

### 6.2 REST API Surface

All routes are versioned under `/api/v1`. Authentication: bearer JWT in the `Authorization` header (issued at login; login/signup flow is standard email+password or OAuth — implement simply, it is not the focus of this spec).

**Projects**
- `GET /api/v1/projects` — list projects for the authenticated user.
- `POST /api/v1/projects` — create a project. Body: `name` (optional, can be auto-generated later).
- `GET /api/v1/projects/{projectId}` — project detail, including `git_remote_url`, `status`, `sandbox_id` (nullable).
- `PATCH /api/v1/projects/{projectId}` — update fields such as `git_remote_url`.
- `DELETE /api/v1/projects/{projectId}`.

**Skills**
- `GET /api/v1/skills` — list all skills for the account.
- `POST /api/v1/skills` — create manually. Body: `name`, `description`, `content`.
- `POST /api/v1/skills/upload` — multipart file upload of one or more `.md` files; backend parses filename as skill name and file content as skill content.
- `DELETE /api/v1/skills/{skillId}`.

**Chat / Agent Sessions**
- `GET /api/v1/projects/{projectId}/messages` — full message history for the project's chat session.
- `POST /api/v1/projects/{projectId}/messages` — send a new user message. This is the single entry point that triggers the Master Agent (Section 7.2). Returns immediately with the created message plus a `job_id` for the resulting agent turn; actual agent response streams over WebSocket.

**Sandbox / File System (proxied through backend to Daytona)**
- `GET /api/v1/projects/{projectId}/files?path=/` — list directory contents at `path` inside the active sandbox.
- `GET /api/v1/projects/{projectId}/files/content?path=/src/main.go` — return file content for preview.
- `GET /api/v1/projects/{projectId}/git/status` — list of changed files with status codes.
- `GET /api/v1/projects/{projectId}/git/diff?path=...` — unified diff for one file.
- `POST /api/v1/projects/{projectId}/git/push` — body: optional `commit_message`; triggers the stage/commit/push job. Requires a GitHub PAT secret to already be registered for the project (see Secrets below), otherwise returns HTTP 428 with an error code the frontend uses to trigger the credential prompt.

**Secrets**
- `POST /api/v1/projects/{projectId}/secrets` — body: `type` (enum: `github_pat`, `azure_credentials`, `huggingface_token`, `nvidia_nim_token`), and the credential fields appropriate to that type (see Section 9.4). The backend writes the value(s) to Key Vault immediately and stores only the Key Vault secret reference (name/version) in Postgres — the raw value is never persisted in the app database and never echoed back in any subsequent GET.
- `GET /api/v1/projects/{projectId}/secrets` — list secret **types present** (booleans only, e.g. `{"github_pat": true, "azure_credentials": false, ...}`), never values.

**Jobs**
- `GET /api/v1/jobs/{jobId}` — job status snapshot (`queued`, `running`, `succeeded`, `failed`, `cancelled`), plus a `result` object whose shape depends on job type (see each agent's section).

**Terminal**
- `POST /api/v1/projects/{projectId}/terminal/session` — ensures a live Daytona sandbox exists for the project (creating one if necessary) and returns a short-lived WebSocket URL + token scoped to that sandbox's PTY.

### 6.3 WebSocket API Surface

- `WS /api/v1/projects/{projectId}/stream` — a single multiplexed connection per active project view. Server pushes JSON-framed events of these kinds:
  - `chat_token` — incremental assistant text (`message_id`, `delta`).
  - `chat_message_complete` — signals a message is finished (`message_id`).
  - `system_status` — agent progress events (`job_id`, `agent`, `text`, `level`: info/success/error).
  - `git_status_changed` — pushes a fresh uncommitted-files list whenever the sandbox git state changes (avoids polling).
  - `job_update` — job state transitions (`job_id`, `status`, optional `result`).
- `WS /api/v1/terminal/{sessionToken}` — raw bidirectional byte stream between the browser's xterm.js instance and the Daytona sandbox PTY (backend proxies this; do not expose Daytona's own endpoint directly to the browser — always broker through the backend so tokens/credentials stay server-side).

### 6.4 Authentication & Multi-Tenancy

- Every project, skill, secret, and job row is scoped by `user_id` (and optionally `org_id` if you add team support later — not required for v1, but design the schema with a nullable `org_id` column so it is not a breaking change to add).
- Every REST/WebSocket handler must verify the authenticated user owns the `projectId`/`jobId` in the path before doing any work.

### 6.5 Async Task / Job Model

- A `jobs` table: `id`, `project_id`, `type` (enum matching each agent's task types, see Section 7), `status`, `payload` (JSONB — the task input), `result` (JSONB, nullable), `error` (text, nullable), `created_at`, `updated_at`.
- Jobs are enqueued into Redis (via `asynq` or an equivalent reliable queue library) and consumed by the `cmd/worker` process, decoupled from the API process so long agent runs never block HTTP handlers.
- Worker publishes progress via Redis Pub/Sub, which the API process's WebSocket handler subscribes to and relays to the correct connected browser session.

---

## 7. Agent System (Google ADK, Go)

### 7.1 Orchestration Framework Choice

Primary recommendation: implement all five agents natively in **Google ADK for Go**, using ADK's multi-agent composition (a root/orchestrator agent with sub-agents) and ADK's tool-calling for every external action (Daytona, Azure, Docker, GitHub, Hugging Face, NIM). ADK Go 2.0 supports graph-based workflow agents and parallel/loop execution primitives, which map directly onto the "Master routes to exactly one of four slaves, then the slave runs a multi-step tool-calling loop" pattern described below.

If the builder prefers, **LangGraph** (Python) may be used instead as the orchestration layer for the Master Agent's routing/state-machine logic, running as a separate internal microservice that the Go backend calls over gRPC or HTTP, with the four slave agents still implemented as ADK Go agents/tools invoked by that service. This is a valid alternative architecture — pick one and be consistent; do not mix partial LangGraph and partial ADK routing for the same decision point.

Model backing: any ADK-supported model works for the agents' own reasoning (Gemini by default). This is independent from the LLMs the **LLM Deployer Agent** provisions on behalf of the user — those are separate, user-chosen open-weight models served via vLLM or NIM.

### 7.2 Master Agent — Intent Analyzer & Orchestrator

**Input:** every new user chat message for a project (`POST /messages`), plus the full prior conversation state for that project's session.

**Responsibilities:**
1. Classify the message into one of five intents: `build_app`, `deploy_app`, `deploy_llm`, `maintain_app`, `general_or_other`.
2. If the project already has an in-flight multi-turn interaction with a slave agent (e.g. the App Developer Agent is mid-way through asking follow-up questions), route the new message to **that same agent's session** rather than reclassifying from scratch — the Master Agent must track "which slave, if any, owns the current conversational turn" as session state.
3. For `general_or_other`, answer directly as a conversational assistant (no slave agent invocation) — e.g. answering questions about the platform itself, or general programming questions.
4. For the other four intents, hand off to the matching slave agent (Sections 7.3–7.6), passing the user's message plus full project context (existing codebase presence, sandbox ID, registered secrets, git remote).
5. Emit a `system_status` event announcing which agent was activated, then stream that agent's questions/output back into the chat as normal assistant messages.
6. Persist intent + routing decisions to the session state so the same classification is not repeated needlessly turn-to-turn.

### 7.3 App Developer Agent

**Intent:** `build_app`.

**Flow:**
1. Receives the initial build prompt.
2. Asks clarifying follow-up questions **as normal chat turns** until it has enough information to proceed — at minimum: target stack/framework (if not already specified), key features/pages, data model basics, whether authentication is needed. Do not hardcode a rigid fixed question list — the agent should ask only what is still ambiguous given what the user already said, but must not start generating code until it has asked at least one round of clarification unless the user's initial prompt was already fully specified.
3. Once sufficient detail is gathered, enqueues a `codegen` job.
4. The job, running in the worker: creates (or reuses) a Daytona sandbox for the project, uses the Code Generation LLM (tool call) to produce a project structure and file contents, uses the Daytona filesystem tool to write every file into the sandbox, and uses the Daytona git tool to `git init`/commit locally inside the sandbox if no repo exists yet.
5. On completion, emits a `system_status` success event with a file count summary, and triggers a `git_status_changed` push so the sidebar updates.

**Tools this agent needs:** Daytona Sandbox (create/reuse), Daytona Filesystem (write files, list files), Code Generation LLM call, Daytona Git (init/add/commit locally — pushing to a remote is deliberately left to the Uncommitted Files → Push flow or the App Maintainer Agent, not this agent, to keep responsibilities separated).

### 7.4 App Deployer Agent

**Intent:** `deploy_app`.

**Flow:**
1. Analyzes the existing codebase in the project's sandbox (reads file tree/contents) to infer language/framework and how it should be containerized.
2. If no Azure credentials secret exists for the project, asks the user (in chat) to provide them, and the frontend's secrets flow (Section 5.3.6-style credential prompt, generalized) collects `azure_client_id`, `azure_client_secret`, `azure_tenant_id`, `azure_subscription_id` and stores them via the Secrets endpoint as type `azure_credentials`.
3. Generates a Dockerfile if one does not already exist (based on inferred stack), then builds the image inside the sandbox using the Docker Engine API (Go client), tags it, and pushes it to a container registry (Azure Container Registry is the default choice — provision one if missing).
4. Provisions an Azure VM (via `armcompute`) sized appropriately for the workload, using `azidentity` credentials built from the stored secret, opens the necessary inbound port(s), pulls and runs the built image on the VM (via a startup script or SSH command executed through the Daytona sandbox, which has network access to the new VM).
5. Returns the deployed app's public IP/URL and any relevant connection details as the job `result`, and posts a `system_status` success message summarizing them.

**Tools this agent needs:** Daytona Filesystem/exec (codebase analysis, Dockerfile generation, docker build/push), Azure Identity + `armcompute` + `armnetwork` + `armresources` (VM + networking provisioning), Azure Container Registry client, Secrets (read Azure credential, prompt-and-store if missing).

### 7.5 LLM Deployer Agent

**Intent:** `deploy_llm`.

**Flow — clarification phase (ask as normal chat turns, in this order, skipping any already answered):**
1. Which Hugging Face model (repo ID) to deploy, and whether it is gated (if gated, a Hugging Face token is required — collect and store as secret type `huggingface_token` if not already present).
2. Purpose of the deployment: **agentic coding** vs **general usage** (this affects default context length / sampling defaults and which model family is suggested if the user is undecided).
3. Expected number of concurrent users / load tier (rough bucket: light / moderate / heavy) — used to size the VM/cluster.
4. Deployment topology, offered as an explicit choice: **(a)** single Azure VM running **vLLM**, **(b)** **AKS + Azure Load Balancer** running a vLLM (or NIM) container across multiple pods for horizontal scaling, or **(c)** Azure VM running an **NVIDIA NIM** microservice (requires an NVIDIA NIM/NGC auth token — collect and store as secret type `nvidia_nim_token` if not already present).

**Flow — execution phase (dispatches to one of three sub-paths based on the user's topology choice):**
- **(a) VM + vLLM:** provision a GPU-enabled Azure VM (`armcompute`, GPU-family SKU), install NVIDIA drivers/CUDA + vLLM, download the chosen model from Hugging Face Hub (using the stored HF token if the model is gated), and launch vLLM's OpenAI-compatible server (`vllm serve <model>`) bound to a fixed port, opening that port on the VM's NSG.
- **(b) AKS + Load Balancer:** provision or reuse an AKS cluster (`armcontainerservice`), deploy a Kubernetes Deployment running a vLLM (or NIM) container image sized per the requested load tier, and expose it via a Kubernetes `Service` of `type: LoadBalancer`, which provisions an Azure Standard Load Balancer with a public IP automatically.
- **(c) VM + NVIDIA NIM:** provision a GPU-enabled Azure VM, authenticate to NVIDIA's container registry (`nvcr.io`) using the stored NIM/NGC token, pull and run the appropriate NIM container for the chosen model, exposing its OpenAI-compatible port.

**Completion:** in all three sub-paths, the job `result` must include: the deployment topology chosen, the public endpoint URL, the port, the API path convention (OpenAI-compatible `/v1/chat/completions` etc.), whether an API key/token is required to call it, and the underlying model name — post this as a clearly formatted `system_status` success message so the user can copy the endpoint details directly.

**Tools this agent needs:** Daytona exec (driver/tooling install commands run through the sandbox's SSH/exec channel against the target VM, or directly on the VM via cloud-init), Azure Identity + `armcompute` + `armnetwork` + `armcontainerservice`, Hugging Face Hub client (model download / gated-repo auth), NVIDIA NGC/NIM auth (docker login to `nvcr.io` using the stored token), Secrets (HF token, NIM token, Azure credentials).

### 7.6 App Maintainer Agent

**Intent:** `maintain_app`.

**Flow:**
1. Receives a maintenance/bug-fix prompt (e.g. "users report a 500 error on checkout").
2. If no `git_remote_url` is set on the project, or no `github_pat` secret exists, asks the user for the repository URL and/or PAT (via the standard secrets prompt flow) before proceeding.
3. Clones the full repository from the configured remote into a **fresh Daytona sandbox** using the stored PAT for auth.
4. Attempts to reproduce the reported issue by installing dependencies and running the project **inside the sandbox** (inferring the run command from project manifests — e.g. `package.json` scripts, `go.mod` + `main.go`, `requirements.txt`/`pyproject.toml`, etc.), capturing logs/stack traces.
5. Uses those logs plus the codebase to locate and apply a fix (LLM-driven code edit via the Daytona filesystem tool).
6. Re-runs the reproduction steps to verify the fix resolves the issue (best-effort verification; if it cannot be fully automated, at least confirm the app builds/starts cleanly).
7. Commits the change with a descriptive message and pushes to the same remote using the stored PAT.
8. Reports a summary of the diagnosis, the fix applied, and the commit that was pushed as a `system_status` success message; on failure to reproduce or fix, reports what was tried and where it got stuck rather than failing silently.

**Tools this agent needs:** Daytona Sandbox (fresh per maintenance job), Daytona Git (clone using PAT, add/commit/push), Daytona Filesystem/exec (dependency install, run/reproduce, read logs, apply edits), GitHub REST API client (PAT-authenticated — used for anything beyond plain git, e.g. verifying remote details), Secrets (`github_pat`, and the stored `git_remote_url` from the project record).

### 7.7 Shared Agent Infrastructure

- **Tool layer:** implement each external integration (Daytona, Azure family, Docker, GitHub, Hugging Face, NIM) as a standalone Go package under `/backend/internal/agents/shared`, exposing a small typed interface per capability (e.g. a `SandboxProvider` interface with `CreateSandbox`, `WriteFile`, `ListFiles`, `Exec`, `GitStatus`, `GitCommit`, `GitPush`). Register these as ADK tools so every agent calls the same underlying implementation — do not duplicate Daytona/Azure client code per agent.
- **Session/state:** use ADK's session and state management so each agent's mid-conversation follow-up-question flow survives across chat turns without the frontend needing any special "multi-step form" handling — this is what makes Section 5.4's plain-chat follow-up UX work.
- **Guardrails:** every tool call that provisions billable cloud resources (VM creation, AKS cluster creation) must be preceded by a `system_status` message stating what is about to be created (resource type, rough size/cost tier) — do not silently spend the user's cloud budget.

---

## 8. Daytona Sandbox Integration

- One Daytona sandbox per **project** for the App Developer / App Deployer / Terminal / File Explorer flows (created lazily on first need, reused across turns), and one **fresh, disposable** sandbox per App Maintainer job (since it needs a clean clone of the real repository, decoupled from whatever else is going on in the project's main sandbox).
- All sandbox interaction goes through the Go SDK's sandbox lifecycle, filesystem, process/exec, and git-adjacent capabilities — the backend must never shell out to a local Docker daemon for user code; the sandbox is the only place user/agent-generated code executes.
- The Terminal modal's WebSocket session (Section 6.3) must attach to the **same live sandbox** the agents are using for that project, not a separate one, so the user can literally watch/interact with what the agents are doing.
- Configure sandbox idle/auto-stop behavior conservatively (e.g. auto-stop after a period of no activity) to control cost, and auto-resume transparently the next time the project is opened or an agent needs it.

---

## 9. Data & Storage Layer

### 9.1 PostgreSQL — Core Entities

- `users` — id, email, password hash (or OAuth identity), plan tier, created_at.
- `projects` — id, user_id, name, status, git_remote_url, sandbox_id (nullable), created_at, updated_at.
- `messages` — id, project_id, role, content, created_at.
- `skills` — id, user_id, name, description, content, source (`uploaded`/`manual`), created_at.
- `secrets_refs` — id, project_id, type, keyvault_secret_name, keyvault_secret_version, created_at (never a raw value column).
- `jobs` — as defined in Section 6.5.
- `deployments` — id, project_id, job_id, kind (`app`/`llm`), endpoint_url, details (JSONB: topology, model, ports, etc.), created_at.

### 9.2 Redis

- Job queue backing store (`asynq` or equivalent).
- Pub/Sub channels for job progress and chat token streaming, consumed by the API process and relayed to WebSocket clients.
- Optional short-lived caching of sandbox file-tree listings to avoid hammering Daytona on rapid File Explorer refreshes.

### 9.3 Azure Blob Storage

- Store build/deploy job logs (full stdout/stderr) as blobs keyed by `job_id`, with only a truncated tail persisted in Postgres for quick display, and a link to fetch the full blob on demand.
- Store generated-codebase archive snapshots (e.g. a zip of the sandbox workspace at key milestones) for download/audit purposes.

### 9.4 Secrets / Vault

Store the following secret types, each written to Azure Key Vault immediately on submission, never persisted in plaintext elsewhere:

- `github_pat` — single string value (the PAT).
- `azure_credentials` — a JSON object with `client_id`, `client_secret`, `tenant_id`, `subscription_id` (service principal credentials used by `azidentity` for all Azure provisioning on the user's behalf).
- `huggingface_token` — single string value.
- `nvidia_nim_token` — single string value (NGC API key used for `nvcr.io` auth and/or the NIM cloud endpoint).

The backend's secrets package should expose a single `GetSecret(ctx, projectID, secretType) (value, error)` used by every agent tool — agents never touch Key Vault directly, only through this package, and only ever hold the decrypted value in memory for the duration of the single tool call that needs it.

---

## 10. Security Requirements

- All inbound traffic over HTTPS/WSS only; terminate TLS at a load balancer/ingress in front of the Go API.
- JWT auth on every REST and WebSocket route; WebSocket auth via a short-lived token issued by a REST call just before connecting (do not pass long-lived JWTs as query params on the WS URL).
- Principle of least privilege for the Azure service principal instructions given to users: document (in-app help text near the Azure credential prompt) that the service principal should be scoped to a dedicated resource group where possible.
- Rate-limit the message-send and job-creation endpoints per user to prevent runaway cloud spend from accidental rapid-fire requests.
- Sanitize/validate all file paths passed to the File Explorer and Daytona filesystem endpoints to prevent path traversal outside the sandbox workspace root.

---

## 11. Observability, Logging, Job Queue, Monitoring

- Structured (JSON) logs from both the API and worker processes, including `project_id`/`job_id` correlation fields on every log line related to an agent run.
- Prometheus metrics: job duration histograms per job type, job success/failure counters, active WebSocket connection gauge, queue depth gauge.
- Grafana dashboards built on top of those metrics (job throughput, failure rate by agent type, average time-to-deploy).
- Alerting on sustained job failure rate or queue backlog growth.

---

## 12. Deployment Architecture (Production-Grade)

- Containerize the frontend (static build served via Nginx or a small Go/Node static server) and the backend's two binaries (`api`, `worker`) as separate Docker images.
- Run `api` and `worker` as independently scalable deployments (Kubernetes recommended for the platform's own infrastructure, mirroring the same AKS pattern the LLM Deployer Agent offers to users) so a burst of long-running agent jobs never starves incoming HTTP/WebSocket traffic.
- Managed PostgreSQL (e.g. Azure Database for PostgreSQL) and managed Redis (e.g. Azure Cache for Redis) rather than self-hosted, for production durability.
- CI/CD: build/test/lint on every push, build and push versioned container images on merge to main, deploy via a standard rolling-update strategy.
- Environment configuration strictly via environment variables / mounted config, never hardcoded — including all Azure tenant defaults, Key Vault URI, Postgres/Redis connection strings, and the Daytona API key/target region used by the platform's own backend to talk to Daytona (distinct from any per-project user secrets).

---

## 13. Build Order / Implementation Roadmap

Build in this order so the system is testable at every stage:

1. **Skeleton:** Go module + `chi`/`gin` server with health-check route; Vite React app with the two-column shell and dark theme; Postgres schema migrations for `users`, `projects`, `messages`.
2. **Auth + Projects CRUD:** login/signup, JWT middleware, Projects REST endpoints, Projects modal in the frontend, project-switching context.
3. **Chat plumbing without agents:** `POST /messages` that just echoes back a canned assistant reply, WebSocket streaming of that reply token-by-token, full chat UI (composer, suggestion chips, empty state, message list) wired end-to-end.
4. **Daytona integration:** backend wrapper around the Daytona Go SDK; Terminal modal + WebSocket PTY proxy working against a real sandbox; File Explorer modal reading a real sandbox file tree; Uncommitted Git Files sidebar list reading real `git status` from a sandbox.
5. **Secrets + Key Vault:** secrets REST endpoints, Key Vault wrapper, the generic credential-prompt frontend flow, wired first to `github_pat` so Git Remote URL + Push can go fully live end-to-end.
6. **Skills:** Skills REST endpoints + modal (upload and manual create), stored and retrievable, even before any agent actually consumes them.
7. **Master Agent + App Developer Agent:** ADK agent runtime stood up, intent classification working for at least `build_app` vs `general_or_other`, App Developer's follow-up-question loop, codegen job writing real files into a real sandbox, Uncommitted Files list reflecting the new files.
8. **App Maintainer Agent:** since it reuses git clone/PAT plumbing already built in step 5, add repo clone → reproduce → fix → push.
9. **App Deployer Agent:** Azure credential collection, Dockerfile generation, image build/push, VM provisioning, app running and reachable.
10. **LLM Deployer Agent:** all three topology sub-paths (VM+vLLM, AKS+LB, VM+NIM), HF and NIM token collection, endpoint-details reporting.
11. **Hardening pass:** rate limiting, path-traversal checks, structured logging, Prometheus metrics, CI/CD, production deployment manifests.

---

## 14. Reference Documentation

**Agent framework**
- Google ADK for Go — getting started: https://google.github.io/adk-docs/get-started/go/
- Google ADK for Go — package reference: https://pkg.go.dev/google.golang.org/adk
- Google ADK — agents & multi-agent concepts: https://google.github.io/adk-docs/agents/
- Google ADK Go source: https://github.com/google/adk-go
- LangGraph (optional orchestration alternative): https://docs.langchain.com/oss/python/langgraph/overview and https://www.langchain.com/langgraph

**Sandboxing**
- Daytona documentation home: https://www.daytona.io/docs/en/
- Daytona Go SDK reference: https://www.daytona.io/docs/en/go-sdk/daytona/
- Daytona getting started: https://www.daytona.io/docs/en/getting-started/
- Daytona sandbox lifecycle reference: https://www.daytona.io/docs/en/sandboxes/

**Frontend terminal**
- xterm.js official site and quick start: https://xtermjs.org/
- xterm.js GitHub repo (addons, API surface): https://github.com/xtermjs/xterm.js/

**Azure SDK for Go**
- Overview of Azure SDK for Go management libraries: https://learn.microsoft.com/en-us/azure/developer/go/management-libraries
- Control-plane operations guide (VM/compute provisioning pattern): https://learn.microsoft.com/en-us/azure/developer/go/control-plane
- `azidentity` authentication reference (via package docs linked from the above)
- Azure Key Vault Go client (secrets — `azsecrets`): https://learn.microsoft.com/en-us/azure/key-vault/secrets/quick-create-go and https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets
- AKS public standard load balancer (for the LLM Deployer's AKS+LB path): https://learn.microsoft.com/en-us/azure/aks/load-balancer-standard
- AKS internal load balancer reference: https://learn.microsoft.com/en-us/azure/aks/internal-lb

**Containerization**
- Docker Engine API / Go SDK guide: https://docs.docker.com/reference/api/engine/sdk/
- Docker Engine SDK usage examples (Go): https://docs.docker.com/reference/api/engine/sdk/examples/

**LLM serving**
- vLLM OpenAI-compatible server docs: https://docs.vllm.ai/en/latest/serving/online_serving/openai_compatible_server/
- NVIDIA NIM overview: https://docs.nvidia.com/nim/index.html
- NVIDIA NIM deployment guide (NGC auth, `nvcr.io` login): https://docs.nvidia.com/brev/latest/guides/inference-deployment/deploying-nims
- NVIDIA developer NIM guide (API key / catalog flow): https://developer.nvidia.com/blog/a-simple-guide-to-deploying-generative-ai-with-nvidia-nim/

**Model source**
- Hugging Face Hub authentication / tokens: https://huggingface.co/docs/huggingface_hub/quick-start
- Hugging Face Hub API reference: https://huggingface.co/docs/huggingface_hub/package_reference/hf_api

**Source control**
- GitHub REST API — authenticating (PAT usage): https://docs.github.com/en/rest/authentication
- Managing personal access tokens: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens
- Getting started with the GitHub REST API: https://docs.github.com/en/rest/guides/getting-started-with-the-rest-api

---

**End of specification.** Implement strictly in the order given in Section 13; every section above is a hard requirement, not a suggestion, unless explicitly marked optional.
