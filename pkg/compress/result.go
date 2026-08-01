package compress

// Result contains statistics about the compression operation.
type Result struct {
	FilesTotal     int
	FilesProcessed int
	OriginalSize   uint64
	CompressedSize uint64
	Errors         []error
}

// CompressionRatio returns the compression ratio as a percentage.
func (r *Result) CompressionRatio() float64 {
	if r.OriginalSize == 0 {
		return 0
	}
	return float64(r.CompressedSize) / float64(r.OriginalSize) * 100
}

// Success returns true if all files were processed without errors.
func (r *Result) Success() bool {
	return len(r.Errors) == 0 && r.FilesProcessed == r.FilesTotal
}

func (r *Result) GetFilesTotal() int        { return r.FilesTotal }
func (r *Result) GetFilesProcessed() int    { return r.FilesProcessed }
func (r *Result) GetErrors() []error        { return r.Errors }
func (r *Result) GetOriginalSize() uint64   { return r.OriginalSize }
func (r *Result) GetCompressedSize() uint64 { return r.CompressedSize }
