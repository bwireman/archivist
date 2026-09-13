package embed

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

func TestWorkerOnceDrainsWholeQueue(t *testing.T) {
	st := openStore(t)
	const n = 20
	for i := 0; i < n; i++ {
		upsertQueued(t, st, fmt.Sprintf("rec_%02d", i), fmt.Sprintf("Title %d", i))
	}
	w := &Worker{Stores: []*store.Store{st}, Embedder: &FakeEmbedder{Dim: 8}, Model: "test"}
	got, err := w.Run(context.Background(), WorkerOptions{Once: true, Concurrency: 4})
	if err != nil {
		t.Fatal(err)
	}
	if got != n {
		t.Fatalf("embedded %d, want %d", got, n)
	}
	depth, _ := st.QueueDepth()
	if depth != 0 {
		t.Fatalf("queue depth %d, want 0", depth)
	}
}

func TestWorkerContinuesAfterOneFailure(t *testing.T) {
	st := openStore(t)
	upsertQueued(t, st, "rec_a", "alpha")
	upsertQueued(t, st, "rec_b", "fail-please")
	upsertQueued(t, st, "rec_c", "charlie")
	var calls atomic.Int32
	w := &Worker{
		Stores: []*store.Store{st},
		Model:  "test",
		Embedder: embedFunc(func(_ context.Context, text string) ([]float32, error) {
			calls.Add(1)
			if strings.Contains(text, "fail-please") {
				return nil, errors.New("boom")
			}
			return []float32{1, 0, 0, 0}, nil
		}),
	}
	got, err := w.Run(context.Background(), WorkerOptions{Once: true, Concurrency: 2})
	if err == nil {
		t.Fatal("expected embed error")
	}
	if got != 2 {
		t.Fatalf("embedded %d, want 2 siblings", got)
	}
	depth, _ := st.QueueDepth()
	if depth != 1 {
		t.Fatalf("failed item should remain, depth %d", depth)
	}
}

func TestWorkerDropsGhostQueueItems(t *testing.T) {
	st := openStore(t)
	w := &Worker{Stores: []*store.Store{st}, Embedder: &FakeEmbedder{Dim: 8}, Model: "test"}
	ok, err := w.processOne(context.Background(), st, store.QueueItem{RecordID: "rec_missing"})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("missing record should not count as embedded")
	}
}

func TestWorkerEmbedsHomeStoreWhenRepoIsEmpty(t *testing.T) {
	repo := openStore(t)
	home := openStore(t)
	upsertQueued(t, home, "rec_home", "Home record")
	w := &Worker{Stores: []*store.Store{repo, home}, Embedder: &FakeEmbedder{Dim: 8}, Model: "test"}
	got, err := w.Run(context.Background(), WorkerOptions{Once: true, Concurrency: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("embedded %d, want 1", got)
	}
	depth, _ := home.QueueDepth()
	if depth != 0 {
		t.Fatalf("home queue depth %d, want 0", depth)
	}
}

func TestWorkerFindsRecordOnOtherStore(t *testing.T) {
	repo := openStore(t)
	home := openStore(t)
	upsertQueued(t, home, "rec_home", "Home record")
	w := &Worker{Stores: []*store.Store{repo, home}, Embedder: &FakeEmbedder{Dim: 8}, Model: "test"}
	ok, err := w.processOne(context.Background(), repo, store.QueueItem{RecordID: "rec_home"})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("should embed using the store that has the record")
	}
	depth, _ := home.QueueDepth()
	if depth != 0 {
		t.Fatalf("home queue depth %d after embedding, want 0", depth)
	}
}

func TestWorkerWithoutOnceDrainsQueue(t *testing.T) {
	st := openStore(t)
	for i := 0; i < 5; i++ {
		upsertQueued(t, st, fmt.Sprintf("rec_%d", i), fmt.Sprintf("Title %d", i))
	}
	w := &Worker{Stores: []*store.Store{st}, Embedder: &FakeEmbedder{Dim: 8}, Model: "test"}
	got, err := w.Run(context.Background(), WorkerOptions{Concurrency: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got != 5 {
		t.Fatalf("embedded %d, want 5", got)
	}
	depth, _ := st.QueueDepth()
	if depth != 0 {
		t.Fatalf("queue depth %d, want 0", depth)
	}
}

type embedFunc func(context.Context, string) ([]float32, error)

func (f embedFunc) Embed(ctx context.Context, text string) ([]float32, error) {
	return f(ctx, text)
}

func (f embedFunc) Dimensions() int { return 4 }

func openStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func upsertQueued(t *testing.T, st *store.Store, id, title string) {
	t.Helper()
	rec := &record.Record{
		ID:         id,
		Slug:       id,
		Type:       record.TypeDecision,
		Scope:      record.ScopeRepo,
		Title:      title,
		Status:     record.StatusAccepted,
		Body:       title + " body",
		SourcePath: "docs/decisions/" + id + ".md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
}
