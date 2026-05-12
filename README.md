# gitgetter

A modern, keyboard-driven **TUI** for everyday git operations — built with
[Bubble Tea](https://github.com/charmbracelet/bubbletea) and the Charm stack.

```
 gitgetter                        ╭─ branch: main → origin/main  ↑2 ─╮
 Status  Log  Branches  Remote  Stash
────────────────────────────────────────────────────────────────────────
  Staged
 ❯ ✔  A  src/app.go
    A  README.md

  Modified
    M  go.mod

  Untracked
    ?  scratch.txt
────────────────────────────────────────────────────────────────────────
space select · a all · s stage · u unstage · S stage all · c commit · q quit
```

## Features

| Tab          | Operations                                                            |
| ------------ | --------------------------------------------------------------------- |
| **Status**   | Multi-select files, stage / unstage / stage-all, commit, amend, stash |
| **Log**      | Browse history, multi-select commits to squash, amend HEAD            |
| **Branches** | List local & remote branches, checkout, create, delete                |
| **Remote**   | Push, pull, fetch, force-push (with lease) + ahead/behind indicator   |
| **Stash**    | List stashes, push with message, pop, drop                            |

### Keyboard quick-reference

| Key              | Action                            |
| ---------------- | --------------------------------- |
| `↑ / k`, `↓ / j` | Move cursor                       |
| `ctrl+u / d`     | Page up / down                    |
| `g / G`          | Jump to top / bottom              |
| `tab / ⇧tab`     | Switch tabs                       |
| `space`          | Toggle selection                  |
| `a`              | Select all / deselect all         |
| `s`              | Stage (selected or cursor)        |
| `u`              | Unstage                           |
| `S`              | Stage all changes                 |
| `c`              | Commit staged                     |
| `A`              | Amend last commit                 |
| `x`              | Squash selected commits (Log tab) |
| `p / P`          | Push / force-push with lease      |
| `l`              | Pull                              |
| `f`              | Fetch all remotes                 |
| `o / ↵`          | Checkout branch                   |
| `n`              | New branch                        |
| `d`              | Delete branch / drop stash        |
| `t / T`          | Stash / pop stash                 |
| `r`              | Refresh all git state             |
| `q`              | Quit                              |

## Installation

### Nix (recommended)

```bash
# Run without installing
nix run github:yourname/gitgetter

# Install into NixOS or nix-darwin via the module:
# flake.nix
{
  inputs.gitgetter.url = "github:yourname/gitgetter";

  outputs = { nixpkgs, gitgetter, ... }: {
    nixosConfigurations.myhost = nixpkgs.lib.nixosSystem {
      modules = [
        gitgetter.nixosModules.default
        {
          programs.gitgetter.enable = true;
          programs.gitgetter.alias  = "gg";   # optional short alias
        }
      ];
    };
  };
}
```

### Go (any platform)

```bash
go install github.com/yourname/gitgetter@latest
```

### Build from source

```bash
git clone https://github.com/yourname/gitgetter
cd gitgetter
go build -o gitgetter .
```

## Development

```bash
nix develop          # enter dev shell (Go + tools pre-loaded)
go run .             # run directly
go build ./...       # build
go test ./...        # test
nix fmt              # format all sources (Go + Nix + Markdown)
nix flake check      # run checks including format verification
```

## Architecture

```
gitgetter/
├── main.go                   Entry point — starts Bubble Tea program
├── internal/
│   ├── git/
│   │   └── git.go            All git shell invocations, pure functions
│   ├── styles/
│   │   └── styles.go         Tokyo Night palette + Lipgloss styles
│   └── app/
│       ├── model.go          Root model, tea.Msg types, helpers
│       ├── keys.go           All key bindings (bubbles/key)
│       ├── update.go         tea.Update — all state transitions
│       └── view.go           tea.View — all rendering
├── nix/
│   └── module.nix            NixOS / nix-darwin program module
├── flake.nix                 Nix flake (devShell, package, formatter)
└── treefmt.toml              treefmt formatting rules
```

## License

MIT
