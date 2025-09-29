package fsutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func ResolveAbs(p string) (string, error) {
	if p == "" {
		return "", errors.New("empty path")
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(cwd, p)), nil
}

func EnsureDir(p string) error {
	return os.MkdirAll(p, 0o755)
}

func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func CopyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return EnsureDir(target)
		}
		return CopyFile(path, target)
	})
}

// CreateSymlink removes destination if exists and creates a symlink.
func CreateSymlink(src, dst string) error {
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	if _, err := os.Lstat(dst); err == nil {
		if err := os.RemoveAll(dst); err != nil {
			return err
		}
	}
	return os.Symlink(src, dst)
}

// MaterializeSymlink replaces symlink with actual content.
func MaterializeSymlink(path string) (string, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if (fi.Mode() & os.ModeSymlink) == 0 {
		return "", fmt.Errorf("not a symlink: %s", path)
	}
	target, err := os.Readlink(path)
	if err != nil {
		return "", err
	}
	// Determine if link points to dir by checking the link path
	// We must evaluate relative target from link dir
	linkDir := filepath.Dir(path)
	absTarget := target
	if !filepath.IsAbs(absTarget) {
		absTarget = filepath.Clean(filepath.Join(linkDir, target))
	}
	stat, err := os.Stat(absTarget)
	if err != nil {
		return target, err
	}
	// create temp location next to link
	tmp := path + ".gw_unlink_tmp"
	if stat.IsDir() {
		if err := CopyDir(absTarget, tmp); err != nil {
			return target, err
		}
	} else {
		if err := CopyFile(absTarget, tmp); err != nil {
			return target, err
		}
	}
	if err := os.Remove(path); err != nil {
		return target, err
	}
	if err := os.Rename(tmp, path); err != nil {
		return target, err
	}
	return target, nil
}
