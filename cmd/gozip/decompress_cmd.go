package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"

	"github.com/creativeyann17/go-zip/internal/multipart"
	"github.com/creativeyann17/go-zip/pkg/decompress"
)

func init() {
	rootCmd.AddCommand(decompressCmd())
}

func decompressCmd() *cobra.Command {
	var inputPath, outputPath string
	var maxThreads int
	var verbose, quiet, overwrite bool

	cmd := &cobra.Command{
		Use:   "decompress",
		Short: "Decompress ZIP archive(s) to a directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			inputPath = multipart.ResolveInputPath(inputPath)

			opts := &decompress.Options{
				InputPath:  inputPath,
				OutputPath: outputPath,
				MaxThreads: maxThreads,
				Verbose:    verbose,
				Quiet:      quiet,
				Overwrite:  overwrite,
			}
			if err := opts.Validate(); err != nil {
				return err
			}

			log := func(format string, args ...interface{}) {
				if !quiet {
					fmt.Printf(format+"\n", args...)
				}
			}

			log("Starting decompression...")
			log("  Input:       %s", opts.InputPath)
			log("  Output:      %s", opts.OutputPath)
			if overwrite {
				log("  Mode:        OVERWRITE")
			}
			log("")

			var progressCb decompress.ProgressCallback
			var progress *mpb.Progress
			if !quiet && !verbose {
				progressCb, progress = decompress.ProgressBarCallback()
			}

			result, err := decompress.Decompress(opts, progressCb)
			if progress != nil {
				progress.Wait()
			}
			if err != nil {
				return err
			}

			fmt.Println()
			fmt.Print(decompress.FormatSummary(result))
			if len(result.Errors) > 0 {
				return fmt.Errorf("finished with %d errors", len(result.Errors))
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input ZIP archive (required)")
	cmd.Flags().StringVarP(&outputPath, "output", "o", ".", "Output directory")
	cmd.Flags().IntVarP(&maxThreads, "threads", "t", runtime.NumCPU(), "Max concurrent threads")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing files")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed output")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Minimal output")
	_ = cmd.MarkFlagRequired("input")

	return cmd
}
