# gw (Go)

Git worktree power tool, re-implemented in Go to mirror the existing fish function `gw`.

## Install

前提

- Go 1.21+
- Git
- インタラクティブ選択は内蔵UI（promptui）で動作します。追加の外部ツールは不要です。

インストール（スクリプトのみ）

```fish
# リポジトリを取得
git clone https://github.com/sh0o0/gw.git
cd gw

# デフォルト: ~/.local/bin/gw にインストール
sh scripts/install.sh

# インストール先を変えたい場合（例: /usr/local）
env PREFIX=/usr/local sh scripts/install.sh

# 動作確認
gw --help
```

## Usage

- `gw link <path>`: Move file to primary worktree and symlink back
- `gw unlink <path>`: Replace symlink with real file/dir
- `gw switch` or `gw switch <branch>`
- `gw checkout <branch>`
- `gw restore <branch>`
- `gw list`
- `gw prune`
- `gw remove [--force] [branch ...]`

Note: Changing directories from a child process cannot affect your shell session. Use `gw shell-init` to install a wrapper that updates your shell automatically, or combine with `cd $(gw switch ...)` if you prefer manual control.

## Shell integration

Fish:

```fish
source (gw shell-init fish | psub)
```

Bash / Zsh:

```bash
eval "$(gw shell-init bash)"
```
