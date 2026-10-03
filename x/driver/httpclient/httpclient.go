// Package httpclient is the HTTP client capability.
//
// [Driver.Client] is the client tool downloads and GitHub API calls use.
// The native implementation registers from [github.com/lewtec/lewkit/x/driver/httpclient/native].
package httpclient

import (
	"context"
	"fmt"
	"net/http"

	"github.com/lewtec/lewkit/x/driver"
)

// Driver provides the HTTP client for this process.
type Driver interface {
	Client() *http.Client
}

// Client is the process HTTP client.
func Client(ctx context.Context) (*http.Client, error) {
	source, err := driver.Get[Driver](ctx)
	if err != nil {
		return nil, err
	}
	client := source.Client()
	if client == nil {
		return nil, fmt.Errorf("%w: nil client", driver.ErrUnavailable)
	}
	return client, nil
}
