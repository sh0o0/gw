# gw (Go)

Git worktree power tool.

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

### Core Commands

- `gw go [branch]`: Fuzzy search and cd to worktree, or switch directly by branch name
  - `--show-path`: Display worktree path in fuzzy finder
- `gw new <branch>`: Create new worktree with a new branch
  - `--from <ref>`: Create from specific ref (branch, tag, or commit)
  - `--from-current`: Create from current branch
  - `--editor`, `-e`: Open editor after creating worktree
  - `--no-editor`: Do not open editor (override config)
  - `--editor-cmd <cmd>`: Editor command to use (default: $EDITOR)
  - `--hook-bg`: Run post-create hook in background
  - `--hook-fg`: Run post-create hook in foreground (override config)
  - `--verbose`, `-v`: Show each symlink created
- `gw add <branch>`: Create new worktree for an existing branch (fetches from origin first)
  - `--verbose`, `-v`: Show each symlink created
  - `--hook-bg`: Run post-create hook in background
  - `--hook-fg`: Run post-create hook in foreground (override config)
- `gw rm [--force] [branch ...]`: Remove worktree(s) by fuzzy select or by branch names
  - `--force`: Force remove
  - `--show-path`: Display worktree path in fuzzy finder
  - `--merged`: Remove all merged branches (interactive selection to exclude)
  - `--bg`: Run removal in background
- `gw list`: List all worktrees
- `gw clean`: Clean up stale worktree references
- `gw mv <old-branch> <new-branch>`: Rename branch and relocate worktree

### Symlink Management

- `gw link <path>`: Move file to primary worktree and create symlink back
- `gw unlink <path>`: Replace symlink with real file/dir
- `gw sync`: Sync symlinks from primary worktree to current worktree
  - `--verbose`, `-v`: Show each symlink created

### Editor & AI Integration

- `gw editor [branch]` (alias: `gw ed`): Open worktree in editor
  - `--editor`, `-e <cmd>`: Editor command to use (default: $EDITOR or gw.editor config)
  - `--show-path`: Display worktree path in fuzzy finder
- `gw ai [branch]`: Open worktree in AI CLI
  - `--ai`, `-a <cmd>`: AI CLI command to use (default: gw.ai config)
  - `--show-path`: Display worktree path in fuzzy finder

### Configuration Management

- `gw config get <key>`: Get configuration value
- `gw config set <key> <value>`: Set configuration value
- `gw config list`: List all gw configuration

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

All configuration is stored via `git config`, supporting both local (repo-scoped) and global settings. Keys use kebab-case format.

### Configuration Keys

| Key | Type | Description | Default |
|-----|------|-------------|---------|
| `gw.new.open-editor` | boolean | Auto-open editor when creating new worktree | false |
| `gw.hooks.background` | boolean | Run post-create hooks in background | false |
| `gw.hooks.post-create` | string (multi-value) | Post-create hook commands | (none) |
| `gw.editor` | string | Default editor command | $EDITOR |
| `gw.ai` | string | AI CLI command to use | (none) |
| `gw.symlink.include` | string (multi-value) | Glob patterns for symlinking | (see default.gitconfig) |
| `gw.symlink.exclude` | string (multi-value) | Glob patterns to exclude from symlinking | (see default.gitconfig) |

### Configuration Examples

```bash
# Set default editor
gw config set editor code

# Enable auto-open editor on new worktree
gw config set new.open-editor true

# Set AI CLI command
gw config set ai claude

# Run hooks in background by default
gw config set hooks.background true

# View all configuration
gw config list
```

### Hooks

Hook commands are stored in git config and executed via `sh -c`. Multiple commands run in order. Global hooks run first, then local hooks.

```bash
# Add a post-create hook
git config --local --add gw.hooks.post-create 'echo "Created worktree for $GW_BRANCH"'

# Add multiple hooks (executed in order)
git config --local --add gw.hooks.post-create 'npm install'

# View configured hooks
git config --local --get-all gw.hooks.post-create

# Remove all hooks
git config --local --unset-all gw.hooks.post-create
```

Environment variables available in hooks:

- `GW_HOOK_NAME` - Hook name (`post-create`)
- `GW_BRANCH` - Branch name
- `GW_PATH` - Worktree path

Hook output is logged to `<worktreePath>/gw-hook-post-create.log` when running in background.

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

## Fuzzy Finder Features

When using interactive selection (`gw go`, `gw rm`, `gw editor`, `gw ai`), the fuzzy finder shows:

- Branch name
- PR status indicators: `DRAFT`, `IN PROGRESS`, `MERGED`, `LINEAR` (when available via `gh` CLI)
- Assignees (when available)
- Worktree path (with `--show-path` flag)

Multi-select is available in `gw rm` (use TAB to mark, ENTER to confirm).
