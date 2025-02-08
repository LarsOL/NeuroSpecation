# Project Overview

This repository contains a collection of tools and utilities to interface with OpenAI APIs using Go, manage AI-driven processes, and handle file system interactions efficiently. The components are organized into directories, each catering to specific functionalities.

## Directory Structure

### 1. `aihelpers`

- **Description**: Contains Go source files to interface with OpenAI APIs.
- **Key Business Processes**:
  - Initialize an OpenAI client.
  - Set the AI model for requests.
  - Handle prompt requests to OpenAI and manage responses.
- **Architectural Patterns**:
  - Client initialization pattern.
  - Singleton-like structure for ensuring consistent client configuration.
  - Error handling for network operations.
- **Main File**:
  - `aihelpers.go`: Implements all main functionalities for OpenAI client operations.
- **Dependencies**: Relies on the OpenAI Go SDK and `github.com/openai/openai-go` for API interactions.

### 2. `cmd`

- **Description**: A command-line interface (CLI) for managing AI processes, such as AI prompting, knowledge base updating, pull request reviews, and README generation.
- **Key Business Processes**:
  - Generate AI prompts and responses.
  - Update knowledge bases in YAML format.
  - Automate GitHub pull request reviews with AI.
  - Generate README files with AI summaries.
- **Architectural Patterns**:
  - CLI utility implemented with the `cobra` library.
  - Concurrent processing via `sync.WaitGroup`.
  - Structured logging with `context` and `log/slog`.
  - Configuration management via `viper`.
  - File operations for logging and writing outputs.
- **Key Files**:
  - `helpers.go`: Utility functions for AI prompt handling and logging.
  - `knowledgebase.go`: Manages knowledge base operations.
  - `pr.go`: Facilitates AI-assisted pull request reviews.
  - `readme.go`: Handles README generation.
  - `root.go`: Main command interface setup.
  - `version.go`: Outputs software version details.
- **External Dependencies**:
  - Cobra and Viper for CLI and configuration management.
  - OpenAI for generating AI responses.
- **Note**: Supports dry-run mode for testing and environmental variables for sensitive data management.

### 3. `dirhelper`

- **Description**: Provides utilities for traversing directories and handling files.
- **Business Processes**:
  - Directory and file processing.
  - Filtering based on specific criteria.
  - Custom directory actions.
- **Architectural Patterns**:
  - Functional programming with callback functions for flexibility.
  - Separation of concerns with distinct operation functions.
- **Key Files**:
  - `dirhelper.go`: Main implementation for directory operations.
- **Key Functions**:
  - `WalkDirectories`: Core directory traversal function with customizable callbacks.
  - `readDirectoryContents`: Reads directory contents.
  - `IsCodeFile`: Checks if a file is a code file.
  - `FilterNodes`: Filters unwanted files/directories.
- **Additional Notes**: Focuses on code file processing, with flexible logic for filtering directories such as `.git`, `.idea`, etc.

## General Information

- **Key External Dependencies**: Includes OpenAI SDK, Cobra, Viper, and logging packages.
- **Noteworthy Capabilities**: Implements rate-limiting and environmental variable handling for enhanced manageability.

This README provides a structural overview and key insights into the components, dependencies, and architectural patterns used within this repository. For detailed usage instructions or contributions, refer to the source code or internal documentation within each directory.