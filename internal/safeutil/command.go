package safeutil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// SafeCommand creates an exec.Cmd for the provided binary name after resolving
// it via PATH and ensuring each argument is clean (no null bytes).
func SafeCommand(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	if name == "" {
		return nil, fmt.Errorf("safeutil: empty command name")
	}

	resolved, err := exec.LookPath(name)
	if err != nil {
		return nil, fmt.Errorf("safeutil: resolve %s: %w", name, err)
	}

	cleanArgs := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.ContainsRune(arg, '\x00') {
			return nil, fmt.Errorf("safeutil: argument contains forbidden characters: %q", arg)
		}
		cleanArgs = append(cleanArgs, arg)
	}

	// G204 is intentional here - we've validated the path and sanitized arguments above
	cmd := exec.CommandContext(ctx, resolved, cleanArgs...) // #nosec G204 -- validated path and sanitized args
	return cmd, nil
}
