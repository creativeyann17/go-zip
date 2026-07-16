# go-zip

[![Test and Release](https://github.com/creativeyann17/go-zip/actions/workflows/test-and-release.yml/badge.svg)](https://github.com/creativeyann17/go-zip/actions/workflows/test-and-release.yml)
[![GitHub release](https://img.shields.io/github/v/release/creativeyann17/go-zip)](https://github.com/creativeyann17/go-zip/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/creativeyann17/go-zip)](go.mod)
[![License](https://img.shields.io/github/license/creativeyann17/go-zip)](LICENSE)
[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-support-orange?logo=buy-me-a-coffee&logoColor=white)](https://buymeacoffee.com/creativeyann17)

Parallel ZIP compression tool and Go library for file-tree backups.

ZIP-only: no format flag. Single-thread writes `archive.zip`; multi-thread writes `archive_01.zip`, `archive_02.zip`, …

## Features

- **True parallel ZIP** — one archive part per busy worker (no shared-writer mutex)
- **Deflate levels 1–9** (level 1 = store)
- **`.gitignore` support** (nested)
- **`--no-gc`** lower GC latency spikes, capped to available RAM (never uncapped — see below)
- **Multi-part aware** decompress and verify
- **Zip-slip protection** on extract
- **CLI and library** API

## Installation

### From source

```bash
git clone https://github.com/creativeyann17/go-zip.git
cd go-zip
make build
```

Binary lands in `bin/gozip`.

```bash
make test
make build-all   # multi-platform dist/ (release)
```

## CLI

```bash
# Single archive
gozip compress -i /data -o backup -t 1 -l 6

# Multi-part (4 workers)
gozip compress -i /data -o backup -t 4 -l 9 --verbose

gozip verify -i backup_01.zip --data
gozip decompress -i backup_01.zip -o /restore --overwrite

# Single-part verify/extract
gozip verify -i backup.zip --data
gozip decompress -i backup.zip -o /restore
```

### Compress flags

| Flag | Description |
|------|-------------|
| `-i, --input` | Input file or directory (required) |
| `-o, --output` | Output base path (default `archive`) |
| `-t, --threads` | Worker count (default: CPU count) |
| `-l, --level` | Deflate 1–9 (default 5) |
| `--gitignore` | Honor `.gitignore` |
| `--no-gc` | Lower GC latency during compress, capped to available RAM |
| `--dry-run` | Simulate only |
| `--verbose` / `--quiet` | Logging |

No `--zip` flag: output is always ZIP.

### `--no-gc`

Turns off percentage-based GC for fewer pauses, but always pairs it with a soft
memory limit (Go's `debug.SetMemoryLimit`) sized to ~70% of currently available
RAM. The runtime still forces a GC pass before the process grows past that cap
— it cannot grow unbounded, swap-thrash, or OOM the machine. If available RAM
can't be read (non-Linux, or `/proc/meminfo` unreadable) or is too low for a
meaningful cap, `--no-gc` is silently downgraded to default GC and a warning is
printed (unless `--quiet`).

### Naming rules

| Threads | Output |
|---------|--------|
| `1` | `base.zip` |
| `> 1` | `base_01.zip`, `base_02.zip`, … (contiguous, lazy — idle workers create nothing) |

Decompress/verify:

- Plain `*.zip` (no `_NN` suffix) → single file only
- `*_NN.zip` → discover sibling parts `base_01`…`base_99`

## Library

```go
import (
    "github.com/creativeyann17/go-zip/pkg/compress"
    "github.com/creativeyann17/go-zip/pkg/decompress"
)

opts := compress.DefaultOptions()
opts.InputPath = "/data"
opts.OutputPath = "backup"
opts.MaxThreads = 8
opts.Level = 9
result, err := compress.Compress(opts, nil)

dopts := decompress.DefaultOptions()
dopts.InputPath = "backup_01.zip"
dopts.OutputPath = "/restore"
dopts.Overwrite = true
_, err = decompress.Decompress(dopts, nil)
```

Library-only: set `opts.Files` for a custom file list instead of `InputPath`.

## Layout

```
cmd/gozip/          CLI
pkg/compress/       compress API
pkg/decompress/     decompress API
pkg/verify/         verify API
internal/multipart  part naming + discovery
```

## License

MIT
