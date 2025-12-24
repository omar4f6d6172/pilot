package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/spf13/cobra"
)

// Data Structure
type SystemdConfig struct {
	Username string
	UID      string
	Port     string
	IdleTime string
}

// 1. SOCKET: Listens on the file, triggers the proxy
const socketTmpl = `[Unit]
Description=Public Socket for {{.Username}}

[Socket]
ListenStream=/run/pilot/{{.Username}}.sock
SocketMode=0666
Service=rest-api-proxy.service

[Install]
WantedBy=sockets.target
`

// 2. PROXY: The "Brain" of the operation
// --exit-idle-time: Kills the proxy if no traffic flows for X time
const proxyTmpl = `[Unit]
Description=Socket Proxy for {{.Username}}
Requires=rest-api.service
After=rest-api.service

[Service]
# Point to the internal localhost port (UID)
# Exit if idle for {{.IdleTime}}
ExecStart=/usr/lib/systemd/systemd-socket-proxyd --exit-idle-time={{.IdleTime}} 127.0.0.1:{{.Port}}
NonBlocking=true
`

// 3. BACKEND: The "Dumb" Worker
// StopWhenUnneeded=true: Dies automatically when the proxy dies
const serviceTmpl = `[Unit]
Description=User REST API Backend
StopWhenUnneeded=true
PartOf=rest-api-proxy.service

[Service]
ExecStart=/usr/local/bin/user-rest-api
Environment=PORT={{.Port}}
Type=simple
ExecStartPost=/bin/sleep 1
`

var setupTenantName string
var setupIdleTime string

var setupSystemdCmd = &cobra.Command{
	Use:   "setup-systemd",
	Short: "Sets up autoscaling systemd units",
	Run: func(cmd *cobra.Command, args []string) {
		if err := SetupSystemd(setupTenantName, setupIdleTime); err != nil {
			log.Fatalf("❌ Error: %v", err)
		}
	},
}

// SetupSystemd configures the systemd units for a user
func SetupSystemd(username, idleTime string) error {
	u, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("could not find user %s: %v", username, err)
	}

	// Calculate a high port based on UID (e.g., UID + 10000)
	// This avoids "permission denied" on ports < 1024
	uidInt, err := strconv.Atoi(u.Uid)
	if err != nil {
		return fmt.Errorf("invalid UID %s: %v", u.Uid, err)
	}
	port := uidInt + 10000

	config := SystemdConfig{
		Username: username,
		UID:      u.Uid,
		Port:     fmt.Sprintf("%d", port),
		IdleTime: idleTime,
	}

	// Ensure shared socket directory exists and is writable
	socketDir := "/run/pilot"
	if err := os.MkdirAll(socketDir, 0777); err != nil {
		return fmt.Errorf("failed to create socket directory %s: %v", socketDir, err)
	}
	// Force permissions (MkdirAll respects umask)
	if err := os.Chmod(socketDir, 0777); err != nil {
		return fmt.Errorf("failed to chmod socket directory %s: %v", socketDir, err)
	}

	fmt.Printf("🔧 Configuring Autoscaling (Idle: %s) for %s...\n", config.IdleTime, config.Username)

	// Render Templates
	socketContent, err := renderTemplate("socket", socketTmpl, config)
	if err != nil {
		return err
	}
	proxyContent, err := renderTemplate("proxy", proxyTmpl, config)
	if err != nil {
		return err
	}
	serviceContent, err := renderTemplate("service", serviceTmpl, config)
	if err != nil {
		return err
	}

	// Write Files
	systemdDir := filepath.Join(u.HomeDir, ".config/systemd/user")
	if err := runAsUser(config.Username, "mkdir", "-p", systemdDir); err != nil {
		return err
	}

	if err := writeAsUser(config.Username, socketContent, filepath.Join(systemdDir, "rest-api.socket")); err != nil {
		return err
	}
	if err := writeAsUser(config.Username, proxyContent, filepath.Join(systemdDir, "rest-api-proxy.service")); err != nil {
		return err
	}
	if err := writeAsUser(config.Username, serviceContent, filepath.Join(systemdDir, "rest-api.service")); err != nil {
		return err
	}

	// Reload & Enable only the Socket
	if err := runAsUser(config.Username, "systemctl", "--user", "daemon-reload"); err != nil {
		return err
	}
	if err := runAsUser(config.Username, "systemctl", "--user", "enable", "--now", "rest-api.socket"); err != nil {
		return err
	}

	fmt.Println("✅ Autoscaling Active. Service will die after", config.IdleTime, "of silence.")
	return nil
}

// Helper to execute a template string
func renderTemplate(name, tmplStr string, data SystemdConfig) (string, error) {
	t, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %v", name, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %v", name, err)
	}

	return buf.String(), nil
}

func init() {
	rootCmd.AddCommand(setupSystemdCmd)
	setupSystemdCmd.Flags().StringVarP(&setupTenantName, "name", "n", "", "Tenant Name")
	// Default to 5 minutes, but allowing "10s" is great for demos/testing
	setupSystemdCmd.Flags().StringVarP(&setupIdleTime, "idle", "i", "5min", "Time before service dies (e.g. 10s, 5min)")
	_ = setupSystemdCmd.MarkFlagRequired("name")
}
