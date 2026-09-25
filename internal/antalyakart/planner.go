package antalyakart

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type TripPlanResult struct {
	Summary                   string            `json:"summary"`
	Language                  string            `json:"language"`
	Origin                    StopRef           `json:"origin"`
	Destination               StopRef           `json:"destination"`
	DirectRouteOptions        []TripRouteOption `json:"direct_route_options"`
	Notes                     []string          `json:"notes,omitempty"`
	OriginStopCandidates      []StopRef         `json:"origin_stop_candidates,omitempty"`
	DestinationStopCandidates []StopRef         `json:"destination_stop_candidates,omitempty"`
}

type StopArrivalSummary struct {
	Summary          string                 `json:"summary"`
	Language         string                 `json:"language"`
	Stop             StopRef                `json:"stop"`
	RouteArrivalInfo []RouteArrivalEstimate `json:"route_arrival_info"`
	Notes            []string               `json:"notes,omitempty"`
}

type StopRef struct {
	StopID   string  `json:"stop_id"`
	StopName string  `json:"stop_name"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

type TripRouteOption struct {
	RouteCode                string       `json:"route_code"`
	DisplayRouteCode         string       `json:"display_route_code"`
	RouteName                string       `json:"route_name"`
	HeadSign                 string       `json:"head_sign"`
	Direction                string       `json:"direction"`
	StopsBetween             int          `json:"stops_between"`
	EstimatedWaitMinutes     *int         `json:"estimated_wait_minutes,omitempty"`
	UpcomingVehiclesAtOrigin []VehicleETA `json:"upcoming_vehicles_at_origin,omitempty"`
}

type VehicleETA struct {
	BusID            string `json:"bus_id"`
	Plate            string `json:"plate,omitempty"`
	DisplayRouteCode string `json:"display_route_code"`
	Direction        string `json:"direction"`
	TimeDiffMinutes  *int   `json:"time_diff_minutes,omitempty"`
	StopDiff         *int   `json:"stop_diff,omitempty"`
}

type RouteArrivalEstimate struct {
	RouteCode        string       `json:"route_code"`
	DisplayRouteCode string       `json:"display_route_code"`
	RouteName        string       `json:"route_name"`
	Direction        string       `json:"direction"`
	VehicleETAs      []VehicleETA `json:"vehicle_etas"`
}

func (c *Client) BuildTripPlan(originQuery, destinationQuery, language, outputMode string, maxRoutes int) (*TripPlanResult, error) {
	lang := normalizeLanguage(language)
	mode := normalizeOutputMode(outputMode)
	if maxRoutes <= 0 {
		maxRoutes = 3
	}

	originResp, err := c.SearchRoutesAndStopsTyped(originQuery)
	if err != nil {
		return nil, fmt.Errorf("origin stop search failed: %w", err)
	}
	destinationResp, err := c.SearchRoutesAndStopsTyped(destinationQuery)
	if err != nil {
		return nil, fmt.Errorf("destination stop search failed: %w", err)
	}

	originStop, originCandidates, err := pickBestStop(originQuery, originResp.StopList)
	if err != nil {
		return nil, err
	}
	destinationStop, destinationCandidates, err := pickBestStop(destinationQuery, destinationResp.StopList)
	if err != nil {
		return nil, err
	}

	originNearest, err := c.NearestBusesTyped(originStop.Lat, originStop.Lng, originStop.StopID)
	if err != nil {
		return nil, fmt.Errorf("origin nearest buses lookup failed: %w", err)
	}
	destinationNearest, err := c.NearestBusesTyped(destinationStop.Lat, destinationStop.Lng, destinationStop.StopID)
	if err != nil {
		return nil, fmt.Errorf("destination nearest buses lookup failed: %w", err)
	}

	originRoutes := routesByCode(originNearest.RouteList)
	destinationRoutes := routesByCode(destinationNearest.RouteList)

	var options []TripRouteOption
	seen := map[string]struct{}{}
	pathCache := map[string]*PathInfoResponse{}

	for routeCode, originRouteItems := range originRoutes {
		destinationRouteItems, ok := destinationRoutes[routeCode]
		if !ok {
			continue
		}

		for _, originRoute := range originRouteItems {
			displayCode := nonEmpty(originRoute.DisplayRouteCode, originRoute.RouteCode)
			if strings.TrimSpace(displayCode) == "" {
				continue
			}

			directions := map[string]struct{}{"0": {}, "1": {}}
			directions[strings.TrimSpace(originRoute.Direction)] = struct{}{}
			for _, dr := range destinationRouteItems {
				directions[strings.TrimSpace(dr.Direction)] = struct{}{}
			}

			for dir := range directions {
				if dir != "0" && dir != "1" {
					continue
				}
				cacheKey := displayCode + ":" + dir
				path, ok := pathCache[cacheKey]
				if !ok {
					path, err = c.RoutePathInfoTyped(displayCode, dir, "111111")
					if err != nil {
						continue
					}
					pathCache[cacheKey] = path
				}

				pathOptions := collectRouteOptionsFromPath(path, routeCode, displayCode, originRoute.Name, dir, originStop, destinationStop, originNearest.BusList)
				for _, option := range pathOptions {
					uniqueKey := option.DisplayRouteCode + ":" + option.Direction + ":" + option.HeadSign
					if _, exists := seen[uniqueKey]; exists {
						continue
					}
					seen[uniqueKey] = struct{}{}
					options = append(options, option)
				}
			}
		}
	}

	sort.Slice(options, func(i, j int) bool {
		mi := largeWhenNil(options[i].EstimatedWaitMinutes)
		mj := largeWhenNil(options[j].EstimatedWaitMinutes)
		if mi == mj {
			return options[i].StopsBetween < options[j].StopsBetween
		}
		return mi < mj
	})
	if len(options) > maxRoutes {
		options = options[:maxRoutes]
	}

	result := &TripPlanResult{
		Summary:            tripSummary(lang, originStop.StopName, destinationStop.StopName, len(options)),
		Language:           lang,
		Origin:             originStop,
		Destination:        destinationStop,
		DirectRouteOptions: options,
	}
	if len(options) == 0 {
		result.Notes = append(result.Notes, noDirectRouteNote(lang))
	}
	if mode == "detailed" {
		result.OriginStopCandidates = originCandidates
		result.DestinationStopCandidates = destinationCandidates
	}
	return result, nil
}

func (c *Client) SummarizeStopArrivals(stopID string, lat, lng float64, language, outputMode string) (*StopArrivalSummary, error) {
	lang := normalizeLanguage(language)
	mode := normalizeOutputMode(outputMode)

	nearest, err := c.NearestBusesTyped(lat, lng, stopID)
	if err != nil {
		return nil, err
	}

	routeMap := map[string]RouteArrivalEstimate{}
	for _, route := range nearest.RouteList {
		key := route.RouteCode + ":" + route.Direction
		routeMap[key] = RouteArrivalEstimate{
			RouteCode:        route.RouteCode,
			DisplayRouteCode: route.DisplayRouteCode,
			RouteName:        route.Name,
			Direction:        route.Direction,
		}
	}

	for _, bus := range nearest.BusList {
		matched := false
		for key, routeInfo := range routeMap {
			if routeInfo.RouteCode == bus.RouteCode && routeInfo.Direction == bus.Direction {
				entry := routeMap[key]
				entry.VehicleETAs = append(entry.VehicleETAs, newVehicleETA(bus))
				routeMap[key] = entry
				matched = true
				break
			}
		}
		if !matched {
			key := bus.RouteCode + ":" + bus.Direction
			entry := routeMap[key]
			entry.RouteCode = bus.RouteCode
			entry.DisplayRouteCode = bus.DisplayRouteCode
			entry.Direction = bus.Direction
			entry.VehicleETAs = append(entry.VehicleETAs, newVehicleETA(bus))
			routeMap[key] = entry
		}
	}

	var arrivals []RouteArrivalEstimate
	for _, item := range routeMap {
		sort.Slice(item.VehicleETAs, func(i, j int) bool {
			return largeWhenNil(item.VehicleETAs[i].TimeDiffMinutes) < largeWhenNil(item.VehicleETAs[j].TimeDiffMinutes)
		})
		if mode == "compact" && len(item.VehicleETAs) > 2 {
			item.VehicleETAs = item.VehicleETAs[:2]
		}
		arrivals = append(arrivals, item)
	}
	sort.Slice(arrivals, func(i, j int) bool {
		if len(arrivals[i].VehicleETAs) == 0 {
			return false
		}
		if len(arrivals[j].VehicleETAs) == 0 {
			return true
		}
		return largeWhenNil(arrivals[i].VehicleETAs[0].TimeDiffMinutes) < largeWhenNil(arrivals[j].VehicleETAs[0].TimeDiffMinutes)
	})

	stopName := nearest.StopInfo.BusStopName
	if strings.TrimSpace(stopName) == "" {
		stopName = stopID
	}

	summary := &StopArrivalSummary{
		Summary:          stopArrivalSummaryText(lang, stopName, len(arrivals)),
		Language:         lang,
		Stop:             StopRef{StopID: stopID, StopName: stopName, Lat: lat, Lng: lng},
		RouteArrivalInfo: arrivals,
	}
	if len(arrivals) == 0 {
		summary.Notes = append(summary.Notes, noArrivalNote(lang))
	}
	return summary, nil
}

func collectRouteOptionsFromPath(path *PathInfoResponse, routeCode, displayCode, routeName, direction string, originStop, destinationStop StopRef, buses []BusSummary) []TripRouteOption {
	var out []TripRouteOption
	for _, pathEntry := range path.PathList {
		originSeq := -1
		destinationSeq := -1
		for _, stop := range pathEntry.BusStopList {
			seq := parseInt(stop.Seq)
			if stop.StopID == originStop.StopID {
				originSeq = seq
			}
			if stop.StopID == destinationStop.StopID {
				destinationSeq = seq
			}
		}
		if originSeq < 0 || destinationSeq < 0 || originSeq >= destinationSeq {
			continue
		}

		vehicles := collectMatchingVehicles(buses, routeCode, displayCode, direction)
		var wait *int
		if len(vehicles) > 0 {
			wait = vehicles[0].TimeDiffMinutes
		}
		out = append(out, TripRouteOption{
			RouteCode:                routeCode,
			DisplayRouteCode:         nonEmpty(pathEntry.DisplayRouteCode, displayCode),
			RouteName:                routeName,
			HeadSign:                 pathEntry.HeadSign,
			Direction:                nonEmpty(pathEntry.Direction, direction),
			StopsBetween:             destinationSeq - originSeq,
			EstimatedWaitMinutes:     wait,
			UpcomingVehiclesAtOrigin: vehicles,
		})
	}
	return out
}

func collectMatchingVehicles(buses []BusSummary, routeCode, displayCode, direction string) []VehicleETA {
	var out []VehicleETA
	for _, bus := range buses {
		sameRoute := (bus.RouteCode != "" && bus.RouteCode == routeCode) || (bus.DisplayRouteCode != "" && bus.DisplayRouteCode == displayCode)
		if !sameRoute {
			continue
		}
		if direction != "" && bus.Direction != "" && bus.Direction != direction {
			continue
		}
		out = append(out, newVehicleETA(bus))
	}
	sort.Slice(out, func(i, j int) bool {
		return largeWhenNil(out[i].TimeDiffMinutes) < largeWhenNil(out[j].TimeDiffMinutes)
	})
	return out
}

func newVehicleETA(bus BusSummary) VehicleETA {
	return VehicleETA{
		BusID:            bus.BusID,
		Plate:            nonEmpty(bus.Plate, bus.PlateNumber),
		DisplayRouteCode: bus.DisplayRouteCode,
		Direction:        bus.Direction,
		TimeDiffMinutes:  parseIntPointer(bus.TimeDiff),
		StopDiff:         parseIntPointer(bus.StopDiff),
	}
}

func routesByCode(routes []RouteSummary) map[string][]RouteSummary {
	out := map[string][]RouteSummary{}
	for _, route := range routes {
		if strings.TrimSpace(route.RouteCode) == "" {
			continue
		}
		out[route.RouteCode] = append(out[route.RouteCode], route)
	}
	return out
}

func pickBestStop(query string, stops []StopSummary) (StopRef, []StopRef, error) {
	type rankedStop struct {
		stop  StopRef
		score int
	}
	var ranked []rankedStop
	for _, stop := range stops {
		stopID := strings.TrimSpace(stop.StopID)
		name := strings.TrimSpace(nonEmpty(stop.StopName, stop.Name))
		if stopID == "" || name == "" {
			continue
		}
		lat, errLat := strconv.ParseFloat(strings.TrimSpace(stop.Lat), 64)
		lng, errLng := strconv.ParseFloat(strings.TrimSpace(stop.Lng), 64)
		if errLat != nil || errLng != nil {
			continue
		}
		ref := StopRef{
			StopID:   stopID,
			StopName: name,
			Lat:      lat,
			Lng:      lng,
		}
		ranked = append(ranked, rankedStop{stop: ref, score: stopMatchScore(query, name)})
	}
	if len(ranked) == 0 {
		return StopRef{}, nil, fmt.Errorf("no matching stops found for query %q", query)
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].stop.StopName < ranked[j].stop.StopName
		}
		return ranked[i].score < ranked[j].score
	})

	var candidates []StopRef
	limit := 5
	if len(ranked) < limit {
		limit = len(ranked)
	}
	for i := 0; i < limit; i++ {
		candidates = append(candidates, ranked[i].stop)
	}

	return ranked[0].stop, candidates, nil
}

func stopMatchScore(query, stopName string) int {
	q := strings.ToLower(strings.TrimSpace(query))
	s := strings.ToLower(strings.TrimSpace(stopName))
	switch {
	case s == q:
		return 0
	case strings.HasPrefix(s, q):
		return 1
	case strings.Contains(s, q):
		return 2
	default:
		diff := len(s) - len(q)
		if diff < 0 {
			diff *= -1
		}
		return 10 + diff
	}
}

func normalizeLanguage(language string) string {
	lang := strings.ToLower(strings.TrimSpace(language))
	if lang == "tr" {
		return "tr"
	}
	return "en"
}

func normalizeOutputMode(outputMode string) string {
	mode := strings.ToLower(strings.TrimSpace(outputMode))
	if mode == "detailed" {
		return "detailed"
	}
	return "compact"
}

func parseInt(value string) int {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return -1
	}
	return n
}

func parseIntPointer(value string) *int {
	n := parseInt(value)
	if n < 0 {
		return nil
	}
	return &n
}

func largeWhenNil(v *int) int {
	if v == nil {
		return 99999
	}
	return *v
}

func nonEmpty(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}

func tripSummary(lang, origin, destination string, optionCount int) string {
	if lang == "tr" {
		if optionCount == 0 {
			return fmt.Sprintf("%s ile %s arasında doğrudan hat bulunamadı.", origin, destination)
		}
		return fmt.Sprintf("%s ile %s arasında %d doğrudan güzergah bulundu.", origin, destination, optionCount)
	}
	if optionCount == 0 {
		return fmt.Sprintf("No direct route was found between %s and %s.", origin, destination)
	}
	return fmt.Sprintf("Found %d direct route options between %s and %s.", optionCount, origin, destination)
}

func stopArrivalSummaryText(lang, stopName string, routeCount int) string {
	if lang == "tr" {
		return fmt.Sprintf("%s durağı için %d hatta canlı varış bilgisi listelendi.", stopName, routeCount)
	}
	return fmt.Sprintf("Live arrivals for %d routes at stop %s.", routeCount, stopName)
}

func noDirectRouteNote(lang string) string {
	if lang == "tr" {
		return "Doğrudan hat bulunamadı. Yakın aktarma alternatifleri için farklı başlangıç/bitiş duraklarını deneyin."
	}
	return "No direct route was found. Try nearby stop alternatives for a transfer-based trip."
}

func noArrivalNote(lang string) string {
	if lang == "tr" {
		return "Bu durak için anlık araç verisi alınamadı."
	}
	return "No live vehicle data was returned for this stop."
}
