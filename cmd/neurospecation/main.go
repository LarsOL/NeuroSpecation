package main

import (
	"context"
	"fmt"
	"github.com/LarsOL/NeuroSpecation/aihelpers"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
	"path/filepath"
)

type Options struct {
	dryRun                 bool
	debug                  bool
	logPrompt              bool
	updateKnowledge        bool
	model                  string
	createReadme           bool
	reviewPR               bool
	concurrencyRPMThrottle int
	targetBranch           string
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	o := &Options{}

	rootCmd := &cobra.Command{
		Use:   "neurospecation",
		Short: "NeuroSpecation CLI",
	}

	rootCmd.PersistentFlags().BoolVarP(&o.debug, "debug", "d", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&o.dryRun, "dry-run", false, "Enable dry-run mode")
	rootCmd.PersistentFlags().StringVarP(&o.model, "model", "m", "gpt-4o", "The model to use for AI requests")
	rootCmd.PersistentFlags().StringVarP(&o.targetBranch, "dir", "", "", "Directory to run on")
	rootCmd.PersistentFlags().IntVar(&o.concurrencyRPMThrottle, "throttle", 500, "API limit in requests per minute")

	rootCmd.AddCommand(createReadmeCmd(o))
	rootCmd.AddCommand(reviewPRCmd(o))
	rootCmd.AddCommand(updateKnowledgeCmd(o))
	rootCmd.AddCommand(versionCmd(o))

	err := rootCmd.Execute()
	if err != nil {
		slog.Error("Error:", "err", err)
		os.Exit(1)
	}
}
func versionCmd(o *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print neurospectation version",
		Run: func(cmd *cobra.Command, args []string) {
			slog.Info("Details:", "version", version, "commit", commit, "releaseDate", date)
		},
	}
}
func createReadmeCmd(o *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "create-readme",
		Short: "Create a summary of the directory",
		Run: func(cmd *cobra.Command, args []string) {
			lvl := new(slog.LevelVar)
			if o.debug {
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

			slog.Debug("Command line arguments", "args", os.Args)
			directory := cmd.Flag("dir").Value.String()
			if directory == "" {
				slog.Debug("directory command line argument not set")
				directory = os.Getenv("GITHUB_WORKSPACE")
				if directory == "" {
					slog.Debug("GITHUB_WORKSPACE argument not set, using current directory")
					directory = "."
				} else {
					slog.Debug("using directory from GITHUB_WORKSPACE", "dir", directory)
				}
			} else {
				slog.Debug("using directory from cmd argument", "dir", directory)
			}

			if o.dryRun {
				slog.Info("Dry-run mode enabled")
			} else {
				slog.Debug("Dry-run mode disabled")
			}

			var aiClient *aihelpers.AIClient
			if !o.dryRun {
				apiKey := os.Getenv("OPENAI_API_KEY")
				if apiKey == "" {
					slog.Error("API key is not set")
					os.Exit(1)
				}
				aiClient = aihelpers.NewOpenAIClient(apiKey, o.model)
			}

			slog.Info("Creating AI README")
			err := CreateReadMe(ctx, directory, aiClient, o)
			if err != nil {
				slog.Error("Error creating readme", "err", err)
				os.Exit(1)
			}
			slog.Info("finished creating readme")
		},
	}
}

func reviewPRCmd(o *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "review-pr",
		Short: "Review pull requests",
		Run: func(cmd *cobra.Command, args []string) {
			lvl := new(slog.LevelVar)
			if o.debug {
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

			slog.Info("Command line arguments", "args", os.Args)
			directory := cmd.Flag("dir").Value.String()
			if directory == "" {
				slog.Debug("directory command line argument not set")
				directory = os.Getenv("GITHUB_WORKSPACE")
				if directory == "" {
					slog.Debug("GITHUB_WORKSPACE argument not set, using current directory")
					directory = "."
				} else {
					slog.Debug("using directory from GITHUB_WORKSPACE", "dir", directory)
				}
			} else {
				slog.Debug("using directory from cmd argument", "dir", directory)
			}

			if o.dryRun {
				slog.Info("Dry-run mode enabled")
			} else {
				slog.Debug("Dry-run mode disabled")
			}

			var aiClient *aihelpers.AIClient
			if !o.dryRun {
				apiKey := os.Getenv("OPENAI_API_KEY")
				if apiKey == "" {
					slog.Error("API key is not set")
					os.Exit(1)
				}
				aiClient = aihelpers.NewOpenAIClient(apiKey, o.model)
			}

			slog.Info("Creating PR review")
			err := ReviewPullRequests(ctx, directory, aiClient, o)
			if err != nil {
				slog.Error("Error reviewing pull requests", "err", err)
				os.Exit(1)
			}
			slog.Info("finished creating PR review")
		},
	}
}

func updateKnowledgeCmd(o *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "update-knowledge-base",
		Short: "Update the knowledge base",
		Run: func(cmd *cobra.Command, args []string) {
			lvl := new(slog.LevelVar)
			if o.debug {
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

			slog.Info("Command line arguments", "args", os.Args)
			directory := cmd.Flag("dir").Value.String()
			if directory == "" {
				slog.Debug("directory command line argument not set")
				directory = os.Getenv("GITHUB_WORKSPACE")
				if directory == "" {
					slog.Debug("GITHUB_WORKSPACE argument not set, using current directory")
					directory = "."
				} else {
					slog.Debug("using directory from GITHUB_WORKSPACE", "dir", directory)
				}
			} else {
				slog.Debug("using directory from cmd argument", "dir", directory)
			}

			if o.dryRun {
				slog.Info("Dry-run mode enabled")
			} else {
				slog.Debug("Dry-run mode disabled")
			}

			var aiClient *aihelpers.AIClient
			if !o.dryRun {
				apiKey := os.Getenv("OPENAI_API_KEY")
				if apiKey == "" {
					slog.Error("API key is not set")
					os.Exit(1)
				}
				aiClient = aihelpers.NewOpenAIClient(apiKey, o.model)
			}

			slog.Info("Updating AI knowledge base")
			err := UpdateKnowledgeBase(ctx, directory, aiClient, o)
			if err != nil {
				slog.Error("Error updating knowledge base", "err", err)
				os.Exit(1)
			}
			slog.Info("finished updating all knowledge base files")
		},
	}
}

func promptAI(ctx context.Context, aiClient *aihelpers.AIClient, prompt string, dryRun bool) (string, error) {
	if dryRun {
		slog.Debug("Dry-run mode, skipping AI prompt")
		return "", nil
	}
	loggerFromCtx(ctx).Debug("Prompting AI", "prompt", prompt)
	ans, err := aiClient.Prompt(ctx, aihelpers.PromptRequest{Prompt: prompt})
	if err != nil {
		return "", fmt.Errorf("failed to prompt AI: %w", err)
	}
	return ans, nil
}

func logPromptToFile(dir, filename, prompt string) error {
	fl, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		slog.Error("failed to create ai prompt file", "err", err)
		return err
	}
	defer fl.Close()

	_, err = fl.WriteString(prompt)
	if err != nil {
		slog.Error("failed to write ai prompt file", "err", err)
		return err
	}
	return nil
}

const Loggerkey = "logger"

func setLoggerToCtx(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, Loggerkey, logger)
}

func loggerFromCtx(ctx context.Context) *slog.Logger {
	return ctx.Value(Loggerkey).(*slog.Logger)
}
