package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:     "gozip",
	Short:   "go-zip - parallel ZIP compression for backups",
	Long:    "go-zip creates efficient multi-threaded ZIP archives from file trees.",
	Version: fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
