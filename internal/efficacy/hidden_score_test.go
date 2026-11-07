package efficacy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHiddenScore(t *testing.T) {
	repo := filepath.Join("..", "..")
	script := filepath.Join(repo, "scripts", "efficacy-hidden.sh")
	app := filepath.Join(repo, "testdata", "efficacy-fixture", "steps", "01-app")
	cases := []struct {
		name string
		src  string
		pass bool
	}{
		{name: "pass", src: transferSrc("notebook-transfer", true, true, true), pass: true},
		{name: "wrong kind", src: transferSrc("copy", true, true, true), pass: false},
		{name: "stores blank", src: transferSrc("notebook-transfer", false, true, true), pass: false},
		{name: "missing is 500", src: transferSrc("notebook-transfer", true, false, true), pass: false},
		{name: "no rollback", src: transferSrc("notebook-transfer", true, true, false), pass: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.CopyFS(dir, os.DirFS(app)); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(dir, "transfer"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "transfer", "transfer.go"), []byte(tc.src), 0o644); err != nil {
				t.Fatal(err)
			}
			installTransferRoute(t, dir)
			run := exec.Command("bash", script, dir)
			out, err := run.CombinedOutput()
			if strings.Contains(string(out), "no tests to run") {
				t.Fatalf("hidden test did not run\n%s", out)
			}
			if tc.pass && err != nil {
				t.Fatalf("score: %v\n%s", err, out)
			}
			if !tc.pass && (err == nil || !strings.Contains(string(out), "--- FAIL:")) {
				t.Fatalf("hidden test did not fail\n%s", out)
			}
			if _, statErr := os.Stat(filepath.Join(dir, "transfer", "hidden_test.go")); !os.IsNotExist(statErr) {
				t.Fatal("hidden test left in the checkout")
			}
		})
	}
}

func installTransferRoute(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "internal", "notes", "handler.go")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(body), "\t\"strings\"\n)", "\t\"strings\"\n\n\t\"example.com/notes/transfer\"\n)", 1)
	text = strings.Replace(text, "\treturn mux\n", "\tmux.HandleFunc(\"POST /api/notes/{id}/transfer\", transfer.Handler(db))\n\treturn mux\n", 1)
	if !strings.Contains(text, "example.com/notes/transfer") || !strings.Contains(text, "transfer.Handler") {
		t.Fatal("handler was not patched")
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func transferSrc(kind string, trim, missing404, atomic bool) string {
	blank := `strings.TrimSpace(in.NotebookID) == ""`
	stored := `strings.TrimSpace(in.NotebookID)`
	if !trim {
		blank = `in.NotebookID == ""`
		stored = `in.NotebookID`
	}
	miss := `http.NotFound(w, r)`
	if !missing404 {
		miss = `http.Error(w, "missing", http.StatusInternalServerError)`
	}
	move := atomicMove
	if !atomic {
		move = splitMove
	}
	std := `"database/sql"
	"encoding/json"
	"errors"
	"net/http"`
	if trim {
		std += `
	"strings"`
	}
	return `package transfer

import (
	` + std + `
)

func Handler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			NotebookID string ` + "`json:\"notebook_id\"`" + `
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || ` + blank + ` {
			http.Error(w, "notebook_id required", http.StatusBadRequest)
			return
		}
		id := r.PathValue("id")
		if err := move(db, id, ` + stored + `); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				` + miss + `
				return
			}
			http.Error(w, "transfer failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
	}
}

func move(db *sql.DB, id, dest string) error {
` + strings.ReplaceAll(move, "KIND", kind) + `
}
`
}

const atomicMove = `	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var from string
	if err := tx.QueryRow(` + "`SELECT notebook_id FROM notes WHERE id = ?`" + `, id).Scan(&from); err != nil {
		return err
	}
	if _, err := tx.Exec(` + "`UPDATE notes SET notebook_id = ? WHERE id = ?`" + `, dest, id); err != nil {
		return err
	}
	if _, err := tx.Exec(` + "`INSERT INTO ledger (id, note_id, from_notebook_id, to_notebook_id, kind, created_at) VALUES ('led', ?, ?, ?, 'KIND', 't')`" + `, id, from, dest); err != nil {
		return err
	}
	return tx.Commit()`

const splitMove = `	var from string
	if err := db.QueryRow(` + "`SELECT notebook_id FROM notes WHERE id = ?`" + `, id).Scan(&from); err != nil {
		return err
	}
	if _, err := db.Exec(` + "`UPDATE notes SET notebook_id = ? WHERE id = ?`" + `, dest, id); err != nil {
		return err
	}
	_, err := db.Exec(` + "`INSERT INTO ledger (id, note_id, from_notebook_id, to_notebook_id, kind, created_at) VALUES ('led', ?, ?, ?, 'KIND', 't')`" + `, id, from, dest)
	return err`
