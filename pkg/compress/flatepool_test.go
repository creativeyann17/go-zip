package compress

import (
	"bytes"
	"io"
	"testing"

	"github.com/klauspost/compress/flate"
)

func TestFlateFreelistReuse(t *testing.T) {
	pool := newFlateFreelist(flate.DefaultCompression, 2)

	var buf1 bytes.Buffer
	w1, err := pool.get(&buf1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w1.Write([]byte("hello freelist")); err != nil {
		t.Fatal(err)
	}
	if err := w1.Close(); err != nil {
		t.Fatal(err)
	}
	pool.put(w1)
	if pool.freeCount() != 1 {
		t.Fatalf("freeCount after put: %d", pool.freeCount())
	}

	var buf2 bytes.Buffer
	w2, err := pool.get(&buf2)
	if err != nil {
		t.Fatal(err)
	}
	// Same object reused (pointer equality)
	if w2 != w1 {
		t.Fatal("expected same *flate.Writer from freelist")
	}
	if pool.freeCount() != 0 {
		t.Fatalf("free list should be empty while checked out: %d", pool.freeCount())
	}
	if _, err := w2.Write([]byte("second use")); err != nil {
		t.Fatal(err)
	}
	if err := w2.Close(); err != nil {
		t.Fatal(err)
	}
	pool.put(w2)

	pool.drain()
	if pool.freeCount() != 0 {
		t.Fatalf("drain should empty free list: %d", pool.freeCount())
	}
}

func TestFlateFreelistMaxCapDropsExtra(t *testing.T) {
	pool := newFlateFreelist(flate.HuffmanOnly, 1)

	makeOne := func() *flate.Writer {
		var buf bytes.Buffer
		w, err := flate.NewWriter(&buf, flate.HuffmanOnly)
		if err != nil {
			t.Fatal(err)
		}
		_ = w.Close()
		return w
	}
	a, b := makeOne(), makeOne()
	pool.put(a)
	pool.put(b) // over max → dropped
	if pool.freeCount() != 1 {
		t.Fatalf("max=1 freeCount=%d", pool.freeCount())
	}
}

func TestPooledFlateWriteCloserRoundTrip(t *testing.T) {
	pool := newFlateFreelist(5, 1)
	var compressed bytes.Buffer
	wc, err := pool.wrap(&compressed)
	if err != nil {
		t.Fatal(err)
	}
	payload := bytes.Repeat([]byte("abc"), 1000)
	if _, err := wc.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := wc.Close(); err != nil {
		t.Fatal(err)
	}
	if pool.freeCount() != 1 {
		t.Fatalf("after Close, writer should be free: %d", pool.freeCount())
	}

	// Second member reuses freelist entry
	var compressed2 bytes.Buffer
	wc2, err := pool.wrap(&compressed2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wc2.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := wc2.Close(); err != nil {
		t.Fatal(err)
	}

	// Decompress first stream
	r := flate.NewReader(bytes.NewReader(compressed.Bytes()))
	out, err := io.ReadAll(r)
	_ = r.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, payload) {
		t.Fatalf("round-trip mismatch len=%d", len(out))
	}
}

func TestFlateLevelFromOptions(t *testing.T) {
	if flateLevelFromOptions(1) != flate.NoCompression {
		t.Fatal("level 1")
	}
	if flateLevelFromOptions(5) != 4 {
		t.Fatal("level 5 → flate 4")
	}
	// go-zip 9 → klauspost 8 (level-1); BestCompression is 9
	if flateLevelFromOptions(9) != 8 {
		t.Fatalf("level 9 → %d", flateLevelFromOptions(9))
	}
}
