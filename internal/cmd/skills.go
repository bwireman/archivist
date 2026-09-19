package cmd

import (
	"fmt"

	"github.com/bwireman/archivist/internal/skills"
	"github.com/spf13/cobra"
)

func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Install always-on rules and on-demand skills",
	}
	install := &cobra.Command{
		Use:   "install",
		Short: "Generate host-specific skill files",
		RunE: func(cmd *cobra.Command, args []string) error {
			target, _ := cmd.Flags().GetString("target")
			t, err := skills.ParseTarget(target)
			if err != nil {
				return err
			}
			root, _, err := loadEnv()
			if err != nil {
				return err
			}
			if err := skills.Install(root, t); err != nil {
				return err
			}
			fmt.Printf("Installed agent rules and skills for %s\n", target)
			return nil
		},
	}
	install.Flags().String("target", "cursor", "cursor|claude|agents-md|copilot|codex")
	cmd.AddCommand(install)
	return cmd
}
