#!/usr/bin/env bash

# Function to display usage
usage() {
    echo "Usage: $0 [--prompts] [--all]"
    echo "  --prompts  Delete all prompt-related files"
    echo "  --all      Delete all AI-related files"
    exit 1
}

# Check for options
if [[ $# -eq 0 ]]; then
    usage
fi

# Process options
for arg in "$@"; do
    case $arg in
        --prompts)
            rm -rf ai_prompt.txt ai_knowledge_prompt.txt ai_summary_prompt.txt ai_review_prompt.txt
            ;;
        --all)
            rm -rf ai_*
            ;;
        *)
            usage
            ;;
    esac
done
