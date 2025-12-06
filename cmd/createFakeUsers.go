package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-faker/faker/v4"
	"github.com/spf13/cobra"
)

var fakeUserCount int
var fakeIdleTime string

type FakeUser struct {
	Username string `faker:"username"`
}

var createFakeUsersCmd = &cobra.Command{
	Use:   "create-fake-users",
	Short: "Creates n fake users with systemd and caddy config",
	Run: func(cmd *cobra.Command, args []string) {
		for i := 0; i < fakeUserCount; i++ {
			var f FakeUser
			if err := faker.FakeData(&f); err != nil {
				log.Fatalf("❌ Failed to generate fake data: %v", err)
			}

			// Clean and prefix username
			// We ensure the base name is clean, then prepend "test-"
			baseName := cleanUsername(f.Username)
			if len(baseName) == 0 {
				baseName = "user" // Fallback if faker gives empty string after cleaning (unlikely)
			}
			username := "test-" + baseName

			fmt.Printf("\n--- Processing User %d/%d: %s ---\n", i+1, fakeUserCount, username)

			// 1. Create User
			if err := CreateUser(username); err != nil {
				log.Printf("⚠️  Skipping %s: %v\n", username, err)
				continue
			}

			// 2. Setup Systemd
			if err := SetupSystemd(username, fakeIdleTime); err != nil {
				log.Printf("⚠️  Failed systemd for %s: %v\n", username, err)
				continue
			}

			// 3. Setup Caddy
			if err := SetupProxy(username, ""); err != nil {
				log.Printf("⚠️  Failed caddy for %s: %v\n", username, err)
				continue
			}
		}
		fmt.Println("\n✅ All done.")
	},
}

func cleanUsername(s string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func init() {
	rootCmd.AddCommand(createFakeUsersCmd)
	createFakeUsersCmd.Flags().IntVarP(&fakeUserCount, "count", "n", 1, "Number of fake users to create")
	createFakeUsersCmd.Flags().StringVarP(&fakeIdleTime, "idle", "i", "5min", "Idle time for systemd service (e.g. 10s, 1h)")
}
