package compress

import (
	"io"
	"sync"

	"github.com/klauspost/compress/flate"
)

// flateFreelist is an explicit free-list of *flate.Writer — C-style freelist,
// not sync.Pool.
//
// Semantics (closest pure-Go can get to malloc/free for these objects):
//
//	Get  → take from free list or NewWriter (malloc)
//	Put  → Reset(nil) + push free list (free for reuse; memory stays with process)
//	Drain→ drop free list (C free without munmap: GC may reclaim later when on)
//
// Why not sync.Pool: under GC-off it never drops anyway, but we want explicit
// Drain at job end and a hard max of free entries. Why not real free(): Go has
// no public runtime.free for arbitrary heap objects (runtime.freegc is experimental
// and not a user API). klauspost flate already owns its buffers; Reset reuses them.
//
// Bounds flate.Writer memory to O(threads) instead of O(files): one worker
// creates at most `max` live writers over the life of the job, reused via
// Reset for every file it compresses, instead of allocating (and leaking,
// under --no-gc) a fresh flate.Writer per file.
type flateFreelist struct {
	mu    sync.Mutex
	free  []*flate.Writer
	level int
	// max free entries retained. 0 = unlimited. Extra Put drops the ref (GC later).
	max int
}

func newFlateFreelist(level, maxFree int) *flateFreelist {
	return &flateFreelist{level: level, max: maxFree}
}

// get returns a writer bound to dst. Caller must Close via pooled wrapper then put.
func (p *flateFreelist) get(dst io.Writer) (*flate.Writer, error) {
	p.mu.Lock()
	if n := len(p.free); n > 0 {
		w := p.free[n-1]
		p.free[n-1] = nil
		p.free = p.free[:n-1]
		p.mu.Unlock()
		w.Reset(dst)
		return w, nil
	}
	p.mu.Unlock()
	return flate.NewWriter(dst, p.level)
}

// put returns w to the free list after detaching the underlying io.Writer.
// This is our "free" — object is live again for Get, not returned to the OS.
func (p *flateFreelist) put(w *flate.Writer) {
	if w == nil {
		return
	}
	// Drop reference to zip entry writer before storing.
	w.Reset(io.Discard)
	p.mu.Lock()
	if p.max > 0 && len(p.free) >= p.max {
		p.mu.Unlock()
		// Over capacity: drop ref. With GC on, reclaimed soon; with GC off,
		// only happens if callers Put more than max concurrent (should not).
		return
	}
	p.free = append(p.free, w)
	p.mu.Unlock()
}

// drain empties the free list (explicit "delete all free objects" from our
// control). Memory is eligible for GC only when the collector runs.
func (p *flateFreelist) drain() {
	p.mu.Lock()
	for i := range p.free {
		p.free[i] = nil
	}
	p.free = nil
	p.mu.Unlock()
}

func (p *flateFreelist) freeCount() int {
	p.mu.Lock()
	n := len(p.free)
	p.mu.Unlock()
	return n
}

// pooledFlateWriteCloser is what archive/zip RegisterCompressor must return.
// Close finalizes the deflate member, then freelist-Put the underlying Writer.
type pooledFlateWriteCloser struct {
	w    *flate.Writer
	pool *flateFreelist
	done bool
}

func (c *pooledFlateWriteCloser) Write(p []byte) (int, error) {
	return c.w.Write(p)
}

func (c *pooledFlateWriteCloser) Close() error {
	if c.done {
		return nil
	}
	c.done = true
	err := c.w.Close()
	c.pool.put(c.w)
	c.w = nil
	return err
}

// wrap gets a freelist writer and returns a zip-compatible WriteCloser.
func (p *flateFreelist) wrap(dst io.Writer) (io.WriteCloser, error) {
	w, err := p.get(dst)
	if err != nil {
		return nil, err
	}
	return &pooledFlateWriteCloser{w: w, pool: p}, nil
}

// flateLevelFromOptions maps go-zip level 1–9 to klauspost flate levels.
func flateLevelFromOptions(level int) int {
	if level <= 1 {
		return flate.NoCompression
	}
	return min(level-1, flate.BestCompression)
}
