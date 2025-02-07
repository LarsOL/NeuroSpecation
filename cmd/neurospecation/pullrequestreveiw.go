package main

import (
	"context"
	"fmt"
	"github.com/LarsOL/NeuroSpecation/aihelpers"
	"github.com/google/go-github/v69/github"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const ReviewPrompt = "You are a skilled software engineer, review the given pull requests and provide valuable feedback. Look for both high level architectural problems and code level improvements. You will be first given the repo context as distilled by a AI, then the PR."

func ReviewPullRequests(ctx context.Context, dir string, aiClient *aihelpers.AIClient, options *Options) error {
	targetBranch := options.targetBranch
	if targetBranch == "" {
		slog.Debug("no target branch flag set")
		targetBranch = os.Getenv("GITHUB_BASE_REF")
		if targetBranch == "" {
			slog.Debug("no target branch github env set (GITHUB_BASE_REF)")
			defaultBranchName, err := getDefaultBranch(dir)
			if err != nil {
				return err
			}
			targetBranch = defaultBranchName
		} else {
			slog.Debug("target branch github env set (GITHUB_BASE_REF)", "env", targetBranch)
		}
	}

	if options.debug {
		debug(dir)
	}

	if !isInsideGitRepo(dir) {
		return fmt.Errorf("must be run from within a git repo")
	}

	diffOutput, err := getGitDiff(dir, targetBranch)
	if err != nil {
		return err
	}

	if diffOutput == "" {
		return fmt.Errorf("no diff between current commit and %s", targetBranch)
	}

	gitRoot, err := getGitRoot(dir)
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

	if os.Getenv("GITHUB_TOKEN") == "" {
		err := writeReviewFile(dir, reviewOutput, options.dryRun)
		if err != nil {
			return err
		}
	} else {
		err = writeReviewToPR(ctx, reviewOutput)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeReviewToPR(ctx context.Context, reviewOutput string) error {
	client := github.NewClient(nil).WithAuthToken(os.Getenv("GITHUB_TOKEN"))

	repo := os.Getenv("GITHUB_REPOSITORY")
	prNumber := os.Getenv("GITHUB_PR_NUMBER")

	if repo == "" {
		return fmt.Errorf("GITHUB_REPOSITORY environment variable is not set")
	}
	if prNumber == "" {
		return fmt.Errorf("GITHUB_PR_NUMBER environment variable is not set")
	}

	ownerRepo := strings.Split(repo, "/")
	if len(ownerRepo) != 2 {
		return fmt.Errorf("invalid GITHUB_REPOSITORY format, got %s", repo)
	}

	owner, repoName := ownerRepo[0], ownerRepo[1]
	prNum, err := strconv.Atoi(prNumber)
	if err != nil {
		return fmt.Errorf("invalid GITHUB_PR_NUMBER format, got %s: err: %w", prNumber, err)
	}

	comment := &github.IssueComment{Body: &reviewOutput}
	_, _, err = client.Issues.CreateComment(ctx, owner, repoName, prNum, comment)
	if err != nil {
		return fmt.Errorf("failed to create comment on PR, err: %w", err)
	}
	return nil
}

func runGitCommand(dir string, args ...string) *exec.Cmd {
	// Create the command and set its working directory.
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// Explicitly set both GIT_DIR and GIT_WORK_TREE.
	cmd.Env = append(os.Environ(),
		"GIT_DIR="+dir+"/.git",
		"GIT_WORK_TREE="+dir,
	)
	return cmd
}

func getDefaultBranch(dir string) (string, error) {
	cmd := runGitCommand(dir, "rev-parse", "--abbrev-ref", "origin/HEAD")
	defaultBranch, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}
	defaultBranchName := strings.TrimSpace(string(defaultBranch))
	defaultBranchName = strings.TrimPrefix(defaultBranchName, "origin/")
	return defaultBranchName, nil
}

func isInsideGitRepo(dir string) bool {
	cmd := runGitCommand(dir, "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Not a Git repo: %v, output: %s\n", err, output)
		return false
	}

	gitRoot := strings.TrimSpace(string(output))
	slog.Debug("Git root directory: ", "dir", dir, "gitRoot", gitRoot)
	return true
}

func getGitDiff(dir string, target string) (string, error) {
	cmd := runGitCommand(dir, "diff", "origin/"+target+"...HEAD")
	diffOutput, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff between current commit and target: %s err: %w", target, err)
	}
	return string(diffOutput), nil
}

func debug(dir string) {
	debugCommands := []struct {
		name string
		args []string
	}{
		{"ls", []string{"."}},
		{"env", []string{""}},
		{"ls", []string{"-la", "/github/workspace/.git"}},
		{"ls", []string{"-la", "./.git"}},
		{"ls", []string{"-la"}},
		{"ls", []string{"~"}},
		{"ls", []string{".."}},
		{"ls", []string{"../.."}},
		{"pwd", []string{""}},
		{"git", []string{"version"}},
		{"git", []string{"status"}},
		{"git", []string{"-C", ".", "status"}},
		{"git", []string{"-C", "..", "status"}},
		{"git", []string{"-C", "/github/workspace", "status"}},
		{"git", []string{"-C", "../..", "status"}},
		{"git", []string{"rev-parse", "--show-toplevel"}},
		{"git", []string{"branch", "-a"}},
		{"git", []string{"remote", "-v"}},
		{"git", []string{"rev-parse", "--is-inside-work-tree"}},
	}

	for _, cmdInfo := range debugCommands {
		slog.Debug("running: ", "name", cmdInfo.name, "args", cmdInfo.args)
		cmd := exec.Command(cmdInfo.name, cmdInfo.args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_DIR="+dir+"/.git",
			"GIT_WORK_TREE="+dir,
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			slog.Debug(fmt.Sprintf("failed to execute %s %v: %v", cmdInfo.name, cmdInfo.args, err))
		} else {
			slog.Debug(fmt.Sprintf("output of %s %v: %s", cmdInfo.name, cmdInfo.args, string(output)))
		}
	}
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
