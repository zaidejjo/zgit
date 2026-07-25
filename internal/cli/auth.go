package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zaidejjo/zgit/pkg/core/github"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage GitHub authentication",
	Long:  `Authenticate with GitHub using a personal access token or device flow.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with GitHub",
	Long: `Set up GitHub authentication by providing a personal access token.
Supports both classic and fine-grained PATs with 'repo' and 'read:org' scopes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Check if already authenticated
		if token := cfg.GetString("github.token"); token != "" {
			user := cfg.GetString("github.user")
			if user != "" {
				fmt.Fprintf(os.Stdout, "Already authenticated as %s\n", user)
				fmt.Fprintf(os.Stdout, "Use 'zgit auth logout' to re-authenticate\n")
				return nil
			}
		}

		fmt.Fprintf(os.Stdout, "GitHub Personal Access Token\n")
		fmt.Fprintf(os.Stdout, "Create one at: https://github.com/settings/tokens\n")
		fmt.Fprintf(os.Stdout, "Required scopes: repo, read:org\n\n")

		fmt.Fprintf(os.Stdout, "Enter token: ")
		var token string
		_, err := fmt.Scanln(&token)
		if err != nil {
			return fmt.Errorf("read token: %w", err)
		}
		token = stripSpaces(token)
		if token == "" {
			return fmt.Errorf("token cannot be empty")
		}

		// Validate by creating a client and fetching user
		client, err := github.NewCombinedClient(token)
		if err != nil {
			return fmt.Errorf("invalid token: %w", err)
		}

		user, err := client.GetAuthenticatedUser(ctx)
		if err != nil {
			return fmt.Errorf("token validation failed: %w", err)
		}

		// Save to config
		cfg.Set("github.token", token)
		cfg.Set("github.user", user.Login)
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Fprintf(os.Stdout, "\n✓ Authenticated as %s (%s)\n", user.Login, user.Name)
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  `Display the current GitHub authentication status and user.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		token := cfg.GetString("github.token")
		user := cfg.GetString("github.user")

		if token == "" {
			fmt.Fprintln(os.Stdout, "Not authenticated")
			fmt.Fprintln(os.Stdout, "Run 'zgit auth login' to authenticate")
			return nil
		}

		if user != "" {
			fmt.Fprintf(os.Stdout, "Authenticated as %s\n", user)
		} else {
			fmt.Fprintln(os.Stdout, "Token configured, but user unknown")
		}
		fmt.Fprintf(os.Stdout, "Token: %s…\n", maskToken(token))

		// Validate token is still working
		client, err := github.NewCombinedClient(token)
		if err == nil {
			if _, err := client.GetAuthenticatedUser(context.Background()); err != nil {
				fmt.Fprintf(os.Stderr, "⚠ Token validation failed: %v\n", err)
			} else {
				fmt.Fprintln(os.Stdout, "✓ Token is valid")
			}
		}
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove GitHub authentication",
	Long:  `Clear the stored GitHub token from configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.Set("github.token", "")
		cfg.Set("github.user", "")
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Fprintln(os.Stdout, "✓ Authentication removed")
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return token
	}
	return token[:4] + "…" + token[len(token)-4:]
}

func stripSpaces(s string) string {
	result := make([]byte, 0, len(s))
	for _, b := range []byte(s) {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			result = append(result, b)
		}
	}
	return string(result)
}
