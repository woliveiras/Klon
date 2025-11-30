# Release guide

This document describes how to create and publish a release for Klon using
the repository's `Makefile` and `scripts/release.sh` helper. The workflow is
designed to be safe (no `sudo`) and to let you review changes before pushing.

Prerequisites

- You have a working Git remotes configured (typically `origin` points to GitHub).
- You are *not* running commands as `root` — do not use `sudo` for these steps.
- Your working tree is clean (no uncommitted changes) before preparing a release.
- You have permission to push to `origin` or create tags on the repository.

1) Decide the bump level

The release tooling supports two ways to choose the new version:

- Explicit version: `VERSION=v1.2.3`
- Semantic bump token: `RELEASE=major|minor|patch`

If you use a bump token the script will compute the next version automatically
from the most recent tag.

2) Prepare the release (local only)

Running `prepare` will update `CHANGELOG.md`, create a commit and tag *locally*.
It will not push anything to the remote so you can inspect the results first.

Examples:

```
# prepare a patch bump (script finds latest tag and bumps patch)
make prepare RELEASE=patch

# or prepare an explicit version
make prepare VERSION=v1.2.0
```

After `make prepare` you can inspect the created commit and tag:

```
git --no-pager log --decorate --oneline -n 5
git --no-pager tag --list --sort=-v:refname | head -n 5
```

3) Push the release (remote)

When you're happy with the commit and tag, push to the remote. `make push`
will auto-detect the most recent tag if you don't pass `VERSION`.

```
# push using the most recent tag created by prepare
make push

# or push a specific tag
make push VERSION=v1.2.0
```

The `push` target uses `git push origin main --follow-tags` so both the commit
and the tag are sent to the remote.

4) Post-release

- Optionally create a GitHub release from the pushed tag using the web UI.
- pkg.go.dev will automatically index public modules; to force a refresh open
  https://pkg.go.dev/github.com/woliveiras/klon and click "Request".

Safety notes

- Never run `make` with `sudo` in this repository. The scripts intentionally
  refuse to run as root and using `sudo` can create root-owned files in `.git`.
- `make prepare` will commit and tag locally. Inspect before pushing.

Troubleshooting

- If `make prepare` fails because the working tree is not clean, run:

```
git status --porcelain
git add/commit or git stash
```

- If you run into permission problems in `.git` (files owned by `root`) you can fix them:

```
sudo chown -R $(whoami) .git
```

But prefer to avoid `sudo` during normal development and release operations.

Questions or changes

If you'd like the release flow to be more automated (for example automatically
pushing in `prepare` or publishing a GitHub release), I can update the
Makefile and script accordingly — tell me which behaviour you prefer.
