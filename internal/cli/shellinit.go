package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newShellInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "shell-init [shell]",
		Short: "Print shell integration script",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			script, err := buildShellInitScript(resolveShellName(name))
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), script)
			return nil
		},
	}
}

func resolveShellName(arg string) string {
	arg = strings.TrimSpace(arg)
	if arg != "" {
		return strings.ToLower(arg)
	}
	if sh := os.Getenv("SHELL"); sh != "" {
		return strings.ToLower(filepath.Base(sh))
	}
	return ""
}

func buildShellInitScript(shell string) (string, error) {
	if shell == "" {
		shell = "fish"
	}
	switch shell {
	case "fish":
		return fishShellInitScript, nil
	case "bash", "zsh":
		return bashShellInitScript, nil
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
}

const fishShellInitScript = `function gw --description 'Git worktree power tool'
    if test (count $argv) -gt 0; and test $argv[1] = tui
        set -l tmpfile (mktemp)
        env GW_CALLER_CWD=$PWD command gw $argv > $tmpfile
        set -l exit_status $status
        if test $exit_status -eq 0
            set -l raw (cat $tmpfile)
            if test -n "$raw"
                set -l target (string replace -r '\n*$' '' -- $raw)
                if string match -rq '^/' -- $target; and test -d "$target"
                    cd "$target"
                end
            end
        end
        rm -f $tmpfile
        return $exit_status
    end
    set -l raw (env GW_CALLER_CWD=$PWD command gw $argv | string collect)
    set -l status_list $pipestatus
    set -l exit_status $status_list[1]
    if test $exit_status -eq 0
        if test (count $argv) -gt 0
            switch $argv[1]
				case go new add mv
                    set -l target (string replace -r '\n*$' '' -- $raw)
                    if string match -rq '^/' -- $target
                        if test -d "$target"
                            cd "$target"
                            set raw ''
                        end
                    end
            end
        end
    end
    if test -n "$raw"
        printf '%s' $raw
    end
    return $exit_status
end`

const bashShellInitScript = `gw() {
  if [ $# -gt 0 ] && [ "$1" = "tui" ]; then
    local _gw_tmpfile
    _gw_tmpfile="$(mktemp)"
    env GW_CALLER_CWD="$PWD" command gw "$@" > "$_gw_tmpfile"
    local _gw_status=$?
    if [ "$_gw_status" -eq 0 ]; then
      local _gw_out
      _gw_out="$(cat "$_gw_tmpfile")"
      if [ -n "$_gw_out" ]; then
        local _gw_target
        _gw_target="$(printf '%s\n' "$_gw_out" | tail -n 1)"
        if [ -n "$_gw_target" ] && [ "${_gw_target#/}" != "$_gw_target" ] && [ -d "$_gw_target" ]; then
          builtin cd "$_gw_target" || { rm -f "$_gw_tmpfile"; return $?; }
        fi
      fi
    fi
    rm -f "$_gw_tmpfile"
    return $_gw_status
  fi
  local _gw_out
  _gw_out="$(env GW_CALLER_CWD="$PWD" command gw "$@")"
  local _gw_status=$?
  if [ "$_gw_status" -eq 0 ] && [ $# -gt 0 ]; then
		case "$1" in
			go|new|add|mv)
        local _gw_target
        _gw_target="$(printf '%s\n' "$_gw_out" | tail -n 1)"
        if [ -n "$_gw_target" ] && [ "${_gw_target#/}" != "$_gw_target" ] && [ -d "$_gw_target" ]; then
          builtin cd "$_gw_target" || return $?
          _gw_out="$(printf '%s\n' "$_gw_out" | sed '$d')"
        fi
        ;;
    esac
  fi
  if [ -n "$_gw_out" ]; then
    printf '%s\n' "$_gw_out"
  fi
  return $_gw_status
}`
