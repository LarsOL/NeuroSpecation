package cmd

import (
	"context"
	"fmt"
	"github.com/LarsOL/NeuroSpecation/aihelpers"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func promptAI(ctx context.Context, aiClient *aihelpers.AIClient, prompt string, dryRun bool) (string, error) {
	if dryRun {
		slog.Debug("Dry-run mode, skipping AI prompt")
		return "", nil
	}
	loggerFromCtx(ctx).Debug("Prompting AI", "prompt", prompt)
	ans, resp, err := aiClient.Prompt(ctx, aihelpers.PromptRequest{Prompt: prompt})
	if viper.GetBool(debugKey) {
		slog.Debug("openai resp", "resp", resp)
	}
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

func getGitRoot(dir string) (string, error) {
	cmd := runGitCommand(dir, "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git root directory: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

const Loggerkey = "logger"

func setLoggerToCtx(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, Loggerkey, logger)
}

func loggerFromCtx(ctx context.Context) *slog.Logger {
	return ctx.Value(Loggerkey).(*slog.Logger)
}
