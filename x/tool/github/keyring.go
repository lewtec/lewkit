package github

import (
	"context"
	"errors"
	"log/slog"

	"github.com/zalando/go-keyring"
)

// keyringGet is replaced in tests so they do not touch the login keyring.
var keyringGet = func(service, user string) (string, error) {
	return keyring.Get(service, user)
}

// tokenFromKeyring reads the secret gh stores under service "gh:github.com".
// The active user's entry is tried first, then the empty-account slot older
// gh releases used.
func tokenFromKeyring(ctx context.Context) (string, error) {
	accounts := []string{""}
	if user := githubHostUser(); user != "" {
		accounts = []string{user, ""}
	}
	for _, account := range accounts {
		token, err := keyringGet(ghKeyringService, account)
		if err != nil {
			if !errors.Is(err, keyring.ErrNotFound) && !errors.Is(err, keyring.ErrUnsupportedPlatform) {
				slog.DebugContext(ctx, "keyring lookup failed", "error", err)
			}
			continue
		}
		if token = cleanToken(token); token != "" {
			return token, nil
		}
	}
	return "", nil
}
