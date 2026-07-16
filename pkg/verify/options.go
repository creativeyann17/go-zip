package verify

// Options configures the verify operation.
type Options struct {
	InputPath  string
	VerifyData bool
	Verbose    bool
	Quiet      bool
}

// Validate checks if options are valid.
func (o *Options) Validate() error {
	if o.InputPath == "" {
		return ErrInputRequired
	}
	if o.Quiet {
		o.Verbose = false
	}
	return nil
}
