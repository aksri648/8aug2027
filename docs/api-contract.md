# API Contract: SaaS Agent Platform

## Version: v1
Base URL: `/api/v1`

---

## 1. Authentication
All endpoints except `/api/v1/auth/*` require a Bearer token in the `Authorization` header:
`Authorization: Bearer <jwt_token>`

### Endpoints
- `POST /api/v1/auth/login`
  - Request: `{"email": "string", "password": "string"}`
  - Response: `{"token": "string", "user": {"id": "uuid", "email": "string", "plan": "string"}}`
- `POST /api/v1/auth/signup`
  - Request: `{"email": "string", "password": "string"}`
  - Response: `{"token": "string", "user": {"id": "uuid", "email": "string", "plan": "string"}}`

---

## 2. Projects API

### `GET /api/v1/projects`
List all projects for the authenticated user.
- Response: `[{"id": "uuid", "name": "string", "status": "draft|building|deployed|error", "git_remote_url": "string", "sandbox_id": "string", "created_at": "ISO8601", "updated_at": "ISO8601"}]`

### `POST /api/v1/projects`
Create a new project.
- Request: `{"name": "string"}`
- Response: Project object.

### `GET /api/v1/projects/{projectId}`
Get detailed project state.
- Response: Project object.

### `PATCH /api/v1/projects/{projectId}`
Update project attributes (e.g. `git_remote_url`, `name`).
- Request: `{"git_remote_url": "string", "name": "string"}`
- Response: Updated Project object.

### `DELETE /api/v1/projects/{projectId}`
Delete a project.
- Response: `{"status": "deleted"}`

---

## 3. Skills API

### `GET /api/v1/skills`
List skills for user/account.
- Response: `[{"id": "uuid", "name": "string", "description": "string", "content": "string", "source": "uploaded|manual", "updated_at": "ISO8601"}]`

### `POST /api/v1/skills`
Create skill manually.
- Request: `{"name": "string", "description": "string", "content": "string"}`
- Response: Skill object.

### `POST /api/v1/skills/upload`
Upload skill markdown files (`multipart/form-data`).
- Files field: `files` (array of `.md` files)
- Response: `[Skill]`

### `DELETE /api/v1/skills/{skillId}`
Delete a skill.
- Response: `{"status": "deleted"}`

---

## 4. Chat & Agent Messages API

### `GET /api/v1/projects/{projectId}/messages`
Fetch chat message history for project.
- Response: `[{"id": "uuid", "role": "user|assistant|system-status", "content": "string", "created_at": "ISO8601"}]`

### `POST /api/v1/projects/{projectId}/messages`
Send user prompt to Master Agent.
- Request: `{"content": "string"}`
- Response: `{"user_message": Message, "assistant_message_id": "uuid", "job_id": "uuid"}`
- *Note:* Real-time response tokens and agent progress are streamed via WebSocket.

---

## 5. Sandbox Files & Git API

### `GET /api/v1/projects/{projectId}/files?path=/`
List files in project sandbox directory.
- Response: `[{"name": "string", "path": "string", "is_dir": boolean, "size": number}]`

### `GET /api/v1/projects/{projectId}/files/content?path=...`
Read content of a file.
- Response: `{"path": "string", "content": "string"}`

### `GET /api/v1/projects/{projectId}/git/status`
List git status of project sandbox.
- Response: `{"uncommitted": [{"path": "string", "status": "A|M|D|??"}]}`

### `GET /api/v1/projects/{projectId}/git/diff?path=...`
Fetch git unified diff for file.
- Response: `{"path": "string", "diff": "string"}`

### `POST /api/v1/projects/{projectId}/git/push`
Stage, commit, and push project code to remote.
- Request: `{"commit_message": "string"}`
- Response: `{"job_id": "uuid", "status": "queued"}`

---

## 6. Secrets API

### `POST /api/v1/projects/{projectId}/secrets`
Save secret to Azure Key Vault.
- Request:
  ```json
  {
    "type": "github_pat" | "azure_credentials" | "huggingface_token" | "nvidia_nim_token",
    "value": "string" | { ... }
  }
  ```
- Response: `{"status": "stored", "type": "string"}`

### `GET /api/v1/projects/{projectId}/secrets`
Check presence of secret types (never values).
- Response: `{"github_pat": boolean, "azure_credentials": boolean, "huggingface_token": boolean, "nvidia_nim_token": boolean}`

---

## 7. Jobs API

### `GET /api/v1/jobs/{jobId}`
Get job execution status and results.
- Response: `{"id": "uuid", "project_id": "uuid", "type": "string", "status": "queued|running|succeeded|failed|cancelled", "result": {}, "error": "string", "updated_at": "ISO8601"}`

---

## 8. Terminal API

### `POST /api/v1/projects/{projectId}/terminal/session`
Request sandbox PTY session.
- Response: `{"session_token": "string", "websocket_url": "ws://.../api/v1/terminal/{session_token}"}`

---

## 9. WebSocket Endpoints

### `WS /api/v1/projects/{projectId}/stream`
Multiplexed real-time event stream.
Frames emitted by server:
- `{"type": "chat_token", "message_id": "uuid", "delta": "string"}`
- `{"type": "chat_message_complete", "message_id": "uuid"}`
- `{"type": "system_status", "job_id": "uuid", "agent": "string", "text": "string", "level": "info|success|error"}`
- `{"type": "git_status_changed", "uncommitted": [...]}`
- `{"type": "job_update", "job_id": "uuid", "status": "string", "result": {}}`

### `WS /api/v1/terminal/{sessionToken}`
Raw bidirectional WebSocket byte stream for xterm.js PTY interaction.
