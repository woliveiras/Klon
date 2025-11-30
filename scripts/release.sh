#!/usr/bin/env bash

set -euo pipefail

# Always run from the repository root (one level above this script).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}/.."

if [ $# -lt 1 ]; then
  echo "Usage: $0 vX.Y.Z" >&2
  exit 1
fi

VERSION="$1"

# Normalize version to start with 'v'
if [[ "$VERSION" != v* ]]; then
  VERSION="v$VERSION"
fi

echo "Preparing release $VERSION"

# Determine the previous tag, if any
LAST_TAG=""
if LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null); then
  echo "Last tag detected: $LAST_TAG"
else
  LAST_TAG=""
  echo "No previous tags detected; generating changelog from initial commit."
fi

if [ -n "$LAST_TAG" ]; then
  COMMITS=$(git log --oneline "${LAST_TAG}..HEAD")
else
  COMMITS=$(git log --oneline)
fi

DATE=$(date +%F)

ENTRY="## $VERSION - $DATE

"

if [ -z "$COMMITS" ]; then
  ENTRY+="- No changes recorded.

"
else
  while IFS= read -r line; do
    hash=${line%% *}
    msg=${line#* }
    ENTRY+="- $msg ($hash)
"
  done <<< "$COMMITS"
  ENTRY+="
"
fi

if [ -f CHANGELOG.md ]; then
  tmp_file=$(mktemp)
  printf "%s%s" "$ENTRY" "$(cat CHANGELOG.md)" > "$tmp_file"
  mv "$tmp_file" CHANGELOG.md
else
  printf "# Changelog\n\n%s" "$ENTRY" > CHANGELOG.md
fi

git add CHANGELOG.md
git commit -m "chore: release $VERSION"

# Create an annotated tag so that `git push --follow-tags` will include it.
git tag -a "$VERSION" -m "Release $VERSION"

echo "Release $VERSION prepared."
echo "Changelog updated and git tag created."
echo "Next steps:"
echo "  git push origin main --follow-tags"
