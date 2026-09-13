package embed

import (
	"context"
	"fmt"
	"os"
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
}

func (w *Worker) Run(ctx context.Context, opts WorkerOptions) (int, error) {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 1
	}
	total := 0
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n, err := w.drain(ctx, opts)
		total += n
		if opts.Once {
			return total, err
		}
		if w.queueDepth() == 0 {
			return total, err
		}
		if n == 0 {
			if err != nil {
				return total, err
			}
			return total, fmt.Errorf("embed queue not draining (%d items remain)", w.queueDepth())
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (w *Worker) queueDepth() int {
	n := 0
	for _, st := range w.Stores {
		if st == nil {
			continue
		}
		d, err := st.QueueDepth()
		if err == nil {
			n += d
		}
	}
	return n
}

func (w *Worker) drain(ctx context.Context, opts WorkerOptions) (int, error) {
	type job struct {
		st   *store.Store
		item store.QueueItem
	}
	var jobs []job
	for _, st := range w.Stores {
		if st == nil {
			continue
		}
		batch, err := st.DequeueEmbed(0)
		if err != nil {
			return 0, err
		}
		for _, item := range batch {
			jobs = append(jobs, job{st: st, item: item})
		}
	}
	if len(jobs) == 0 {
		return 0, nil
	}

	sem := make(chan struct{}, opts.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	done := 0
	var firstErr error

	for _, j := range jobs {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			ok, err := w.processOne(ctx, j.st, j.item)
			mu.Lock()
			defer mu.Unlock()
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if ok {
				done++
			}
		}(j)
	}
	wg.Wait()
	return done, firstErr
}

func (w *Worker) processOne(ctx context.Context, st *store.Store, item store.QueueItem) (bool, error) {
	rec, recStore, err := w.lookupRecord(item.RecordID, st)
	if err != nil {
		return false, err
	}
	if rec == nil {
		fmt.Fprintf(os.Stderr, "dropping embed queue row with no record: %s\n", item.RecordID)
		return false, st.DropQueueItem(item.RecordID)
	}
	text := rec.EmbedText()
	emb, err := w.Embedder.Embed(ctx, text)
	if err != nil {
		_ = recStore.FailQueueItem(item.RecordID, err.Error())
		if recStore != st {
			_ = st.FailQueueItem(item.RecordID, err.Error())
		}
		return false, fmt.Errorf("embed %s: %w", rec.SourcePath, err)
	}
	model := w.Model
	if model == "" {
		model = config.DefaultEmbedModel
	}
	if err := recStore.SetRecordVector(rec.ID, model, emb); err != nil {
		return false, err
	}
	if recStore != st {
		_ = st.DropQueueItem(item.RecordID)
	}
	return true, nil
}

func (w *Worker) lookupRecord(id string, prefer *store.Store) (*record.Record, *store.Store, error) {
	stores := make([]*store.Store, 0, len(w.Stores)+1)
	if prefer != nil {
		stores = append(stores, prefer)
	}
	for _, st := range w.Stores {
		if st == nil || st == prefer {
			continue
		}
		stores = append(stores, st)
	}
	for _, st := range stores {
		rec, ok, err := st.GetRecordByID(id)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			return rec, st, nil
		}
	}
	return nil, nil, nil
}
