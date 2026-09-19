package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/config"
)

func TestDocsExportTargetSkippedByDefault(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	out, skip := docsExportTarget(root, cfg, "")
	if out != "" {
		t.Fatalf("outDir: %q", out)
	}
	if !strings.Contains(skip, "records.write_docs is false") {
		t.Fatalf("skip: %q", skip)
	}
	if !strings.Contains(skip, config.DefaultArchiveDir) {
		t.Fatalf("skip missing export path: %q", skip)
	}
}

func TestDocsExportTargetWritesWhenEnabled(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Records.WriteDocs = true
	out, skip := docsExportTarget(root, cfg, "")
	if skip != "" {
		t.Fatalf("skip: %q", skip)
	}
	want := filepath.Join(root, config.DefaultArchiveDir)
	if out != want {
		t.Fatalf("outDir: %q want %q", out, want)
	}
}

func TestDocsExportTargetBundleIgnoresWriteDocs(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	bundle := filepath.Join(root, "bundle")
	out, skip := docsExportTarget(root, cfg, bundle)
	if skip != "" {
		t.Fatalf("skip: %q", skip)
	}
	if out != bundle {
		t.Fatalf("outDir: %q want %q", out, bundle)
	}
}
