#!/bin/sh

# Define the source of the hook script
HOOK_SRC="./hooks/pre-commit"

# Define the destination in the .git/hooks directory
HOOK_DEST=".git/hooks/pre-commit"

# Check if the .git directory exists (i.e., the script is run inside a Git repo)
if [ ! -d ".git" ]; then
  echo "This is not a Git repository!"
  exit 1
fi

# Copy the pre-commit hook script to the Git hooks directory
cp "$HOOK_SRC" "$HOOK_DEST"
chmod +x "$HOOK_DEST"

echo "Pre-commit hook installed successfully."
