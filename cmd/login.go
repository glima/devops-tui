package cmd

import (
	"fmt"
	"os"

	"github.com/samuelenocsson/devops-tui/internal/auth"
)

// Login forces re-authentication via device flow
func Login() error {
	authenticator := auth.NewDeviceFlowAuthenticator()

	// Clear existing cached token to force re-authentication
	if authenticator.HasCachedToken() {
		fmt.Println("Clearing existing cached credentials...")
		if err := authenticator.ClearCache(); err != nil {
			return fmt.Errorf("failed to clear cached credentials: %w", err)
		}
	}

	fmt.Println("Starting authentication...")
	fmt.Println()

	// Perform device flow authentication
	_, err := authenticator.GetToken()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Println("You can now run 'devops-tui' to start the application.")
	return nil
}

// ExecuteLogin runs the login command
func ExecuteLogin() {
	if err := Login(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
