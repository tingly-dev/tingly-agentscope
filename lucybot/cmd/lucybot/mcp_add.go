package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tingly-dev/lucybot/internal/config"
	"github.com/tingly-dev/lucybot/internal/mcp"
	"github.com/urfave/cli/v2"
)

// mcpCommand is the parent command for MCP management
var mcpCommand = &cli.Command{
	Name:  "mcp",
	Usage: "Manage MCP (Model Context Protocol) servers",
	Subcommands: []*cli.Command{
		mcpAddCommand,
		mcpListCommand,
	},
}

// mcpAddCommand adds a new MCP server
var mcpAddCommand = &cli.Command{
	Name:      "add",
	Usage:     "Add a new MCP server",
	ArgsUsage: "<name>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "command",
			Aliases: []string{"c"},
			Usage:   "Command to run (for stdio transport)",
		},
		&cli.StringFlag{
			Name:  "args",
			Usage: "Arguments for the command (quoted string, for stdio transport)",
		},
		&cli.StringSliceFlag{
			Name:  "env",
			Usage: "Environment variables (KEY=VALUE format)",
		},
		&cli.StringFlag{
			Name:    "transport",
			Aliases: []string{"t"},
			Usage:   "Transport type (stdio, http, streamable-http)",
			Value:   "stdio",
		},
		&cli.StringFlag{
			Name:    "url",
			Aliases: []string{"u"},
			Usage:   "URL for HTTP-based transports",
		},
		&cli.StringSliceFlag{
			Name:  "header",
			Usage: "HTTP headers (KEY=VALUE format, for HTTP transports)",
		},
		&cli.IntFlag{
			Name:  "timeout",
			Usage: "Connection timeout in seconds",
			Value: 30,
		},
		&cli.BoolFlag{
			Name:    "global",
			Aliases: []string{"g"},
			Usage:   "Add to global config instead of project config",
		},
		&cli.BoolFlag{
			Name:    "force",
			Aliases: []string{"f"},
			Usage:   "Overwrite existing server if it exists",
		},
		&cli.BoolFlag{
			Name:  "lazy-load",
			Usage: "Enable lazy loading for this server",
		},
		&cli.BoolFlag{
			Name:  "no-lazy-load",
			Usage: "Disable lazy loading for this server",
		},
		&cli.StringSliceFlag{
			Name:  "trigger",
			Usage: "Trigger keywords for lazy loading",
		},
		&cli.StringSliceFlag{
			Name:  "preload-with",
			Usage: "Server names to preload with this server",
		},
		&cli.BoolFlag{
			Name:  "skip-test",
			Usage: "Skip connection test",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() == 0 {
			return fmt.Errorf("server name is required")
		}
		name := c.Args().First()

		// Determine transport type
		transportType := determineTransportType(c)

		// Parse args if provided
		var args []string
		if argsStr := c.String("args"); argsStr != "" {
			var err error
			args, err = parseArgs(argsStr)
			if err != nil {
				return fmt.Errorf("failed to parse args: %w", err)
			}
		}

		// Parse env vars
		env, err := parseKeyValuePairs(c.StringSlice("env"))
		if err != nil {
			return fmt.Errorf("failed to parse env: %w", err)
		}

		// Parse headers
		headers, err := parseKeyValuePairs(c.StringSlice("header"))
		if err != nil {
			return fmt.Errorf("failed to parse headers: %w", err)
		}

		// Determine lazy load setting
		var lazyLoad *bool
		if c.IsSet("lazy-load") && c.Bool("lazy-load") {
			trueVal := true
			lazyLoad = &trueVal
		} else if c.IsSet("no-lazy-load") && c.Bool("no-lazy-load") {
			falseVal := false
			lazyLoad = &falseVal
		}

		// Create server config
		serverConfig := mcp.MCPServerConfig{
			Name:        name,
			Type:        transportType,
			Command:     c.String("command"),
			Args:        args,
			Env:         env,
			URL:         c.String("url"),
			Headers:     headers,
			Timeout:     c.Int("timeout"),
			Enabled:     true,
			LazyLoad:    lazyLoad,
			Triggers:    c.StringSlice("trigger"),
			PreloadWith: c.StringSlice("preload-with"),
		}

		// Validate the config
		if err := serverConfig.Validate(); err != nil {
			return fmt.Errorf("invalid server configuration: %w", err)
		}

		// Test connection if not skipped
		if !c.Bool("skip-test") {
			fmt.Printf("Testing connection to MCP server '%s'...\n", name)
			toolCount, err := mcp.TestServerConnection(context.Background(), &serverConfig)
			if err != nil {
				return fmt.Errorf("connection test failed: %w (use --skip-test to bypass)", err)
			}
			if toolCount >= 0 {
				fmt.Printf("  ✓ Connection successful (%d tools available)\n", toolCount)
			} else {
				fmt.Printf("  ✓ Connection successful\n")
			}
		}

		// Load existing config
		cfg, configPath, err := loadConfigForUpdate(c.Bool("global"))
		if err != nil {
			return err
		}

		// Check if server already exists
		if existing, exists := cfg.MCP.Servers[name]; exists && !c.Bool("force") {
			return fmt.Errorf("server '%s' already exists (use --force to overwrite)\n\nExisting config:\n  Type: %s\n  Command: %s\n  URL: %s",
				name, existing.GetType(), existing.Command, existing.URL)
		}

		// Initialize servers map if needed
		if cfg.MCP.Servers == nil {
			cfg.MCP.Servers = make(map[string]mcp.MCPServerConfig)
		}

		// Add/update the server
		cfg.MCP.Servers[name] = serverConfig

		// Save config
		if err := saveMCPConfig(cfg, configPath); err != nil {
			return err
		}

		// Print success message
		fmt.Printf("\n✓ Added MCP server '%s' to %s\n", name, configPath)
		fmt.Printf("\nConfiguration:\n")
		fmt.Printf("  Name:    %s\n", name)
		fmt.Printf("  Type:    %s\n", transportType)
		if serverConfig.Command != "" {
			fmt.Printf("  Command: %s\n", serverConfig.Command)
		}
		if len(serverConfig.Args) > 0 {
			fmt.Printf("  Args:    %s\n", strings.Join(serverConfig.Args, " "))
		}
		if serverConfig.URL != "" {
			fmt.Printf("  URL:     %s\n", serverConfig.URL)
		}
		if lazyLoad != nil {
			fmt.Printf("  Lazy:    %v\n", *lazyLoad)
		}
		if len(serverConfig.Triggers) > 0 {
			fmt.Printf("  Triggers: %s\n", strings.Join(serverConfig.Triggers, ", "))
		}

		return nil
	},
}

// mcpListCommand lists configured MCP servers
var mcpListCommand = &cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List configured MCP servers",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "Show all servers including disabled ones",
		},
	},
	Action: func(c *cli.Context) error {
		cfg, err := config.LoadConfigWithMerge()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(cfg.MCP.Servers) == 0 {
			fmt.Println("No MCP servers configured.")
			fmt.Println("\nUse 'lucybot mcp add <name>' to add a server.")
			return nil
		}

		showAll := c.Bool("all")

		fmt.Println("\nConfigured MCP Servers:")
		fmt.Println(strings.Repeat("=", 60))

		for name, server := range cfg.MCP.Servers {
			if !server.Enabled && !showAll {
				continue
			}

			status := "enabled"
			if !server.Enabled {
				status = "disabled"
			}

			fmt.Printf("\n  %s (%s, %s)\n", name, server.GetType(), status)

			if server.Command != "" {
				fmt.Printf("    Command: %s\n", server.Command)
			}
			if len(server.Args) > 0 {
				fmt.Printf("    Args:    %s\n", strings.Join(server.Args, " "))
			}
			if server.URL != "" {
				fmt.Printf("    URL:     %s\n", server.URL)
			}
			if server.LazyLoad != nil && *server.LazyLoad {
				fmt.Printf("    Lazy:    true\n")
			}
			if len(server.Triggers) > 0 {
				fmt.Printf("    Triggers: %s\n", strings.Join(server.Triggers, ", "))
			}
		}

		fmt.Println()
		return nil
	},
}

// determineTransportType determines the transport type from flags
func determineTransportType(c *cli.Context) string {
	transport := c.String("transport")

	// Auto-detect based on other flags if not explicitly set
	if !c.IsSet("transport") {
		if c.String("url") != "" {
			// Check if streamable-http is implied
			// For now, default to http if URL is provided without explicit transport
			return "http"
		}
	}

	return transport
}

// parseArgs parses a quoted argument string into a slice
func parseArgs(argsStr string) ([]string, error) {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range argsStr {
		switch r {
		case '"', '\'':
			if !inQuote {
				inQuote = true
				quoteChar = r
			} else if r == quoteChar {
				inQuote = false
				quoteChar = 0
			} else {
				current.WriteRune(r)
			}
		case ' ', '\t':
			if inQuote {
				current.WriteRune(r)
			} else {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args, nil
}

// parseKeyValuePairs parses KEY=VALUE pairs from a slice of strings
func parseKeyValuePairs(pairs []string) (map[string]string, error) {
	result := make(map[string]string)

	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid key=value pair: %s", pair)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}

		result[key] = value
	}

	return result, nil
}

// loadConfigForUpdate loads the config for modification
func loadConfigForUpdate(global bool) (*config.Config, string, error) {
	var configPath string

	if global {
		configPath = config.GetGlobalConfigPath()
		if configPath == "" {
			return nil, "", fmt.Errorf("could not determine global config path")
		}
	} else {
		configPath = config.GetProjectConfigPath()
	}

	// Load existing config or create new one
	var cfg *config.Config
	if _, err := os.Stat(configPath); err == nil {
		// File exists, load it
		var loadErr error
		cfg, loadErr = config.LoadConfig(configPath)
		if loadErr != nil {
			// Try to load with merge to get defaults
			cfg, _ = config.LoadConfigWithMerge()
		}
	} else {
		// Create new config with defaults
		cfg = config.GetDefaultConfig()
	}

	return cfg, configPath, nil
}

// saveMCPConfig saves the MCP configuration to the specified path
func saveMCPConfig(cfg *config.Config, path string) error {
	// Ensure directory exists
	dir := path[:strings.LastIndex(path, string(os.PathSeparator))]
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save the config
	if err := config.SaveConfig(cfg, path); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}
