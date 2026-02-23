# Repository Guidelines

## Project Structure & Module Organization
This repository is a Go CLI for managing dotfiles symlinks.
- `main.go`: entrypoint for the `dotfiles` command.
- `src/`: core implementation and unit tests.
  - Command logic: `src/command_*.go` (`clone`, `link`, `list`, `clean`, `update`).
  - Supporting modules: `src/mappings.go`, `src/repository.go`, `src/version.go`, `src/absolute_path.go`.
  - Tests live next to code as `*_test.go`.
- `.github/workflows/`: CI (`ci.yml`) and release (`release.yml`).

## Build, Test, and Development Commands
Use standard Go tooling from the repository root:
- `go build` builds the CLI binary.
- `go test -v ./src` runs unit tests.
- `go test -v -race -coverprofile=coverage.txt -covermode=atomic ./src` matches CI coverage/race settings.
- `go run . list` runs the app locally without installing.

## Coding Style & Naming Conventions
- Follow idiomatic Go and keep code `gofmt`-formatted (`gofmt -w .`).
- Keep packages/files focused; command handlers use `command_<name>.go`.
- Use descriptive exported names (`CamelCase`) and unexported helpers (`camelCase`).
- Prefer table-driven tests for command/mapping behavior.

## Testing Guidelines
- Add/extend tests for every behavior change in `src/*_test.go`.
- Name tests with `Test<FunctionOrBehavior>` (example: `TestLinkCommandDryRun`).
- Ensure tests pass with race detection before opening a PR.

## Commit & Pull Request Guidelines
Recent history favors short, imperative commit subjects (examples: `update dependencies`, `fix build script`).
- Keep subject lines concise and action-oriented.
- Group related changes into a single commit when practical.
- PRs should include: purpose, behavior changes, test evidence (command + result), and linked issue if applicable.
- For user-facing CLI changes, include sample command/output snippets.

## Release & Configuration Notes
- Releases are tag-driven (`v*.*.*`) via GoReleaser.
- Keep `CHANGELOG.md` and version-related code in sync when preparing a release.
