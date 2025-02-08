package cmd

import (
	"context"
	"fmt"
	"github.com/LarsOL/NeuroSpecation/aihelpers"
	"github.com/LarsOL/NeuroSpecation/dirhelper"
	"github.com/spf13/viper"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var readmeCmd = &cobra.Command{
	Use:   "readme",
	Short: "Create a summary of the directory",
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

		slog.Info("Creating AI README")
		err := CreateReadMe(ctx, directory, aiClient)
		if err != nil {
			slog.Error("Error creating readme", "err", err)
			os.Exit(1)
		}
		slog.Info("finished creating readme")
	},
}

func init() {
	rootCmd.AddCommand(readmeCmd)
}

const ReadmePrompt = "Create a README file for this directory. This should contain a concise representation of all the key information needed for a skilled software engineer to understand the repo. Do not guess at any information. Only use the provided text. Reply with a markdown file."

func CreateReadMe(ctx context.Context, dir string, aiClient *aihelpers.AIClient) error {
	prompt, err := gatherAIKnowledgeForReadMe(dir)
	if err != nil {
		return err
	}

	if viper.GetBool(logPromptKey) {
		if err := logPromptToFile(dir, "ai_readme_prompt.txt", prompt); err != nil {
			return err
		}
	}

	ans, err := promptAI(ctx, aiClient, prompt, viper.GetBool(dryRunKey))
	if err != nil {
		return err
	}

	return writeReadMe(dir, ans, viper.GetBool(dryRunKey))
}

func gatherAIKnowledgeForReadMe(dir string) (string, error) {
	var prompt strings.Builder
	prompt.WriteString(ReadmePrompt)
	prompt.WriteString("\n<Summarised AI knowledge base>\n")
	err := dirhelper.WalkDirectories(dir, func(d string, files []dirhelper.FileContent, subdirs []string) error {
		slog.Debug("Processing Directory", "Dir", d)
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
