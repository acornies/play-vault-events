# Copilot Instructions

## Project Overview

This repository demonstrates **HashiCorp Vault Enterprise event notifications** (v1.21+ Enterprise-only feature) using three components that work together:

1. **Docker Compose** – Runs Vault Enterprise in dev mode (root token: `root`, port 8200)
2. **vault-simulate** – Go CLI that generates synthetic KV write traffic to produce Vault events
3. **godot/** – Godot 4.6 GDScript client that subscribes to Vault's WebSocket event stream and visualizes events in real time

Typical workflow: start Vault via Docker → run `vault-simulate` to write secrets → Godot client receives events over WebSocket and animates.

## Build & Test (vault-simulate)

All commands run from the `vault-simulate/` directory.

```bash
# Build
go build -o vault-simulate .

# Run all tests
go test -v

# Run a single test
go test -run TestParseRequestTypes -v
go test -run TestValidateRequestTypes -v
go test -run TestContainsType -v

# Run with coverage
go test -cover
```

There is no Makefile or build automation — only direct `go` commands.

## Architecture

### vault-simulate (`vault-simulate/main.go`)

- Uses the official Vault Go SDK (`github.com/hashicorp/vault/api v1.22.0`) via `api.DefaultConfig()` which reads `VAULT_ADDR` and `VAULT_TOKEN` from the environment.
- CLI flags: `-duration`, `-num-requests`, `-min-interval`, `-max-interval`, `-request-types` (comma-delimited: `kv`, `ldap`, `database`), `-kv-path`
- **Only `kv` is implemented**; `ldap` and `database` are validated but rejected at runtime via `checkUnimplemented()`.
- Generates random intervals across the total duration via `generateRandomIntervals()`, then fires KV writes with unique keys (`simulate/key-{n}-{UnixNano}`).
- Graceful shutdown: listens for SIGINT/SIGTERM, then calls `cleanupKVMount()` which unmounts and re-enables the KV v2 mount (deletes all written secrets).

### godot/ (`godot/main.gd`)

- `@export var websocket_url` and `@export var auth_token` are configured in the Godot scene inspector (defaults: `ws://localhost:8200/v1/sys/events/subscribe/kv-v2/data-*?json=true` and `root`).
- Authenticates via custom WebSocket header `X-Vault-Token`.
- In `_process(delta)`, polls `WebSocketPeer` state and parses incoming JSON packets, emitting a `vault_event` signal and calling `logo.pulsate()` on each event.

### Docker

```bash
docker compose up -d       # Start Vault Enterprise (requires VAULT_LICENSE in .env)
docker compose down        # Stop
docker compose logs -f vault
```

`.env` holds the `VAULT_LICENSE` value (git-ignored).

## Key Conventions

### Error handling in vault-simulate
Fail-fast at startup: validate all CLI flags (`validateRequestTypes`), check required mounts (`checkMounts`), and reject unimplemented types (`checkUnimplemented`) before any requests are made.

### Go module path
`github.com/acornies/play-vault-events/vault-simulate` — used in import paths and test output.

### Vault SDK configuration
Always use `api.DefaultConfig()` to pick up `VAULT_ADDR` and `VAULT_TOKEN` from the environment; do not hardcode addresses.

### KV write format
Secrets are written to KV v2 at path `simulate/key-{requestNum}-{UnixNano}` with fields `timestamp` and `value`. The mount path is controlled by `-kv-path` (default: `simulate-secret`).

### Godot conventions
- Scripts use `extends Node` with `@onready` to cache node references.
- `@export` variables are configured in the scene inspector, not hardcoded.
- Animations are driven by `AnimationPlayer` in `logo.tscn` via `logo.pulsate()`.

### What's gitignored
`.env`, `docker-compose.override.yml`, `.LICENSE.txt`, and the compiled `vault-simulate/vault-simulate` binary.
