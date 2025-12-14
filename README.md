# gw (Go)

Git worktree power tool, re-implemented in Go to mirror the existing fish function `gw`.

## Install

前提

- Go 1.24+
- Git
- インタラクティブ選択は内蔵UI（go-fuzzyfinder）で動作します。追加の外部ツールは不要です。

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
- `gw move <old-branch> <new-branch>`
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

## Configuration

All configuration is stored via `git config --local`, making it repo-scoped and not committed by default.

### Hooks

Hook commands are stored in git config and executed via `sh -c`. Multiple commands run in order.

```bash
# Add a post-checkout hook
git config --local --add gw.hooks.postCheckout 'echo "switched to $GW_NEW_BRANCH"'

# Add multiple hooks (executed in order)
git config --local --add gw.hooks.postCheckout 'npm install'

# View configured hooks
git config --local --get-all gw.hooks.postCheckout

# Remove all hooks
git config --local --unset-all gw.hooks.postCheckout
```

Environment variables available in hooks:

- `GW_HOOK_NAME` - Hook name (e.g., `post-checkout`)
- `GW_PREV_BRANCH` - Previous branch name
- `GW_NEW_BRANCH` - New branch name

### Symlink Patterns

Control which gitignored files are symlinked to new worktrees.

```bash
# Add include patterns
git config --global --add gw.symlink.include '**/.env*'
git config --global --add gw.symlink.include '**/.vscode/*'
git config --global --add gw.symlink.include '**/CLAUDE.local.md'

# Add exclude patterns
git config --global --add gw.symlink.exclude '**/node_modules/**'

# Or use local (repo-scoped) config
git config --local --add gw.symlink.include '**/.env*'

# View current patterns
git config --get-all gw.symlink.include
git config --get-all gw.symlink.exclude
```

You can use the provided default config file:

```bash
# Option 1: Include in your ~/.gitconfig
cat >> ~/.gitconfig <<'EOF'
[include]
    path = /path/to/gw/docs/default.gitconfig
EOF

# Option 2: Copy contents directly
cat docs/default.gitconfig >> ~/.gitconfig
```
