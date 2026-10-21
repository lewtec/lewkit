package terminal

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

type backend struct{}

func (backend) Prompt(_ context.Context, prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	return "", scanner.Err()
}

func (backend) Confirm(_ context.Context, message string) (bool, error) {
	fmt.Printf("%s [y/N]: ", message)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		text := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return text == "y" || text == "yes", nil
	}
	return false, scanner.Err()
}
