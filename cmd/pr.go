package cmd

import (
	"context"
	"fmt"
	"github.com/LarsOL/NeuroSpecation/aihelpers"
	"github.com/google/go-github/v69/github"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Review pull requests",
	Run: func(cmd *cobra.Command, args []string) {
		lvl := new(slog.LevelVar)
		if viper.GetBool(debugKey) {
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

		if viper.GetBool(dryRunKey) {
			slog.Info("Dry-run mode enabled")
		} else {
			slog.Debug("Dry-run mode disabled")
		}

		var aiClient *aihelpers.AIClient
		if !viper.GetBool(dryRunKey) {
			apiKey := os.Getenv("OPENAI_API_KEY")
			if apiKey == "" {
				slog.Error("API key is not set")
				os.Exit(1)
			}
			aiClient = aihelpers.NewOpenAIClient(apiKey, viper.GetString(modelKey))
		}

		slog.Info("Creating PR review")
		err := ReviewPullRequests(ctx, directory, aiClient)
		if err != nil {
			slog.Error("Error reviewing pull requests", "err", err)
			os.Exit(1)
		}
		slog.Info("finished creating PR review")
	},
}

const targetBranchKey = "target-branch"

func init() {
	rootCmd.AddCommand(prCmd)

	prCmd.PersistentFlags().String(targetBranchKey, "", "Target branch for pull request reviews")

	err := viper.BindPFlags(prCmd.PersistentFlags())
	if err != nil {
		slog.Error("could not bind to persistent flags:", "err", err)
		os.Exit(1)
	}

}

const ReviewPrompt = "You are a extremely skilled software engineer, review the given pull request and only provide valuable feedback. Only provide feedback if it is a strong point, do not include small or obvious suggestions. Provide two sections of feedback, 1. high level architectural problems, 2. code level improvements. Ensure to also consider non-functional concerns like security, performance & maintainability (testing). You will be first given the repo context as distilled by a AI, then the PR diff."

func ReviewPullRequests(ctx context.Context, dir string, aiClient *aihelpers.AIClient) error {
	targetBranch := viper.GetString(targetBranchKey)
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

	if viper.GetBool(debugKey) {
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

	prompt, err := createReviewPrompt(ctx, gitRoot, diffOutput)
	if err != nil {
		return err
	}

	if viper.GetBool(logPromptKey) {
		if err := logPromptToFile(dir, "ai_review_prompt.txt", prompt); err != nil {
			return err
		}
	}

	reviewOutput, err := promptAI(ctx, aiClient, prompt, viper.GetBool(dryRunKey))
	if err != nil {
		return err
	}

	if os.Getenv("GITHUB_TOKEN") == "" {
		err := writeReviewFile(dir, reviewOutput, viper.GetBool(dryRunKey))
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

	if viper.GetBool(debugKey) {
		// Print context of PR & check we at least have read permission
		pr, _, err := client.PullRequests.Get(ctx, owner, repoName, prNum)
		if err != nil {
			slog.Error("unable to get pr", "err", err)
		}
		slog.Debug("Adding comment to this PR", "pr", pr)
	}

	// Find existing comments and update the first one if it exists
	comments, _, err := client.Issues.ListComments(ctx, owner, repoName, prNum, nil)
	if err != nil {
		return fmt.Errorf("failed to list comments on PR, err: %w", err)
	}

	const UniqueTag = "# NeuroSpecation AI Review\n"
	reviewOutput = UniqueTag + reviewOutput
	var commentID int64
	for _, c := range comments {
		if strings.Contains(*c.Body, UniqueTag) {
			commentID = *c.ID
			break
		}
	}

	if commentID != 0 {
		// Update the existing comment
		_, _, err = client.Issues.EditComment(ctx, owner, repoName, commentID, &github.IssueComment{Body: &reviewOutput})
		if err != nil {
			return fmt.Errorf("failed to update comment on PR, err: %w", err)
		}
	} else {
		// Create a new comment if no existing comment is found
		comment := &github.IssueComment{Body: &reviewOutput}
		_, _, err = client.Issues.CreateComment(ctx, owner, repoName, prNum, comment)
		if err != nil {
			return fmt.Errorf("failed to create comment on PR, err: %w", err)
		}
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
		{"pwd", []string{""}},
		{"ls", []string{"-la"}},
		{"git", []string{"version"}},
		{"git", []string{"status"}},
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

func createReviewPrompt(ctx context.Context, gitRoot, diffOutput string) (string, error) {
	reviewPrompt := ReviewPrompt
	if os.Getenv("GITHUB_TOKEN") != "" {
		title, body, err := getPRInfo(ctx)
		if err != nil {
			return "", err
		}
		reviewPrompt = reviewPrompt + "<PR Details>\n" + "Title: " + title + "\nBody: " + body + "\n</PR Details>\n"
	}

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
	reviewPrompt = reviewPrompt + "\n<Repo Context>\n" + knowledgeContent + "\n</Repo Context>\n" + "\n<Diff>\n" + diffOutput + "\n</Diff>\n"
	return reviewPrompt, nil
}

func getPRInfo(ctx context.Context) (string, string, error) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		return "", "", nil
	}

	client := github.NewClient(nil).WithAuthToken(os.Getenv("GITHUB_TOKEN"))

	repo := os.Getenv("GITHUB_REPOSITORY")
	prNumber := os.Getenv("GITHUB_PR_NUMBER")

	if repo == "" {
		return "", "", fmt.Errorf("GITHUB_REPOSITORY environment variable is not set")
	}
	if prNumber == "" {
		return "", "", fmt.Errorf("GITHUB_PR_NUMBER environment variable is not set")
	}

	ownerRepo := strings.Split(repo, "/")
	if len(ownerRepo) != 2 {
		return "", "", fmt.Errorf("invalid GITHUB_REPOSITORY format, got %s", repo)
	}

	owner, repoName := ownerRepo[0], ownerRepo[1]
	prNum, err := strconv.Atoi(prNumber)
	if err != nil {
		return "", "", fmt.Errorf("invalid GITHUB_PR_NUMBER format, got %s: err: %w", prNumber, err)
	}

	pr, _, err := client.PullRequests.Get(ctx, owner, repoName, prNum)
	if err != nil {
		return "", "", fmt.Errorf("unable to get pr. err: %w", err)
	}

	if pr == nil {
		return "", "", fmt.Errorf("expected pr data to be not nil")
	}

	if pr.Title == nil && pr.Body == nil {
		return "", "", fmt.Errorf("expected pr title & body to be not nil")
	}

	return pr.GetTitle(), pr.GetBody(), nil
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
