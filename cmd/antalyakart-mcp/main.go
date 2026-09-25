package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"

	"github.com/antelman107/mcp/internal/antalyakart"
)

type SearchRoutesAndStopsArgs struct {
	Keyword string `json:"keyword" jsonschema:"description=Optional text to search in route names route codes and stop names"`
}

type NearbyPlacesArgs struct {
	Latitude  float64 `json:"latitude" jsonschema:"required,description=Latitude in decimal degrees"`
	Longitude float64 `json:"longitude" jsonschema:"required,description=Longitude in decimal degrees"`
}

type NearestBusesArgs struct {
	Latitude  float64 `json:"latitude" jsonschema:"required,description=Latitude in decimal degrees"`
	Longitude float64 `json:"longitude" jsonschema:"required,description=Longitude in decimal degrees"`
	BusStopID string  `json:"bus_stop_id" jsonschema:"required,description=Stop identifier for the target stop for example 10828"`
}

type RoutePathAndVehiclesArgs struct {
	DisplayRouteCode string `json:"display_route_code" jsonschema:"required,description=Public route code for example 106 or LC07A"`
	Direction        int    `json:"direction" jsonschema:"required,description=Direction index where 0 and 1 are usually outbound and inbound"`
	DataMode         string `json:"data_mode" jsonschema:"description=Choose full for geometry stops and schedule or live-only for active vehicle list only"`
}

type PlanDirectTripArgs struct {
	OriginQuery      string `json:"origin_query" jsonschema:"required,description=Search text for origin stop for example otogar"`
	DestinationQuery string `json:"destination_query" jsonschema:"required,description=Search text for destination stop for example markantalya"`
	Language         string `json:"language" jsonschema:"description=Output language tr or en. Default en"`
	OutputMode       string `json:"output_mode" jsonschema:"description=compact or detailed. Default compact"`
	MaxRoutes        int    `json:"max_routes" jsonschema:"description=Maximum number of direct route options to return. Default 3"`
}

type SummarizeStopArrivalsArgs struct {
	BusStopID  string  `json:"bus_stop_id" jsonschema:"required,description=Stop identifier for example 10828"`
	Latitude   float64 `json:"latitude" jsonschema:"required,description=Latitude in decimal degrees"`
	Longitude  float64 `json:"longitude" jsonschema:"required,description=Longitude in decimal degrees"`
	Language   string  `json:"language" jsonschema:"description=Output language tr or en. Default en"`
	OutputMode string  `json:"output_mode" jsonschema:"description=compact or detailed. Default compact"`
}

func main() {
	done := make(chan struct{})

	client := antalyakart.NewClient(
		envOrDefault("ANTALYAKART_BASE_URL", antalyakart.DefaultBaseURL),
		envOrDefault("ANTALYAKART_REGION", antalyakart.DefaultRegion),
		envOrDefault("ANTALYAKART_LANG", antalyakart.DefaultLang),
		envOrDefault("ANTALYAKART_AUTH_TYPE", antalyakart.DefaultAuthType),
	)

	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())

	err := server.RegisterTool(
		"search_routes_and_stops",
		"Search Antalya route, stop, and place data using optional text keyword.",
		func(arguments SearchRoutesAndStopsArgs) (*mcp_golang.ToolResponse, error) {
			payload, err := client.SearchRoutesAndStops(arguments.Keyword)
			if err != nil {
				return nil, err
			}
			return jsonToolResponse(payload), nil
		},
	)
	if err != nil {
		panic(err)
	}

	err = server.RegisterTool(
		"nearby_places_stops_and_kiosks",
		"Find nearby points of interest, bus stops, and kiosk/card top-up points for a coordinate.",
		func(arguments NearbyPlacesArgs) (*mcp_golang.ToolResponse, error) {
			payload, err := client.NearbyPlacesStopsAndKiosks(arguments.Latitude, arguments.Longitude)
			if err != nil {
				return nil, err
			}
			return jsonToolResponse(payload), nil
		},
	)
	if err != nil {
		panic(err)
	}

	err = server.RegisterTool(
		"nearest_buses_for_stop",
		"Return nearest active buses and route options for a selected stop near a coordinate.",
		func(arguments NearestBusesArgs) (*mcp_golang.ToolResponse, error) {
			if strings.TrimSpace(arguments.BusStopID) == "" {
				return nil, fmt.Errorf("bus_stop_id is required")
			}
			payload, err := client.NearestBuses(arguments.Latitude, arguments.Longitude, arguments.BusStopID)
			if err != nil {
				return nil, err
			}
			return jsonToolResponse(payload), nil
		},
	)
	if err != nil {
		panic(err)
	}

	err = server.RegisterTool(
		"route_path_and_vehicles",
		"Get detailed route path with stops and schedule or a lightweight live vehicle view.",
		func(arguments RoutePathAndVehiclesArgs) (*mcp_golang.ToolResponse, error) {
			if strings.TrimSpace(arguments.DisplayRouteCode) == "" {
				return nil, fmt.Errorf("display_route_code is required")
			}
			if arguments.Direction != 0 && arguments.Direction != 1 {
				return nil, fmt.Errorf("direction must be 0 or 1")
			}

			mode := strings.ToLower(strings.TrimSpace(arguments.DataMode))
			if mode == "" {
				mode = "full"
			}

			resultType := "111111"
			if mode == "live-only" {
				resultType = "010000"
			} else if mode != "full" {
				return nil, fmt.Errorf("data_mode must be full or live-only")
			}

			payload, err := client.RoutePathInfo(arguments.DisplayRouteCode, strconv.Itoa(arguments.Direction), resultType)
			if err != nil {
				return nil, err
			}
			return jsonToolResponse(payload), nil
		},
	)
	if err != nil {
		panic(err)
	}

	err = server.RegisterTool(
		"plan_direct_trip_between_stops",
		"Plan direct bus routes between two stop queries with ETA-focused summaries.",
		func(arguments PlanDirectTripArgs) (*mcp_golang.ToolResponse, error) {
			if strings.TrimSpace(arguments.OriginQuery) == "" {
				return nil, fmt.Errorf("origin_query is required")
			}
			if strings.TrimSpace(arguments.DestinationQuery) == "" {
				return nil, fmt.Errorf("destination_query is required")
			}
			result, err := client.BuildTripPlan(
				arguments.OriginQuery,
				arguments.DestinationQuery,
				arguments.Language,
				arguments.OutputMode,
				arguments.MaxRoutes,
			)
			if err != nil {
				return nil, err
			}
			return objectToolResponse(result)
		},
	)
	if err != nil {
		panic(err)
	}

	err = server.RegisterTool(
		"summarize_stop_arrivals",
		"Summarize upcoming bus arrivals at a stop grouped by route and direction.",
		func(arguments SummarizeStopArrivalsArgs) (*mcp_golang.ToolResponse, error) {
			if strings.TrimSpace(arguments.BusStopID) == "" {
				return nil, fmt.Errorf("bus_stop_id is required")
			}
			result, err := client.SummarizeStopArrivals(
				arguments.BusStopID,
				arguments.Latitude,
				arguments.Longitude,
				arguments.Language,
				arguments.OutputMode,
			)
			if err != nil {
				return nil, err
			}
			return objectToolResponse(result)
		},
	)
	if err != nil {
		panic(err)
	}

	if err := server.Serve(); err != nil {
		panic(err)
	}

	<-done
}

func jsonToolResponse(payload []byte) *mcp_golang.ToolResponse {
	return mcp_golang.NewToolResponse(
		mcp_golang.NewTextContent(string(payload)),
	)
}

func objectToolResponse(payload any) (*mcp_golang.ToolResponse, error) {
	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	return jsonToolResponse(out), nil
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
