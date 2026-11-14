package cmd

import (
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
	mcpsrv "github.com/bwireman/archivist/internal/mcp"
	"github.com/bwireman/archivist/internal/skills"
	"github.com/bwireman/archivist/internal/store"
)

func TestUserMCPServesSearchWithoutPathFlag(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	orig := repoPath
	repoPath = root
	t.Cleanup(func() { repoPath = orig })
	if err := skills.InstallUser(""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Open(config.StorePath(root)); err != nil {
		t.Fatal(err)
	}
	loaded, cfg, err := loadEnv()
	if err != nil {
		t.Fatal(err)
	}
	repo, homeDB, extras, err := openReadArchives(loaded, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	defer homeDB.Close()
	defer closeExtras(extras)
	srv := mcpsrv.New(loaded, cfg, repo, homeDB, embed.OptionalFromConfig(t.Context(), cfg.Ollama))
	if srv.MCPServer().GetTool("search") == nil {
		t.Fatal("mcp did not serve search")
	}
}
