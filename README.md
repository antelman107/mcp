# AntalyaKart MCP

Transit tools for Antalya buses: routes, stops, nearby kiosks, live arrivals, and direct trip plans. The data comes from the same API as [online.antalyakart.com.tr](https://online.antalyakart.com.tr/#/home).

The server is already running. Point a client at:

`https://mcp.goturkey.club/antalyakart`

## Connect

### Cursor

Project file `.cursor/mcp.json`, or global `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "antalyakart": {
      "url": "https://mcp.goturkey.club/antalyakart"
    }
  }
}
```

### Claude Code

```bash
claude mcp add --scope user --transport http antalyakart https://mcp.goturkey.club/antalyakart
```

### Codex

User file `~/.codex/config.toml`, or project `.codex/config.toml`:

```toml
[mcp_servers.antalyakart]
url = "https://mcp.goturkey.club/antalyakart"
```

## Try it

1. Search routes and stops for `otogar`.
2. Show nearby stops and kiosks around `36.897087, 30.713777`.
3. Summarize arrivals at stop `10828`.
4. Plan a direct trip from `otogar` to `markantalya`.

## Tools

- `search_routes_and_stops` — routes, stops, and places by keyword
- `nearby_places_stops_and_kiosks` — places, stops, and card top-up points near a coordinate
- `nearest_buses_for_stop` — buses approaching a stop
- `route_path_and_vehicles` — route geometry, stops, schedule, or live vehicles only
- `plan_direct_trip_between_stops` — direct buses between two stop searches
- `summarize_stop_arrivals` — upcoming arrivals at a stop, grouped by route

## Run it yourself

```bash
go install github.com/antelman107/mcp/cmd/antalyakart-mcp@latest
antalyakart-mcp
```

That listens on `http://127.0.0.1:8090/antalyakart`. `GET /healthz` returns `ok`. Another server on the same host gets its own path, for example `/another-mcp`.

From a checkout:

```bash
go build -o bin/antalyakart-mcp ./cmd/antalyakart-mcp
```

Optional environment variables:

| Variable | Default |
|---|---|
| `MCP_ADDR` | `:8090` |
| `MCP_PATH` | `/antalyakart` |
| `ANTALYAKART_BASE_URL` | `https://service.kentkart.com/rl1` |
| `ANTALYAKART_REGION` | `026` |
| `ANTALYAKART_LANG` | `tr` |
| `ANTALYAKART_AUTH_TYPE` | `4` |

API notes live in `docs/investigation-notes.md` and `docs/antalyakart-openapi.yaml`.

## Telegram advisor

`antalyakart-advisor` is the Telegram bot `@antalyakart_advisor_bot`. It answers bus questions with a Google ADK for Go agent (`LlmAgent` and `Runner`, at most six model rounds) and the same AntalyaKart tools as the MCP server. After deploy, Telegram posts updates to `https://mcp.goturkey.club/antalyakart-advisor/webhook`.

Chat history is one JSON file, a map of chat id to messages. The default name is `chat-history.json` next to the executable, so the process can be started from another directory. `CHAT_HISTORY_PATH` overrides it; a relative value is still resolved from the executable directory.

```bash
go build -o bin/antalyakart-advisor ./cmd/antalyakart-advisor
```

The binary listens on `127.0.0.1:8091`. `GET /healthz` returns `ok`.

| Variable | Required | Default |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | yes | |
| `GOOGLE_API_KEY` | yes | Gemini / ADK key |
| `GEMINI_MODEL` | no | `gemini-flash-latest` |
| `WEBHOOK_URL` | no | empty skips webhook registration |
| `TELEGRAM_WEBHOOK_SECRET` | no | empty accepts any caller |
| `BOT_ADDR` | no | `127.0.0.1:8091` |
| `BOT_PATH` | no | `/antalyakart-advisor` |
| `CHAT_HISTORY_PATH` | no | `chat-history.json` beside the executable |

Put the token and API key in a gitignored env file or in the server env. `.env.example` lists the names. Deploy writes `/var/www/mcp/antalyakart-advisor.env` from GitHub Actions secrets `TELEGRAM_BOT_TOKEN` and `GOOGLE_API_KEY`, using the same SSH host and workflow shape as `antalyakart-mcp` (`.github/workflows/deploy-advisor.yml`).
