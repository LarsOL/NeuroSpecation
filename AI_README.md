# NeuroSpecation Repository

## Overview
NeuroSpecation is a tool designed to assist software engineers by utilizing AI to review code changes and summarize directories. The application follows a layered architecture that enhances maintainability.

## Key Business Processes
- **Identify and Explain Key Business Processes**: Analyzes and summarizes crucial business structures and workflows.
  
## Module Description
- **Name**: NeuroSpecation
- **Version**: 0.73.0
- **Created By**: Lars Lawoko
- **License**: MIT

## Architectural Patterns
- **Layered Architecture**: Enhances maintainability through the separation of concerns.
- **Command-Line Interface (CLI)**: Interacts with users via command-line arguments.
- **Adapter Pattern**: Integrates with external services (OpenAI API) to adapt responses to the tool's requirements.

## Key Files
- `cmd/neurospecation/main.go`: Main entry point managing command-line arguments and orchestrating actions.
- `aihelpers/aihelpers.go`: AI Client for interacting with the OpenAI API.
- `dirhelper/dirhelper.go`: Utility functions for directory traversal and file filtering.
- `.aider.chat.history.md`: History of AI chat interactions.
- `clean.sh`: Script to remove all AI-related files.
- `run.sh`: Script to run the application.
- `go.mod` and `go.sum`: Go module dependency files.
- `LICENSE`: License file.

## Key Links to Other Modules
- **aihelpers**: Handles interactions with AI services.
- **dirhelper**: Provides directory manipulation and file reading/writing functionalities.

## Usage Instructions
1. Run the application using `./run.sh`.
2. Use `./clean.sh` to remove all AI-related files.

## Environment Variables
- **OPENAI_API_KEY**: Required for authenticating requests to the OpenAI API.

## Additional Information
- The application generates YAML or markdown summaries based on AI-generated insights from provided files and helps in reviewing pull requests by comparing differences between branches.
- Key flags for execution:
  - `dryRun`: Skips actual execution and logs intended actions.
  - `debug`: Enables detailed logging.
  - `logPrompt`: Logs prompts sent to the AI.
  - `updateKnowledge`: Triggers the update of the knowledge base.
  - `createReadme`: Generates a summary of the directory.
  - `reviewPR`: Triggers pull request reviews.

---

For further information, refer to the comments and documentation within the codebase.