package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	osuser "os/user"
	"strconv"
	"strings"
)

// LogAction logs an administrative action to the centralized log file
func LogAction(action, user, status string) {
	log.Printf("ACTION=%s USER=%s STATUS=%s", action, user, status)
}

// runAsUser executes a command as a specific user using runuser.
// It assumes the current process has root privileges for runuser.
func runAsUser(username string, command ...string) error {
	// 1. Get the UID (needed for the path /run/user/UID)
	u, err := osuser.Lookup(username) // Use osuser.Lookup
	if err != nil {
		return fmt.Errorf("user lookup failed: %v", err)
	}

	// 2. Construct the environment variables manually
	// This tells systemctl exactly where to find the bus
	xdgRuntime := fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%s", u.Uid)
	dbusAddr := fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/%s/bus", u.Uid)

	// 3. Prepare the command
	// We use "runuser" with "-u user -- command"
	// But we wrap it in /bin/bash to inject the variables cleanly
	fullCmd := fmt.Sprintf("export %s; export %s; %s", xdgRuntime, dbusAddr, strings.Join(command, " "))

	cmd := exec.Command("runuser", "-u", username, "--", "/bin/bash", "-c", fullCmd)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command as user '%s': %v", username, err)
	}
	return nil
}

// writeAsUser writes content to a file and sets its ownership to the specified user.
// It assumes the current process has root privileges to write and chown files.
func writeAsUser(username string, content string, filePath string) error {
	// Write content to file
	err := os.WriteFile(filePath, []byte(content), 0644) // Default file permissions
	if err != nil {
		return fmt.Errorf("failed to write file %s: %v", filePath, err)
	}

	// Lookup user to get UID and GID
	u, err := osuser.Lookup(username) // Use osuser.Lookup
	if err != nil {
		return fmt.Errorf("could not find user %s for chown: %v", username, err)
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return fmt.Errorf("invalid UID for user %s: %v", username, err)
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return fmt.Errorf("invalid GID for user %s: %v", username, err)
	}

	// Change file ownership
	err = os.Chown(filePath, uid, gid)
	if err != nil {
		return fmt.Errorf("failed to change ownership of file %s to user %s: %v", filePath, username, err)
	}
	fmt.Printf("💾 Wrote file %s and set ownership to %s.\n", filePath, username)
	return nil
}
