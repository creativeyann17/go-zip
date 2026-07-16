package compress_test

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/creativeyann17/go-zip/pkg/compress"
	"github.com/creativeyann17/go-zip/pkg/decompress"
	"github.com/creativeyann17/go-zip/pkg/verify"
)

func writeTree(t *testing.T, inputDir string, files map[string]string) map[string]string {
	t.Helper()
	hashes := make(map[string]string)
	for relPath, content := range files {
		fullPath := filepath.Join(inputDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
		hash := md5.Sum([]byte(content))
		hashes[relPath] = fmt.Sprintf("%x", hash)
	}
	return hashes
}

func TestCompressSingleThreadNaming(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}
	_ = writeTree(t, inputDir, map[string]string{
		"a.txt": "hello single thread",
		"b.txt": "more content here",
	})

	outBase := filepath.Join(tempDir, "out")
	result, err := compress.Compress(&compress.Options{
		InputPath:  inputDir,
		OutputPath: outBase,
		MaxThreads: 1,
		Level:      5,
		Quiet:      true,
	}, nil)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	if result.FilesProcessed != 2 {
		t.Fatalf("files: got %d", result.FilesProcessed)
	}

	single := outBase + ".zip"
	if _, err := os.Stat(single); err != nil {
		t.Fatalf("expected %s: %v", single, err)
	}
	if _, err := os.Stat(outBase + "_01.zip"); err == nil {
		t.Fatal("must not create _01.zip when MaxThreads=1")
	}

	// Round-trip
	extractDir := filepath.Join(tempDir, "extract")
	_, err = decompress.Decompress(&decompress.Options{
		InputPath:  single,
		OutputPath: extractDir,
		MaxThreads: 1,
		Overwrite:  true,
		Quiet:      true,
	}, nil)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(extractDir, "a.txt"))
	if err != nil || string(data) != "hello single thread" {
		t.Fatalf("round-trip a.txt: %v %q", err, data)
	}

	vresult, err := verify.Verify(&verify.Options{InputPath: single, VerifyData: true, Quiet: true}, nil)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !vresult.IsValid() {
		t.Fatalf("verify invalid: %s", vresult.Summary())
	}
}

func TestCompressMultiPartNaming(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for i := 0; i < 20; i++ {
		files[fmt.Sprintf("f%02d.txt", i)] = fmt.Sprintf("content for file %d with padding....\n", i)
	}
	hashes := writeTree(t, inputDir, files)

	outBase := filepath.Join(tempDir, "multi")
	result, err := compress.Compress(&compress.Options{
		InputPath:  inputDir,
		OutputPath: outBase + ".zip",
		MaxThreads: 4,
		Level:      6,
		Quiet:      true,
	}, nil)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	if result.FilesProcessed != len(files) {
		t.Fatalf("files: got %d want %d", result.FilesProcessed, len(files))
	}

	if _, err := os.Stat(outBase + ".zip"); err == nil {
		t.Fatal("must not create plain multi.zip when MaxThreads>1")
	}
	first := outBase + "_01.zip"
	if _, err := os.Stat(first); err != nil {
		t.Fatalf("expected multi-part %s: %v", first, err)
	}

	extractDir := filepath.Join(tempDir, "extract")
	dresult, err := decompress.Decompress(&decompress.Options{
		InputPath:  first,
		OutputPath: extractDir,
		MaxThreads: 4,
		Overwrite:  true,
		Quiet:      true,
	}, nil)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	if dresult.FilesProcessed != len(files) {
		t.Fatalf("decompressed %d want %d (errors: %v)", dresult.FilesProcessed, len(files), dresult.Errors)
	}

	for rel, wantHash := range hashes {
		data, err := os.ReadFile(filepath.Join(extractDir, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		got := fmt.Sprintf("%x", md5.Sum(data))
		if got != wantHash {
			t.Errorf("hash mismatch %s", rel)
		}
	}
}

func TestCompressDryRun(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	_ = os.MkdirAll(inputDir, 0755)
	_ = os.WriteFile(filepath.Join(inputDir, "x.txt"), []byte("dry"), 0644)

	outBase := filepath.Join(tempDir, "dry")
	_, err := compress.Compress(&compress.Options{
		InputPath:  inputDir,
		OutputPath: outBase,
		MaxThreads: 1,
		DryRun:     true,
		Quiet:      true,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outBase + ".zip"); err == nil {
		t.Fatal("dry-run must not write archive")
	}
}

func TestCompressLevels(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	_ = os.MkdirAll(inputDir, 0755)
	// compressible content
	content := make([]byte, 4096)
	for i := range content {
		content[i] = byte('A' + i%3)
	}
	_ = os.WriteFile(filepath.Join(inputDir, "data.bin"), content, 0644)

	for _, level := range []int{1, 5, 9} {
		out := filepath.Join(tempDir, fmt.Sprintf("lvl%d", level))
		_, err := compress.Compress(&compress.Options{
			InputPath:  inputDir,
			OutputPath: out,
			MaxThreads: 1,
			Level:      level,
			Quiet:      true,
		}, nil)
		if err != nil {
			t.Fatalf("level %d: %v", level, err)
		}
	}
}

func TestInvalidLevel(t *testing.T) {
	opts := &compress.Options{
		InputPath:  "/tmp",
		OutputPath: "x",
		Level:      15,
	}
	if err := opts.Validate(); err != compress.ErrInvalidLevel {
		t.Fatalf("want ErrInvalidLevel, got %v", err)
	}
}
