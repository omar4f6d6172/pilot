package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var (
	ctName   string
	ctDomain string
	ctIdle   string
)

var createTenantCmdFull = &cobra.Command{
	Use:   "create-tenant",
	Short: "Full provisioning of a tenant (User, DB, Systemd, Proxy)",
	Long: `Orchestrates the entire provisioning process for a new tenant:
1. Creates a Linux System User (with lingering enabled).
2. Creates a PostgreSQL Role and Database (Peer Auth).
3. Generates and starts Systemd Socket & Service units.
4. Configures Caddy Reverse Proxy to route traffic.`,
	Run: func(cmd *cobra.Command, args []string) {
		if ctName == "" {
			log.Fatal("Tenant name is required (--name)")
		}

		log.Printf("🚀 Starting provisioning for tenant '%s'...\n", ctName)

		// 1. Create Linux User
		if err := CreateUser(ctName); err != nil {
			log.Fatalf("❌ User creation failed: %v", err)
		}

		// 2. Setup Database
		if err := SetupDatabase(ctName); err != nil {
			log.Fatalf("❌ Database setup failed: %v", err)
		}

		// 3. Setup Systemd
		if err := SetupSystemd(ctName, ctIdle); err != nil {
			log.Fatalf("❌ Systemd setup failed: %v", err)
		}

		// 4. Setup Proxy
		if err := SetupProxy(ctName, ctDomain, ""); err != nil {
			log.Fatalf("❌ Proxy setup failed: %v", err)
		}

		log.Printf("🎉 Success! Tenant '%s' is fully provisioned and ready.\n", ctName)
	},
}

func init() {
	rootCmd.AddCommand(createTenantCmdFull)

	createTenantCmdFull.Flags().StringVarP(&ctName, "name", "n", "", "Tenant Name (linux username) [Required]")
	createTenantCmdFull.Flags().StringVarP(&ctDomain, "domain", "d", "", "Custom Domain (e.g. app.example.com)")
	createTenantCmdFull.Flags().StringVarP(&ctIdle, "idle", "i", "5min", "Idle timeout for socket activation")

	_ = createTenantCmdFull.MarkFlagRequired("name")
}
