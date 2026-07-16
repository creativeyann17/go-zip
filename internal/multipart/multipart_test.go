package multipart

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsMultiPartName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"archive.zip", false},
		{"archive_01.zip", true},
		{"archive_99.zip", true},
		{"foo_bar.zip", false},
		{"foo_1.zip", false},
		{"foo_001.zip", false},
		{"backup_data_02.zip", true},
		{"nozip", false},
	}
	for _, tc := range cases {
		if got := IsMultiPartName(tc.name); got != tc.want {
			t.Errorf("IsMultiPartName(%q)=%v want %v", tc.name, got, tc.want)
		}
	}
}

func TestOutputPath(t *testing.T) {
	if got := OutputPath("backup", false, 1); got != "backup.zip" {
		t.Errorf("single: got %q", got)
	}
	if got := OutputPath("backup", true, 1); got != "backup_01.zip" {
		t.Errorf("multi part1: got %q", got)
	}
	if got := OutputPath("backup", true, 3); got != "backup_03.zip" {
		t.Errorf("multi part3: got %q", got)
	}
}

func TestDiscoverParts_Single(t *testing.T) {
	dir := t.TempDir()
	single := filepath.Join(dir, "backup.zip")
	// Sibling multi-part must NOT be merged when input is plain backup.zip
	_ = os.WriteFile(single, []byte("PK"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "backup_01.zip"), []byte("PK"), 0644)

	parts, err := DiscoverParts(single)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 1 || parts[0] != single {
		t.Fatalf("want only single path, got %v", parts)
	}
}

func TestDiscoverParts_Multi(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "set_01.zip")
	p2 := filepath.Join(dir, "set_02.zip")
	_ = os.WriteFile(p1, []byte("PK"), 0644)
	_ = os.WriteFile(p2, []byte("PK"), 0644)

	parts, err := DiscoverParts(p1)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 2 {
		t.Fatalf("want 2 parts, got %v", parts)
	}
}

func TestResolveInputPath(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "data")
	_ = os.WriteFile(base+".zip", []byte("PK"), 0644)

	got := ResolveInputPath(base)
	if got != base+".zip" {
		t.Errorf("prefer .zip: got %q", got)
	}

	// Only multi exists
	base2 := filepath.Join(dir, "onlymulti")
	_ = os.WriteFile(base2+"_01.zip", []byte("PK"), 0644)
	got = ResolveInputPath(base2)
	if got != base2+"_01.zip" {
		t.Errorf("fallback _01: got %q", got)
	}
}
