/*
Copyright © 2025 OMAR ALTANBAKJI & NOAH EID
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"time"

	"github.com/spf13/cobra"
)

// Variable to store the flag value
var tenantName string

// CreateUser creates a new system user, enables lingering, and ensures the user service is running
func CreateUser(username string) error {
	// 1. Create the user
	fmt.Printf("👤 Creating user '%s'...\n", username)

	// -m: Create home directory
	// -s: Set shell to bash
	userCmd := exec.Command("useradd", "-m", "-s", "/bin/bash", username)

	if out, err := userCmd.CombinedOutput(); err != nil {
		// Ignore error if user already exists (exit code 9)
		// But CombinedOutput doesn't give exit code easily without type assertion.
		// For robustness in "fix this now" mode, let's fail if it fails, assuming clean slate or manual cleanup.
		return fmt.Errorf("failed to create user: %v, Output: %s", err, string(out))
	}

	// 2. Lookup the user to get UID (needed for systemctl and wait loop)
	u, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("failed to lookup user %s after creation: %v", username, err)
	}

	// 3. Enable Lingering
	fmt.Printf("⚙️  Enabling systemd lingering for '%s'...\n", username)
	lingerCmd := exec.Command("loginctl", "enable-linger", username)

	if out, err := lingerCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable linger: %v, Output: %s", err, string(out))
	}

	// 4. Explicitly start the user service
	// This forces systemd to create /run/user/<UID> and the bus socket immediately
	serviceName := fmt.Sprintf("user@%s.service", u.Uid)
	fmt.Printf("🚀 Starting systemd service '%s'...\n", serviceName)
	startCmd := exec.Command("systemctl", "start", serviceName)
	if out, err := startCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start user service: %v, Output: %s", err, string(out))
	}

	// 5. Wait for Systemd User Manager (DBus socket)
	busPath := fmt.Sprintf("/run/user/%s/bus", u.Uid)
	fmt.Printf("⏳ Waiting for user bus at %s...\n", busPath)

	timeout := 10 * time.Second
	start := time.Now()
	for time.Since(start) < timeout {
		if _, err := os.Stat(busPath); err == nil {
			fmt.Printf("✅ User bus is ready.\n")
			fmt.Printf("✅ Success! Tenant '%s' is ready.\n", username)
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for user bus at %s (is systemd-logind running?)", busPath)
}

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
		if err := CreateUser(tenantName); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(createTenantCmd)

	// Define the --name flag
	createTenantCmd.Flags().StringVarP(&tenantName, "name", "n", "", "Name of the tenant (linux username)")

	// Mark the flag as required
	_ = createTenantCmd.MarkFlagRequired("name")
}
