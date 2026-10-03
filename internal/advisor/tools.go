package advisor

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/functiontool"

	"github.com/antelman107/mcp/internal/antalyakart"
)

const maxToolChars = 8000

type toolText struct {
	Text string `json:"text"`
}

type searchArgs struct {
	Keyword string `json:"keyword"` // optional text for route names, route codes, and stop names
}

type nearbyArgs struct {
	Latitude  float64 `json:"latitude"`  // latitude in decimal degrees
	Longitude float64 `json:"longitude"` // longitude in decimal degrees
}

type nearestArgs struct {
	Latitude  float64 `json:"latitude"`    // latitude in decimal degrees
	Longitude float64 `json:"longitude"`   // longitude in decimal degrees
	BusStopID string  `json:"bus_stop_id"` // stop identifier, for example 10828
}

type routeArgs struct {
	DisplayRouteCode string `json:"display_route_code"` // public route code, for example 106 or LC07A
	Direction        int    `json:"direction"`          // 0 or 1
	DataMode         string `json:"data_mode"`          // full, or live-only for active vehicles
}

type tripArgs struct {
	OriginQuery      string `json:"origin_query"`      // search text for the origin stop
	DestinationQuery string `json:"destination_query"` // search text for the destination stop
	Language         string `json:"language"`          // tr or en
	OutputMode       string `json:"output_mode"`       // compact or detailed
	MaxRoutes        int    `json:"max_routes"`        // maximum direct route options
}

type arrivalsArgs struct {
	BusStopID  string  `json:"bus_stop_id"` // stop identifier, for example 10828
	Latitude   float64 `json:"latitude"`    // latitude in decimal degrees
	Longitude  float64 `json:"longitude"`   // longitude in decimal degrees
	Language   string  `json:"language"`    // tr or en
	OutputMode string  `json:"output_mode"` // compact or detailed
}

func transitTools(client *antalyakart.Client) ([]tool.Tool, error) {
	search, err := functiontool.New(functiontool.Config{
		Name:        "search_routes_and_stops",
		Description: "Search Antalya routes, stops, and places by an optional keyword.",
	}, func(_ agent.Context, args searchArgs) (toolText, error) {
		payload, err := client.SearchRoutesAndStops(args.Keyword)
		if err != nil {
			return toolText{}, err
		}
		return toolText{Text: clip(string(payload))}, nil
	})
	if err != nil {
		return nil, err
	}

	nearby, err := functiontool.New(functiontool.Config{
		Name:        "nearby_places_stops_and_kiosks",
		Description: "Find places, bus stops, and card top-up kiosks near a coordinate.",
	}, func(_ agent.Context, args nearbyArgs) (toolText, error) {
		payload, err := client.NearbyPlacesStopsAndKiosks(args.Latitude, args.Longitude)
		if err != nil {
			return toolText{}, err
		}
		return toolText{Text: clip(string(payload))}, nil
	})
	if err != nil {
		return nil, err
	}

	nearest, err := functiontool.New(functiontool.Config{
		Name:        "nearest_buses_for_stop",
		Description: "List buses approaching a stop near a coordinate.",
	}, func(_ agent.Context, args nearestArgs) (toolText, error) {
		if strings.TrimSpace(args.BusStopID) == "" {
			return toolText{}, fmt.Errorf("bus_stop_id is required")
		}
		payload, err := client.NearestBuses(args.Latitude, args.Longitude, args.BusStopID)
		if err != nil {
			return toolText{}, err
		}
		return toolText{Text: clip(string(payload))}, nil
	})
	if err != nil {
		return nil, err
	}

	route, err := functiontool.New(functiontool.Config{
		Name:        "route_path_and_vehicles",
		Description: "Get a route path with stops and schedule, or live vehicles only.",
	}, func(_ agent.Context, args routeArgs) (toolText, error) {
		if strings.TrimSpace(args.DisplayRouteCode) == "" {
			return toolText{}, fmt.Errorf("display_route_code is required")
		}
		if args.Direction != 0 && args.Direction != 1 {
			return toolText{}, fmt.Errorf("direction must be 0 or 1")
		}
		mode := strings.ToLower(strings.TrimSpace(args.DataMode))
		if mode == "" {
			mode = "full"
		}
		resultType := "111111"
		if mode == "live-only" {
			resultType = "010000"
		} else if mode != "full" {
			return toolText{}, fmt.Errorf("data_mode must be full or live-only")
		}
		payload, err := client.RoutePathInfo(args.DisplayRouteCode, fmt.Sprintf("%d", args.Direction), resultType)
		if err != nil {
			return toolText{}, err
		}
		return toolText{Text: clip(string(payload))}, nil
	})
	if err != nil {
		return nil, err
	}

	trip, err := functiontool.New(functiontool.Config{
		Name:        "plan_direct_trip_between_stops",
		Description: "Plan direct buses between two stop searches and include ETAs.",
	}, func(_ agent.Context, args tripArgs) (toolText, error) {
		if strings.TrimSpace(args.OriginQuery) == "" || strings.TrimSpace(args.DestinationQuery) == "" {
			return toolText{}, fmt.Errorf("origin_query and destination_query are required")
		}
		result, err := client.BuildTripPlan(args.OriginQuery, args.DestinationQuery, args.Language, args.OutputMode, args.MaxRoutes)
		if err != nil {
			return toolText{}, err
		}
		return asText(result)
	})
	if err != nil {
		return nil, err
	}

	arrivals, err := functiontool.New(functiontool.Config{
		Name:        "summarize_stop_arrivals",
		Description: "Summarize upcoming arrivals at a stop, grouped by route and direction.",
	}, func(_ agent.Context, args arrivalsArgs) (toolText, error) {
		if strings.TrimSpace(args.BusStopID) == "" {
			return toolText{}, fmt.Errorf("bus_stop_id is required")
		}
		result, err := client.SummarizeStopArrivals(args.BusStopID, args.Latitude, args.Longitude, args.Language, args.OutputMode)
		if err != nil {
			return toolText{}, err
		}
		return asText(result)
	})
	if err != nil {
		return nil, err
	}

	return []tool.Tool{search, nearby, nearest, route, trip, arrivals}, nil
}

func asText(payload any) (toolText, error) {
	out, err := json.Marshal(payload)
	if err != nil {
		return toolText{}, err
	}
	return toolText{Text: clip(string(out))}, nil
}

func clip(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "no data"
	}
	if len(text) <= maxToolChars {
		return text
	}
	return text[:maxToolChars] + "\n...truncated"
}
