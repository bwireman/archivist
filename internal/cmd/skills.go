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
		RunE: func(cmd *cobra.Command, _ []string) error {
			user, _ := cmd.Flags().GetBool("user")
			cli, _ := cmd.Flags().GetString("cli")
			if user {
				if err := skills.InstallUser(cli); err != nil {
					return err
				}
				fmt.Println("Installed user skills and the archivist MCP server")
				return nil
			}
			target, _ := cmd.Flags().GetString("target")
			t, err := skills.ParseTarget(target)
			if err != nil {
				return err
			}
			root, _, err := loadEnv()
			if err != nil {
				return err
			}
			if err := skills.InstallCLI(root, t, cli); err != nil {
				return err
			}
			fmt.Printf("Installed agent rules and skills for %s\n", target)
			return nil
		},
	}
	install.Flags().String("target", "cursor", "cursor|claude|agents-md|copilot|codex")
	install.Flags().String("cli", "", "command written into rules and skills in place of `archivist` (for example ./bin/archivist)")
	install.Flags().Bool("user", false, "install skills for every workspace under the home directory and add the user MCP server")
	cmd.AddCommand(install)
	return cmd
}
