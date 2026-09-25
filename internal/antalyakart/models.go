package antalyakart

type SearchResponse struct {
	Result    ResultInfo     `json:"result"`
	RouteList []RouteSummary `json:"routeList"`
	StopList  []StopSummary  `json:"stopList"`
	PlaceList []PlaceSummary `json:"placeList"`
}

type NearbyResponse struct {
	Result    ResultInfo     `json:"result"`
	PlaceList []PlaceSummary `json:"placeList"`
	KioskList []KioskSummary `json:"kioskList"`
	StopList  []StopSummary  `json:"stopList"`
}

type NearestBusResponse struct {
	Result    ResultInfo     `json:"result"`
	StopInfo  StopInfo       `json:"stopInfo"`
	BusList   []BusSummary   `json:"busList"`
	RouteList []RouteSummary `json:"routeList"`
}

type PathInfoResponse struct {
	Result   ResultInfo      `json:"result"`
	PathList []PathInfoEntry `json:"pathList"`
}

type ResultInfo struct {
	Cmd     string `json:"cmd"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Date    string `json:"dateTime"`
}

type RouteSummary struct {
	RouteCode        string `json:"routeCode"`
	DisplayRouteCode string `json:"displayRouteCode"`
	Name             string `json:"name"`
	HeadSign         string `json:"headSign"`
	Direction        string `json:"direction"`
	RouteColor       string `json:"routeColor"`
	RouteType        string `json:"routeType"`
}

type StopSummary struct {
	StopID   string `json:"stopId"`
	StopName string `json:"stopName"`
	Name     string `json:"name"`
	Lat      string `json:"lat"`
	Lng      string `json:"lng"`
	Routes   string `json:"routes"`
}

type PlaceSummary struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Lat      string  `json:"lat"`
	Lng      string  `json:"lng"`
	Distance float64 `json:"distance"`
}

type KioskSummary struct {
	Title    string `json:"title"`
	Address  string `json:"address"`
	Lat      string `json:"lat"`
	Lng      string `json:"lng"`
	Distance string `json:"distance"`
}

type StopInfo struct {
	BusStopID   string `json:"busStopid"`
	BusStopName string `json:"busStopName"`
	Lat         string `json:"lat"`
	Lng         string `json:"lng"`
}

type BusSummary struct {
	BusID            string `json:"busId"`
	Plate            string `json:"plate"`
	PlateNumber      string `json:"plateNumber"`
	RouteCode        string `json:"routeCode"`
	DisplayRouteCode string `json:"displayRouteCode"`
	Direction        string `json:"direction"`
	TimeDiff         string `json:"timeDiff"`
	StopDiff         string `json:"stopDiff"`
	Lat              string `json:"lat"`
	Lng              string `json:"lng"`
}

type PathInfoEntry struct {
	HeadSign         string               `json:"headSign"`
	DisplayRouteCode string               `json:"displayRouteCode"`
	Direction        string               `json:"direction"`
	PathCode         string               `json:"path_code"`
	PointList        []PathPoint          `json:"pointList"`
	BusList          []BusSummary         `json:"busList"`
	BusStopList      []PathStopSummary    `json:"busStopList"`
	ScheduleList     []PathScheduleWindow `json:"scheduleList"`
}

type PathPoint struct {
	Seq string `json:"seq"`
	Lat string `json:"lat"`
	Lng string `json:"lng"`
}

type PathStopSummary struct {
	Seq      string `json:"seq"`
	StopID   string `json:"stopId"`
	StopName string `json:"stopName"`
}

type PathScheduleWindow struct {
	Description string             `json:"description"`
	TimeList    []PathScheduleTime `json:"timeList"`
}

type PathScheduleTime struct {
	DepartureTime string `json:"departureTime"`
}
