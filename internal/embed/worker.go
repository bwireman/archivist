package embed

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

type Worker struct {
	Stores   []*store.Store
	Embedder Embedder
	Model    string
}

type WorkerOptions struct {
	Once        bool
	Concurrency int
	BatchSize   int
}

func DefaultWorkerOptions() WorkerOptions {
	return WorkerOptions{Concurrency: 2, BatchSize: 16}
}

func (w *Worker) Run(ctx context.Context, opts WorkerOptions) (int, error) {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 1
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 16
	}
	total := 0
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n, err := w.drain(ctx, opts)
		total += n
		if err != nil {
			return total, err
		}
		if opts.Once || n == 0 {
			return total, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (w *Worker) drain(ctx context.Context, opts WorkerOptions) (int, error) {
	var items []store.QueueItem
	for _, st := range w.Stores {
		if st == nil {
			continue
		}
		batch, err := st.DequeueEmbed(opts.BatchSize)
		if err != nil {
			return 0, err
		}
		items = append(items, batch...)
	}
	if len(items) == 0 {
		return 0, nil
	}

	sem := make(chan struct{}, opts.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	done := 0
	var firstErr error

	for _, item := range items {
		wg.Add(1)
		go func(item store.QueueItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := w.processOne(ctx, item); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			mu.Lock()
			done++
			mu.Unlock()
		}(item)
	}
	wg.Wait()
	return done, firstErr
}

func (w *Worker) processOne(ctx context.Context, item store.QueueItem) error {
	for _, st := range w.Stores {
		if st == nil {
			continue
		}
		rec, ok, err := st.GetRecordByID(item.RecordID)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		text := rec.EmbedText()
		emb, err := w.Embedder.Embed(ctx, text)
		if err != nil {
			_ = st.FailQueueItem(item.RecordID, err.Error())
			return fmt.Errorf("embed %s: %w", rec.SourcePath, err)
		}
		model := w.Model
		if model == "" {
			model = config.DefaultEmbedModel
		}
		return st.SetRecordVector(rec.ID, model, emb)
	}
	return nil
}

// OpenWorkerStores returns repo and home stores for embedding.
func OpenWorkerStores(repoRoot string) ([]*store.Store, error) {
	var out []*store.Store
	repo, err := store.Open(config.StorePath(repoRoot))
	if err != nil {
		return nil, err
	}
	out = append(out, repo)
	homePath := config.HomeStorePath()
	if homePath != "" {
		home, err := store.Open(homePath)
		if err != nil {
			_ = repo.Close()
			return nil, err
		}
		out = append(out, home)
	}
	return out, nil
}

// EnqueueRecord upserts a record and queues embedding.
func EnqueueRecord(st *store.Store, rec *record.Record) error {
	return st.UpsertRecord(rec)
}
