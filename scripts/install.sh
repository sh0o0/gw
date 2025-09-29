#!/usr/bin/env sh

# Minimal installer for gw (POSIX sh)
# Usage:
#   PREFIX=$HOME/.local sh scripts/install.sh
#
# Env vars:
#   PREFIX   install prefix (default: $HOME/.local)

set -e

err() { printf '%s\n' "$*" >&2; }

command -v go >/dev/null 2>&1 || { err "go is required"; exit 1; }
command -v git >/dev/null 2>&1 || { err "git is required"; exit 1; }

PREFIX=${PREFIX:-"$HOME/.local"}
SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
PROJECT_ROOT=$(CDPATH='' cd -- "$SCRIPT_DIR/.." && pwd -P)
cd "$PROJECT_ROOT"

TMP_BIN=$(mktemp -t gw.XXXXXX)
trap 'rm -f "$TMP_BIN" >/dev/null 2>&1 || true' EXIT INT TERM
go build -o "$TMP_BIN" ./cmd/gw || { err "build failed"; exit 1; }

BINDIR="$PREFIX/bin"
mkdir -p "$BINDIR" || { err "failed to create: $BINDIR"; exit 1; }
TARGET="$BINDIR/gw"

if command -v install >/dev/null 2>&1; then
  install -m 0755 "$TMP_BIN" "$TARGET"
else
  mv -f "$TMP_BIN" "$TARGET"
  chmod 0755 "$TARGET" || true
fi

rm -f "$TMP_BIN" >/dev/null 2>&1 || true
trap - EXIT INT TERM

printf 'Installed: %s\n' "$TARGET"

case ":$PATH:" in
  *":$BINDIR:"*) : ;;
  *) printf 'Note: %s is not on PATH. Add it to your shell.\n' "$BINDIR";;
esac

exit 0
