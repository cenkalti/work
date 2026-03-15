#!/bin/bash
set -euo pipefail

REPO_DIR="$(cd "$(dirname "$0")" && pwd)"

# Install binary
echo "Installing work binary..."
go install "${REPO_DIR}/cmd/work/"

# Source zsh helper
SOURCE_LINE="source ${REPO_DIR}/shell/work.zsh"
if ! grep -qF "$SOURCE_LINE" ~/.zshrc 2>/dev/null; then
  echo "Adding shell integration to ~/.zshrc..."
  echo "$SOURCE_LINE" >> ~/.zshrc
else
  echo "Shell integration already in ~/.zshrc"
fi

# Copy slash commands
echo "Copying slash commands..."
mkdir -p ~/.claude/commands
cp "${REPO_DIR}/commands/"*.md ~/.claude/commands/

echo "Done. Restart your shell or run: source ~/.zshrc"
