# NeuroSpectation

AI repo assistant

# Getting started

## CLI

```
A ai repo assistant

Usage:
  neurospecation [command]

Available Commands:
  completion    Generate the autocompletion script for the specified shell
  help          Help about any command
  knowledgebase Update the knowledge base
  pr            Review pull requests
  readme        Create a summary of the directory
  version       Print neurospectation version

Flags:
      --config string   config file (default is $HOME/.NeuroSpecation.yaml)
  -d, --debug           Enable debug logging
      --dir string      Directory to run on
      --dry-run         Enable dry-run mode
  -h, --help            help for neurospecation
      --log-prompts     Debug: Log prompts to file
  -m, --model string    The model to use for AI requests (default "gpt-4o")

Use "neurospecation [command] --help" for more information about a command.

```

## CI/CD
Add a new workflow to <project>/.github/workflows/pr.yml

```yaml
name: PR Review

on:
  pull_request:
    branches:
      - main

permissions:
  contents: read
  pull-requests: write

jobs:
  PRReview:
    name: Neurospection PR Review
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        id: checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Ensures we have full history

      - name: Fetch target branch
        run: |
          TARGET_BRANCH="${{ github.base_ref }}"  # Use the PR's base branch
          echo "Fetching target branch: $TARGET_BRANCH"
          git fetch origin $TARGET_BRANCH
          git branch --track $TARGET_BRANCH origin/$TARGET_BRANCH || true

      - name: Neurospecation Review
        uses: LarsOL/NeuroSpecation@v0.0.3
        with:
          review: "pr"
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          GITHUB_PR_NUMBER: ${{ github.event.number }}

```


