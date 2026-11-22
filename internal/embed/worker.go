package embed

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/index"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

// maxAttempts bounds how many times a single record may fail to embed before
// it is dropped from the queue. Without it one permanently bad record makes
// every later `archivist embed` run fail.
const maxAttempts = 5

type Worker struct {
	Stores   []*store.Store
	Embedder Embedder
	Model    string
}

type WorkerOptions struct {
	Once        bool
	Concurrency int
}

func (w *Worker) resolvedModel() string {
	if strings.TrimSpace(w.Model) == "" {
		return config.DefaultEmbedModel
	}
	return w.Model
}

func (w *Worker) requeueMismatches(ctx context.Context) error {
	if w.Embedder == nil {
		return fmt.Errorf("embed worker has no embedder")
	}
	probe, err := w.Embedder.Embed(ctx, "archivist")
	if err != nil {
		return err
	}
	model := w.resolvedModel()
	for _, st := range w.Stores {
		if st == nil {
			continue
		}
		if err := st.RequeueVectorMismatches(model, len(probe)); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) Run(ctx context.Context, opts WorkerOptions) (int, error) {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 2
	}
	if err := w.requeueMismatches(ctx); err != nil {
		return 0, err
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
		if err != nil {
			continue
		}
		n += d
		fd, err := st.FileQueueDepth()
		if err != nil {
			continue
		}
		n += fd
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
		return w.drainFiles(ctx)
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
	fn, ferr := w.drainFiles(ctx)
	if firstErr == nil {
		firstErr = ferr
	}
	return done + fn, firstErr
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
	textHash := rec.ContentHash
	if item.TextHash != "" && item.TextHash != textHash {
		return false, nil
	}
	emb, err := w.Embedder.Embed(ctx, rec.EmbedText())
	if err != nil {
		return w.failRecord(st, recStore, item, rec, err)
	}
	committed, err := recStore.CommitRecordVector(rec.ID, w.resolvedModel(), textHash, emb)
	if err != nil {
		return false, err
	}
	if !committed {
		return false, nil
	}
	if recStore != st {
		if err := st.DropQueueItemHash(item.RecordID, textHash); err != nil {
			return false, err
		}
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

func (w *Worker) drainFiles(ctx context.Context) (int, error) {
	var n int
	var first error
	for _, st := range w.Stores {
		if st == nil {
			continue
		}
		items, err := st.DequeueFileEmbed()
		if err != nil {
			return n, err
		}
		for _, item := range items {
			ok, err := w.processFile(ctx, st, item)
			if err != nil && first == nil {
				first = err
			}
			if ok {
				n++
			}
		}
	}
	return n, first
}

func (w *Worker) processFile(ctx context.Context, st *store.Store, item store.FileQueueItem) (bool, error) {
	syms, err := st.SymbolsForFile(item.Checkout, item.Path)
	if err != nil {
		return false, err
	}
	text := index.FileEmbedText(item.Path, syms)
	if text == "" {
		return false, st.DropFileEmbed(item.Checkout, item.Path)
	}
	hash := index.FileEmbedHash(text)
	if item.TextHash != "" && item.TextHash != hash {
		return false, nil
	}
	emb, err := w.Embedder.Embed(ctx, text)
	if err != nil {
		return w.failFile(st, item, hash, err)
	}
	committed, err := st.CommitFileVector(item.Checkout, item.Path, w.resolvedModel(), hash, emb)
	if err != nil {
		return false, err
	}
	return committed, nil
}

func (w *Worker) failRecord(st, recStore *store.Store, item store.QueueItem, rec *record.Record, embedErr error) (bool, error) {
	live, ok, err := st.EmbedQueueItem(item.RecordID)
	if err != nil {
		return false, err
	}
	if item.TextHash != "" && (!ok || live.TextHash != item.TextHash) {
		return false, nil
	}
	attempts := item.Attempts
	if ok {
		attempts = live.Attempts
	}
	if attempts+1 < maxAttempts {
		if ferr := recStore.FailQueueItem(item.RecordID, embedErr.Error()); ferr != nil {
			return false, ferr
		}
		if recStore != st {
			if ferr := st.FailQueueItem(item.RecordID, embedErr.Error()); ferr != nil {
				return false, ferr
			}
		}
		return false, fmt.Errorf("embed %s: %w", rec.SourcePath, embedErr)
	}
	fmt.Fprintf(os.Stderr, "giving up on %s after %d attempts: %v\n", rec.SourcePath, attempts+1, embedErr)
	if gerr := recStore.RecordGiveup(rec.ID, attempts+1, embedErr.Error()); gerr != nil {
		return false, gerr
	}
	drop := func(db *store.Store) error {
		if item.TextHash != "" {
			return db.DropQueueItemHash(item.RecordID, item.TextHash)
		}
		return db.DropQueueItem(item.RecordID)
	}
	if gerr := drop(recStore); gerr != nil {
		return false, gerr
	}
	if recStore != st {
		return false, drop(st)
	}
	return false, nil
}

func (w *Worker) failFile(st *store.Store, item store.FileQueueItem, hash string, embedErr error) (bool, error) {
	live, ok, err := st.FileEmbedQueueItem(item.Checkout, item.Path)
	if err != nil {
		return false, err
	}
	snapshot := item.TextHash
	if snapshot == "" {
		snapshot = hash
	}
	if snapshot != "" && (!ok || live.TextHash != snapshot) {
		return false, nil
	}
	attempts := item.Attempts
	if ok {
		attempts = live.Attempts
	}
	if attempts+1 < maxAttempts {
		if ferr := st.FailFileQueueItem(item.Checkout, item.Path, embedErr.Error()); ferr != nil {
			return false, ferr
		}
		return false, fmt.Errorf("embed %s: %w", item.Path, embedErr)
	}
	fmt.Fprintf(os.Stderr, "giving up on %s after %d attempts: %v\n", item.Path, attempts+1, embedErr)
	if gerr := st.RecordFileGiveup(item.Checkout, item.Path, attempts+1, embedErr.Error()); gerr != nil {
		return false, gerr
	}
	if snapshot != "" {
		return false, st.DropFileQueueItemHash(item.Checkout, item.Path, snapshot)
	}
	return false, st.DropFileQueueItem(item.Checkout, item.Path)
}
