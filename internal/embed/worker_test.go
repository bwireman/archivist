package embed

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

func TestWorkerOnceDrainsWholeQueue(t *testing.T) {
	st := openStore(t)
	const n = 20
	for i := range n {
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
	for i := range 5 {
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

func openStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestWorkerRequeuesModelAndDimMismatch(t *testing.T) {
	st := openStore(t)
	match := []float32{9, 9, 9, 9, 9, 9, 9, 9}
	ids := []string{"rec_match", "rec_model", "rec_dim"}
	for _, id := range ids {
		upsertQueued(t, st, id, id)
	}
	if err := st.SetRecordVector("rec_match", "test", match); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRecordVector("rec_model", "other", match); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRecordVector("rec_dim", "test", []float32{1, 0}); err != nil {
		t.Fatal(err)
	}
	w := &Worker{Stores: []*store.Store{st}, Embedder: &FakeEmbedder{Dim: 8}, Model: "test"}
	if _, err := w.Run(context.Background(), WorkerOptions{Once: true}); err != nil {
		t.Fatal(err)
	}
	got, model, ok, err := st.GetRecordVector("rec_match")
	if err != nil || !ok || model != "test" || !floatEq(got, match) {
		t.Fatalf("match model %s ok %v vec %v err %v", model, ok, got, err)
	}
	for _, id := range []string{"rec_model", "rec_dim"} {
		vec, model, ok, err := st.GetRecordVector(id)
		if err != nil || !ok || model != "test" || len(vec) != 8 {
			t.Fatalf("%s model %s dim %d ok %v err %v", id, model, len(vec), ok, err)
		}
	}
}

func TestWorkerEmptyModelUsesDefault(t *testing.T) {
	st := openStore(t)
	match := []float32{9, 9, 9, 9, 9, 9, 9, 9}
	upsertQueued(t, st, "rec_keep", "keep")
	if err := st.SetRecordVector("rec_keep", config.DefaultEmbedModel, match); err != nil {
		t.Fatal(err)
	}
	w := &Worker{Stores: []*store.Store{st}, Embedder: &FakeEmbedder{Dim: 8}}
	if _, err := w.Run(context.Background(), WorkerOptions{Once: true}); err != nil {
		t.Fatal(err)
	}
	got, model, ok, err := st.GetRecordVector("rec_keep")
	if err != nil || !ok || model != config.DefaultEmbedModel || !floatEq(got, match) {
		t.Fatalf("model %s ok %v vec %v err %v", model, ok, got, err)
	}
}

func TestWorkerProbeFailureDoesNotDrain(t *testing.T) {
	st := openStore(t)
	upsertQueued(t, st, "rec_wait", "waiting")
	w := &Worker{
		Stores: []*store.Store{st},
		Model:  "test",
		Embedder: embedFunc(func(_ context.Context, text string) ([]float32, error) {
			if text == "archivist" {
				return nil, errors.New("probe down")
			}
			return []float32{1, 0}, nil
		}),
	}
	if _, err := w.Run(context.Background(), WorkerOptions{Once: true}); err == nil || !strings.Contains(err.Error(), "probe down") {
		t.Fatalf("err %v", err)
	}
	depth, _ := st.QueueDepth()
	if depth != 1 {
		t.Fatalf("queue depth %d", depth)
	}
	if _, _, ok, err := st.GetRecordVector("rec_wait"); err != nil || ok {
		t.Fatalf("vector ok %v err %v", ok, err)
	}
}

func TestGiveupDropsBothQueueRows(t *testing.T) {
	repo := openStore(t)
	home := openStore(t)
	upsertQueued(t, home, "rec_home", "Home record")
	if err := repo.SeedEmbedQueue("rec_home", "x", 4); err != nil {
		t.Fatal(err)
	}
	w := &Worker{
		Stores: []*store.Store{repo, home},
		Model:  "test",
		Embedder: embedFunc(func(context.Context, string) ([]float32, error) {
			return nil, errors.New("nope")
		}),
	}
	ok, err := w.processOne(context.Background(), repo, store.QueueItem{RecordID: "rec_home", Attempts: 4})
	if err != nil || ok {
		t.Fatalf("ok %v err %v", ok, err)
	}
	for _, st := range []*store.Store{repo, home} {
		n, err := st.QueueDepth()
		if err != nil || n != 0 {
			t.Fatalf("queue %d err %v", n, err)
		}
	}
	gave, err := home.EmbedGiveups()
	if err != nil || len(gave) != 1 || gave[0].ID != "rec_home" {
		t.Fatalf("giveups %+v err %v", gave, err)
	}
	if err := home.SetRecordVector("rec_home", "test", []float32{1, 0}); err != nil {
		t.Fatal(err)
	}
	gave, err = home.EmbedGiveups()
	if err != nil || len(gave) != 0 {
		t.Fatalf("giveup remained %+v err %v", gave, err)
	}
}

func floatEq(a, b []float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
