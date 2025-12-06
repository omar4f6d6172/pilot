/*
Copyright © 2025 OMAR ALTANBAKJI & NOAH EID
*/
package cmd

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/spf13/cobra"
)

// Variable to store the flag value
var tenantName string

// createTenantCmd represents the create-tenant command
var createTenantCmd = &cobra.Command{
	Use:   "create-user",
	Short: "Creates a new system user with lingering enabled",
	Long: `Creates a new Linux user for a specific tenant.
    
This command performs two main system operations:
1. useradd -m -s /bin/bash <name>: Creates the user and home directory.
2. loginctl enable-linger <name>: Allows the user's systemd instance to run at boot without login.

Example:
  pilot create-user --name="omar"`,
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Create the user
		fmt.Printf("👤 Creating user '%s'...\n", tenantName)

		// -m: Create home directory
		// -s: Set shell to bash (helpful for debugging, even if not logged in often)
		userCmd := exec.Command("useradd", "-m", "-s", "/bin/bash", tenantName)

		if out, err := userCmd.CombinedOutput(); err != nil {
			log.Fatalf("❌ Failed to create user: %v\nOutput: %s", err, string(out))
		}

		// 2. Enable Lingering
		// This is critical for Socket Activation to work for non-logged-in users
		fmt.Printf("⚙️  Enabling systemd lingering for '%s'...\n", tenantName)
		lingerCmd := exec.Command("loginctl", "enable-linger", tenantName)

		if out, err := lingerCmd.CombinedOutput(); err != nil {
			log.Fatalf("❌ Failed to enable linger: %v\nOutput: %s", err, string(out))
		}

		fmt.Printf("✅ Success! Tenant '%s' is ready.\n", tenantName)
	},
}

func init() {
	rootCmd.AddCommand(createTenantCmd)

	// Define the --name flag
	createTenantCmd.Flags().StringVarP(&tenantName, "name", "n", "", "Name of the tenant (linux username)")

	// Mark the flag as required
	_ = createTenantCmd.MarkFlagRequired("name")
}
