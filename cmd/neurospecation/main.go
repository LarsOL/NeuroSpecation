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
		err := UpdateKnowledgeBase(ctx, directory, aiClient, o)
		if err != nil {
			slog.Error("Error updating knowledge base", "err", err)
			os.Exit(1)
		}
	}
	if o.createReadme {
		err := CreateReadMe(ctx, directory, aiClient, o)
		if err != nil {
			slog.Error("Error creating readme", "err", err)
			os.Exit(1)
		}
	}
	if o.reviewPR {
		err := ReviewPullRequests(ctx, directory, aiClient, o)
		if err != nil {
			slog.Error("Error reviewing pull requests", "err", err)
			os.Exit(1)
		}
	}
}

func CreateReadMe(ctx context.Context, dir string, aiClient *aihelpers.AIClient, options *Options) error {
	prompt := ReadmePrompt + "\n<Summarised AI knowledge base>\n"
	gatherAIKnowledge := func(d string, files []dirhelper.FileContent, subdirs []string) error {
		slog.Info("Processing Directory", "Dir", d)

		if len(files) == 0 {
			slog.Debug("No knowledge files in directory", "Dir", d)
		} else {
			for _, file := range files {
				slog.Debug("Processing file", "File", file.Name)
				prompt += "- " + file.FullPath() + "\n"
				prompt += file.Content + "\n"
			}
		}
		return nil
	}
	onlyAIKnowledgeFiles := func(node fs.DirEntry) bool {
		if node.IsDir() {
			slog.Debug("Allowing sub dir", "dir", node.Name())
			return true
		}
		if node.Name() == "ai_knowledge.yaml" {
			slog.Debug("Allowing file", "File", node.Name())
			return true
		}
		slog.Debug("Rejecting file", "File", node.Name())
		return false
	}

	err := dirhelper.WalkDirectories(dir, gatherAIKnowledge, onlyAIKnowledgeFiles)
	if err != nil {
		slog.Error("Error walking directories", "err", err)
		os.Exit(1)
	}
	prompt += "</Summarised AI knowledge base>\n"

	slog.Debug("Prompting AI", "prompt", prompt)
	var ans string
	if !options.dryRun {
		var err error
		ans, err = aiClient.Prompt(context.TODO(), aihelpers.PromptRequest{
			Prompt: prompt,
		})
		if err != nil {
			return fmt.Errorf("failed to prompt AI: %w", err)
		}
	}

	if options.logPrompt {
		fl, err := os.Create(filepath.Join(dir, "ai_summary_prompt.txt"))
		if err != nil {
			slog.Error("failed to create ai prompt file", "err", err)
			return err
		}

		_, err = fl.WriteString(prompt)
		if err != nil {
			slog.Error("failed to write ai prompt file", "err", err)
			return err
		}
	}

	ymlPath := filepath.Join(dir, "AI_README.md")
	if options.dryRun {
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

	_, err = f.WriteString(ans)
	if err != nil {
		slog.Error("failed to write yaml file", "err", err)
		return err
	}
	return nil
}

func UpdateKnowledgeBase(ctx context.Context, dir string, aiClient *aihelpers.AIClient, options *Options) error {
	updateAIKnowledge := func(d string, files []dirhelper.FileContent, subdirs []string) error {
		slog.Info("Processing Directory", "Dir", d)
		prompt := KnowledgeBasePrompt + "\n<Directory Information>\n"
		prompt += "Directory: " + d + "\n"

		if len(subdirs) == 0 {
			prompt += "No subdirectories\n"
		} else {
			prompt += "Subdirectories:\n"
			for _, subdir := range subdirs {
				prompt += "- " + subdir + "\n"
			}
		}

		if len(files) == 0 {
			slog.Debug("No valid files in dir", "dir", dir)
			return nil
		} else {
			prompt += "Files:\n"
			for _, file := range files {
				prompt += "- " + file.Name + "\n"
				prompt += file.Content + "\n"
			}
		}

		prompt += "</Directory Information>\nDo not guess at any information. Only use the provided text. Is it useful to write a summary of this directory? If it is, reply with the yaml file. If it is not, reply with 'no'."
		slog.Debug("Prompting AI", "prompt", prompt)

		if options.logPrompt {
			fl, err := os.Create(filepath.Join(d, "ai_knowledge_prompt.txt"))
			if err != nil {
				slog.Error("failed to create ai prompt file", "err", err)
				return err
			}

			_, err = fl.WriteString(prompt)
			if err != nil {
				slog.Error("failed to write ai prompt file", "err", err)
				return err
			}
		}

		ymlPath := filepath.Join(d, "ai_knowledge.yaml")
		var ans string
		if !options.dryRun {
			var err error
			ans, err = aiClient.Prompt(context.TODO(), aihelpers.PromptRequest{
				Prompt: prompt,
			})
			if err != nil {
				return fmt.Errorf("failed to prompt AI: %w", err)
			}
		}

		if options.dryRun {
			slog.Debug("skipping AI prompt, would have written file to:", "path", ymlPath)
			return nil
		}

		if ans == "no" {
			slog.Debug("AI did not find the directory useful", "dir", d, "ans", ans)
			return nil
		}
		ans = strings.TrimPrefix(ans, "```yaml\n")
		ans = strings.TrimSuffix(ans, "\n```")
		f, err := os.Create(ymlPath)
		if err != nil {
			slog.Error("failed to create yaml file", "err", err)
			return err
		}

		_, err = f.WriteString(ans)
		if err != nil {
			slog.Error("failed to write yaml file", "err", err)
			return err
		}

		return nil
	}
	err := dirhelper.WalkDirectories(dir, updateAIKnowledge, nil)
	if err != nil {
		return fmt.Errorf("failed to walk directories: %w", err)
	}
	return nil
}

func ReviewPullRequests(ctx context.Context, dir string, aiClient *aihelpers.AIClient, options *Options) error {
	// Get the current branch name
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	currentBranch := strings.TrimSpace(string(output))

	// Get the default branch name (usually 'main' or 'master')
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "origin/HEAD")
	defaultBranch, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get default branch: %w", err)
	}
	defaultBranchName := strings.TrimSpace(string(defaultBranch))
	defaultBranchName = strings.TrimPrefix(defaultBranchName, "origin/")

	// Get the diff between the current branch and the default branch
	cmd = exec.Command("git", "diff", string(currentBranch), defaultBranchName)
	diffOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get diff between currentbranch %s and default branch %s: %w", currentBranch, defaultBranch, err)
	}

	// Gather context from ai_knowledge.yaml for changed files
	changedFiles := strings.Split(string(diffOutput), "\n")
	knowledgeContent := ""

	for _, line := range changedFiles {
		if strings.HasPrefix(line, "diff --git") {
			parts := strings.Split(line, " ")
			if len(parts) > 2 {
				filePath := strings.TrimPrefix(parts[2], "a/")
				// Rather than the passed dir, figure out the git root and use that to calculate the full path. ai!
				fullPath := filepath.Join(dir, filePath)
				dirPath := filepath.Dir(fullPath)
				knowledgePath := filepath.Join(dirPath, "ai_knowledge.yaml")
				content, err := os.ReadFile(knowledgePath)
				if err == nil {
					knowledgeContent += string(content) + "\n"
				}
			}
		}
	}

	prompt := ReviewPrompt + "\n<Repo Context>\n" + string(knowledgeContent) + "\n</Repo Context>\n" + "\n<Diff>\n" + string(diffOutput) + "\n</Diff>\n"

	slog.Debug("Prompting AI for review", "prompt", prompt)
	var reviewOutput string
	if !options.dryRun {
		var err error
		reviewOutput, err = aiClient.Prompt(ctx, aihelpers.PromptRequest{
			Prompt: prompt,
		})
		if err != nil {
			return fmt.Errorf("failed to prompt AI: %w", err)
		}
	}

	if options.logPrompt {
		fl, err := os.Create(filepath.Join(dir, "ai_review_prompt.txt"))
		if err != nil {
			slog.Error("failed to create ai review prompt file", "err", err)
			return err
		}

		_, err = fl.WriteString(prompt)
		if err != nil {
			slog.Error("failed to write ai review prompt file", "err", err)
			return err
		}
	}

	reviewFilePath := filepath.Join(dir, "AI-Review.md")
	if options.dryRun {
		slog.Debug("skipping AI review, would have written file to:", "path", reviewFilePath)
		return nil
	}

	f, err := os.Create(reviewFilePath)
	if err != nil {
		slog.Error("failed to create review file", "err", err)
		return err
	}

	_, err = f.WriteString(reviewOutput)
	if err != nil {
		slog.Error("failed to write review file", "err", err)
		return err
	}

	return nil
}
