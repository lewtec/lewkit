package sops

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"hash"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const unencryptedSuffix = "_unencrypted"

// macOnlyInit matches the SOPS sequence mixed into a MAC that covers only
// encrypted values, so that MAC cannot collide with a MAC over the whole tree.
var macOnlyInit = []byte{
	0x8a, 0x3f, 0xd2, 0xad, 0x54, 0xce, 0x66, 0x52,
	0x7b, 0x10, 0x34, 0xf3, 0xd1, 0x47, 0xbe, 0x0b,
	0x0b, 0x97, 0x5b, 0x3b, 0xf4, 0x4f, 0x72, 0xc6,
	0xfd, 0xad, 0xec, 0x81, 0x76, 0xf2, 0x7d, 0x69,
}

var encValue = regexp.MustCompile(`^ENC\[AES256_GCM,data:(.+),iv:(.+),tag:(.+),type:(.+)\]`)

type item struct {
	key string
	val any
}

func decryptTree(items []item, meta *sopsMeta, key []byte) ([]item, error) {
	docs, err := decryptBranches([][]item{items}, meta, key)
	if err != nil {
		return nil, err
	}
	return docs[0], nil
}

// decryptBranches walks each document with one MAC, in file order.
func decryptBranches(docs [][]item, meta *sopsMeta, key []byte) ([][]item, error) {
	if err := meta.prepare(); err != nil {
		return nil, err
	}
	sum := sha512.New()
	if meta.MACOnlyEncrypted {
		sum.Write(macOnlyInit)
	}
	out := make([][]item, len(docs))
	for i, doc := range docs {
		branch, err := decryptBranch(doc, nil, meta, key, sum)
		if err != nil {
			return nil, err
		}
		out[i] = branch
	}
	if err := checkMAC(meta, key, fmt.Sprintf("%X", sum.Sum(nil))); err != nil {
		return nil, err
	}
	return out, nil
}

func checkMAC(meta *sopsMeta, key []byte, got string) error {
	ts, err := time.Parse(time.RFC3339, meta.LastModified)
	if err != nil {
		return fmt.Errorf("sops: lastmodified: %w", err)
	}
	mac, err := decryptLeaf(meta.MAC, key, ts.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("sops: mac: %w", err)
	}
	macText, ok := mac.(string)
	if !ok || subtle.ConstantTimeCompare([]byte(macText), []byte(got)) != 1 {
		return ErrMAC
	}
	return nil
}

func decryptBranch(items []item, path []string, meta *sopsMeta, key []byte, sum hash.Hash) ([]item, error) {
	out := make([]item, len(items))
	for i, it := range items {
		next := append(append([]string{}, path...), it.key)
		val, err := decryptValue(it.val, next, meta, key, sum)
		if err != nil {
			return nil, err
		}
		out[i] = item{key: it.key, val: val}
	}
	return out, nil
}

func decryptSeq(items []any, path []string, meta *sopsMeta, key []byte, sum hash.Hash) ([]any, error) {
	out := make([]any, len(items))
	for i, it := range items {
		val, err := decryptValue(it, path, meta, key, sum)
		if err != nil {
			return nil, err
		}
		out[i] = val
	}
	return out, nil
}

func decryptValue(v any, path []string, meta *sopsMeta, key []byte, sum hash.Hash) (any, error) {
	switch v := v.(type) {
	case []item:
		return decryptBranch(v, path, meta, key, sum)
	case []any:
		return decryptSeq(v, path, meta, key, sum)
	case nil:
		return nil, nil
	default:
		return finishLeaf(v, path, meta, key, sum)
	}
}

func finishLeaf(v any, path []string, meta *sopsMeta, key []byte, sum hash.Hash) (any, error) {
	enc := shouldEncrypt(path, meta)
	out := v
	if enc {
		text, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("sops: encrypted %s has type %T", strings.Join(path, "."), v)
		}
		if text != "" {
			plain, err := decryptLeaf(text, key, strings.Join(path, ":")+":")
			if err != nil {
				return nil, fmt.Errorf("sops: %s: %w", strings.Join(path, "."), err)
			}
			out = plain
		}
	}
	if !meta.MACOnlyEncrypted || enc {
		raw, err := toBytes(out)
		if err != nil {
			return nil, err
		}
		sum.Write(raw)
	}
	return out, nil
}

func shouldEncrypt(path []string, meta *sopsMeta) bool {
	encrypted := true
	if meta.UnencryptedSuffix != "" {
		for _, part := range path {
			if strings.HasSuffix(part, meta.UnencryptedSuffix) {
				encrypted = false
				break
			}
		}
	}
	if meta.EncryptedSuffix != "" {
		encrypted = false
		for _, part := range path {
			if strings.HasSuffix(part, meta.EncryptedSuffix) {
				encrypted = true
				break
			}
		}
	}
	if meta.UnencryptedRegex != "" {
		for _, part := range path {
			matched, _ := regexp.Match(meta.UnencryptedRegex, []byte(part))
			if matched {
				encrypted = false
				break
			}
		}
	}
	if meta.EncryptedRegex != "" {
		encrypted = false
		for _, part := range path {
			matched, _ := regexp.Match(meta.EncryptedRegex, []byte(part))
			if matched {
				encrypted = true
				break
			}
		}
	}
	return encrypted
}

func decryptLeaf(ciphertext string, key []byte, aad string) (any, error) {
	if ciphertext == "" {
		return "", nil
	}
	match := encValue.FindStringSubmatch(ciphertext)
	if match == nil {
		return nil, fmt.Errorf("value is not sops data")
	}
	data, err := base64.StdEncoding.DecodeString(match[1])
	if err != nil {
		return nil, fmt.Errorf("data: %w", err)
	}
	iv, err := base64.StdEncoding.DecodeString(match[2])
	if err != nil {
		return nil, fmt.Errorf("iv: %w", err)
	}
	tag, err := base64.StdEncoding.DecodeString(match[3])
	if err != nil {
		return nil, fmt.Errorf("tag: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, len(iv))
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, iv, append(data, tag...), []byte(aad))
	if err != nil {
		return nil, err
	}
	text := string(plain)
	switch match[4] {
	case "str":
		return text, nil
	case "int":
		return strconv.Atoi(text)
	case "float":
		return strconv.ParseFloat(text, 64)
	case "bytes":
		return plain, nil
	case "bool":
		return strconv.ParseBool(text)
	case "time":
		var ts time.Time
		if err := ts.UnmarshalText(plain); err != nil {
			return nil, err
		}
		return ts, nil
	default:
		return nil, fmt.Errorf("unknown sops type %s", match[4])
	}
}

func toBytes(v any) ([]byte, error) {
	switch v := v.(type) {
	case string:
		return []byte(v), nil
	case int:
		return []byte(strconv.Itoa(v)), nil
	case int64:
		return []byte(strconv.FormatInt(v, 10)), nil
	case float64:
		return []byte(strconv.FormatFloat(v, 'f', -1, 64)), nil
	case bool:
		if v {
			return []byte("True"), nil
		}
		return []byte("False"), nil
	case []byte:
		return v, nil
	case time.Time:
		return v.MarshalText()
	default:
		return nil, fmt.Errorf("sops: cannot authenticate %T", v)
	}
}
