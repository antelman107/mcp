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
