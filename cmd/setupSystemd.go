package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

// 1. Data Structure for Templates
type SystemdConfig struct {
	Username     string
	UID          string
	ExternalPort int
}

// 2. The Templates
// We use raw string literals (backticks) for clean multi-line text.

const socketTmpl = `[Unit]
Description=Public Socket for {{.Username}}

[Socket]
ListenStream=/run/pilot/{{.Username}}.sock
SocketMode=0666
Service=rest-api-proxy.service

[Install]
WantedBy=sockets.target
`

const proxyTmpl = `[Unit]
Description=Socket Proxy for {{.Username}}
Requires=rest-api.service
After=rest-api.service

[Service]
# Proxy traffic from the FD passed by systemd to the internal localhost port
ExecStart=/usr/lib/systemd/systemd-socket-proxyd 127.0.0.1:{{.UID}}
NonBlocking=true
`

const serviceTmpl = `[Unit]
Description=User REST API Backend
# Stop the service if it sits idle for 5 minutes (Resource Saving!)
StopWhenUnneeded=true

[Service]
ExecStart=/usr/local/bin/user-rest-api
# The App listens on its own UID as a port
Environment=PORT={{.UID}}
Type=simple
`

// Flags
var setupTenantName string
var setupExternalPort int

var setupSystemdCmd = &cobra.Command{
	Use:   "setup-systemd",
	Short: "Generates systemd units using Go templates",
	Run: func(cmd *cobra.Command, args []string) {

		// A. Gather Data
		u, err := user.Lookup(setupTenantName)
		if err != nil {
			log.Fatalf("❌ Could not find user %s: %v", setupTenantName, err)
		}

		config := SystemdConfig{
			Username:     setupTenantName,
			UID:          u.Uid,
			ExternalPort: setupExternalPort,
		}

		// Ensure shared socket directory exists and is writable
		socketDir := "/run/pilot"
		if err := os.MkdirAll(socketDir, 0777); err != nil {
			log.Fatalf("❌ Failed to create socket directory %s: %v", socketDir, err)
		}
		// Force permissions (MkdirAll respects umask)
		if err := os.Chmod(socketDir, 0777); err != nil {
			log.Fatalf("❌ Failed to chmod socket directory %s: %v", socketDir, err)
		}

		fmt.Printf("🔧 Generating config for %s (Socket %s/%s.sock -> Port %s)...\n", config.Username, socketDir, config.Username, config.UID)

		// B. Render Templates
		socketContent := renderTemplate("socket", socketTmpl, config)
		proxyContent := renderTemplate("proxy", proxyTmpl, config)
		serviceContent := renderTemplate("service", serviceTmpl, config)

		// C. Write to Disk
		systemdDir := filepath.Join(u.HomeDir, ".config/systemd/user")
		runAsUser(config.Username, "mkdir", "-p", systemdDir)

		writeAsUser(config.Username, socketContent, filepath.Join(systemdDir, "rest-api.socket"))
		writeAsUser(config.Username, proxyContent, filepath.Join(systemdDir, "rest-api-proxy.service"))
		writeAsUser(config.Username, serviceContent, filepath.Join(systemdDir, "rest-api.service"))

		// D. Reload & Enable
		fmt.Println("🚀 Reloading Systemd...")
		runAsUser(config.Username, "systemctl", "--user", "daemon-reload")
		runAsUser(config.Username, "systemctl", "--user", "enable", "--now", "rest-api.socket")

		fmt.Println("✅ Success.")
	},
}

// Helper to execute a template string
func renderTemplate(name, tmplStr string, data SystemdConfig) string {
	t, err := template.New(name).Parse(tmplStr)
	if err != nil {
		log.Fatalf("❌ Failed to parse template %s: %v", name, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		log.Fatalf("❌ Failed to execute template %s: %v", name, err)
	}

	return buf.String()
}

// (Ensure you have the runAsUser and writeAsUser helpers from previous steps)
func init() {
	rootCmd.AddCommand(setupSystemdCmd)
	setupSystemdCmd.Flags().StringVarP(&setupTenantName, "name", "n", "", "Tenant Name")
	setupSystemdCmd.Flags().IntVarP(&setupExternalPort, "port", "p", 8080, "External Port")
	_ = setupSystemdCmd.MarkFlagRequired("name")
}