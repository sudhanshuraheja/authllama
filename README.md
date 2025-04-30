# Auth Lllama

A lightweight proxy for Ollama that adds file-based authentication and request control.

# Features

- File-based authentication
  - Local file with service-name and expected auth header
  - Reject unauthorized requests with 401
  - Reloads config on file change and via `/admin/reload`
  - All logic driven by config file (`config/config.json`)

- Full Ollama API passthrough
  - Supports `/api/generate`, `/api/chat`, `/api/tags`, `/api/show`, `/api/pull`, `/api/push`, `/api/create`, `/api/delete`
  - Supports GET and POST
  - Logs basic request metadata
  - Controlled by per-service config

- Streaming support
  - Detects `"stream": true` requests
  - Streams responses with flush and chunking
  - Timeout on inactive stream
  - Handles write/read/timeout errors

- Rate limiting per service
  - Defined as TPM (transactions per minute) in config
  - Tracks usage per service
  - Returns 429 on limit breach
  - Logs rate-limit events
  - Reloadable from config file and via `/admin/reload`

- Config reload
  - Uses `fsnotify` to detect file changes
  - Also supports `/admin/reload` endpoint with `X-Admin-Key` protection
  - Reloads both auth and rate-limit configs

- Observability
  - Tracks: request count, auth failures, rate-limit responses, stream sessions
  - Logs metrics every 30 seconds to stdout

- Graceful shutdown
  - Handles `SIGINT`/`SIGTERM` and cleanly shuts down the HTTP server