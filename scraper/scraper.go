// Package scraper provides a Go interface to the Mack-Bot Python scraper CLI.
// It runs `python -m scraper <cmd>` as a subprocess and parses the JSON output.
package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// run executes the Python scraper CLI with the given arguments and decodes
// the JSON output into dst. Stderr is captured and included in any error.
func run(dst any, args ...string) error {
	fullArgs := append([]string{"-m", "scraper"}, args...)
	cmd := exec.Command("python3", fullArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Try to parse error JSON from stderr
		errOut := strings.TrimSpace(stderr.String())
		var errJSON struct {
			Error string `json:"error"`
		}
		if jsonErr := json.Unmarshal([]byte(errOut), &errJSON); jsonErr == nil && errJSON.Error != "" {
			return fmt.Errorf("scraper: %s", errJSON.Error)
		}
		return fmt.Errorf("scraper %v: %w\n%s", args, err, errOut)
	}

	if err := json.Unmarshal(stdout.Bytes(), dst); err != nil {
		return fmt.Errorf("scraper parse: %w\noutput: %s", err, stdout.String())
	}
	return nil
}
