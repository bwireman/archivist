package efficacy

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
	_ "modernc.org/sqlite"
)

func TestGoldenFactsSynthetic(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "index.db")
		st := openWritable(t, path)
		mustUpsert(t, st, &record.Record{
			ID: "rec_rule", Slug: "keep-id", Type: record.TypeRule, Scope: record.ScopeRepo,
			Title: "Keep the note id", Status: record.StatusAccepted,
			Body:       "KEEP-NOTE-ID and WRITE-LEDGER-ROW",
			SourcePath: "docs/decisions/keep.md",
		})
		mustUpsert(t, st, &record.Record{
			ID: "rec_pit", Slug: "decoy", Type: record.TypePitfall, Scope: record.ScopeRepo,
			Title: "Copy then delete", Status: record.StatusAccepted,
			Body:       "COPY-THEN-DELETE drops the id",
			SourcePath: "docs/decisions/decoy.md",
		})
		if err := st.Close(); err != nil {
			t.Fatal(err)
		}
		if err := CheckGolden(path); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("fail", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "index.db")
		st := openWritable(t, path)
		mustUpsert(t, st, &record.Record{
			ID: "rec_bad", Slug: "copy", Type: record.TypeRule, Scope: record.ScopeRepo,
			Title: "Copy", Status: record.StatusAccepted,
			Body:       "COPY-THEN-DELETE is required",
			SourcePath: "docs/decisions/bad.md",
		})
		if err := st.Close(); err != nil {
			t.Fatal(err)
		}
		if err := CheckGolden(path); err == nil {
			t.Fatal("expected golden failure")
		}
	})
}

func TestGoldenFactsOnDB(t *testing.T) {
	path := os.Getenv("EFFICACY_DB")
	if path == "" {
		t.Skip("EFFICACY_DB unset")
	}
	if err := CheckGolden(path); err != nil {
		t.Fatal(err)
	}
}

func TestCheckpointWAL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.db")
	st := openWritable(t, path)
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	if err := Checkpoint(path); err != nil {
		t.Fatal(err)
	}
}

func TestCheckpointWALOnDB(t *testing.T) {
	path := os.Getenv("EFFICACY_CHECKPOINT")
	if path == "" {
		t.Skip("EFFICACY_CHECKPOINT unset")
	}
	if err := Checkpoint(path); err != nil {
		t.Fatal(err)
	}
}

func CheckGolden(path string) error {
	st, err := store.OpenReadOnly(path)
	if err != nil {
		return err
	}
	defer st.Close()
	recs, err := st.AllRecords()
	if err != nil {
		return err
	}
	var rules, pitfalls []string
	for _, r := range recs {
		if r.Status != record.StatusAccepted || r.SupersededBy != "" {
			continue
		}
		switch r.Type {
		case record.TypeRule, record.TypeDecision:
			rules = append(rules, r.Body)
		case record.TypePitfall:
			pitfalls = append(pitfalls, r.Body)
		}
	}
	if !oneHasBoth(rules, "KEEP-NOTE-ID", "WRITE-LEDGER-ROW") {
		return errors.New("missing KEEP-NOTE-ID and WRITE-LEDGER-ROW on one accepted rule or decision")
	}
	if !anyHas(pitfalls, "COPY-THEN-DELETE") {
		return errors.New("missing COPY-THEN-DELETE on an accepted pitfall")
	}
	if anyHas(rules, "COPY-THEN-DELETE") {
		return errors.New("accepted rule or decision contains COPY-THEN-DELETE")
	}
	return nil
}

func Checkpoint(path string) error {
	db, err := sql.Open("sqlite", path+"?_busy_timeout=5000&_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}

func oneHasBoth(bodies []string, a, b string) bool {
	for _, body := range bodies {
		if strings.Contains(body, a) && strings.Contains(body, b) {
			return true
		}
	}
	return false
}

func anyHas(bodies []string, token string) bool {
	for _, b := range bodies {
		if strings.Contains(b, token) {
			return true
		}
	}
	return false
}

func openWritable(t *testing.T, path string) *store.Store {
	t.Helper()
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func mustUpsert(t *testing.T, st *store.Store, r *record.Record) {
	t.Helper()
	if err := st.UpsertRecord(r); err != nil {
		t.Fatal(err)
	}
}
