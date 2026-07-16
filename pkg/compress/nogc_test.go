package compress

import (
	"runtime/debug"
	"testing"
)

func TestAvailableRAMBytes(t *testing.T) {
	n, err := availableRAMBytes()
	if err != nil {
		t.Skipf("/proc/meminfo not available: %v", err)
	}
	if n < 1<<20 {
		t.Fatalf("available RAM suspiciously low: %d", n)
	}
}

func TestFormatBytes(t *testing.T) {
	if got := formatBytes(1536); got != "1.5 KiB" {
		t.Fatalf("got %q", got)
	}
	if got := formatBytes(3 << 30); got != "3.00 GiB" {
		t.Fatalf("got %q", got)
	}
}

// TestEnableLowLatencyGCSetsAndRestoresCap checks the mechanism itself
// (memory limit lowered while active, restored after) without asserting
// anything about the host's actual available RAM — a fixed RAM-threshold
// assertion here would be flaky across machines/CI runners.
func TestEnableLowLatencyGCSetsAndRestoresCap(t *testing.T) {
	if _, err := availableRAMBytes(); err != nil {
		t.Skipf("/proc/meminfo not available: %v", err)
	}

	baseline := debug.SetMemoryLimit(-1)
	restore := enableLowLatencyGC(true)
	defer restore()

	active := debug.SetMemoryLimit(-1)
	if active == baseline {
		t.Skip("host free RAM too low for a safe cap (below minSafeMemCap) — nothing to assert")
	}
	if active <= 0 {
		t.Fatalf("expected a positive memory cap, got %d", active)
	}

	restore()
	restored := debug.SetMemoryLimit(-1)
	if restored != baseline {
		t.Fatalf("restore did not put back the original memory limit: baseline=%d restored=%d", baseline, restored)
	}
}

// TestEnableLowLatencyGCRespectsExistingLowerLimit verifies a tighter
// GOMEMLIMIT the user already configured is never raised by --no-gc.
func TestEnableLowLatencyGCRespectsExistingLowerLimit(t *testing.T) {
	if _, err := availableRAMBytes(); err != nil {
		t.Skipf("/proc/meminfo not available: %v", err)
	}

	const tinyButSafe = minSafeMemCap // exactly at the floor, still "safe"
	baseline := debug.SetMemoryLimit(tinyButSafe)
	defer debug.SetMemoryLimit(baseline)

	restore := enableLowLatencyGC(true)
	defer restore()

	active := debug.SetMemoryLimit(-1)
	if active > tinyButSafe {
		t.Fatalf("expected cap to stay at or below the pre-existing GOMEMLIMIT %d, got %d", tinyButSafe, active)
	}
}
