package graph

import (
	"context"
	"time"
)

// Watcher runs background incremental indexing loop.
type Watcher struct {
	indexer  *Indexer
	interval time.Duration
}

// NewWatcher creates a watcher instance.
func NewWatcher(indexer *Indexer, interval time.Duration) *Watcher {
	if interval <= 0 {
		interval = 1 * time.Second
	}
	return &Watcher{
		indexer:  indexer,
		interval: interval,
	}
}

// Watch runs until context is canceled.
func (w *Watcher) Watch(ctx context.Context, onChange func(prog *IndexProgress)) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			prog, err := w.indexer.IndexRepository()
			if err == nil && prog != nil && prog.IndexedFiles > 0 {
				if onChange != nil {
					onChange(prog)
				}
			}
		}
	}
}
