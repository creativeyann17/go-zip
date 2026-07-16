package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/creativeyann17/go-zip/internal/multipart"
	"github.com/creativeyann17/go-zip/pkg/verify"
)

func init() {
	rootCmd.AddCommand(verifyCmd())
}

func verifyCmd() *cobra.Command {
	var inputPath string
	var verifyData, verbose, quiet bool

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify ZIP archive integrity",
		Long: `Verify the integrity of a ZIP archive.

By default, performs structural validation (headers, metadata).
Use --data to also verify data integrity by decompressing all content.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			inputPath = multipart.ResolveInputPath(inputPath)

			opts := &verify.Options{
				InputPath:  inputPath,
				VerifyData: verifyData,
				Verbose:    verbose,
				Quiet:      quiet,
			}
			if err := opts.Validate(); err != nil {
				return err
			}

			log := func(format string, args ...interface{}) {
				if !quiet {
					fmt.Printf(format+"\n", args...)
				}
			}

			log("Verifying archive: %s", inputPath)
			if verifyData {
				log("Mode: Full data integrity check")
			} else {
				log("Mode: Structural validation only")
			}
			log("")

			var progressCb verify.ProgressCallback
			if !quiet && !verbose {
				progressCb = func(event verify.ProgressEvent) {
					switch event.Type {
					case verify.EventFileVerify:
						if event.Current%100 == 0 || event.Current == event.Total {
							fmt.Printf("\r  Progress: %d files", event.Current)
						}
					case verify.EventComplete:
						fmt.Printf("\r  Progress: %d files\n", event.Current)
					}
				}
			} else if verbose {
				progressCb = func(event verify.ProgressEvent) {
					switch event.Type {
					case verify.EventFileVerify:
						fmt.Printf("  [%d] %s\n", event.Current, event.FilePath)
					case verify.EventComplete:
						fmt.Println("Verification complete")
					}
				}
			}

			result, err := verify.Verify(opts, progressCb)
			if err != nil && result == nil {
				return err
			}

			fmt.Println()
			fmt.Print(result.Summary())

			if !result.IsValid() {
				return fmt.Errorf("archive verification failed")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input ZIP archive (required)")
	cmd.Flags().BoolVar(&verifyData, "data", false, "Verify data integrity by decompressing all content")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed output")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Minimal output")
	_ = cmd.MarkFlagRequired("input")

	return cmd
}
