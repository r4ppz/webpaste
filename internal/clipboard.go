package webpaste

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

const clipboardCommand = "wl-paste"

func ClipboardAvailable() error {
	if _, err := exec.LookPath(clipboardCommand); err != nil {
		return fmt.Errorf("%s not found in PATH: %w", clipboardCommand, err)
	}
	return nil
}

func getClipboard(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, clipboardCommand, "-n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s failed: %w - output: %s", clipboardCommand, err, string(out))
	}

	return string(out), nil
}
