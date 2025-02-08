package cmd

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "neurospecation",
	Short: "A ai repo assistant",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		slog.Error("Error:", "err", err)
		os.Exit(1)
	}
}

const dryRunKey = "dry-run"
const debugKey = "debug"
const modelKey = "model"
const dirKey = "dir"
const logPromptKey = "log-prompts"

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.NeuroSpecation.yaml)")
	rootCmd.PersistentFlags().BoolP(debugKey, "d", false, "Enable debug logging")
	rootCmd.PersistentFlags().Bool(dryRunKey, false, "Enable dry-run mode")
	rootCmd.PersistentFlags().StringP(modelKey, "m", "gpt-4o", "The model to use for AI requests")
	rootCmd.PersistentFlags().StringP(dirKey, "", "", "Directory to run on")
	rootCmd.PersistentFlags().Bool(logPromptKey, false, "Debug: Log prompts to file")

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		lvl := new(slog.LevelVar)
		if viper.GetBool("debug") {
			lvl.Set(slog.LevelDebug)
			slog.Info("Debug logging enabled")
		} else {
			lvl.Set(slog.LevelInfo)
			slog.Debug("Debug logging disabled")
		}
		l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: lvl,
		}))
		slog.SetDefault(l)

		ctx := context.Background()
		ctx = setLoggerToCtx(ctx, l)
		cmd.SetContext(ctx)
	}

	err := viper.BindPFlags(rootCmd.Flags())
	if err != nil {
		slog.Error("could not bind to flags:", "err", err)
		os.Exit(1)
	}
	err = viper.BindPFlags(rootCmd.PersistentFlags())
	if err != nil {
		slog.Error("could not bind to persistent flags:", "err", err)
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			slog.Error("Could not get home directory to load config", "err", err)
		} else {
			viper.AddConfigPath(home)
		}

		gitRoot, err := getGitRoot(".")
		if err != nil {
			slog.Error("Could not get git root to load config", "err", err)
		} else {
			viper.AddConfigPath(gitRoot)
		}
		viper.SetConfigType("yaml")
		viper.SetConfigName(".neurospecation")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		slog.Info("Using config file:", "file", viper.ConfigFileUsed())
	}

	viper.OnConfigChange(func(e fsnotify.Event) {
		slog.Info("Config file changed:", "file", e.Name)
	})
	viper.WatchConfig()
}
