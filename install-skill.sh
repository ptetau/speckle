#!/usr/bin/env bash
# install-skill.sh — install the speckle Claude Code skill globally
# Run from the speckle repo root: ./install-skill.sh
set -euo pipefail

REPO_DIR="$(cd "$(dirname "$0")" && pwd)"
SKILL_SRC="$REPO_DIR/.claude/skills/speckle"
SKILL_DEST="$HOME/.claude/skills/speckle"

echo "Installing speckle skill to $SKILL_DEST ..."
mkdir -p "$SKILL_DEST"
cp "$SKILL_SRC/SKILL.md" "$SKILL_DEST/SKILL.md"
echo "  Skill installed."

echo "Installing speckle binary (go install) ..."
go install github.com/ptetau/speckle@latest
echo "  Binary installed."

echo ""
echo "Done. /speckle is now available in all Claude Code sessions."
echo "Run ./install-skill.sh again at any time to update to the latest version."
