package cmd

import (
	"fmt"
	"log"
	"os/exec"
	"os/user"

	"github.com/spf13/cobra"
)

var checkLogsName string

var checkLogsCmd = &cobra.Command{
	Use:   "check-logs",
	Short: "View logs for a specific tenant",
	Run: func(cmd *cobra.Command, args []string) {
		if checkLogsName == "" {
			log.Fatal("Tenant name is required (--name)")
		}
		if err := CheckLogs(checkLogsName); err != nil {
			log.Fatalf("❌ Error: %v", err)
		}
	},
}

func CheckLogs(username string) error {
	// 1. Get UID to find the user
	u, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("could not find user %s: %v", username, err)
	}

	fmt.Printf("📜 Fetching logs for tenant '%s' (UID %s)...\n", username, u.Uid)

	// 2. Fetch logs using journalctl
	// We use _UID match to see everything running as that user (proxy, service, etc.)
	// or --user-unit if we want to be specific. _UID is broader and often better for debugging everything.
	cmd := exec.Command("journalctl", fmt.Sprintf("_UID=%s", u.Uid), "--no-pager", "-n", "50")

	// Connect stdout/stderr to current terminal
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run journalctl: %v", err)
	}

	fmt.Println(string(out))
	return nil
}

func init() {
	rootCmd.AddCommand(checkLogsCmd)
	checkLogsCmd.Flags().StringVarP(&checkLogsName, "name", "n", "", "Tenant Name (Required)")
	_ = checkLogsCmd.MarkFlagRequired("name")
}
