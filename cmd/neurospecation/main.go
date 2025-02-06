package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/LarsOL/NeuroSpecation/aihelpers"
	"log/slog"
	"os"
	"path/filepath"
)

type Options struct {
	dryRun              bool
	debug               bool
	logPrompt           bool
	updateKnowledge     bool
	model               string
	createReadme        bool
	reviewPR            bool
	concurrencyRPMLimit int
	targetBranch        string
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	debug := flag.Bool("d", false, "Enable debug logging")
	dryRun := flag.Bool("dr", false, "Enable dry-run mode")
	logPrompt := flag.Bool("lp", false, "Log the prompt sent to the AI")
	updateKnowledge := flag.Bool("uk", false, "Update the knowledge base")
	model := flag.String("m", "gpt-4o", "The model to use for AI requests")
	createReadme := flag.Bool("cr", false, "Create a summary of the directory")
	reviewPR := flag.Bool("r", false, "Review pull requests")
	concurrencyLimit := flag.Int("cl", 500, "Concurrency limit for updating knowledge base")
	targetBranch := flag.String("tb", "", "Target branch for pull request reviews")
	help := flag.Bool("h", false, "Show help")
	ver := flag.Bool("v", false, "Show version")

	flag.Usage = func() {
		slog.Info("Usage: neurospecation <directory> [flags]")
		slog.Info("Flags:")
		flag.PrintDefaults()
		slog.Info("Details:", "version", version, "commit", commit, "releaseDate", date)
	}
	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if *ver {
		slog.Info("Details:", "version", version, "commit", commit, "releaseDate", date)
		os.Exit(0)
	}

	o := &Options{
		dryRun:              *dryRun,
		debug:               *debug,
		logPrompt:           *logPrompt,
		updateKnowledge:     *updateKnowledge,
		model:               *model,
		createReadme:        *createReadme,
		reviewPR:            *reviewPR,
		concurrencyRPMLimit: *concurrencyLimit,
		targetBranch:        *targetBranch,
	}

	directory := flag.Arg(0)
	if directory == "" {
		directory = "."
	}

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

	if o.dryRun {
		slog.Info("Dry-run mode enabled")
	} else {
		slog.Debug("Dry-run mode disabled")
	}

	slog.Debug("Command line arguments", "args", os.Args)

	var aiClient *aihelpers.AIClient
	if !o.dryRun {
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			slog.Error("API key is not set")
			os.Exit(1)
		}
		aiClient = aihelpers.NewOpenAIClient(apiKey, o.model)
	}

	if o.updateKnowledge {
		slog.Info("Updating AI knowledge base")
		err := UpdateKnowledgeBase(ctx, directory, aiClient, o)
		if err != nil {
			slog.Error("Error updating knowledge base", "err", err)
			os.Exit(1)
		}
		slog.Info("finished updating all knowledge base files")
	}
	if o.createReadme {
		slog.Info("Creating AI README")
		err := CreateReadMe(ctx, directory, aiClient, o)
		if err != nil {
			slog.Error("Error creating readme", "err", err)
			os.Exit(1)
		}
		slog.Info("finished creating readme")
	}
	if o.reviewPR {
		slog.Info("Creating PR review")
		err := ReviewPullRequests(ctx, directory, aiClient, o)
		if err != nil {
			slog.Error("Error reviewing pull requests", "err", err)
			os.Exit(1)
		}
		slog.Info("finished creating PR review")
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
