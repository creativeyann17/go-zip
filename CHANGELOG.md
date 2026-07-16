# Changelog

## v0.1.0

- Initial release: parallel ZIP compress / decompress / verify
- Single-thread `base.zip`, multi-thread `base_NN.zip` naming
- CLI `gozip` and library packages under `pkg/`
- `--no-gc`: percentage-based GC off, backed by a soft memory limit (`debug.SetMemoryLimit`) capped to ~70% of available RAM — runtime forces GC before the cap instead of growing unbounded. Falls back to default GC (with a warning) when available RAM can't be read or is too low for a safe cap.
- Flate freelist (`Reset` reuse per worker): O(threads) flate memory, not O(files)
