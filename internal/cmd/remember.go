package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bwireman/archivist/internal/archive"
	"github.com/bwireman/archivist/internal/check"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/retrieve"
	"github.com/spf13/cobra"
)

func newRememberCmd() *cobra.Command {
	var recType, scope, title, body, severity string
	var appliesTo, tags []string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "remember",
		Short: "Create a new archive record",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			repo, home, err := openStores(root)
			if err != nil {
				return err
			}
			defer repo.Close()
			defer home.Close()
			svc := archive.New(root, cfg, repo, home)
			rec := &record.Record{
				Type:      record.Type(recType),
				Scope:     record.Scope(scope),
				Title:     title,
				Body:      body,
				Status:    record.StatusAccepted,
				Severity:  record.Severity(severity),
				AppliesTo: appliesTo,
				Tags:      tags,
			}
			id, err := svc.Remember(rec)
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(map[string]string{"id": id})
			}
			fmt.Printf("Remembered %s\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&recType, "type", "decision", "record type: decision, rule, feature, guide, map, pitfall")
	cmd.Flags().StringVar(&scope, "scope", "repo", "scope: dev|repo|global")
	cmd.Flags().StringVar(&title, "title", "", "record title")
	cmd.Flags().StringVar(&body, "body", "", "record body markdown")
	cmd.Flags().StringVar(&severity, "severity", "", "rule severity")
	cmd.Flags().StringSliceVar(&appliesTo, "applies-to", nil, "path globs")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "tags")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("body")
	return cmd
}

func newUpdateCmd() *cobra.Command {
	var title, body, status string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an existing record",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			repo, home, err := openStores(root)
			if err != nil {
				return err
			}
			defer repo.Close()
			defer home.Close()
			svc := archive.New(root, cfg, repo, home)
			err = svc.Update(args[0], func(r *record.Record) error {
				if title != "" {
					r.Title = title
				}
				if body != "" {
					r.Body = body
				}
				if status != "" {
					r.Status = record.Status(status)
				}
				return nil
			})
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "ok"})
			}
			fmt.Println("Updated")
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "new title")
	cmd.Flags().StringVar(&body, "body", "", "new body")
	cmd.Flags().StringVar(&status, "status", "", "new status")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}

func newRetireCmd() *cobra.Command {
	var supersededBy string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "retire <id>",
		Short: "Mark a record superseded",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			repo, home, err := openStores(root)
			if err != nil {
				return err
			}
			defer repo.Close()
			defer home.Close()
			svc := archive.New(root, cfg, repo, home)
			if err := svc.Retire(args[0], supersededBy); err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "ok"})
			}
			fmt.Println("Retired")
			return nil
		},
	}
	cmd.Flags().StringVar(&supersededBy, "superseded-by", "", "replacement record id")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}

func newCheckCmd() *cobra.Command {
	var paths []string
	var diff string
	var strict bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "check [description]",
		Short: "Check rules against a change",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			repo, home, err := openStores(root)
			if err != nil {
				return err
			}
			defer repo.Close()
			defer home.Close()
			embedder := embed.OptionalFromConfig(cmd.Context(), cfg.Ollama)
			engine := &retrieve.Engine{Repo: repo, Home: home}
			res, err := check.Run(cmd.Context(), engine, embedder, check.Options{
				Description: strings.Join(args, " "),
				Paths:       paths,
				Diff:        diff,
				Strict:      strict,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(res)
			}
			for _, m := range res.Matches {
				fmt.Printf("[%s] %s — %s (%s)\n", m.Severity, m.Record.Title, m.Reason, m.Record.SourcePath)
			}
			if strict && res.HasViolation {
				return fmt.Errorf("enforceable rule violations found")
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&paths, "paths", nil, "touched file paths")
	cmd.Flags().StringVar(&diff, "diff", "", "git diff text")
	cmd.Flags().BoolVar(&strict, "strict", false, "exit nonzero when a must/must-not rule's applies_to matches the changed paths")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}
