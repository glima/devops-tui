package cmd

import (
	"fmt"
	"os"

	"github.com/samuelenocsson/devops-tui/internal/auth"
)

// Logout clears the cached OAuth token
func Logout() error {
	authenticator := auth.NewDeviceFlowAuthenticator()

	if !authenticator.HasCachedToken() {
		fmt.Println("No cached credentials found.")
		return nil
	}

	if err := authenticator.ClearCache(); err != nil {
		return fmt.Errorf("failed to clear cached credentials: %w", err)
	}

	fmt.Println("✓ Successfully logged out. Cached credentials have been removed.")
	fmt.Println("  You will need to re-authenticate on next run.")
	return nil
}

// ExecuteLogout runs the logout command
func ExecuteLogout() {
	if err := Logout(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
