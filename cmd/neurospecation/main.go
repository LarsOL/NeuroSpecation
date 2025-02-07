package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/LarsOL/NeuroSpecation/aihelpers"
	"log/slog"
)

type Options struct {
	DryRun              bool
	Debug               bool
	LogPrompt           bool
	UpdateKnowledge     bool
	Model               string
	CreateReadme        bool
	ReviewPR            bool
	ConcurrencyRPMLimit int
	TargetBranch        string
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Define command-line flags
	debug := flag.Bool("d", false, "Enable debug logging")
	dryRun := flag.Bool("dr", false, "Enable dry-run mode")
	logPrompt := flag.Bool("lp", false, "Log the prompt sent to the AI")
	updateKnowledge := flag.Bool("uk", false, "Update the knowledge base")
	model := flag.String("m", "gpt-4o", "The model to use for AI requests")
	createReadme := flag.Bool("cr", false, "Create a summary of the directory")
	reviewPR := flag.Bool("r", false, "Review pull requests")
	concurrencyLimit := flag.Int("cl", 500, "Concurrency limit for updating knowledge base")
	targetBranch := flag.String("tb", "", "Target branch for pull request reviews")
	dir := flag.String("dir", "", "Directory to run on")
	help := flag.Bool("h", false, "Show help")
	ver := flag.Bool("v", false, "Show version")

	// Custom usage message
	flag.Usage = func() {
		slog.Info("Usage: neurospecation <directory> [flags]")
		slog.Info("Flags:")
		flag.PrintDefaults()
		slog.Info("Details:", "version", version, "commit", commit, "releaseDate", date)
	}
	flag.Parse()

	// Handle help and version requests
	if *help {
		flag.Usage()
		os.Exit(0)
	}
	if *ver {
		slog.Info("Details:", "version", version, "commit", commit, "releaseDate", date)
		os.Exit(0)
	}

	// Build options struct
	opts := Options{
		DryRun:              *dryRun,
		Debug:               *debug,
		LogPrompt:           *logPrompt,
		UpdateKnowledge:     *updateKnowledge,
		Model:               *model,
		CreateReadme:        *createReadme,
		ReviewPR:            *reviewPR,
		ConcurrencyRPMLimit: *concurrencyLimit,
		TargetBranch:        *targetBranch,
	}

	// Initialize logging
	levelVar := new(slog.LevelVar)
	if opts.Debug {
		levelVar.Set(slog.LevelDebug)
		slog.Info("Debug logging enabled")
	} else {
		levelVar.Set(slog.LevelInfo)
		slog.Debug("Debug logging disabled")
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: levelVar,
	}))
	slog.SetDefault(logger)

	// Create context with the logger
	ctx := context.Background()
	ctx = setLoggerToCtx(ctx, logger)

	slog.Info("Command line arguments", "args", os.Args)

	// Determine target directory
	targetDir := *dir
	if targetDir == "" {
		slog.Debug("Directory flag not set; checking GITHUB_WORKSPACE environment variable")
		targetDir = os.Getenv("GITHUB_WORKSPACE")
		if targetDir == "" {
			slog.Debug("GITHUB_WORKSPACE not set; using current directory")
			targetDir = "."
		} else {
			slog.Debug("Using directory from GITHUB_WORKSPACE", "dir", targetDir)
		}
	} else {
		slog.Debug("Using directory from command line flag", "dir", targetDir)
	}

	// Ensure at least one operation mode is selected
	if !opts.CreateReadme && !opts.ReviewPR && !opts.UpdateKnowledge {
		slog.Info("Please select one or more modes: Update Knowledgebase, Create README, or Review PR")
		flag.Usage()
		os.Exit(0)
	}

	// Report dry-run mode status
	if opts.DryRun {
		slog.Info("Dry-run mode enabled")
	} else {
		slog.Debug("Dry-run mode disabled")
	}

	// Initialize the AI client if not in dry-run mode
	var aiClient *aihelpers.AIClient
	if !opts.DryRun {
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			slog.Error("API key is not set")
			os.Exit(1)
		}
		aiClient = aihelpers.NewOpenAIClient(apiKey, opts.Model)
	}

	// Execute selected operations
	if opts.UpdateKnowledge {
		slog.Info("Updating AI knowledge base")
		if err := UpdateKnowledgeBase(ctx, targetDir, aiClient, &opts); err != nil {
			slog.Error("Error updating knowledge base", "err", err)
			os.Exit(1)
		}
		slog.Info("Finished updating knowledge base")
	}

	if opts.CreateReadme {
		slog.Info("Creating AI README")
		if err := CreateReadMe(ctx, targetDir, aiClient, &opts); err != nil {
			slog.Error("Error creating README", "err", err)
			os.Exit(1)
		}
		slog.Info("Finished creating README")
	}

	if opts.ReviewPR {
		slog.Info("Creating PR review")
		if err := ReviewPullRequests(ctx, targetDir, aiClient, &opts); err != nil {
			slog.Error("Error reviewing pull requests", "err", err)
			os.Exit(1)
		}
		slog.Info("Finished creating PR review")
	}
}

// promptAI prompts the AI client with the provided prompt.
func promptAI(ctx context.Context, aiClient *aihelpers.AIClient, prompt string, dryRun bool) (string, error) {
	if dryRun {
		slog.Debug("Dry-run mode, skipping AI prompt")
		return "", nil
	}
	loggerFromCtx(ctx).Debug("Prompting AI", "prompt", prompt)
	answer, err := aiClient.Prompt(ctx, aihelpers.PromptRequest{Prompt: prompt})
	if err != nil {
		return "", fmt.Errorf("failed to prompt AI: %w", err)
	}
	return answer, nil
}

// logPromptToFile writes the given prompt to a file in the specified directory.
func logPromptToFile(dir, filename, prompt string) error {
	filePath := filepath.Join(dir, filename)
	fl, err := os.Create(filePath)
	if err != nil {
		slog.Error("Failed to create AI prompt file", "err", err)
		return err
	}
	defer fl.Close()

	if _, err := fl.WriteString(prompt); err != nil {
		slog.Error("Failed to write AI prompt to file", "err", err)
		return err
	}
	return nil
}

const loggerKey = "logger"

// setLoggerToCtx stores the logger in the given context.
func setLoggerToCtx(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// loggerFromCtx retrieves the logger from the given context.
func loggerFromCtx(ctx context.Context) *slog.Logger {
	return ctx.Value(loggerKey).(*slog.Logger)
}
