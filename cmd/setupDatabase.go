package cmd

import (
	"fmt"
	"os/exec"
	"strings"
)

// SetupDatabase creates a PostgreSQL user and database for the tenant
func SetupDatabase(username string) error {
	// 0. Validate Username
	if err := ValidateUsername(username); err != nil {
		return fmt.Errorf("security check failed: %v", err)
	}

	fmt.Printf("🐘 Configuring PostgreSQL for %s...\n", username)

	// 1. Check if role exists
	// We use "sudo -u postgres psql -tAc ..." to check safely.
	checkCmd := exec.Command("sudo", "-u", "postgres", "psql", "-tAc", fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname='%s'", username))
	out, _ := checkCmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(out))

	if outputStr != "1" {
		// 2. Create Role
		fmt.Printf("   ➕ Creating DB Role '%s'...\n", username)
		createRoleCmd := exec.Command("sudo", "-u", "postgres", "createuser", "-S", "-R", "-D", "-l", username)
		if out, err := createRoleCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create db user: %v, output: %s", err, string(out))
		}
	} else {
		fmt.Printf("   ℹ️  DB Role '%s' already exists.\n", username)
	}

	// 3. Check if Database exists
	checkDbCmd := exec.Command("sudo", "-u", "postgres", "psql", "-tAc", fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname='%s'", username))
	out, _ = checkDbCmd.CombinedOutput()
	outputStr = strings.TrimSpace(string(out))

	if outputStr != "1" {
		// 4. Create Database
		fmt.Printf("   ➕ Creating Database '%s'...\n", username)
		createDbCmd := exec.Command("sudo", "-u", "postgres", "createdb", "-O", username, username)
		if out, err := createDbCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create database: %v, output: %s", err, string(out))
		}
		fmt.Printf("✅ Database ready.\n")
	} else {
		fmt.Printf("   ℹ️  Database '%s' already exists.\n", username)
	}

	return nil
}
