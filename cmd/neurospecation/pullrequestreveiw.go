package main

import (
	"context"
	"fmt"
	"github.com/LarsOL/NeuroSpecation/aihelpers"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const ReviewPrompt = "You are a skilled software engineer, review the given pull requests and provide valuable feedback. Look for both high level architectural problems and code level improvements. You will be first given the repo context as distilled by a AI, then the PR."

func ReviewPullRequests(ctx context.Context, dir string, aiClient *aihelpers.AIClient, options *Options) error {
	currentBranch, defaultBranchName, err := getGitBranches()
	if err != nil {
		return err
	}

	targetBranch := options.targetBranch
	if targetBranch == "" {
		targetBranch = defaultBranchName
	}

	if currentBranch == targetBranch {
		return fmt.Errorf("current branch %s, same as target branch %s", currentBranch, targetBranch)
	}

	diffOutput, err := getGitDiff(currentBranch, targetBranch)
	if err != nil {
		return err
	}

	gitRoot, err := getGitRoot()
	if err != nil {
		return err
	}

	prompt, err := createReviewPrompt(gitRoot, diffOutput)
	if err != nil {
		return err
	}

	if options.logPrompt {
		if err := logPromptToFile(dir, "ai_review_prompt.txt", prompt); err != nil {
			return err
		}
	}

	reviewOutput, err := promptAI(ctx, aiClient, prompt, options.dryRun)
	if err != nil {
		return err
	}

	return writeReviewFile(dir, reviewOutput, options.dryRun)
}

func getGitBranches() (string, string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get current branch: %w", err)
	}
	currentBranch := strings.TrimSpace(string(output))

	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "origin/HEAD")
	defaultBranch, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get default branch: %w", err)
	}
	defaultBranchName := strings.TrimSpace(string(defaultBranch))
	defaultBranchName = strings.TrimPrefix(defaultBranchName, "origin/")
	return currentBranch, defaultBranchName, nil
}

func getGitDiff(currentBranch, defaultBranchName string) (string, error) {
	cmd := exec.Command("git", "diff", defaultBranchName, currentBranch)
	diffOutput, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff between currentbranch %s and default branch %s: %w", currentBranch, defaultBranchName, err)
	}
	return string(diffOutput), nil
}

func getGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git root directory: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func createReviewPrompt(gitRoot, diffOutput string) (string, error) {
	changedFiles := strings.Split(diffOutput, "\n")
	knowledgeContent := ""
	for _, line := range changedFiles {
		if strings.HasPrefix(line, "diff --git") {
			parts := strings.Split(line, " ")
			if len(parts) > 2 {
				filePath := strings.TrimPrefix(parts[2], "a/")
				fullPath := filepath.Join(gitRoot, filePath)
				dirPath := filepath.Dir(fullPath)
				knowledgePath := filepath.Join(dirPath, "ai_knowledge.yaml")
				content, err := os.ReadFile(knowledgePath)
				if err == nil {
					knowledgeContent += string(content) + "\n"
				}
			}
		}
	}
	reviewPrompt := ReviewPrompt + "\n<Repo Context>\n" + knowledgeContent + "\n</Repo Context>\n" + "\n<Diff>\n" + diffOutput + "\n</Diff>\n"
	return reviewPrompt, nil
}

func writeReviewFile(dir, reviewOutput string, dryRun bool) error {
	reviewFilePath := filepath.Join(dir, "ai_Review.md")
	if dryRun {
		slog.Debug("skipping AI review, would have written file to:", "path", reviewFilePath)
		return nil
	}

	f, err := os.Create(reviewFilePath)
	if err != nil {
		slog.Error("failed to create review file", "err", err)
		return err
	}
	defer f.Close()

	_, err = f.WriteString(reviewOutput)
	if err != nil {
		slog.Error("failed to write review file", "err", err)
		return err
	}
	return nil
}
