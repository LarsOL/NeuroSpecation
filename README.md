# NeuroSpectation 

This repository contains components designed for enhancing interactions with OpenAI's API and supporting AI-based operations through command-line tools. Below is a summary of the available directories and their functionalities.

## Usage
```yaml
name: Example Workflow

on:
  workflow_dispatch:

jobs:
  example:
    name: neurospecation
    runs-on: ubuntu-latest

    steps:
      - name: Example
        id: example
        uses: actions/neurospecation@main
```

## Directories Overview

### 1. aihelpers

- **Description**: A package for communicating with OpenAI's API via a client-server architectural pattern.
- **Primary Use**: Facilitates prompt-based text generation by interacting with OpenAI's API.
- **Implemented In**: Go
- **Key Components**:
  - `AIClient`: Data structure representing an OpenAI client with essential fields such as `APIKey`, `Model`, and `Client`.
  - `NewOpenAIClient`: Initializes a new AI client using the API key and model.
  - `SetModel`: Method to change the AI model for requests.
  - `PromptRequest`: Structure for making prompt requests, including fields for prompt text, max tokens, and temperature.
  - `Prompt`: Executes prompts and retrieves responses from OpenAI.
- **Key File**: `aihelpers.go`: Houses the core logic for API interaction.
- **Dependencies**: Relies on the `openai-go` library for API calls.

### 2. cmd/neurospecation

- **Description**: Contains CLI-based code for various AI operations such as updating a knowledge base, generating READMEs, and reviewing pull requests.
- **Purpose**: Offers tools for AI-based document creation and code analysis.
- **Architectural Pattern**: CLI, allowing users to manage operations with command-line flags.
- **Key File**: `main.go`: Acts as the entry point, setting up configurations and operations.
- **Functional Features**:
  - Update AI knowledge base.
  - Generate documentation.
  - Review pull requests.
- **Execution**: Controlled via command-line flags (e.g., `-uk` for knowledge update, `-dr` for dry-run).
- **Dependencies**: Uses OpenAI API, configured by the `OPENAI_API_KEY` environment variable.

### 3. dirhelper

- **Description**: Supports directory traversal and file filtering.
- **Purpose**: Efficiently processes directories and files, identifying code files through customizable filters.
- **Key File**: `dirhelper.go`: Implements traversal, filtering, and content reading logic.
- **Functions**:
  - `IsCodeFile`: Identifies code files based on extensions.
  - `FilterNodes`: Excludes non-code files and unwanted directories.
  - `WalkDirectories`: Traverses directories using `Walk` function.
- **Features**: Modular design, utilizing Go's filesystem libraries.

## Development and Maintenance

- **Enhancements**:
  - Convert `Prompt` function in `aihelpers` to a streaming version for large data.
- **Dependencies**:
  - OpenAI API and `openai-go` library are central dependencies.
  
For any further details or assistance with using this repository, please refer to in-file comments and function documentation.
```
