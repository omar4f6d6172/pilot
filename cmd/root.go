package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

var logFile *os.File

func init() {
	// Initialize Logging
	logPath := "/var/log/pilot.log"

	// Create/Open log file
	var err error
	logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Fallback to local file if /var/log permission denied (dev mode)
		logPath = "pilot.log"
		logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}

	if err == nil {
		// Multiwriter: Log to both file and stdout
		// BUT: Cobra prints to stdout/stderr too, so maybe just set output of standard log
		log.SetOutput(logFile)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pilot",
	Short: "Control-Plane CLI for systemd socket activation orchestration",
	Long: `Pilot is a specialized Control-Plane CLI tool designed to orchestrate 
systemd Socket Activation on Linux systems. 

It automates the complex configuration of Linux users, systemd units, 
PostgreSQL database roles, and Caddy reverse proxy routes to enable 
resource-efficient, lazy-loaded web service provisioning.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pilot.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
