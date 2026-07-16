package compress

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

// noGCMemFraction is the fraction of available RAM the heap may grow into
// before the soft memory limit forces a GC pass. Leaves headroom for the OS,
// page cache, and other processes.
const noGCMemFraction = 0.7

// minSafeMemCap: below this, capping isn't worth it — GC would thrash near
// the ceiling instead of giving any real latency benefit.
const minSafeMemCap = 256 << 20 // 256 MiB

// availableRAMBytes reads MemAvailable from /proc/meminfo (Linux).
// Falls back to a conservative fraction of MemTotal if MemAvailable is missing.
func availableRAMBytes() (uint64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	var memAvailable, memTotal, memFree, buffers, cached uint64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		v, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		v *= 1024 // kB → bytes
		switch fields[0] {
		case "MemAvailable:":
			memAvailable = v
		case "MemTotal:":
			memTotal = v
		case "MemFree:":
			memFree = v
		case "Buffers:":
			buffers = v
		case "Cached:":
			cached = v
		}
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	if memAvailable > 0 {
		return memAvailable, nil
	}
	if approx := memFree + buffers + cached; approx > 0 {
		return approx, nil
	}
	if memTotal > 0 {
		return memTotal / 2, nil
	}
	return 0, fmt.Errorf("could not parse /proc/meminfo")
}

// enableLowLatencyGC configures the runtime for --no-gc. Percentage-based GC
// is turned off for fewer, less frequent pauses, but a soft memory limit
// (debug.SetMemoryLimit) is always set as a hard backstop, sized from
// currently available RAM. Unlike disabling GC outright, the runtime still
// forces a GC pass before the process grows past the cap — that's what
// actually prevents --no-gc from swap-thrashing or OOM-killing the machine,
// instead of relying on a one-shot pre-flight guess of job memory use.
//
// If available RAM can't be determined (non-Linux, or /proc/meminfo
// unreadable) or is too low to set a meaningful cap, GC is left at its
// default percent instead of being disabled — there would be no safe way to
// bound growth. Returns a restore func to run via defer.
func enableLowLatencyGC(quiet bool) func() {
	oldLimit := debug.SetMemoryLimit(-1) // read current without changing it

	avail, err := availableRAMBytes()
	if err != nil {
		if !quiet {
			fmt.Fprintf(os.Stderr, "warning: --no-gc: could not read available RAM (%v); keeping default GC (no safe cap to set)\n", err)
		}
		return func() {}
	}

	memCap := int64(float64(avail) * noGCMemFraction)
	if oldLimit != math.MaxInt64 && oldLimit < memCap {
		memCap = oldLimit // respect a tighter GOMEMLIMIT the user already set
	}
	if memCap < minSafeMemCap {
		if !quiet {
			fmt.Fprintf(os.Stderr, "warning: --no-gc: only %s available, too low for a safe memory cap; keeping default GC\n", formatBytes(avail))
		}
		return func() {}
	}

	runtime.GC()
	oldPercent := debug.SetGCPercent(-1)
	debug.SetMemoryLimit(memCap)

	if !quiet {
		fmt.Fprintf(os.Stderr, "info: --no-gc: GC percent disabled, soft memory cap set to %s\n", formatBytes(uint64(memCap)))
	}

	return func() {
		debug.SetGCPercent(oldPercent)
		debug.SetMemoryLimit(oldLimit)
		runtime.GC()
		debug.FreeOSMemory()
	}
}

func formatBytes(b uint64) string {
	const (
		ki = 1 << 10
		mi = 1 << 20
		gi = 1 << 30
	)
	switch {
	case b >= gi:
		return fmt.Sprintf("%.2f GiB", float64(b)/float64(gi))
	case b >= mi:
		return fmt.Sprintf("%.1f MiB", float64(b)/float64(mi))
	case b >= ki:
		return fmt.Sprintf("%.1f KiB", float64(b)/float64(ki))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
