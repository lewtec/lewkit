// Package httpclient is the HTTP client capability.
//
// [Driver.Client] is the client tool downloads and GitHub API calls use.
// The native implementation registers from [github.com/lewtec/lewkit/x/driver/httpclient/native].
package httpclient

import "net/http"

// Driver provides the HTTP client for this process.
type Driver interface {
	Client() *http.Client
}
