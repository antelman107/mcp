# AntalyaKart API Investigation Notes

This document describes how the public transit API behind `https://online.antalyakart.com.tr/#/home` was discovered and validated.

## 1) Frontend artifact inspection

- Downloaded the HTML shell from:
  - `https://online.antalyakart.com.tr/`
- Extracted bundle names:
  - `main-es2015.31659180b508c3361b7a.js`
  - `runtime-es2015.a7844c1fa5e04a3c2ee6.js`
  - `scripts.dc30bdbae7c6dccef0c1.js`

## 2) Config discovery

By inspecting minified bundle strings, the frontend config file location was found:

- `https://online.antalyakart.com.tr/assets/config/appConfig.json`

Important fields in that file:

- `apiServer.url`: `https://service.kentkart.com/rl1/`
- `apiServer.authUrl`: `https://auth.kentkart.com/rl1/`
- default query parameters:
  - `region = "026"`
  - `lang = "tr"`
  - `authType = "4"`

## 3) Transit endpoint discovery

Routes used by the public transit features in the SPA:

- `GET /web/nearest/find`
- `GET /web/nearest/place`
- `GET /web/nearest/bus`
- `GET /web/pathInfo`

## 4) Request and behavior notes

### `GET /web/nearest/find`
- Params: `region`, `lang`, `authType`, optional `keyword`
- Returns: `routeList`, `stopList`, `placeList`
- Use case: search routes/stops by text query.

### `GET /web/nearest/place`
- Params: `region`, `lang`, `authType`, `lat`, `lng`
- Returns: nearby `placeList`, `kioskList`, `stopList`
- Use case: "what is around this coordinate?"

### `GET /web/nearest/bus`
- Params: `region`, `lang`, `authType`, `lat`, `lng`, `busStopId`, optional `accuracy` (frontend sends `"0"`)
- Returns: `stopInfo`, live `busList`, and route alternatives in `routeList`
- Use case: show buses currently approaching/near one stop.

### `GET /web/pathInfo`
- Params: `region`, `lang`, `authType`, `displayRouteCode`, `direction`, `resultType`
- Observed `resultType` values:
  - `111111`: full payload (geometry `pointList`, `busStopList`, `scheduleList`, live `busList`)
  - `010000`: lightweight payload, primarily live `busList`
- Use case: route line visualization and schedule.

## 5) Validation

Each endpoint above was called directly against `https://service.kentkart.com/rl1` and returned HTTP 200 with JSON payloads matching frontend expectations.

## 6) Notes about headers/signatures

The SPA contains an interceptor for custom headers (`X-KK-Credential`, `X-KK-Date`, `X-KK-Signature`) generated through a wasm helper.  
For the transit endpoints listed above, direct calls without these custom headers still succeeded in live testing.
