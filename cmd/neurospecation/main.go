package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/LarsOL/NeuroSpecation/aihelpers"
	"github.com/LarsOL/NeuroSpecation/dirhelper"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const KnowledgeBasePrompt = "Create a YML file with all the key details about this software directory, this should contain a concise representation of all the information needed to: Identify & explain the key business processes, Explain the module, Explain the architectural patterns, Identify key files, Identify key links to other modules, plus anything else that would be useful for a skilled software engineer to understand the directory."
const ReadmePrompt = "Create a README file for this directory. This should contain a concise representation of all the key information needed for a skilled software engineer to understand the repo. Do not guess at any information. Only use the provided text. Reply with a markdown file."
const ReviewPrompt = "You are a skilled software engineer, review the given pull requests and provide valuable feedback. Look for both high level architectural problems and code level improvements. You will be first given the repo context as distilled by a AI, then the PR."

type Options struct {
	dryRun          bool
	debug           bool
	logPrompt       bool
	updateKnowledge bool
	model           string
	createReadme    bool
	reviewPR        bool
}

func main() {
	debug := flag.Bool("d", false, "Enable debug logging")
	dryRun := flag.Bool("dr", false, "Enable dry-run mode")
	logPrompt := flag.Bool("lp", false, "Log the prompt sent to the AI")
	updateKnowledge := flag.Bool("uk", false, "Update the knowledge base")
	model := flag.String("m", "gpt-4o-mini", "The model to use for AI requests")
	createReadme := flag.Bool("cr", false, "Create a summary of the directory")
	reviewPR := flag.Bool("r", false, "Review pull requests")
	help := flag.Bool("h", false, "Show help")

	flag.Usage = func() {
		slog.Error("Usage: repotraversal <directory> [flags]")
		slog.Error("Flags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	ctx := context.Background()
	o := &Options{
		dryRun:          *dryRun,
		debug:           *debug,
		logPrompt:       *logPrompt,
		updateKnowledge: *updateKnowledge,
		model:           *model,
		createReadme:    *createReadme,
		reviewPR:        *reviewPR,
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
	}
	if o.createReadme {
		slog.Info("Creating AI README")
		err := CreateReadMe(ctx, directory, aiClient, o)
		if err != nil {
			slog.Error("Error creating readme", "err", err)
			os.Exit(1)
		}
	}
	if o.reviewPR {
		slog.Info("Creating PR review")
		err := ReviewPullRequests(ctx, directory, aiClient, o)
		if err != nil {
			slog.Error("Error reviewing pull requests", "err", err)
			os.Exit(1)
		}
	}
}

func CreateReadMe(ctx context.Context, dir string, aiClient *aihelpers.AIClient, options *Options) error {
	prompt, err := gatherAIKnowledgeForReadMe(dir)
	if err != nil {
		return err
	}

	if options.logPrompt {
		if err := logPromptToFile(dir, "ai_summary_prompt.txt", prompt); err != nil {
			return err
		}
	}

	ans, err := promptAI(ctx, aiClient, prompt, options.dryRun)
	if err != nil {
		return err
	}

	return writeReadMe(dir, ans, options.dryRun)
}

func gatherAIKnowledgeForReadMe(dir string) (string, error) {
	var prompt strings.Builder
	prompt.WriteString(ReadmePrompt)
	prompt.WriteString("\n<Summarised AI knowledge base>\n")
	err := dirhelper.WalkDirectories(dir, func(d string, files []dirhelper.FileContent, subdirs []string) error {
		slog.Info("Processing Directory", "Dir", d)
		for _, file := range files {
			slog.Debug("Processing file", "File", file.Name)
			prompt.WriteString("- " + file.FullPath() + "\n")
			prompt.WriteString(file.Content + "\n")
		}
		return nil
	}, func(node fs.DirEntry) bool {
		return node.IsDir() || node.Name() == "ai_knowledge.yaml"
	})
	if err != nil {
		return "", fmt.Errorf("error walking directories: %w", err)
	}
	prompt.WriteString("</Summarised AI knowledge base>\n")
	return prompt.String(), nil
}

func promptAI(ctx context.Context, aiClient *aihelpers.AIClient, prompt string, dryRun bool) (string, error) {
	if dryRun {
		slog.Debug("Dry-run mode, skipping AI prompt")
		return "", nil
	}
	slog.Debug("Prompting AI", "prompt", prompt)
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

func writeReadMe(dir, ans string, dryRun bool) error {
	ymlPath := filepath.Join(dir, "ai_README.md")
	if dryRun {
		slog.Debug("skipping AI prompt, would have written file to:", "path", ymlPath)
		return nil
	}

	if ans == "no" {
		slog.Debug("AI did not find the directory useful", "dir", dir, "ans", ans)
		return nil
	}
	ans = strings.TrimPrefix(ans, "```markdown\n")
	ans = strings.TrimSuffix(ans, "\n```")
	f, err := os.Create(ymlPath)
	if err != nil {
		slog.Error("failed to create yaml file", "err", err)
		return err
	}
	defer f.Close()

	_, err = f.WriteString(ans)
	if err != nil {
		slog.Error("failed to write yaml file", "err", err)
		return err
	}
	return nil
}

func UpdateKnowledgeBase(ctx context.Context, dir string, aiClient *aihelpers.AIClient, options *Options) error {
	err := dirhelper.WalkDirectories(dir, func(dir string, files []dirhelper.FileContent, subdirs []string) error {
		prompt := createKnowledgeBasePrompt(dir, files, subdirs)
		if options.logPrompt {
			if err := logPromptToFile(dir, "ai_knowledge_prompt.txt", prompt); err != nil {
				return err
			}
		}

		ans, err := promptAI(ctx, aiClient, prompt, options.dryRun)
		if err != nil {
			return err
		}

		return writeKnowledgeBase(dir, ans, options.dryRun)
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to walk directories: %w", err)
	}
	return nil
}

func createKnowledgeBasePrompt(dir string, files []dirhelper.FileContent, subdirs []string) string {
	var prompt strings.Builder
	prompt.WriteString(KnowledgeBasePrompt)
	prompt.WriteString("\n<Directory Information>\n")
	prompt.WriteString("Directory: " + dir + "\n")

	if len(subdirs) == 0 {
		prompt.WriteString("No subdirectories\n")
	} else {
		prompt.WriteString("Subdirectories:\n")
		for _, subdir := range subdirs {
			prompt.WriteString("- " + subdir + "\n")
		}
	}

	if len(files) == 0 {
		slog.Debug("No valid files in dir", "dir", dir)
		return ""
	} else {
		prompt.WriteString("Files:\n")
		for _, file := range files {
			prompt.WriteString("- " + file.Name + "\n")
			prompt.WriteString(file.Content + "\n")
		}
	}

	prompt.WriteString("</Directory Information>\nDo not guess at any information. Only use the provided text. Is it useful to write a summary of this directory? If it is, reply with the yaml file. If it is not, reply with 'no'.")
	return prompt.String()
}

func writeKnowledgeBase(dir, ans string, dryRun bool) error {
	ymlPath := filepath.Join(dir, "ai_knowledge.yaml")
	if dryRun {
		slog.Debug("skipping AI prompt, would have written file to:", "path", ymlPath)
		return nil
	}

	if ans == "no" {
		slog.Debug("AI did not find the directory useful", "dir", dir, "ans", ans)
		return nil
	}
	ans = strings.TrimPrefix(ans, "```yaml\n")

	ans, _, _ = strings.Cut(ans, "```")
	f, err := os.Create(ymlPath)
	if err != nil {
		slog.Error("failed to create yaml file", "err", err)
		return err
	}
	defer f.Close()

	_, err = f.WriteString(ans)
	if err != nil {
		slog.Error("failed to write yaml file", "err", err)
		return err
	}
	return nil
}

func ReviewPullRequests(ctx context.Context, dir string, aiClient *aihelpers.AIClient, options *Options) error {
	currentBranch, defaultBranchName, err := getGitBranches()
	if err != nil {
		return err
	}

	if currentBranch == defaultBranchName {
		return fmt.Errorf("current branch %s, same as default branch %s", currentBranch, defaultBranchName)
	}

	diffOutput, err := getGitDiff(currentBranch, defaultBranchName)
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
	cmd := exec.Command("git", "diff", currentBranch, defaultBranchName)
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
