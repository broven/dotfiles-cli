`dotfiles` Command
==================
[![CI](https://github.com/broven/dotfiles-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/broven/dotfiles-cli/actions/workflows/ci.yml)

This repository provides `dotfiles` command to manage your [dotfiles](http://dotfiles.github.io/).  It manages your dotfiles repository and symbolic links to use the configurations.
This fork is maintained independently and is not intended to be merged back upstream.

This command has below goals:

- **One binary executable**: If you want to set configuration files in a remote server, all you have to do is sending a binary to the remote.
- **Do one thing and do it well**: This command manages only a dotfiles repository.  Does not handle any other dependencies.  If you want full-setup including dependencies, you should use more suitable tool such as [Ansible](https://www.ansible.com/).  And then use `dotfiles` command from it.
- **Less dependency**: Only depends on `git` command.
- **Sensible defaults**: Many sensible default symbolic link mappings are pre-defined.  You need not to specify the mappings for almost all configuration files.


## Getting Started

1. Download [a released executable](https://github.com/broven/dotfiles-cli/releases) and put it in `$PATH` (or build locally with `go build`).
2. Change current directory to the directory you want to put a dotfiles repository.
3. Clone your dotfiles repository with `$ dotfiles clone`.
4. Enter the repository and run `$ dotfiles link --dry` to check which symlinks will be generated.
5. Write `.dotfiles/mappings.yaml` if needed.
6. `$ dotfiles link`
7. After you no longer need your configuration, remove all links with `$ dotfiles clean`.


## Usage

```
$ dotfiles {subcommand} [arguments]
```

### `clone` subcommand

Clone your dotfiles repository from remote.

```sh
# Clone git@github.com:<github-user>/dotfiles.git into current directory
$ dotfiles clone <github-user>

# Clone https://github.com/<github-user>/dotfiles.git into current directory
$ dotfiles clone <github-user> --https

# You can explicitly specify the repository name
$ dotfiles clone <github-user>/<dotfiles-repo>

# You can also use full-path
$ dotfiles clone git@bitbucket.org:<workspace>/<dotfiles-repo>.git
$ dotfiles clone https://your.site.com/dotfiles.git
```

### `link` subcommand

Set symbolic links to put your configuration files into proper places.

```sh
$ dotfiles link [options] [files...]
```

You can dry-run this command with `--dry` option.

If some `files` in dotfiles repository are specified, only they will be linked.

### `list` subcommand

Show all links set by this command.

```sh
$ dotfiles list
```

### `clean` subcommand

Remove all symbolic link put by `dotfiles link`.

```sh
$ dotfiles clean
```

### `update` subcommand

`git pull` your dotfiles repository from anywhere.

```sh
$ dotfiles update
```

### `selfupdate` subcommand

Update `dotfiles` binary (or `dotfiles.exe` on Windows) itself.

```sh
$ dotfiles selfupdate
```

## Default Mappings

It depends on your platform. Please see [source code](src/mappings.go).

## Symbolic Link Mappings

`dotfiles` command has sensible default mappings from configuration files in dotfiles repository to symbolic links put by `dotfiles link`.  And you can flexibly specify the mappings for your dotfiles manner.  Please create a `.dotfiles` directory and put a `.dotfiles/mappings.yaml` file in the root of your dotfiles repository.

Below is a complete `mappings.yaml` demo.  You can use `~` to represent a home directory.  At least one namespace from `link`, `partial_link`, `npm`, or `homebrew` is required.

```yaml
# Symlink mappings: source in repo -> destination on local machine
link:
  gitignore: ~/.global.gitignore
  cabal_config: ~/.cabal/config
  vimrc:
    - ~/.vimrc
    - ~/.config/nvim/init.vim

# Expand one source directory into many links.
# Example: if repo has zsh/.zshrc and zsh/.zprofile,
# they become ~/.zshrc and ~/.zprofile.
partial_link:
  zsh: ~

# Global npm packages (run: npm install -g <package>)
npm:
  - typescript
  - pnpm
  - eslint

# Homebrew packages
homebrew:
  # Additional taps (run first: brew tap <tap>)
  tap:
    - hashicorp/tap
    - homebrew/cask-fonts
  # Formula packages (run: brew install <formula>)
  formula:
    - wget
    - ripgrep
    - fd
  # GUI apps (run: brew install --cask <cask>)
  cask:
    - iterm2
    - visual-studio-code
```

`dotfiles link` will install/update configured `npm` and `homebrew` packages after linking files.  If `npm` or `brew` command is not found, it skips that namespace with a notification.

`homebrew` also supports a shorthand list.  This is treated as `formula` packages:

```yaml
homebrew:
  - ripgrep
  - fd
```

In addition, you can define platform specific mappings with below mappings YAML files.

- `.dotfiles/mappings_unixlike.yaml`: Will link the mappings in Linux or macOS.
- `.dotfiles/mappings_linux.yaml`: Will link the mappings in Linux.
- `.dotfiles/mappings_darwin.yaml`: Will link the mappings in macOS.
- `.dotfiles/mappings_windows.yaml`: Will link the mappings in Windows.

Below is an example of `.dotfiles/mappings_darwin.yaml`.

```yaml
link:
  keyremap4macbook.xml: ~/Library/Application Support/Karabiner/private.xml
  mac.vimrc: ~/.mac.vimrc
```

Values of the mappings object are basically strings representing destination paths, but they also can be arrays of strings. In the case, multiple symbolic links will be created for the source file.

Note: Keys can usually be written without quotes, but if a key may conflict with YAML syntax (for example it contains `:` or looks like a boolean/number), wrap the key in quotes.

For example, the following configuration will make two symbolic links `~/.vimrc` and `~/.config/nvim/init.vim` for `vimrc` source file.


```yaml
link:
  vimrc:
    - ~/.vimrc
    - ~/.config/nvim/init.vim
```

Real world usage: keep your mapping files under `.dotfiles/` in your own dotfiles repository.

## License

Licensed under [the MIT license](LICENSE.txt).
