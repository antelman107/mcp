# AntalyaKart MCP Server (Go)

Unofficial MCP server that exposes AntalyaKart public transit endpoints as tools for agents.

## What this server provides

This MCP server wraps transit APIs used by:

- `https://online.antalyakart.com.tr/#/home`

and exposes them as MCP tools:

- `search_routes_and_stops`
- `nearby_places_stops_and_kiosks`
- `nearest_buses_for_stop`
- `route_path_and_vehicles`

## API investigation artifacts

- Reverse engineering notes: `docs/investigation-notes.md`
- OpenAPI description: `docs/antalyakart-openapi.yaml`

## Build

```bash
go build -o bin/antalyakart-mcp ./cmd/antalyakart-mcp
```

## Run (stdio MCP server)

```bash
./bin/antalyakart-mcp
```

## Environment variables (optional)

Defaults are already set for Antalya.

- `ANTALYAKART_BASE_URL` (default: `https://service.kentkart.com/rl1`)
- `ANTALYAKART_REGION` (default: `026`)
- `ANTALYAKART_LANG` (default: `tr`)
- `ANTALYAKART_AUTH_TYPE` (default: `4`)

## Cursor MCP configuration example

```json
{
  "mcpServers": {
    "antalyakart": {
      "command": "/absolute/path/to/antalyakart-mcp/bin/antalyakart-mcp",
      "args": []
    }
  }
}
```

## Tool quick examples

- Search:
  - `search_routes_and_stops` with `{ "keyword": "otogar" }`
- Nearby places/stops:
  - `nearby_places_stops_and_kiosks` with `{ "latitude": 36.897087, "longitude": 30.713777 }`
- Nearest buses for stop:
  - `nearest_buses_for_stop` with `{ "latitude": 36.897087, "longitude": 30.713777, "bus_stop_id": "10828" }`
- Route path + live fleet:
  - `route_path_and_vehicles` with `{ "display_route_code": "106", "direction": 0, "data_mode": "full" }`
  - `route_path_and_vehicles` with `{ "display_route_code": "106", "direction": 0, "data_mode": "live-only" }`
