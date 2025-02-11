package cmd

import (
	"context"
	"fmt"
	"github.com/LarsOL/NeuroSpecation/aihelpers"
	"github.com/LarsOL/NeuroSpecation/dirhelper"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var knowledgebaseCmd = &cobra.Command{
	Use:   "knowledgebase",
	Short: "Update the knowledge base",
	Run: func(cmd *cobra.Command, args []string) {
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

		slog.Info("Updating AI knowledge base")
		err := UpdateKnowledgeBase(cmd.Context(), directory, aiClient)
		if err != nil {
			slog.Error("Error updating knowledge base", "err", err)
			os.Exit(1)
		}
		slog.Info("finished updating all knowledge base files")
	},
}

const throttleKey = "throttle"

func init() {
	rootCmd.AddCommand(knowledgebaseCmd)

	knowledgebaseCmd.PersistentFlags().Int(throttleKey, 500, "API limit in requests per minute")

	err := viper.BindPFlags(knowledgebaseCmd.PersistentFlags())
	if err != nil {
		slog.Error("could not bind to persistent flags:", "err", err)
		os.Exit(1)
	}
}

const KnowledgeBasePrompt = "You are a seasoned staff software engineer. Your task is to analyze the given code directory and generate a detailed YAML summary that captures all the essential knowledge needed to understand its purpose and role within the larger codebase. Although the output is for machine consumption, it must be clear, logically organized, and information-dense.\n\nYour YAML summary should include the following sections:\n\n- **business_processes**: Identify and explain the core business processes or domain-specific operations that this directory supports.\n- **module_overview**: Provide a concise description of the module’s purpose, responsibilities, and primary functionality.\n- **architectural_patterns**: Describe any architectural patterns, design principles, or frameworks used within the directory.\n- **key_files**: List and explain the most critical files or components, highlighting their roles.\n- **inter_module_relationships**: Identify and describe the key dependencies, integrations, or links to other modules in the codebase.\n- **additional_insights**: Include any other relevant details (such as performance considerations, security concerns, testing strategies, or scalability issues) that would be valuable for a skilled engineer to understand this directory.\n\nOutput only valid YAML.\n\nBelow is the content or description of the directory:\n"

// TODO: Move concurency throttle into aiClient so that it is respected globally
func UpdateKnowledgeBase(ctx context.Context, dir string, aiClient *aihelpers.AIClient) error {
	// Rate limit to concurrencyRPMThrottle requests per minute
	reqPerMin := viper.GetInt(throttleKey)
	throttle := make(chan time.Time, reqPerMin)
	go func() {
		ticker := time.NewTicker(time.Minute / time.Duration(reqPerMin))
		defer ticker.Stop()
		for t := range ticker.C {
			select {
			case throttle <- t:
			case <-ctx.Done():
				return // exit goroutine when surrounding function returns
			}
		}
	}()

	var wg sync.WaitGroup
	err := dirhelper.WalkDirectories(dir, func(dir string, files []dirhelper.FileContent, subdirs []string) error {
		l := loggerFromCtx(ctx)
		l.With("dir", dir)
		ctx = setLoggerToCtx(ctx, l)
		if len(files) == 0 {
			slog.Debug("Skipping directory with no valid files", "dir", dir)
			return nil
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			prompt := createKnowledgeBasePrompt(dir, files, subdirs)
			if viper.GetBool(logPromptKey) {
				if err := logPromptToFile(dir, "ai_knowledge_prompt.txt", prompt); err != nil {
					slog.Error("error logging prompt", "dir", dir, "err", err)
				}
			}
			<-throttle
			ans, err := promptAI(ctx, aiClient, prompt, viper.GetBool(dryRunKey))
			if err != nil {
				slog.Error("error prompting AI", "dir", dir, "err", err)
				return
			}

			err = writeKnowledgeBase(dir, ans, viper.GetBool(dryRunKey))
			if err != nil {
				slog.Error("error writing knowledge base file", "dir", dir, "err", err)
				return
			}
		}()
		return nil
	}, nil)
	wg.Wait()
	if err != nil {
		return fmt.Errorf("failed to walk directories: %w", err)
	}
	slog.Info("finished updating all knowledge base files")
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

	if strings.EqualFold(ans, "no") || strings.EqualFold(ans, "no.") {
		slog.Debug("AI did not find the directory useful", "dir", dir, "ans", ans)
		return nil
	}
	if strings.Count(ans, "```") < 2 {
		slog.Error("expected a code block as answer", "dir", dir, "ans", ans)
		return nil
	}
	ans, err := extractBlock(ans, "yaml")
	if err != nil {
		slog.Error("expected knowledge base file to contain a yaml block", "err", err)
	}
	if ans == "" {
		return err
	}
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
