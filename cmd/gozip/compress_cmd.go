package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"

	"github.com/creativeyann17/go-zip/pkg/compress"
)

func init() {
	rootCmd.AddCommand(compressCmd())
}

func compressCmd() *cobra.Command {
	var inputPath, outputPath string
	var maxThreads int
	var dryRun, verbose, quiet, useGitignore, disableGC bool
	var compressLevel int

	cmd := &cobra.Command{
		Use:   "compress",
		Short: "Compress file or directory into ZIP archive(s)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if outputPath == "" {
				outputPath = "archive"
			}
			outputPath = strings.TrimSuffix(outputPath, ".zip")

			opts := &compress.Options{
				InputPath:    inputPath,
				OutputPath:   outputPath,
				MaxThreads:   maxThreads,
				Level:        compressLevel,
				DryRun:       dryRun,
				Verbose:      verbose,
				Quiet:        quiet,
				UseGitignore: useGitignore,
				DisableGC:    disableGC,
			}
			if err := opts.Validate(); err != nil {
				return err
			}

			log := func(format string, args ...interface{}) {
				if !quiet {
					fmt.Printf(format+"\n", args...)
				}
			}

			log("Starting compression...")
			log("  Format:      ZIP")
			log("  Input:       %s", opts.InputPath)
			if opts.MaxThreads == 1 {
				log("  Output:      %s.zip", opts.OutputPath)
			} else {
				log("  Output:      %s_01.zip … (%d threads)", opts.OutputPath, opts.MaxThreads)
			}
			log("  Threads:     %d", opts.MaxThreads)
			log("  Level:       %d", opts.Level)
			if dryRun {
				log("  Mode:        DRY-RUN")
			}
			if useGitignore {
				log("  Gitignore:   enabled")
			}
			if disableGC {
				log("  GC Mode:     disabled (pooled buffers)")
			}
			log("")

			var progressCb compress.ProgressCallback
			var progress *mpb.Progress
			if !quiet && !verbose {
				progressCb, progress = compress.ProgressBarCallback()
			}

			result, err := compress.Compress(opts, progressCb)
			if progress != nil {
				progress.Wait()
			}
			if err != nil {
				return err
			}

			fmt.Println()
			fmt.Print(compress.FormatSummary(result, opts))
			if len(result.Errors) > 0 {
				return fmt.Errorf("finished with %d errors", len(result.Errors))
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input file or directory (required)")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output archive path base (default: archive)")
	cmd.Flags().IntVarP(&maxThreads, "threads", "t", runtime.NumCPU(), "Max concurrent threads")
	cmd.Flags().IntVarP(&compressLevel, "level", "l", 5, "Compression level 1-9 (1=store/fast, 9=best deflate)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate without writing")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed output")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Minimal output")
	cmd.Flags().BoolVar(&useGitignore, "gitignore", false, "Respect .gitignore files")
	cmd.Flags().BoolVar(&disableGC, "no-gc", false, "Disable GC during compression (pooled buffers)")
	_ = cmd.MarkFlagRequired("input")

	return cmd
}
