# AntalyaKart MCP Server (Go)

Unofficial MCP server that exposes AntalyaKart public transit endpoints as tools.

This server wraps the APIs used by:

- `https://online.antalyakart.com.tr/#/home`

## What tools you get

- `search_routes_and_stops`
- `nearby_places_stops_and_kiosks`
- `nearest_buses_for_stop`
- `route_path_and_vehicles`
- `plan_direct_trip_between_stops` (high-level direct route planner)
- `summarize_stop_arrivals` (route-grouped ETA summary)

---

## 1) Install / build the binary

### Prerequisites

- Go 1.22+ installed
- Git installed

### Build steps

```bash
git clone <your-fork-or-repo-url>
cd <repo-folder>
go build -o bin/antalyakart-mcp ./cmd/antalyakart-mcp
```

### Optional: install globally on your machine

```bash
mkdir -p "$HOME/.local/bin"
cp ./bin/antalyakart-mcp "$HOME/.local/bin/antalyakart-mcp"
chmod +x "$HOME/.local/bin/antalyakart-mcp"
```

Then ensure `~/.local/bin` is in your `PATH`.

### Optional: quick local sanity check

```bash
./bin/antalyakart-mcp
```

It is a stdio MCP server, so it waits for an MCP client and prints nothing human-friendly by default.

---

## 2) Configure in Cursor

Cursor supports project-level and global MCP config.

- Project-level (recommended for sharing with teammates):
  - `.cursor/mcp.json` in your repo root
- Global:
  - `~/.cursor/mcp.json`

Add:

```json
{
  "mcpServers": {
    "antalyakart": {
      "type": "stdio",
      "command": "/absolute/path/to/antalyakart-mcp",
      "args": []
    }
  }
}
```

If you built in this repo, `command` can be:

- `/absolute/path/to/repo/bin/antalyakart-mcp`

After saving:
1. Open Cursor MCP settings/panel
2. Ensure `antalyakart` server shows as connected
3. Ask your agent to call one of the tools (example prompts below)

---

## 3) Configure in Claude Code

You can add server config with CLI, or by editing `.mcp.json`.

### Option A: Add with CLI (recommended)

```bash
claude mcp add --scope user antalyakart -- /absolute/path/to/antalyakart-mcp
```

Notes:
- `--scope user` makes it available in all your projects.
- You can also use `--scope project` for project-local config.
- The `--` separator is required before the server command.

### Option B: Add manually with config file

Claude Code supports:
- project file: `.mcp.json`
- user-level file: `~/.claude.json`

Example config entry:

```json
{
  "mcpServers": {
    "antalyakart": {
      "type": "stdio",
      "command": "/absolute/path/to/antalyakart-mcp",
      "args": []
    }
  }
}
```

Then run:

```bash
claude mcp list
```

to confirm it is registered.

---

## 4) Configure in Codex

Codex MCP servers are configured in TOML:

- user-level: `~/.codex/config.toml`
- project-level: `.codex/config.toml` (trusted projects)

Add:

```toml
[mcp_servers.antalyakart]
command = "/absolute/path/to/antalyakart-mcp"
args = []
```

Optional fields you can add (if needed later):
- `env`
- `env_vars`
- `cwd`
- startup/tool timeout settings

---

## 5) Optional environment variables

Defaults already target Antalya production backend.

- `ANTALYAKART_BASE_URL` (default: `https://service.kentkart.com/rl1`)
- `ANTALYAKART_REGION` (default: `026`)
- `ANTALYAKART_LANG` (default: `tr`)
- `ANTALYAKART_AUTH_TYPE` (default: `4`)

Example (if you want explicit env in client config):

```json
{
  "mcpServers": {
    "antalyakart": {
      "type": "stdio",
      "command": "/absolute/path/to/antalyakart-mcp",
      "args": [],
      "env": {
        "ANTALYAKART_BASE_URL": "https://service.kentkart.com/rl1",
        "ANTALYAKART_REGION": "026",
        "ANTALYAKART_LANG": "tr",
        "ANTALYAKART_AUTH_TYPE": "4"
      }
    }
  }
}
```

---

## 6) First prompts to verify it works

Use these in your MCP-capable client:

1. "Search Antalya routes and stops for `otogar`."
2. "Show nearby stops and kiosks around lat 36.897087 lng 30.713777."
3. "Summarize arrivals at stop 10828."
4. "Plan a direct trip from otogar to markantalya."

---

## API investigation artifacts

- Reverse engineering notes: `docs/investigation-notes.md`
- OpenAPI description: `docs/antalyakart-openapi.yaml`
