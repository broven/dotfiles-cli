# Design: `require_target` option for link mappings

## Problem

When managing application configs (e.g., GOI plugins in `~/Library/Application Support/GOI/plugin/`) through dotfiles, the app may not be installed yet on a fresh Mac. Currently, `link()` calls `os.MkdirAll()` to create parent directories, which creates the directory structure even when the app isn't installed. This is undesirable.

## Solution

Add a `require_target` option to the `link` namespace. When `require_target: true`, skip the link if the target's parent directory doesn't exist instead of creating it.

## Config Format

```yaml
link:
  # Existing formats — unchanged
  vimrc: ~/.vimrc
  vimrc:
    - ~/.vimrc
    - ~/.config/nvim/init.vim

  # New object format with require_target
  goi_plugin:
    path: ~/Library/Application Support/GOI/plugin/myscript.sh
    require_target: true

  # Object format with multiple paths
  goi_scripts:
    path:
      - ~/Library/Application Support/GOI/plugin/script1.sh
      - ~/Library/Application Support/GOI/plugin/script2.sh
    require_target: true
```

## Behavior

- `require_target: true`: Before linking, check if the parent directory of the target path exists. If not, skip with message `Skip (target not found): <path>`. No `os.MkdirAll`, no error.
- `require_target: false` or not set: Current behavior preserved (create parent dirs as needed).

## Implementation

All changes in `src/mappings.go`:

1. **Data model**: Extend `Mappings` type to track which paths have `require_target: true`. Add a parallel `RequireTarget` set (e.g., `map[string]bool`) to the `Config` struct, or embed the flag alongside the path data.

2. **Parsing** (`parseRawMappings`, ~line 219): Add a `map[string]interface{}` case to handle the object format. Extract `path` (string or `[]string`) and `require_target` (bool, default false).

3. **Link logic** (`link()`, ~line 631): Before `os.MkdirAll`, if `require_target` is set for this path, check if parent directory exists via `os.Stat()`. If not found, print skip message and return `false, nil`.

## Scope

- Only affects `link` namespace
- No changes to: CLI flags, `partial_link`, `npm`, `homebrew`, or `relink` logic
- Fully backward compatible with existing config formats
