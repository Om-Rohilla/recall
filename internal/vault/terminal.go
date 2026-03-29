package vault

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// readPasswordFromTerminal reads a password securely (no echo) from the terminal.
// Falls back to buffered stdin read if not a terminal (e.g., piped input).
func readPasswordFromTerminal(prompt string) (string, error) {
	fd := int(os.Stdin.Fd())

	if term.IsTerminal(fd) {
		fmt.Fprint(os.Stderr, prompt)
		password, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("reading password: %w", err)
		}
		return string(password), nil
	}

	// Not a terminal — read line from stdin
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading password from stdin: %w", err)
	}
	return strings.TrimSpace(line), nil
}
