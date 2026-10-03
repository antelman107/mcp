# MCP Servers Collection (Go)

This repository is structured as a **multi-server MCP collection**.  
Today it contains one server:

| Server | Binary | Package path |
|---|---|---|
| AntalyaKart transit tools | `antalyakart-mcp` | `github.com/antelman107/mcp/cmd/antalyakart-mcp` |

The same repo layout can host more MCPs over time under new `cmd/<server-name>` directories.

---

## Distribution roadmap options (A / B / C)

If we want to simplify onboarding further over time, these are the planned options:

### Option A (next best step): GitHub Releases + package managers
- Keep the Go binary authoritative.
- Auto-build release artifacts for macOS/Linux/Windows.
- Publish install channels such as:
  - Homebrew tap (macOS/Linux)
  - Scoop bucket (Windows)
  - direct release binaries with checksums

**Best for**: local installs without requiring Go.

### Option B: Docker image + deployment recipe
- Publish a container image that runs this MCP server.
- Provide a deployment recipe for teams who want infra-managed execution.

**Best for**: teams that standardize on containers.

### Option C: Combined rollout (recommended order)
1. Deliver Option A first (fastest user impact for installs).
2. Add Option B after that for infra/container users.

This repo currently ships the `go install` path and is ready for Option A next.

---

## Quick install (recommended): `go install`

Install from GitHub:

```bash
go install github.com/antelman107/mcp/cmd/antalyakart-mcp@latest
```

This puts the binary in:
- `$GOBIN` (if set), or
- `$(go env GOPATH)/bin`

Make sure that folder is in your `PATH`, then start it:

```bash
antalyakart-mcp
```

It listens for MCP Streamable HTTP on `:8090` (override with `MCP_ADDR`). The endpoint is `/mcp`. A liveness check is `GET /healthz`.

---

## Alternative install methods

### Build from source in this repo

```bash
git clone https://github.com/antelman107/mcp.git
cd mcp
go build -o bin/antalyakart-mcp ./cmd/antalyakart-mcp
```

### Install locally from current checkout

```bash
go install ./cmd/antalyakart-mcp
```

---

## Configure MCP clients

Point the client at the HTTP endpoint. Locally that is `http://127.0.0.1:8090/mcp`. On the Turkish host it is `https://mcp.goturkey.club/mcp`.

### 1) Cursor

Config file:
- project-level: `.cursor/mcp.json`
- global: `~/.cursor/mcp.json`

```json
{
  "mcpServers": {
    "antalyakart": {
      "url": "https://mcp.goturkey.club/mcp"
    }
  }
}
```

### 2) Claude Code

```bash
claude mcp add --scope user --transport http antalyakart http://127.0.0.1:8090/mcp
```

### 3) Codex

Config file:
- user: `~/.codex/config.toml`
- project: `.codex/config.toml`

```toml
[mcp_servers.antalyakart]
url = "http://127.0.0.1:8090/mcp"
```

---

## AntalyaKart server details

This server wraps APIs used by:
- `https://online.antalyakart.com.tr/#/home`

### Tools exposed

- `search_routes_and_stops`
- `nearby_places_stops_and_kiosks`
- `nearest_buses_for_stop`
- `route_path_and_vehicles`
- `plan_direct_trip_between_stops`
- `summarize_stop_arrivals`

### Optional environment variables

- `MCP_ADDR` (default: `:8090`)
- `ANTALYAKART_BASE_URL` (default: `https://service.kentkart.com/rl1`)
- `ANTALYAKART_REGION` (default: `026`)
- `ANTALYAKART_LANG` (default: `tr`)
- `ANTALYAKART_AUTH_TYPE` (default: `4`)

### First prompts to test

1. Search Antalya routes and stops for `otogar`.
2. Show nearby stops and kiosks around `36.897087, 30.713777`.
3. Summarize arrivals at stop `10828`.
4. Plan a direct trip from `otogar` to `markantalya`.

---

## Adding more MCP servers to this repo

Suggested pattern:

- `cmd/<new-server>/main.go` for the MCP entrypoint
- `internal/<domain>/...` for shared implementation
- `docs/<server>-...` for endpoint analysis / OpenAPI notes

Then users can install each server independently with:

```bash
go install github.com/antelman107/mcp/cmd/<new-server>@latest
```

---

## API investigation artifacts (AntalyaKart)

- Reverse engineering notes: `docs/investigation-notes.md`
- OpenAPI description: `docs/antalyakart-openapi.yaml`
