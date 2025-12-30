package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// CaddyClient handles interactions with the Caddy Admin API
type CaddyClient struct {
	BaseURL string
	Client  *http.Client
}

// NewCaddyClient creates a new client (defaulting to localhost:2019)
func NewCaddyClient(baseURL string) *CaddyClient {
	if baseURL == "" {
		baseURL = "http://localhost:2019"
	}
	return &CaddyClient{
		BaseURL: baseURL,
		Client:  &http.Client{},
	}
}

// RouteExists checks if a route ID exists
func (c *CaddyClient) RouteExists(id string) (bool, error) {
	resp, err := c.Client.Get(fmt.Sprintf("%s/id/%s", c.BaseURL, id))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return true, nil
	}
	if resp.StatusCode == 404 {
		return false, nil
	}
	return false, fmt.Errorf("unexpected status %d", resp.StatusCode)
}

// AddRoute adds a new route to srv0
func (c *CaddyClient) AddRoute(id, domain, upstream string) error {
	route := buildRoute(id, domain, upstream)
	payload, err := json.Marshal(route)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/config/apps/http/servers/srv0/routes", c.BaseURL)
	return c.postRequest(url, payload)
}

// UpdateRoute updates an existing route by ID
func (c *CaddyClient) UpdateRoute(id, domain, upstream string) error {
	route := buildRoute(id, domain, upstream)
	payload, err := json.Marshal(route)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/id/%s", c.BaseURL, id)
	return c.putRequest(url, payload)
}

// InitServer ensures the basics (http app, srv0) exist
func (c *CaddyClient) InitServer() error {
	config := map[string]interface{}{
		"apps": map[string]interface{}{
			"http": map[string]interface{}{
				"servers": map[string]interface{}{
					"srv0": map[string]interface{}{
						"listen": []string{":80"},
						"routes": []interface{}{},
					},
				},
			},
		},
	}
	payload, _ := json.Marshal(config)
	return c.postRequest(fmt.Sprintf("%s/config/", c.BaseURL), payload)
}

// Helper methods

func (c *CaddyClient) postRequest(url string, payload []byte) error {
	resp, err := c.Client.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResponse(resp)
}

func (c *CaddyClient) putRequest(url string, payload []byte) error {
	req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResponse(resp)
}

func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api error: %s", string(body))
	}
	return nil
}

// Data structures

type CaddyRoute struct {
	ID     string                   `json:"@id,omitempty"`
	Match  []CaddyMatch             `json:"match,omitempty"`
	Handle []map[string]interface{} `json:"handle,omitempty"`
}

type CaddyMatch struct {
	Host []string `json:"host,omitempty"`
}

func buildRoute(id, domain, upstream string) CaddyRoute {
	// Determine dial address (unix vs tcp)
	dial := upstream // Assume TCP (host:port) by default
	if strings.HasPrefix(upstream, "/") || strings.HasPrefix(upstream, ".") {
		dial = "unix/" + upstream
	}

	return CaddyRoute{
		ID: id,
		Match: []CaddyMatch{{
			Host: []string{domain},
		}},
		Handle: []map[string]interface{}{{
			"handler": "reverse_proxy",
			"upstreams": []map[string]string{{
				"dial": dial,
			}},
		}},
	}
}
