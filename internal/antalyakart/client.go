package antalyakart

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBaseURL  = "https://service.kentkart.com/rl1"
	DefaultRegion   = "026"
	DefaultLang     = "tr"
	DefaultAuthType = "4"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	region     string
	lang       string
	authType   string
}

func NewClient(baseURL, region, lang, authType string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultBaseURL
	}
	if strings.TrimSpace(region) == "" {
		region = DefaultRegion
	}
	if strings.TrimSpace(lang) == "" {
		lang = DefaultLang
	}
	if strings.TrimSpace(authType) == "" {
		authType = DefaultAuthType
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		baseURL:  strings.TrimRight(baseURL, "/"),
		region:   region,
		lang:     lang,
		authType: authType,
	}
}

func (c *Client) SearchRoutesAndStops(keyword string) ([]byte, error) {
	params := map[string]string{}
	if strings.TrimSpace(keyword) != "" {
		params["keyword"] = keyword
	}
	return c.get("/web/nearest/find", params)
}

func (c *Client) NearbyPlacesStopsAndKiosks(lat, lng float64) ([]byte, error) {
	params := map[string]string{
		"lat": fmt.Sprintf("%.7f", lat),
		"lng": fmt.Sprintf("%.7f", lng),
	}
	return c.get("/web/nearest/place", params)
}

func (c *Client) NearestBuses(lat, lng float64, busStopID string) ([]byte, error) {
	params := map[string]string{
		"accuracy":  "0",
		"lat":       fmt.Sprintf("%.7f", lat),
		"lng":       fmt.Sprintf("%.7f", lng),
		"busStopId": busStopID,
	}
	return c.get("/web/nearest/bus", params)
}

func (c *Client) RoutePathInfo(displayRouteCode, direction, resultType string) ([]byte, error) {
	params := map[string]string{
		"displayRouteCode": displayRouteCode,
		"direction":        direction,
		"resultType":       resultType,
	}
	return c.get("/web/pathInfo", params)
}

func (c *Client) get(endpoint string, extraParams map[string]string) ([]byte, error) {
	reqURL, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse request url: %w", err)
	}

	query := reqURL.Query()
	query.Set("region", c.region)
	query.Set("lang", c.lang)
	query.Set("authType", c.authType)
	for key, value := range extraParams {
		if strings.TrimSpace(value) == "" {
			continue
		}
		query.Set(key, value)
	}
	reqURL.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Ensure responses returned by tools are always valid JSON.
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode response json: %w", err)
	}

	pretty, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("format response json: %w", err)
	}

	return pretty, nil
}
