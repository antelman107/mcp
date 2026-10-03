package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/antelman107/mcp/internal/antalyakart"
)

type SearchRoutesAndStopsArgs struct {
	Keyword string `json:"keyword,omitempty" jsonschema:"Optional text to search in route names, route codes, and stop names"`
}

type NearbyPlacesArgs struct {
	Latitude  float64 `json:"latitude" jsonschema:"Latitude in decimal degrees"`
	Longitude float64 `json:"longitude" jsonschema:"Longitude in decimal degrees"`
}

type NearestBusesArgs struct {
	Latitude  float64 `json:"latitude" jsonschema:"Latitude in decimal degrees"`
	Longitude float64 `json:"longitude" jsonschema:"Longitude in decimal degrees"`
	BusStopID string  `json:"bus_stop_id" jsonschema:"Stop identifier for the target stop, for example 10828"`
}

type RoutePathAndVehiclesArgs struct {
	DisplayRouteCode string `json:"display_route_code" jsonschema:"Public route code, for example 106 or LC07A"`
	Direction        int    `json:"direction" jsonschema:"Direction index where 0 and 1 are usually outbound and inbound"`
	DataMode         string `json:"data_mode,omitempty" jsonschema:"Choose full for geometry, stops, and schedule, or live-only for the active vehicle list"`
}

type PlanDirectTripArgs struct {
	OriginQuery      string `json:"origin_query" jsonschema:"Search text for the origin stop, for example otogar"`
	DestinationQuery string `json:"destination_query" jsonschema:"Search text for the destination stop, for example markantalya"`
	Language         string `json:"language,omitempty" jsonschema:"Output language tr or en. Default en"`
	OutputMode       string `json:"output_mode,omitempty" jsonschema:"compact or detailed. Default compact"`
	MaxRoutes        int    `json:"max_routes,omitempty" jsonschema:"Maximum number of direct route options to return. Default 3"`
}

type SummarizeStopArrivalsArgs struct {
	BusStopID  string  `json:"bus_stop_id" jsonschema:"Stop identifier, for example 10828"`
	Latitude   float64 `json:"latitude" jsonschema:"Latitude in decimal degrees"`
	Longitude  float64 `json:"longitude" jsonschema:"Longitude in decimal degrees"`
	Language   string  `json:"language,omitempty" jsonschema:"Output language tr or en. Default en"`
	OutputMode string  `json:"output_mode,omitempty" jsonschema:"compact or detailed. Default compact"`
}

func main() {
	addr := envOrDefault("MCP_ADDR", ":8090")
	endpoint := envOrDefault("MCP_PATH", "/antalyakart")
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	client := antalyakart.NewClient(
		envOrDefault("ANTALYAKART_BASE_URL", antalyakart.DefaultBaseURL),
		envOrDefault("ANTALYAKART_REGION", antalyakart.DefaultRegion),
		envOrDefault("ANTALYAKART_LANG", antalyakart.DefaultLang),
		envOrDefault("ANTALYAKART_AUTH_TYPE", antalyakart.DefaultAuthType),
	)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "antalyakart",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_routes_and_stops",
		Description: "Search Antalya route, stop, and place data using optional text keyword.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args SearchRoutesAndStopsArgs) (*mcp.CallToolResult, any, error) {
		payload, err := client.SearchRoutesAndStops(args.Keyword)
		if err != nil {
			return nil, nil, err
		}
		return textResult(payload), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "nearby_places_stops_and_kiosks",
		Description: "Find nearby points of interest, bus stops, and kiosk/card top-up points for a coordinate.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args NearbyPlacesArgs) (*mcp.CallToolResult, any, error) {
		payload, err := client.NearbyPlacesStopsAndKiosks(args.Latitude, args.Longitude)
		if err != nil {
			return nil, nil, err
		}
		return textResult(payload), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "nearest_buses_for_stop",
		Description: "Return nearest active buses and route options for a selected stop near a coordinate.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args NearestBusesArgs) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(args.BusStopID) == "" {
			return nil, nil, fmt.Errorf("bus_stop_id is required")
		}
		payload, err := client.NearestBuses(args.Latitude, args.Longitude, args.BusStopID)
		if err != nil {
			return nil, nil, err
		}
		return textResult(payload), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "route_path_and_vehicles",
		Description: "Get detailed route path with stops and schedule or a lightweight live vehicle view.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args RoutePathAndVehiclesArgs) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(args.DisplayRouteCode) == "" {
			return nil, nil, fmt.Errorf("display_route_code is required")
		}
		if args.Direction != 0 && args.Direction != 1 {
			return nil, nil, fmt.Errorf("direction must be 0 or 1")
		}

		mode := strings.ToLower(strings.TrimSpace(args.DataMode))
		if mode == "" {
			mode = "full"
		}

		resultType := "111111"
		if mode == "live-only" {
			resultType = "010000"
		} else if mode != "full" {
			return nil, nil, fmt.Errorf("data_mode must be full or live-only")
		}

		payload, err := client.RoutePathInfo(args.DisplayRouteCode, strconv.Itoa(args.Direction), resultType)
		if err != nil {
			return nil, nil, err
		}
		return textResult(payload), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "plan_direct_trip_between_stops",
		Description: "Plan direct bus routes between two stop queries with ETA-focused summaries.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args PlanDirectTripArgs) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(args.OriginQuery) == "" {
			return nil, nil, fmt.Errorf("origin_query is required")
		}
		if strings.TrimSpace(args.DestinationQuery) == "" {
			return nil, nil, fmt.Errorf("destination_query is required")
		}
		result, err := client.BuildTripPlan(
			args.OriginQuery,
			args.DestinationQuery,
			args.Language,
			args.OutputMode,
			args.MaxRoutes,
		)
		if err != nil {
			return nil, nil, err
		}
		return objectResult(result)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "summarize_stop_arrivals",
		Description: "Summarize upcoming bus arrivals at a stop grouped by route and direction.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args SummarizeStopArrivalsArgs) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(args.BusStopID) == "" {
			return nil, nil, fmt.Errorf("bus_stop_id is required")
		}
		result, err := client.SummarizeStopArrivals(
			args.BusStopID,
			args.Latitude,
			args.Longitude,
			args.Language,
			args.OutputMode,
		)
		if err != nil {
			return nil, nil, err
		}
		return objectResult(result)
	})

	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		SessionTimeout: 30 * time.Minute,
	})

	mux := http.NewServeMux()
	mux.Handle(endpoint, mcpHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte("ok\n"))
		}
	})

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	log.Printf("antalyakart MCP listening on %s (endpoint %s)", addr, endpoint)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func textResult(payload []byte) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(payload)},
		},
	}
}

func objectResult(payload any) (*mcp.CallToolResult, any, error) {
	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return textResult(out), nil, nil
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
