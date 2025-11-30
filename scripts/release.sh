#!/usr/bin/env bash

set -euo pipefail

# Always run from the repository root (one level above this script).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}/.."

if [ $# -lt 1 ]; then
  echo "Usage: $0 <version|major|minor|patch>" >&2
  exit 1
fi

INPUT="$1"

# Fail early if running as root — avoids creating root-owned git files.
if [ "$(id -u)" -eq 0 ]; then
  echo "Refusing to run as root. Run as your user (no sudo)." >&2
  exit 1
fi

# helper: normalize a plain semver (without leading v) and strip pre-release/build
normalize() {
  local ver="$1"
  # strip leading v
  ver="${ver#v}"
  # strip anything after a hyphen (pre-release) or plus (build)
  ver="$(echo "$ver" | sed -E 's/[-+].*$//')"
  echo "$ver"
}

# helper: bump semver
bump() {
  local ver="$1"
  local part="$2" # major|minor|patch
  IFS=. read -r major minor patch <<<"$ver"
  major=${major:-0}
  minor=${minor:-0}
  patch=${patch:-0}
  case "$part" in
    major)
      major=$((major + 1))
      minor=0
      patch=0
      ;;
    minor)
      minor=$((minor + 1))
      patch=0
      ;;
    patch)
      patch=$((patch + 1))
      ;;
    *)
      echo "unknown bump: $part" >&2; return 1
      ;;
  esac
  echo "$major.$minor.$patch"
}

# Determine target version
case "$INPUT" in
  major|minor|patch)
    # find latest tag
    if LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null); then
      base_ver=$(normalize "$LAST_TAG")
      new_ver=$(bump "$base_ver" "$INPUT")
    else
      # no tags yet: choose sensible defaults
      case "$INPUT" in
        major) new_ver="1.0.0" ;;
        minor) new_ver="0.1.0" ;;
        patch) new_ver="0.0.1" ;;
      esac
    fi
    VERSION="v${new_ver}"
    ;;
  *)
    # treat as explicit version
    ver_norm=$(normalize "$INPUT")
    VERSION="v${ver_norm}"
    ;;
esac

echo "Preparing release $VERSION"

# Ensure working tree is clean
if [ -n "$(git status --porcelain)" ]; then
  echo "Working tree is not clean. Commit or stash changes before running release." >&2
  git status --porcelain
  exit 1
fi

# Ensure tag does not already exist
if git rev-parse --verify --quiet "refs/tags/$VERSION" >/dev/null; then
  echo "Tag $VERSION already exists. Aborting." >&2
  exit 1
fi

# Determine the previous tag, if any (for changelog generation)
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

# Write changelog safely using a temporary file and explicit permissions
if [ -f CHANGELOG.md ]; then
  tmp_file=$(mktemp)
  printf "%s%s" "$ENTRY" "$(cat CHANGELOG.md)" > "$tmp_file"
  chmod --reference=CHANGELOG.md "$tmp_file" || true
  mv "$tmp_file" CHANGELOG.md
else
  printf "# Changelog\n\n%s" "$ENTRY" > CHANGELOG.md
fi

# Update README version snippet (e.g., VERSION=v1.2.3) so docs show the latest tag.
if grep -qE "VERSION=v[0-9]+\.[0-9]+\.[0-9]+" README.md; then
  tmp_readme=$(mktemp)
  sed -E "s/VERSION=v[0-9]+\.[0-9]+\.[0-9]+/VERSION=${VERSION}/" README.md > "$tmp_readme"
  mv "$tmp_readme" README.md
else
  echo "Warning: README.md does not contain a VERSION=vX.Y.Z snippet; skipping README version update."
fi

# Stage and commit changelog explicitly
git add -- CHANGELOG.md
git add -- README.md || true
git commit -m "chore: release $VERSION"

# Create an annotated tag so that `git push --follow-tags` will include it.
git tag -a "$VERSION" -m "Release $VERSION"

echo "Release $VERSION prepared."
echo "Changelog updated and git tag created." 
echo "Next steps:"
echo "  make push VERSION=$VERSION"
