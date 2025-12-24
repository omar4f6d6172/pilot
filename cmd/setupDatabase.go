package cmd

import (
	"fmt"
	"os/exec"
)

// SetupDatabase creates a PostgreSQL user and database for the tenant
func SetupDatabase(username string) error {
	fmt.Printf("🐘 Configuring PostgreSQL for %s...\n", username)

	// 1. Check if role exists
	// We use "sudo -u postgres psql" to check.
	// This assumes the host has a "postgres" user with admin rights (standard).
	checkCmd := exec.Command("sudo", "-u", "postgres", "psql", "-tAc", fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname='%s'", username))
	out, _ := checkCmd.CombinedOutput() // Ignore error, empty output means no role

	if string(out) == "" || string(out) == "0\n" {
		// 2. Create Role
		// -S: Not a superuser
		// -R: Cannot create roles
		// -D: Cannot create databases (we create it for them)
		// -l: Can login
		fmt.Printf("   ➕ Creating DB Role '%s'...\n", username)
		createRoleCmd := exec.Command("sudo", "-u", "postgres", "createuser", "-S", "-R", "-D", "-l", username)
		if out, err := createRoleCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create db user: %v, output: %s", err, string(out))
		}
	} else {
		fmt.Printf("   ℹ️  DB Role '%s' already exists.\n", username)
	}

	// 3. Create Database
	// -O owner: Set the owner to the new user
	fmt.Printf("   ➕ Creating Database '%s'...\n", username)
	// Check if DB exists first to avoid error spam
	checkDbCmd := exec.Command("sudo", "-u", "postgres", "psql", "-tAc", fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname='%s'", username))
	out, _ = checkDbCmd.CombinedOutput()

	if string(out) == "" || string(out) == "0\n" {
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
