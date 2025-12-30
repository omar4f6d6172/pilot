package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

// Flags
var proxyTenantName string
var proxyDomain string
var proxyUpstream string // Optional: allow manual upstream override

var setupProxyCmd = &cobra.Command{
	Use:   "setup-proxy",
	Short: "Configures Caddy to route a domain to the tenant's socket",
	Run: func(cmd *cobra.Command, args []string) {
		if err := SetupProxy(proxyTenantName, proxyDomain, proxyUpstream); err != nil {
			log.Fatal(err)
		}
	},
}

// SetupProxy configures Caddy for a user via the REST API
func SetupProxy(username, domain, upstream string) error {
	// 1. Determine Domain
	if domain == "" {
		domain = fmt.Sprintf("%s.localhost", username)
	}

	// 2. Determine Upstream
	if upstream == "" {
		// Default to Unix socket
		upstream = fmt.Sprintf("/run/pilot/%s.sock", username)
	}

	routeID := fmt.Sprintf("tenant-%s", username)
	fmt.Printf("🌐 Configuring Caddy via API: %s -> %s\n", domain, upstream)

	client := NewCaddyClient("") // Use default localhost:2019

	// 3. Check if route exists
	exists, err := client.RouteExists(routeID)
	if err != nil {
		return fmt.Errorf("failed to contact Caddy API: %v (is Caddy running?)", err)
	}

	if exists {
		// Update
		fmt.Printf("🔄 Updating existing route %s...\n", routeID)
		if err := client.UpdateRoute(routeID, domain, upstream); err != nil {
			return fmt.Errorf("failed to update route: %v", err)
		}
	} else {
		// Create
		fmt.Printf("➕ Adding new route %s...\n", routeID)
		if err := client.AddRoute(routeID, domain, upstream); err != nil {
			// Retry with Init check
			fmt.Println("⚠️  Route addition failed, attempting to initialize Caddy server 'srv0'...")
			if initErr := client.InitServer(); initErr != nil {
				return fmt.Errorf("failed to init server: %v (original error: %v)", initErr, err)
			}

			fmt.Println("🔄 Retrying route addition...")
			if retryErr := client.AddRoute(routeID, domain, upstream); retryErr != nil {
				return fmt.Errorf("caddy API error (create): %v", retryErr)
			}
		}
	}

	fmt.Printf("✅ Success! You can now access http://%s\n", domain)
	return nil
}

func init() {
	rootCmd.AddCommand(setupProxyCmd)
	setupProxyCmd.Flags().StringVarP(&proxyTenantName, "name", "n", "", "Tenant Name (Required)")
	setupProxyCmd.Flags().StringVarP(&proxyDomain, "domain", "d", "", "Custom Domain (e.g. app.example.com)")
	setupProxyCmd.Flags().StringVarP(&proxyUpstream, "upstream", "u", "", "Custom Upstream (e.g. localhost:8080 or /run/foo.sock)")
	_ = setupProxyCmd.MarkFlagRequired("name")
}
