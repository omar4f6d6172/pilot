package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/spf13/cobra"
)

// Flags
var proxyTenantName string
var proxyDomain string

// JSON structures for Caddy
type CaddyRoute struct {
	ID     string                   `json:"@id,omitempty"`
	Match  []CaddyMatch             `json:"match,omitempty"`
	Handle []map[string]interface{} `json:"handle,omitempty"`
}

type CaddyMatch struct {
	Host []string `json:"host,omitempty"`
}

var setupProxyCmd = &cobra.Command{
	Use:   "setup-proxy",
	Short: "Configures Caddy to route a domain to the tenant's socket",
	Run: func(cmd *cobra.Command, args []string) {
		if err := SetupProxy(proxyTenantName, proxyDomain); err != nil {
			log.Fatal(err)
		}
	},
}

// SetupProxy configures Caddy for a user via the REST API
func SetupProxy(username, domain string) error {
	// 1. Determine Domain
	if domain == "" {
		domain = fmt.Sprintf("%s.localhost", username)
	}
	routeID := fmt.Sprintf("tenant-%s", username)

	fmt.Printf("🌐 Configuring Caddy via API: %s -> /run/pilot/%s.sock\n", domain, username)

	// Construct Route Object
	route := CaddyRoute{
		ID: routeID,
		Match: []CaddyMatch{{ 
			Host: []string{domain},
		}},
		Handle: []map[string]interface{}{{
			"handler": "reverse_proxy",
			"upstreams": []map[string]string{{
				"dial": fmt.Sprintf("unix/run/pilot/%s.sock", username),
			}},
		}},
	}

	client := &http.Client{}
	payload, err := json.Marshal(route)
	if err != nil {
		return fmt.Errorf("failed to marshal route: %v", err)
	}

	// 2. Check if route exists
	exists, err := caddyIDExists(routeID)
	if err != nil {
		return fmt.Errorf("failed to contact Caddy API: %v (is Caddy running?)", err)
	}

	if exists {
		// Update via PUT /id/<id>
		fmt.Printf("🔄 Updating existing route %s...\n", routeID)
		req, _ := http.NewRequest("PUT", "http://localhost:2019/id/"+routeID, bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("caddy API error (update): %s", string(body))
		}
	} else {
		// Create via POST /config/apps/http/servers/srv0/routes
		fmt.Printf("➕ Adding new route %s...\n", routeID)
		
		// Try adding the route directly
		if err := addCaddyRoute(client, payload); err != nil {
			// If it failed, maybe srv0 is missing. Try initializing it.
			fmt.Println("⚠️  Route addition failed, attempting to initialize Caddy server 'srv0'...")
			if initErr := initCaddyServer(); initErr != nil {
				return fmt.Errorf("failed to init server: %v (original error: %v)", initErr, err)
			}
			
			// Retry adding the route
			fmt.Println("🔄 Retrying route addition...")
			if retryErr := addCaddyRoute(client, payload); retryErr != nil {
				return fmt.Errorf("caddy API error (create): %v", retryErr)
			}
		}
	}

	fmt.Printf("✅ Success! You can now access http://%s\n", domain)
	return nil
}

func addCaddyRoute(client *http.Client, payload []byte) error {
	req, _ := http.NewRequest("POST", "http://localhost:2019/config/apps/http/servers/srv0/routes", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s", string(body))
	}
	return nil
}

func caddyIDExists(id string) (bool, error) {
	resp, err := http.Get("http://localhost:2019/id/" + id)
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

func initCaddyServer() error {

	// Payload to create the http app and srv0 server

	// We post to /config to ensure we create the "apps" key if it's missing

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



		// POST to /config/ merges into root (trailing slash important to avoid 301)



		resp, err := http.Post("http://localhost:2019/config/", "application/json", bytes.NewBuffer(payload))

	if err != nil {

		return err

	}

	defer resp.Body.Close()



	if resp.StatusCode >= 300 {

		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("failed to init http app: %s", string(body))

	}

	return nil

}

func init() {
	rootCmd.AddCommand(setupProxyCmd)
	setupProxyCmd.Flags().StringVarP(&proxyTenantName, "name", "n", "", "Tenant Name (Required)")
	setupProxyCmd.Flags().StringVarP(&proxyDomain, "domain", "d", "", "Custom Domain (e.g. app.example.com)")
	_ = setupProxyCmd.MarkFlagRequired("name")
}