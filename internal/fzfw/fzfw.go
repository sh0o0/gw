package fzfw

import (
	"bufio"
	"bytes"
	"errors"
	"os/exec"
)

// Select runs fzf with given prompt and returns the selected line or empty if cancelled.
func Select(prompt string, items []string) (string, error) {
	if len(items) == 0 {
		return "", errors.New("no items")
	}
	path, err := exec.LookPath("fzf")
	if err != nil {
		return "", errors.New("fzf not found")
	}
	cmd := exec.Command(path, "--prompt="+prompt)
	var out bytes.Buffer
	cmd.Stdout = &out
	in, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	w := bufio.NewWriter(in)
	for _, it := range items {
		w.WriteString(it)
		w.WriteByte('\n')
	}
	w.Flush()
	in.Close()
	if err := cmd.Wait(); err != nil {
		return "", err
	}
	s := out.String()
	if len(s) == 0 {
		return "", nil
	}
	// trim trailing newline
	if s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return s, nil
}
