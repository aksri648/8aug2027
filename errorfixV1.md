# Codebase Audit — Error Fix V1

**Audited:** 2026-08-07  
**Scope:** Go backend, worker/queue, agent integrations, and React frontend.  
**Validation:** `go test ./...` and `npm run build` both pass. No automated tests exist, so passing builds do not exercise the findings below.

## Summary

The application is largely a demo implementation presented as a production platform. Several controls accept user input but do not use it, while deployment, terminal, preview, git, and agent-maintenance endpoints can return successful results without performing the claimed action. Authentication and ownership enforcement are also absent.

## Findings

| ID | Severity | Finding |
|---|---|---|
| EFV1-01 | Critical | Authentication is bypassable and passwords are not verified or safely stored. |
| EFV1-02 | Critical | Every project, message, skill, secret, job, file, and WebSocket endpoint lacks authorization/ownership checks. |
| EFV1-03 | High | The provider test/discovery APIs are server-side request forgery (SSRF) primitives. |
| EFV1-04 | High | Azure, LLM, maintenance, terminal, preview, git, and Daytona features report fabricated success. |
| EFV1-05 | High | The code generator ignores the requested application and selected stack. |
| EFV1-06 | High | Jobs bypass the Redis worker and are marked successful even after agent errors. |
| EFV1-07 | High | Task Wizard deployment parameters are discarded before execution. |
| EFV1-08 | High | Secrets are persisted as plaintext and sensitive configuration is stored in browser local storage. |
| EFV1-09 | Medium | WebSocket origin checks are disabled and broadcasts can concurrently write to the same connection. |
| EFV1-10 | Medium | Invalid input and database errors are ignored, causing panics or false success responses. |
| EFV1-11 | Medium | Deployment can panic on a short project ID and ignores Azure HTTP failures. |
| EFV1-12 | Low | File browsing, Git diff/status, metrics, and preview behavior are incomplete or misleading. |

## Detailed findings and fixes

### EFV1-01 — Authentication is bypassable and passwords are unsafe

**Evidence**

- [server.go](backend/internal/api/server.go#L134) ignores JSON-decoding errors and looks up only the email; the submitted password is never checked.
- If the email is not found, it falls back to the seeded `user-default` account ([server.go](backend/internal/api/server.go#L141)). Any unknown credentials therefore receive a token for that account.
- Tokens are predictable strings (`dev-jwt-token-<user-id>`) and no middleware validates them ([server.go](backend/internal/api/server.go#L146)).
- Signup directly stores the submitted password in `PasswordHash` without hashing ([store.go](backend/internal/store/store.go#L149)).

**Impact:** Account takeover of the default account, plaintext password exposure, and no meaningful session authentication.

**Fix:** Validate input; hash passwords with Argon2id/bcrypt; compare hashes during login; return 401 for unknown/invalid credentials; issue signed, expiring tokens; add authentication middleware for all protected routes.

### EFV1-02 — No authorization or project ownership enforcement

**Evidence**

- Project listing and creation always use `user-default` ([server.go](backend/internal/api/server.go#L166)).
- Direct object routes call `GetProject`, `UpdateProject`, `DeleteProject`, and other store methods by path ID alone, with no authenticated principal or owner comparison ([server.go](backend/internal/api/server.go#L185)).
- Message, secrets, sandbox files, Git data, jobs, terminal sessions, and project WebSockets likewise trust the supplied project ID ([server.go](backend/internal/api/server.go#L266)).

**Impact:** Any caller that can reach the API can read, alter, delete, or trigger actions for arbitrary projects and access their secret-presence and job information.

**Fix:** Derive a user ID from verified authentication; load every project through `WHERE id = ? AND user_id = ?`; apply the same authorization policy to nested resources and WebSockets before upgrading the connection.

### EFV1-03 — Provider endpoints allow SSRF

**Evidence**

- `POST /providers/test` makes a server-side HTTP request to the caller-provided `base_url` ([server.go](backend/internal/api/server.go#L657)).
- `POST /providers/discover` does the same ([server.go](backend/internal/api/server.go#L717)). Neither validates scheme, hostname, DNS result, redirect target, or private/link-local addresses.
- Both endpoints are unauthenticated and forward a caller-supplied bearer token to that URL.

**Impact:** An attacker can make the backend probe internal services or cloud metadata endpoints and potentially leak a supplied API key to an attacker-controlled destination.

**Fix:** Require authentication; use an allowlist of configured provider origins; reject non-HTTPS, loopback, link-local, private, and reserved IP ranges after DNS resolution; disable redirects or revalidate each redirect; never forward credentials outside an approved origin.

### EFV1-04 — Claimed integrations are hardcoded simulations

**Evidence**

- The terminal accepts any token and `processTerminalCmd` returns canned output; unknown commands are reported as successful rather than executed ([websocket.go](backend/internal/api/websocket.go#L93), [websocket.go](backend/internal/api/websocket.go#L140)).
- The live preview serves a fixed HTML mock and manufactures endpoint responses in client-side JavaScript ([server.go](backend/internal/api/server.go#L582)).
- Git push only clears the in-memory Git status; it neither commits nor contacts the configured remote ([server.go](backend/internal/api/server.go#L411)).
- App maintenance writes one canned `main.go` into a *different*, in-memory sandbox, then fabricates a commit hash, diagnosis, verification, and remote URL ([appmaintainer.go](backend/internal/agents/appmaintainer/appmaintainer.go#L33)).
- Remote Daytona calls silently return an emulator/success result after network failure ([daytona.go](backend/internal/agents/shared/daytona.go#L111), [daytona.go](backend/internal/agents/shared/daytona.go#L172)).
- LLM deployment returns a time-derived public IP and fixed GPU details without provisioning any resource ([llmdeployer.go](backend/internal/agents/llmdeployer/llmdeployer.go#L36)).

**Impact:** Users can make infrastructure, source-control, debugging, and support decisions based on actions that never occurred.

**Fix:** Mark demo modes explicitly and prevent them in production. Replace mocks with real integration adapters that return failures when an operation fails. Persist remote workspace/session IDs, execute against those resources, poll asynchronous operations, and only report success after verification.

### EFV1-05 — Code generation discards user requirements

**Evidence**

- The generator reads the prompt then explicitly discards it (`_ = prompt`) ([appdeveloper.go](backend/internal/agents/appdeveloper/appdeveloper.go#L31)).
- It always writes the same five Go/Docker files and always reports `Go 1.22 REST API + Docker`, regardless of the selected stack or requested features ([appdeveloper.go](backend/internal/agents/appdeveloper/appdeveloper.go#L37)).

**Impact:** Requests for React, Python, Next.js, or specific application behavior silently produce the same unrelated Go service, possibly overwriting existing sandbox files.

**Fix:** Pass structured requirements through the job payload; select a generator/template from the requested stack; validate output; avoid overwriting files without an explicit overwrite policy; return the actual generated stack and file set.

### EFV1-06 — Queue is bypassed and job failures are reported as successes

**Evidence**

- API requests create a database job but immediately start `runAsyncJob` in the API process; they never enqueue it in `RedisQueue` ([server.go](backend/internal/api/server.go#L303)).
- `runAsyncJob` ignores every agent error (`res, _ := ...`) and emits `succeeded` afterward ([server.go](backend/internal/api/server.go#L328)).
- The worker only consumes `DequeueJobs`; it is therefore idle for jobs created through the API and also ignores malformed payloads and unknown job types without updating job state ([main.go](backend/cmd/worker/main.go#L39)).
- The in-memory fallback queue loses all queued work on process restart and cannot coordinate separate API/worker processes ([queue.go](backend/internal/queue/queue.go#L23)).

**Impact:** Job status cannot be trusted. Failed work appears successful, retries/DLQ are unused, and running a worker does not process API-created work.

**Fix:** Inject one queue service into the API; enqueue every created job; let workers be the only executors; atomically set `running`, `succeeded`, or `failed` with result/error data; implement retries and DLQ use; do not substitute a non-durable queue in production.

### EFV1-07 — Task Wizard selections do not reach agents

**Evidence**

- The wizard builds `agentPayload` with repository, region, VM size, model, GPU, stack, and test flags, but calls the parent with both values ([TaskWizardModal.jsx](frontend/src/components/modals/TaskWizardModal.jsx#L31)).
- `App` accepts only one callback argument and sends only the plain prompt as `{content}` ([App.jsx](frontend/src/App.jsx#L169)).
- The master agent creates a payload containing only `prompt` and `project_id` ([master.go](backend/internal/agents/master/master.go#L122)).
- The deployer uses hardcoded `eastus` and `Standard_B2s`, and the LLM deployer sees no wizard model/topology/GPU selection ([appdeployer.go](backend/internal/agents/appdeployer/appdeployer.go#L63), [llmdeployer.go](backend/internal/agents/llmdeployer/llmdeployer.go#L36)).

**Impact:** The UI suggests control over deployments and generation, but all structured selections are silently dropped.

**Fix:** Define a typed task request API that accepts `agent_type` and a JSON payload; validate it server-side; persist that payload in the job; have each agent consume only its documented fields.

### EFV1-08 — Secrets are insecurely stored

**Evidence**

- `SaveSecret` stores the submitted value directly in SQLite/Postgres as `SecretValue`; there is no encryption, secret-manager integration, or type validation ([store.go](backend/internal/store/store.go#L285)).
- The frontend configuration screen places Anthropic, OpenAI, DeepSeek, Daytona, Azure, and custom-provider secrets in `localStorage` ([ConfigModal.jsx](frontend/src/components/modals/ConfigModal.jsx#L8), [ConfigModal.jsx](frontend/src/components/modals/ConfigModal.jsx#L43)).
- Configuration values saved in the browser are not wired into server-side integrations, so they are both exposed and ineffective.

**Impact:** Any same-origin script/XSS, browser-profile access, database backup, or DB reader can extract credentials. Users may believe the browser-stored values configure deployments when they do not.

**Fix:** Keep secrets server-side in a managed secret store (or encrypt at rest with envelope encryption); never place long-lived credentials in localStorage; validate secret types; authorize access by project owner; use server-side configuration references rather than browser-only values.

### EFV1-09 — WebSockets accept hostile origins and can race

**Evidence**

- `CheckOrigin` always returns true ([websocket.go](backend/internal/api/websocket.go#L16)).
- Multiple goroutines can call `BroadcastEvent` and invoke `WriteMessage` on the same Gorilla connection concurrently ([websocket.go](backend/internal/api/websocket.go#L51)). Gorilla WebSocket requires at most one concurrent writer.
- Slow/dead connections are written while the hub read lock is held, blocking all hub mutation and broadcasts.

**Impact:** Cross-site WebSocket access is possible; concurrent job/status events can corrupt connections or panic under load; one stalled client can degrade all streams for the project.

**Fix:** Enforce an origin allowlist and authenticated upgrade. Give each connection a buffered outbound channel and one dedicated writer goroutine, with write deadlines and cleanup on failure.

### EFV1-10 — Ignored errors cause panics and misleading responses

**Evidence**

- Signup discards `CreateUser` errors and immediately dereferences `u.ID`; a duplicate email can yield a nil-pointer panic ([server.go](backend/internal/api/server.go#L152)).
- Message creation discards database errors, then dereferences `asstMsg.ID` ([server.go](backend/internal/api/server.go#L272)).
- Delete, create-skill, save-secret, update-job, and many JSON decode operations ignore errors while still returning success ([server.go](backend/internal/api/server.go#L211), [store.go](backend/internal/store/store.go#L279)).
- Git push calls `WriteHeader(428)` and then `writeJSON`, which calls `WriteHeader(428)` again ([server.go](backend/internal/api/server.go#L483)).

**Impact:** Invalid requests or database failures can crash handlers, leave partial data, or incorrectly confirm success. The double header write produces a server warning and is unnecessary.

**Fix:** Reject malformed JSON and validate required fields. Propagate and handle every store/agent error. Use transactions for dependent writes. Call `writeJSON` once per response and return appropriate 4xx/5xx errors.

### EFV1-11 — Azure deployment can panic and treats failed provisioning as success

**Evidence**

- The deployer slices `projectID[:8]` without validating the ID length ([appdeployer.go](backend/internal/agents/appdeployer/appdeployer.go#L49)). Project IDs come directly from routes, and message handling does not first verify project existence/format.
- Missing Azure configuration is replaced with an all-zero subscription ID; a synthetic IP and `simulated` status are produced ([appdeployer.go](backend/internal/agents/appdeployer/appdeployer.go#L44)).
- Resource-group and VM errors/HTTP status codes are ignored, but the job is still updated to `succeeded` ([appdeployer.go](backend/internal/agents/appdeployer/appdeployer.go#L63)). The VM request body also lacks required deployment configuration.

**Impact:** A short project ID can crash the async execution process. Failed/unauthorized Azure calls look like completed public deployments.

**Fix:** Use validated UUIDs/opaque IDs and a safe slug. Fail fast when required Azure credentials/configuration are absent. Check every ARM response status, implement long-running-operation polling, construct valid VM resources, and update the job/project only after successful deployment.

### EFV1-12 — Remaining incomplete/misleading platform behavior

**Evidence**

- `GetGitDiff` generates an all-additions view from the current in-memory file instead of calculating a Git diff; a missing file is labeled as a deletion but shown with `+++` ([daytona.go](backend/internal/agents/shared/daytona.go#L316)).
- `GetGitStatus` returns the internal slice directly, and preview reads `len(sb.Files)` without the sandbox lock, creating race risk with writes ([daytona.go](backend/internal/agents/shared/daytona.go#L307), [server.go](backend/internal/api/server.go#L568)).
- The file explorer renders directory entries as selectable files and never navigates into them ([FileExplorerModal.jsx](frontend/src/components/modals/FileExplorerModal.jsx#L76)).
- Prometheus metrics are declared but no HTTP middleware records request counts/durations, so exported values do not represent API traffic ([metrics.go](backend/internal/metrics/metrics.go#L8)).
- Model discovery returns a fixed model list when the provider fails, making an outage look like a successful discovery ([server.go](backend/internal/api/server.go#L739)).

**Impact:** Users see inaccurate source-control, file-system, service-health, and model-discovery information; concurrent access can race.

**Fix:** Use a real Git backend and return immutable copies of shared data under lock. Add directory navigation. Instrument routes with the declared metrics. Return an explicit provider-unavailable error (or visibly labeled cached data), never invented discovery results.

## Recommended remediation order

1. Disable public exposure until EFV1-01, EFV1-02, EFV1-03, and EFV1-08 are fixed.
2. Remove or clearly gate all simulation paths (EFV1-04 through EFV1-07) before claiming operational integrations.
3. Make the worker and job state machine authoritative, including failure/retry handling.
4. Add unit/integration coverage for authentication, authorization, task-payload propagation, failed agent operations, WebSocket concurrency, and Azure/Daytona adapter errors.
