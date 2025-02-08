package main

import (
	"context"
	"fmt"
	"github.com/LarsOL/NeuroSpecation/aihelpers"
	"github.com/LarsOL/NeuroSpecation/dirhelper"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const ReadmePrompt = "Create a README file for this directory. This should contain a concise representation of all the key information needed for a skilled software engineer to understand the repo. Do not guess at any information. Only use the provided text. Reply with a markdown file."

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
