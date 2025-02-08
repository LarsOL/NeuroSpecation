package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print neurospectation version",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Details:", "version", version, "commit", commit, "releaseDate", date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
