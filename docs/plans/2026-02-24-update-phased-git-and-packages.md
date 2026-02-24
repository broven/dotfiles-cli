# Update Command Phased Git+Packages Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Move Homebrew/NPM sync from `link` to `update`, and make `update` run git and package sync as independent phases where local repo changes skip git pull.

**Architecture:** Keep package-manager sync logic in `src/package_managers.go` and call it from `Update`. Add a small git execution abstraction inside `command_update.go` to support `DOTFILES_GIT_COMMAND`, dirty-worktree detection, and test stubbing. Remove package-manager side effects from `Link` so it only handles symlinks.

**Tech Stack:** Go, standard library `os/exec`, existing repository/config helpers in `src`.

---

### Task 1: Link behavior boundary

**Files:**
- Modify: `src/command_link.go`
- Modify: `src/command_link_test.go`

1. Write failing tests asserting `Link` ignores package manager sections and package-only config no longer succeeds.
2. Run targeted tests and confirm failures.
3. Remove package-sync call from `Link` and keep existing link error behavior.
4. Re-run link tests.

### Task 2: Update phased workflow

**Files:**
- Modify: `src/command_update.go`
- Modify: `src/command_update_test.go`

1. Write failing tests for:
- dirty repo skips git pull but still syncs packages,
- git phase failure still attempts package sync,
- `DOTFILES_GIT_COMMAND` is used,
- returned error aggregates phase failures.
2. Run targeted update tests and confirm failures.
3. Implement phased update flow with injectable helpers for testability.
4. Re-run update tests.

### Task 3: Full verification

**Files:**
- Verify: `src/*`

1. Run `go test -v ./src`.
2. Fix regressions if any.
3. Summarize behavior changes and evidence.
