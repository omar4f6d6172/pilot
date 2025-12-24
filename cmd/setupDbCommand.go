package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var setupDbCmd = &cobra.Command{
	Use:   "setup-database",
	Short: "Configures PostgreSQL user and database",
	Run: func(cmd *cobra.Command, args []string) {
		if err := SetupDatabase(setupTenantName); err != nil {
			log.Fatalf("❌ Error: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(setupDbCmd)
	setupDbCmd.Flags().StringVarP(&setupTenantName, "name", "n", "", "Tenant Name")
	_ = setupDbCmd.MarkFlagRequired("name")
}
