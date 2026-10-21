//go:build android

package terminal

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/driver/launcher"
	"github.com/lewtec/lewkit/x/errs"
)

func (backend) Choose(_ context.Context, opts launcher.ChooseOptions) (*launcher.Item, error) {
	if len(opts.Items) == 0 {
		return nil, launcher.ErrChooseNoItems
	}
	if opts.Prompt != "" {
		fmt.Println(opts.Prompt)
	}
	for i, item := range opts.Items {
		fmt.Printf("%d. %s\n", i+1, launcher.ItemLabel(item))
	}
	fmt.Print("choose: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return nil, scanner.Err()
	}
	text := strings.TrimSpace(scanner.Text())
	if text == "" {
		return nil, nil
	}
	n, err := strconv.Atoi(text)
	if err != nil || n < 1 || n > len(opts.Items) {
		return nil, fmt.Errorf("%w: choose %q", errs.ErrInvalid, text)
	}
	return &opts.Items[n-1], nil
}
