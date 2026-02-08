package cli

import (
	"fmt"
	"strings"

	"github.com/sh0o0/gw/internal/gitx"
	"github.com/spf13/cobra"
)

const (
	configKeyNewOpenEditor   = "gw.new.open-editor"
	configKeyAddOpenEditor   = "gw.add.open-editor"
	configKeyHooksBackground = "gw.hooks.background"
	configKeyEditor          = "gw.editor"
	configKeyAI              = "gw.ai"
	configKeyWorktreeBase    = "gw.worktree.base"
	configKeyWorktreeFlat    = "gw.worktree.flat"
)

type gwConfig struct {
	NewOpenEditor   bool
	AddOpenEditor   bool
	HooksBackground bool
	Editor          string
	AI              string
}

func loadConfig() gwConfig {
	cfg := gwConfig{}

	if v, err := gitx.ConfigGet("", configKeyNewOpenEditor); err == nil {
		cfg.NewOpenEditor = strings.EqualFold(v, "true")
	}
	if v, err := gitx.ConfigGet("", configKeyAddOpenEditor); err == nil {
		cfg.AddOpenEditor = strings.EqualFold(v, "true")
	}
	if v, err := gitx.ConfigGet("", configKeyHooksBackground); err == nil {
		cfg.HooksBackground = strings.EqualFold(v, "true")
	}
	if v, err := gitx.ConfigGet("", configKeyEditor); err == nil {
		cfg.Editor = v
	}
	if v, err := gitx.ConfigGet("", configKeyAI); err == nil {
		cfg.AI = v
	}
	return cfg
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage gw configuration",
	}

	cmd.AddCommand(
		newConfigGetCmd(),
		newConfigSetCmd(),
		newConfigListCmd(),
	)

	return cmd
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := normalizeConfigKey(args[0])
			v, err := gitx.ConfigGet("", key)
			if err != nil {
				return fmt.Errorf("key not found: %s", args[0])
			}
			fmt.Println(v)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := normalizeConfigKey(args[0])
			value := args[1]
			if err := gitx.ConfigSet("", key, value); err != nil {
				return fmt.Errorf("failed to set config: %w", err)
			}
			fmt.Printf("Set %s = %s\n", key, value)
			return nil
		},
	}
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all gw configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			configs, err := gitx.ConfigGetRegexp("", "^gw\\.")
			if err != nil || len(configs) == 0 {
				fmt.Println("No gw configuration found")
				return nil
			}
			for _, kv := range configs {
				fmt.Printf("%s = %s\n", kv.Key, kv.Value)
			}
			return nil
		},
	}
}

func normalizeConfigKey(key string) string {
	switch key {
	case "new.open-editor":
		return configKeyNewOpenEditor
	case "add.open-editor":
		return configKeyAddOpenEditor
	case "hooks.background", "hooks-background":
		return configKeyHooksBackground
	case "hooks.post-create", "post-create":
		return "gw.hooks.post-create"
	case "editor":
		return configKeyEditor
	case "ai":
		return configKeyAI
	case "worktree.base":
		return configKeyWorktreeBase
	case "worktree.flat":
		return configKeyWorktreeFlat
	default:
		if !strings.HasPrefix(key, "gw.") {
			return "gw." + key
		}
		return key
	}
}
