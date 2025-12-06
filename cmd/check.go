/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"strings"
	"text/tabwriter"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check [services...]",
	Short: "Check the status and user of systemd services",
	Long: `Checks if the specified systemd services are running and, 
if active, identifies which user they are running under.

If no services are provided, it defaults to checking:
  - caddy.service
  - postgresql.service`,
	Run: func(cmd *cobra.Command, args []string) {
		// Define defaults
		targetServices := []string{"caddy.service", "postgresql.service"}

		// If user provided args, use those instead
		if len(args) > 0 {
			targetServices = args
		}

		checkServices(targetServices)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

func checkServices(services []string) {
	// 1. Establish connection to systemd D-Bus
	ctx := context.Background()
	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		// If running in a container or non-systemd env, this might fail
		fmt.Fprintf(os.Stderr, "Failed to connect to systemd: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	// 2. Setup pretty printing
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tSTATUS\tRUNNING USER")
	fmt.Fprintln(w, "-------\t------\t------------")

	for _, unitName := range services {
		// Append .service if missing for better UX
		if !strings.Contains(unitName, ".") {
			unitName += ".service"
		}

		// 3. Get Active State
		status := "unknown"
		activeProp, err := conn.GetUnitPropertyContext(ctx, unitName, "ActiveState")

		if err == nil && activeProp != nil {
			// Strip quotes from the dbus string variant
			status = strings.Trim(activeProp.Value.String(), "\"")
		} else {
			// Service not loaded in systemd
			fmt.Fprintf(w, "%s\t%s\t%s\n", unitName, "not-found", "-")
			continue
		}

		// 4. Get Running User (only if active)
		username := "-"
		if status == "active" {
			// ExecMainUID is the numeric UID of the main process
			uidProp, err := conn.GetServicePropertyContext(ctx, unitName, "ExecMainUID")

			if err == nil && uidProp != nil {
				if uid, ok := uidProp.Value.Value().(uint32); ok {
					// Lookup username from UID
					u, err := user.LookupId(fmt.Sprint(uid))
					if err == nil {
						username = u.Username
					} else {
						username = fmt.Sprintf("uid:%d", uid)
					}
				}
			}
		}

		// 5. Output row
		fmt.Fprintf(w, "%s\t%s\t%s\n", unitName, status, username)
	}

	w.Flush()
}
