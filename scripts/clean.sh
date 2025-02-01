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

cd ..
# Process options
for arg in "$@"; do
    case $arg in
        --prompts)
            find . -type f \( -name "ai_prompt.txt" -o -name "ai_knowledge_prompt.txt" -o -name "ai_summary_prompt.txt" -o -name "ai_review_prompt.txt" \) -delete
            ;;
        --all)
            find . -type f -name "ai_*" -delete
            ;;
        *)
            usage
            ;;
    esac
done
