package cmd

import (
	"time"

	"github.com/bwireman/archivist/internal/cmdlog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func wrapArchiveCommandLogs(root *cobra.Command) {
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		if archiveCommand(c) && c.RunE != nil {
			wrapLoggedRunE(c)
		}
		for _, child := range c.Commands() {
			walk(child)
		}
	}
	walk(root)
}

func archiveCommand(c *cobra.Command) bool {
	switch c.Name() {
	case "search", "map", "status", "remember", "update", "retire", "check", "index", "embed", "export", "import", "publish":
		return true
	default:
		return false
	}
}

func wrapLoggedRunE(c *cobra.Command) {
	orig := c.RunE
	c.RunE = func(cmd *cobra.Command, args []string) error {
		logger := cmdlog.Disabled()
		if root, cfg, err := loadEnv(); err == nil {
			logger = cmdlog.FromConfig(root, cfg)
		}
		start := time.Now()
		logger.In("cli", cmd.CommandPath(), cliArgs(cmd, args))
		err := orig(cmd, args)
		logger.Out("cli", cmd.CommandPath(), nil, err, start)
		return err
	}
}

func cliArgs(cmd *cobra.Command, args []string) map[string]any {
	flags := map[string]string{}
	cmd.Flags().Visit(func(f *pflag.Flag) {
		if f.Name == "help" {
			return
		}
		flags[f.Name] = f.Value.String()
	})
	out := map[string]any{}
	if len(args) > 0 {
		out["args"] = args
	}
	if len(flags) > 0 {
		out["flags"] = flags
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
