# gw (Go)

Git worktree power tool, re-implemented in Go to mirror the existing fish function `gw`.

## Install

Build the binary:

- With Go toolchain: `go build ./cmd/gw`

## Usage

- `gw link <path>`: Move file to primary worktree and symlink back
- `gw unlink <path>`: Replace symlink with real file/dir
- `gw switch` or `gw switch <branch>`
- `gw checkout <branch>`
- `gw restore <branch>`
- `gw list`
- `gw prune`
- `gw remove [--force] [branch ...]`

Note: Changing directories from a child process cannot affect your shell session. This CLI prints the target path to stdout on switch/checkout/restore; combine with a shell wrapper to `cd $(gw switch ...)` as needed.
