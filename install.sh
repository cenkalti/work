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

# Symlink slash commands
echo "Symlinking slash commands..."
mkdir -p ~/.claude/commands
for cmd in "${REPO_DIR}/commands/"*.md; do
  ln -sf "$cmd" ~/.claude/commands/
done

# Install SessionStart hook into ~/.claude/settings.json
echo "Installing work context hook into ~/.claude/settings.json..."
python3 - <<'EOF'
import json, os, sys

path = os.path.expanduser("~/.claude/settings.json")
try:
    with open(path) as f:
        settings = json.load(f)
except (FileNotFoundError, json.JSONDecodeError):
    settings = {}

session_start = settings.setdefault("hooks", {}).setdefault("SessionStart", [])
for entry in session_start:
    for h in entry.get("hooks", []):
        if h.get("command") == "work context":
            print("  work context hook already installed")
            sys.exit(0)

session_start.append({"matcher": "", "hooks": [{"type": "command", "command": "work context"}]})
os.makedirs(os.path.dirname(path), exist_ok=True)
with open(path, "w") as f:
    json.dump(settings, f, indent=2)
print("  Added SessionStart hook to ~/.claude/settings.json")
EOF

echo "Done. Restart your shell or run: source ~/.zshrc"
