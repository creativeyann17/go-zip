package decompress

// Result contains statistics about the decompression operation.
type Result struct {
	FilesTotal       int
	FilesProcessed   int
	CompressedSize   uint64
	DecompressedSize uint64
	Errors           []error
}

func (r *Result) Success() bool {
	return len(r.Errors) == 0 && r.FilesProcessed == r.FilesTotal
}

func (r *Result) GetFilesTotal() int        { return r.FilesTotal }
func (r *Result) GetFilesProcessed() int    { return r.FilesProcessed }
func (r *Result) GetErrors() []error        { return r.Errors }
func (r *Result) GetOriginalSize() uint64   { return r.DecompressedSize }
func (r *Result) GetCompressedSize() uint64 { return r.CompressedSize }
