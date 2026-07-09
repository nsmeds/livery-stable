package server

import (
	"context"
	"log"
	"sync"

	"github.com/nsmeds/livery-stable/storage"
)

// Deleter asynchronously removes objects from storage after a file's DB row
// has already been soft-deleted. The DB is the source of truth for what's
// deleted; storage cleanup here is best-effort and only logged on failure.
// Enqueued jobs are in-memory only, so a process restart before a queued
// delete runs leaves the object orphaned in storage - acceptable for MVP.
type Deleter struct {
	store storage.Store
	jobs  chan string
	wg    sync.WaitGroup
}

func NewDeleter(store storage.Store) *Deleter {
	d := &Deleter{store: store, jobs: make(chan string, 64)}
	go d.run()
	return d
}

func (d *Deleter) run() {
	for key := range d.jobs {
		if err := d.store.Delete(context.Background(), key); err != nil {
			log.Printf("deleter: failed to delete storage key %q: %v", key, err)
		}
		d.wg.Done()
	}
}

// Enqueue schedules a storage key for asynchronous deletion.
func (d *Deleter) Enqueue(key string) {
	d.wg.Add(1)
	d.jobs <- key
}

// Wait blocks until all enqueued deletes have completed. Intended for use
// during graceful shutdown so in-flight deletes aren't silently dropped.
func (d *Deleter) Wait() {
	d.wg.Wait()
}
