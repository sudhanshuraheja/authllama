# Auth Lllama

A lightweight proxy for Ollama that adds file-based authentication and request control.

# Tasks

A small wrapper over Ollama to add:

- File-based auth header verification
  - Local file with service-name and expected auth header
  - Reject unauthorized requests with clear error
  - Reload file on change (or via endpoint)
  - All logic must use file-based configuration

- Support for all Ollama APIs (each with file-based config support)
  - Pass-through `/api/generate` endpoint
  - Pass-through `/api/chat` endpoint
  - Pass-through `/api/tags` endpoint
  - Pass-through `/api/show` endpoint
  - Pass-through `/api/pull` endpoint
  - Pass-through `/api/push` endpoint
  - Pass-through `/api/create` endpoint
  - Pass-through `/api/delete` endpoint
  - Support GET, POST, and model-specific endpoints
  - Log basic request metadata
  - All routing and controls based on file configuration

- Support for streaming
  - Handle streamed responses (`Transfer-Encoding: chunked`)
  - Ensure proper error handling during stream interruptions

- Rate limiting per service-name with a local file
  - Define RPM/TPS per service in config
  - Track usage per service
  - Return 429 on limit breach
  - Graceful handling and logging of rate-limit events
  - Entire logic driven by file-based limits

- Config reload
  - Reload auth and rate-limit config file without restart
  - File watcher or HTTP endpoint to trigger reload
  - All reloadable data must originate from files

- Observability
  - Track request count, auth failures, rate-limit events
  - Optionally expose Prometheus-compatible `/metrics` endpoint
  - Metrics config and logging preferences via file

- Optional enhancements (also file-driven)
  - JWT/Bearer token auth support
  - Admin-only `/admin/status` to view loaded config and usage
  - Audit logs for access and rejections
  - Integration test harness for multi-service load testing