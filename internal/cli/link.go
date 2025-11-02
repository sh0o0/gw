package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sh0o0/gw/internal/fsutil"
	"github.com/sh0o0/gw/internal/gitx"
	"github.com/spf13/cobra"
)

func newLinkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "link <path>",
		Short: "Move file to base worktree and create symlink",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := fsutil.ResolveAbs(args[0])
			if err != nil {
				return err
			}
			if _, err := os.Lstat(p); err != nil {
				return fmt.Errorf("file not found: %s", p)
			}
			root, err := gitx.Root("")
			if err != nil {
				return err
			}
			primary, _ := primaryWorktreePath()
			if root == primary {
				return errors.New("you are in the primary worktree; nothing to link")
			}
			if !strings.HasPrefix(p, root+string(os.PathSeparator)) {
				return fmt.Errorf("path must be within current worktree: %s", root)
			}
			rel, _ := filepath.Rel(root, p)
			dst := filepath.Join(primary, rel)
			if _, err := os.Lstat(dst); err == nil {
				return fmt.Errorf("destination already exists: %s", dst)
			}
			if err := fsutil.EnsureDir(filepath.Dir(dst)); err != nil {
				return err
			}
			if err := os.Rename(p, dst); err != nil {
				return err
			}
			if err := os.Symlink(dst, p); err != nil {
				return err
			}
			fmt.Printf("Linked: %s -> %s\n", p, dst)
			return nil
		},
	}
}

func newUnlinkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unlink <path>",
		Short: "Replace symlink with a real file/dir by copying its target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := fsutil.ResolveAbs(args[0])
			if err != nil {
				return err
			}
			target, err := fsutil.MaterializeSymlink(p)
			if err != nil {
				return err
			}
			if target != "" {
				fmt.Printf("Unlinked: %s (copied from: %s)\n", p, target)
			} else {
				fmt.Printf("Unlinked: %s\n", p)
			}
			return nil
		},
	}
}
