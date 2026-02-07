#!/bin/bash
set -e

# update-release-notes.sh
# Generates a changelog from conventional commits since the last tag
# and prepends it to RELEASE_NOTES.md.

NOTES_FILE="RELEASE_NOTES.md"
TEMP_FILE=$(mktemp)

# Find latest tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

if [ -z "$LATEST_TAG" ]; then
  echo "No tags found. Generating changelog from start."
  RANGE="HEAD"
else
  echo "Generating changelog since $LATEST_TAG..."
  RANGE="$LATEST_TAG..HEAD"
fi

# Date
TODAY=$(date +%Y-%m-%d)

# Header
echo "## [Unreleased] - $TODAY" >"$TEMP_FILE"
echo "" >>"$TEMP_FILE"

# Extract and group commits
echo "### Features" >>"$TEMP_FILE"
git log "$RANGE" --pretty=format:"- %s (%h)" --no-merges | grep -E "^- feat" | sed 's/^- feat.*: /- /' >>"$TEMP_FILE" || true
echo "" >>"$TEMP_FILE"

echo "### Fixes" >>"$TEMP_FILE"
git log "$RANGE" --pretty=format:"- %s (%h)" --no-merges | grep -E "^- fix" | sed 's/^- fix.*: /- /' >>"$TEMP_FILE" || true
echo "" >>"$TEMP_FILE"

echo "### Documentation" >>"$TEMP_FILE"
git log "$RANGE" --pretty=format:"- %s (%h)" --no-merges | grep -E "^- docs" | sed 's/^- docs.*: /- /' >>"$TEMP_FILE" || true
echo "" >>"$TEMP_FILE"

echo "### Maintenance" >>"$TEMP_FILE"
git log "$RANGE" --pretty=format:"- %s (%h)" --no-merges | grep -E "^- (chore|test|ci|refactor|perf)" | sed 's/^- [a-z]*.*: /- /' >>"$TEMP_FILE" || true
echo "" >>"$TEMP_FILE"

# Append existing notes
if [ -f "$NOTES_FILE" ]; then
  cat "$NOTES_FILE" >>"$TEMP_FILE"
fi

# Move back
mv "$TEMP_FILE" "$NOTES_FILE"

echo "Updated $NOTES_FILE"
