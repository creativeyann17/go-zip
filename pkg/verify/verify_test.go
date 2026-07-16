package verify_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/creativeyann17/go-zip/pkg/compress"
	"github.com/creativeyann17/go-zip/pkg/verify"
)

// TestVerifyZIP tests verification of ZIP archives
func TestVerifyZIP(t *testing.T) {
	// Create test files
	sourceDir := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "test")

	files := map[string][]byte{
		"file1.txt":        []byte("hello world zip test"),
		"file2.txt":        []byte("more test data for zip"),
		"subdir/file3.txt": []byte("nested content in zip archive"),
	}

	for name, content := range files {
		path := filepath.Join(sourceDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	// Create ZIP archive (single-thread: predictable "test.zip" output path)
	compOpts := &compress.Options{
		InputPath:  sourceDir,
		OutputPath: archivePath,
		Level:      1, // Fast for tests
		MaxThreads: 1,
		Quiet:      true,
	}
	if _, err := compress.Compress(compOpts, nil); err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	actualPath := archivePath + ".zip"

	// Verify archive
	t.Run("StructuralValidation", func(t *testing.T) {
		opts := &verify.Options{
			InputPath:  actualPath,
			VerifyData: false,
		}

		result, err := verify.Verify(opts, nil)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		if !result.HeaderValid {
			t.Error("Header should be valid")
		}
		if !result.StructureValid {
			t.Error("Structure should be valid")
		}
		if result.FileCount != 3 {
			t.Errorf("Expected 3 files, got %d", result.FileCount)
		}
		if !result.IsValid() {
			t.Errorf("Archive should be valid, errors: %v", result.Errors)
		}
	})

	// Verify with data check
	t.Run("DataValidation", func(t *testing.T) {
		opts := &verify.Options{
			InputPath:  actualPath,
			VerifyData: true,
		}

		result, err := verify.Verify(opts, nil)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		if !result.DataVerified {
			t.Error("DataVerified should be true")
		}
		if result.FilesVerified != 3 {
			t.Errorf("Expected 3 files verified, got %d", result.FilesVerified)
		}
		if result.CorruptFiles != 0 {
			t.Errorf("Expected 0 corrupt files, got %d", result.CorruptFiles)
		}
	})
}

// TestVerifyInvalidArchive tests error handling for invalid archives
func TestVerifyInvalidArchive(t *testing.T) {
	// Create invalid archive
	invalidPath := filepath.Join(t.TempDir(), "invalid.zip")
	if err := os.WriteFile(invalidPath, []byte("INVALIDMAGIC"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	opts := &verify.Options{
		InputPath: invalidPath,
	}

	result, err := verify.Verify(opts, nil)
	if err == nil {
		t.Error("Expected error for invalid archive")
	}
	if result.IsValid() {
		t.Error("Invalid archive should not be valid")
	}
}

// TestVerifyNonExistent tests error handling for non-existent files
func TestVerifyNonExistent(t *testing.T) {
	opts := &verify.Options{
		InputPath: "/nonexistent/archive.zip",
	}

	_, err := verify.Verify(opts, nil)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestResultMethods tests Result helper methods
func TestResultMethods(t *testing.T) {
	t.Run("CompressionRatio", func(t *testing.T) {
		r := &verify.Result{
			TotalOrigSize: 1000,
			TotalCompSize: 500,
		}
		ratio := r.CompressionRatio()
		if ratio != 50.0 {
			t.Errorf("Expected ratio 50%%, got %.1f%%", ratio)
		}
	})

	t.Run("SpaceSaved", func(t *testing.T) {
		r := &verify.Result{
			TotalOrigSize: 1000,
			TotalCompSize: 600,
		}
		saved := r.SpaceSaved()
		if saved != 400 {
			t.Errorf("Expected 400 bytes saved, got %d", saved)
		}
	})

	t.Run("IsValid", func(t *testing.T) {
		validResult := &verify.Result{
			HeaderValid:    true,
			StructureValid: true,
			FooterValid:    true,
		}
		if !validResult.IsValid() {
			t.Error("Result should be valid")
		}

		invalidResult := &verify.Result{
			HeaderValid:    true,
			StructureValid: false,
			FooterValid:    true,
		}
		if invalidResult.IsValid() {
			t.Error("Result should be invalid")
		}
	})
}

// TestOptionsValidate tests option validation
func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    *verify.Options
		wantErr error
	}{
		{
			name: "valid options",
			opts: &verify.Options{
				InputPath:  "test.zip",
				VerifyData: false,
			},
			wantErr: nil,
		},
		{
			name: "missing input path",
			opts: &verify.Options{
				VerifyData: true,
			},
			wantErr: verify.ErrInputRequired,
		},
		{
			name: "quiet overrides verbose",
			opts: &verify.Options{
				InputPath: "test.zip",
				Verbose:   true,
				Quiet:     true,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.opts.Quiet && tt.opts.Verbose {
				t.Error("Quiet should override Verbose")
			}
		})
	}
}

// TestProgressCallbacks tests progress callback invocation
func TestProgressCallbacks(t *testing.T) {
	// Create test archive
	sourceDir := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "test")

	files := map[string][]byte{
		"file1.txt": []byte("content 1"),
		"file2.txt": []byte("content 2"),
	}

	for name, content := range files {
		path := filepath.Join(sourceDir, name)
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatal(err)
		}
	}

	compOpts := &compress.Options{
		InputPath:  sourceDir,
		OutputPath: archivePath,
		MaxThreads: 1,
		Quiet:      true,
	}
	if _, err := compress.Compress(compOpts, nil); err != nil {
		t.Fatal(err)
	}

	// Track events
	var events []verify.EventType
	progressCb := func(event verify.ProgressEvent) {
		events = append(events, event.Type)
	}

	opts := &verify.Options{
		InputPath:  archivePath + ".zip",
		VerifyData: false,
	}

	if _, err := verify.Verify(opts, progressCb); err != nil {
		t.Fatal(err)
	}

	if len(events) == 0 {
		t.Error("No progress events received")
	}

	// Check for required events
	hasStart := false
	hasComplete := false
	for _, e := range events {
		if e == verify.EventStart {
			hasStart = true
		}
		if e == verify.EventComplete {
			hasComplete = true
		}
	}

	if !hasStart {
		t.Error("Missing EventStart")
	}
	if !hasComplete {
		t.Error("Missing EventComplete")
	}
}

// TestVerifyTruncatedArchive tests handling of a ZIP that has a valid magic
// but no usable structure (truncated mid-write, e.g. after a crash).
func TestVerifyTruncatedArchive(t *testing.T) {
	truncatedPath := filepath.Join(t.TempDir(), "truncated.zip")

	// Valid local-file-header signature, nothing after it — no central directory.
	if err := os.WriteFile(truncatedPath, []byte("PK\x03\x04"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &verify.Options{
		InputPath: truncatedPath,
	}

	result, err := verify.Verify(opts, nil)
	if err != nil {
		t.Fatalf("Verify itself should not error (failure is recorded in result): %v", err)
	}
	if result.HeaderValid {
		t.Error("HeaderValid should be false for an unreadable ZIP structure")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected an error recorded for the truncated archive")
	}
	if result.IsValid() {
		t.Error("Truncated archive should not be valid")
	}
}

// TestVerifyCorruptedData tests detection of corrupted compressed data
func TestVerifyCorruptedData(t *testing.T) {
	// Create valid archive with larger content to ensure corruption is in compressed data
	sourceDir := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "test")

	testFile := filepath.Join(sourceDir, "test.txt")
	// Use more content to ensure we have substantial compressed data
	content := bytes.Repeat([]byte("Test content for corruption detection\n"), 100)
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	compOpts := &compress.Options{
		InputPath:  sourceDir,
		OutputPath: archivePath,
		MaxThreads: 1,
		Quiet:      true,
	}

	if _, err := compress.Compress(compOpts, nil); err != nil {
		t.Fatal(err)
	}

	// Read and corrupt the archive
	data, err := os.ReadFile(archivePath + ".zip")
	if err != nil {
		t.Fatal(err)
	}

	// Corrupt multiple bytes in the last third (likely compressed data section)
	corruptStart := (len(data) * 2) / 3
	if len(data) > corruptStart+10 {
		for i := 0; i < 10; i++ {
			data[corruptStart+i] ^= 0xFF
		}
	}

	corruptedPath := filepath.Join(t.TempDir(), "corrupted.zip")
	if err := os.WriteFile(corruptedPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// First verify without data check (should still be structurally valid)
	opts := &verify.Options{
		InputPath:  corruptedPath,
		VerifyData: false,
	}

	_, err = verify.Verify(opts, nil)
	if err != nil {
		t.Logf("Structural verification error: %v", err)
	}

	// Now verify with data checking - this should detect corruption
	opts.VerifyData = true
	result, err := verify.Verify(opts, nil)

	// The verification should detect the issue
	if result == nil {
		t.Fatal("Expected result to be returned even on corruption")
	}

	t.Logf("Verification result: err=%v, CorruptFiles=%d, Errors=%v, IsValid=%v",
		err, result.CorruptFiles, len(result.Errors), result.IsValid())

	// Should detect the issue - accept either corrupt files or errors
	if result.CorruptFiles == 0 && len(result.Errors) == 0 {
		t.Error("Should detect corrupted data through CorruptFiles or Errors")
	}

	if result.IsValid() {
		t.Error("Corrupted archive should not be valid")
	}
}

// TestVerifyEmptyFiles tests handling of empty files
func TestVerifyEmptyFiles(t *testing.T) {
	sourceDir := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "test")

	// Create mix of empty and non-empty files
	files := map[string][]byte{
		"empty1.txt":   {},
		"empty2.txt":   {},
		"nonempty.txt": []byte("content"),
	}

	for name, content := range files {
		path := filepath.Join(sourceDir, name)
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatal(err)
		}
	}

	compOpts := &compress.Options{
		InputPath:  sourceDir,
		OutputPath: archivePath,
		MaxThreads: 1,
		Quiet:      true,
	}

	if _, err := compress.Compress(compOpts, nil); err != nil {
		t.Fatal(err)
	}

	opts := &verify.Options{
		InputPath: archivePath + ".zip",
	}

	result, err := verify.Verify(opts, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.EmptyFiles != 2 {
		t.Errorf("Expected 2 empty files, got %d", result.EmptyFiles)
	}
}

// TestVerifyDuplicateDetection tests duplicate path detection
func TestVerifyDuplicateDetection(t *testing.T) {
	// Note: This test would require manually creating an archive with duplicates
	// since the compress package doesn't allow duplicates
	t.Skip("Duplicate detection requires manual archive creation")
}

// TestResultSummary tests the Summary output formatting
func TestResultSummary(t *testing.T) {
	result := &verify.Result{
		ArchivePath:    "/test/archive.zip",
		ArchiveSize:    1024 * 1024,
		FileCount:      10,
		TotalOrigSize:  2 * 1024 * 1024,
		TotalCompSize:  1 * 1024 * 1024,
		HeaderValid:    true,
		FooterValid:    true,
		StructureValid: true,
		MetadataValid:  true,
	}

	summary := result.Summary()
	if summary == "" {
		t.Error("Summary should not be empty")
	}

	// Verify key information is present
	requiredStrings := []string{
		"archive.zip",
		"ZIP",
		"VALID",
	}

	for _, s := range requiredStrings {
		if !contains(summary, s) {
			t.Errorf("Summary missing: %s\nGot: %s", s, summary)
		}
	}
}

// TestResultMetricsEdgeCases tests edge cases in calculations
func TestResultMetricsEdgeCases(t *testing.T) {
	// Zero values
	result := &verify.Result{}

	if result.CompressionRatio() != 0 {
		t.Error("Empty result should have 0 compression ratio")
	}

	if result.SpaceSaved() != 0 {
		t.Error("Empty result should have 0 space saved")
	}

	// Compressed larger than original
	result.TotalOrigSize = 100
	result.TotalCompSize = 150
	if result.SpaceSaved() != 0 {
		t.Error("Negative savings should return 0")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BenchmarkVerifyStructural benchmarks structural verification
func BenchmarkVerifyStructural(b *testing.B) {
	// Create test archive
	sourceDir := b.TempDir()
	archivePath := filepath.Join(b.TempDir(), "bench")

	// Create 50 files
	for i := 0; i < 50; i++ {
		content := bytes.Repeat([]byte("test content\n"), 100)
		path := filepath.Join(sourceDir, fmt.Sprintf("file%03d.txt", i))
		if err := os.WriteFile(path, content, 0644); err != nil {
			b.Fatal(err)
		}
	}

	compOpts := &compress.Options{
		InputPath:  sourceDir,
		OutputPath: archivePath,
		MaxThreads: 1,
		Quiet:      true,
	}

	if _, err := compress.Compress(compOpts, nil); err != nil {
		b.Fatal(err)
	}

	opts := &verify.Options{
		InputPath:  archivePath + ".zip",
		VerifyData: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := verify.Verify(opts, nil); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVerifyData benchmarks full data verification
func BenchmarkVerifyData(b *testing.B) {
	sourceDir := b.TempDir()
	archivePath := filepath.Join(b.TempDir(), "bench")

	// Create 30 files with varied content
	for i := 0; i < 30; i++ {
		content := bytes.Repeat([]byte("benchmark content\n"), 500)
		path := filepath.Join(sourceDir, fmt.Sprintf("file%03d.txt", i))
		if err := os.WriteFile(path, content, 0644); err != nil {
			b.Fatal(err)
		}
	}

	compOpts := &compress.Options{
		InputPath:  sourceDir,
		OutputPath: archivePath,
		MaxThreads: 1,
		Quiet:      true,
	}

	if _, err := compress.Compress(compOpts, nil); err != nil {
		b.Fatal(err)
	}

	opts := &verify.Options{
		InputPath:  archivePath + ".zip",
		VerifyData: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := verify.Verify(opts, nil); err != nil {
			b.Fatal(err)
		}
	}
}
