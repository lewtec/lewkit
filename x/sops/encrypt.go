package sops

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	sopsv3 "github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/aes"
	"github.com/getsops/sops/v3/cmd/sops/common"
	sopsconfig "github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/keyservice"
	jsonstore "github.com/getsops/sops/v3/stores/json"
	"github.com/getsops/sops/v3/version"
)

// ErrNoConfig means no .sops.yaml was found above the output path.
var ErrNoConfig = errors.New("sops config file not found")

// Encrypt encrypts plaintext for path in-process with the sops library.
// The sops encrypt command is package main, so this calls the same library
// steps: the creation rule from .sops.yaml, then the data key, tree, and
// binary store. A missing config file is ErrNoConfig.
func Encrypt(ctx context.Context, path string, plaintext []byte) ([]byte, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	found, err := sopsconfig.LookupConfigFile(path)
	if err != nil {
		if found.Warning != "" {
			return nil, fmt.Errorf("%w: %s: %w", ErrNoConfig, found.Warning, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrNoConfig, err)
	}
	conf, err := sopsconfig.LoadCreationRuleForFile(found.Path, path, map[string]*string{})
	if err != nil {
		return nil, fmt.Errorf("sops: %w", err)
	}
	if conf == nil {
		return nil, fmt.Errorf("sops: %s has no creation rules", found.Path)
	}
	stores, err := sopsconfig.LoadStoresConfig(found.Path)
	if err != nil {
		return nil, fmt.Errorf("sops: %w", err)
	}
	store := jsonstore.NewBinaryStore(&stores.JSONBinary)
	branches, err := store.LoadPlainFile(plaintext)
	if err != nil {
		return nil, fmt.Errorf("sops: %w", err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	tree := &sopsv3.Tree{
		Branches: branches,
		Metadata: sopsv3.Metadata{
			KeyGroups:               conf.KeyGroups,
			ShamirThreshold:         conf.ShamirThreshold,
			UnencryptedSuffix:       conf.UnencryptedSuffix,
			EncryptedSuffix:         conf.EncryptedSuffix,
			UnencryptedRegex:        conf.UnencryptedRegex,
			EncryptedRegex:          conf.EncryptedRegex,
			UnencryptedCommentRegex: conf.UnencryptedCommentRegex,
			EncryptedCommentRegex:   conf.EncryptedCommentRegex,
			MACOnlyEncrypted:        conf.MACOnlyEncrypted,
			Version:                 version.Version,
		},
		FilePath: abs,
	}
	dataKey, errs := tree.GenerateDataKeyWithKeyServices([]keyservice.KeyServiceClient{
		keyservice.NewLocalClient(),
	})
	if len(errs) > 0 {
		return nil, fmt.Errorf("sops: %w", errors.Join(errs...))
	}
	err = common.EncryptTree(common.EncryptTreeOpts{
		DataKey: dataKey,
		Tree:    tree,
		Cipher:  aes.NewCipher(),
	})
	if err != nil {
		return nil, fmt.Errorf("sops: %w", err)
	}
	out, err := store.EmitEncryptedFile(*tree)
	if err != nil {
		return nil, fmt.Errorf("sops: %w", err)
	}
	return out, nil
}
