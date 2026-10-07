package sops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	sopsPrefix    = "sops_"
	mapSeparator  = "__map_"
	listSeparator = "__list_"
)

func decodeDotenv(in []byte) ([]byte, error) {
	if !textFile(in) {
		return in, nil
	}
	items, meta, err := parseDotenv(in)
	if err != nil {
		return nil, err
	}
	if meta == nil || !meta.isSops() {
		return in, nil
	}
	if err := meta.prepare(); err != nil {
		return nil, err
	}
	return finish(items, meta, false, emitDotenv)
}

func parseDotenv(in []byte) ([]item, *sopsMeta, error) {
	var items []item
	flat := map[string]any{}
	sawMeta := false
	sopsFile := looksLikeDotenv(in)
	for _, line := range bytes.Split(in, []byte("\n")) {
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		pos := bytes.IndexByte(line, '=')
		if pos < 0 {
			if sopsFile {
				return nil, nil, fmt.Errorf("sops: invalid dotenv line")
			}
			return nil, nil, nil
		}
		key := string(line[:pos])
		val := strings.ReplaceAll(string(line[pos+1:]), `\n`, "\n")
		if strings.HasPrefix(key, sopsPrefix) {
			sawMeta = true
			flat[key[len(sopsPrefix):]] = val
			continue
		}
		items = append(items, item{key: key, val: val})
	}
	if !sawMeta {
		return nil, nil, nil
	}
	decodeNewLines(flat)
	if err := decodeNonStrings(flat); err != nil {
		return nil, nil, err
	}
	meta, err := unflattenMeta(flat)
	if err != nil {
		return nil, nil, err
	}
	return items, meta, nil
}

func emitDotenv(items []item) ([]byte, error) {
	var buf bytes.Buffer
	for _, it := range items {
		text, ok := it.val.(string)
		if !ok {
			return nil, fmt.Errorf("sops: dotenv value %s has type %T", it.key, it.val)
		}
		value := strings.ReplaceAll(text, "\n", `\n`)
		fmt.Fprintf(&buf, "%s=%s\n", it.key, value)
	}
	return buf.Bytes(), nil
}

func decodeNewLines(m map[string]any) {
	for k, v := range m {
		if s, ok := v.(string); ok {
			m[k] = strings.ReplaceAll(s, `\n`, "\n")
		}
	}
}

func decodeNonStrings(m map[string]any) error {
	if v, ok := m["mac_only_encrypted"]; ok {
		m["mac_only_encrypted"] = v == "true"
	}
	if v, ok := m["shamir_threshold"]; ok {
		switch val := v.(type) {
		case string:
			n, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("sops: shamir_threshold is not an integer: %w", err)
			}
			m["shamir_threshold"] = n
		case int:
			m["shamir_threshold"] = val
		default:
			return fmt.Errorf("sops: shamir_threshold has type %T", val)
		}
	}
	return nil
}

func unflattenMeta(flat map[string]any) (*sopsMeta, error) {
	raw, err := json.Marshal(unflatten(flat))
	if err != nil {
		return nil, err
	}
	var meta sopsMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, fmt.Errorf("sops: dotenv metadata: %w", err)
	}
	return &meta, nil
}

type mapToken struct{ key string }
type listToken struct{ position int }

func tokenize(path string) []any {
	const (
		stateNormal = iota
		stateMap
		stateList
	)
	var tokens []any
	state := stateNormal
	last := 0
	finish := func(i int) {
		switch state {
		case stateNormal:
			tokens = append(tokens, mapToken{path[last:i]})
		case stateMap:
			tokens = append(tokens, mapToken{path[last+len(mapSeparator) : i]})
		case stateList:
			pos, _ := strconv.Atoi(path[last+len(listSeparator) : i])
			tokens = append(tokens, listToken{pos})
		}
		last = i
	}
	for i := 0; i < len(path); {
		if strings.HasPrefix(path[i:], mapSeparator) {
			finish(i)
			state = stateMap
			i += len(mapSeparator)
			continue
		}
		if strings.HasPrefix(path[i:], listSeparator) {
			finish(i)
			state = stateList
			i += len(listSeparator)
			continue
		}
		i++
	}
	finish(len(path))
	return tokens
}

func unflatten(in map[string]any) map[string]any {
	root := map[string]any{}
	for key, value := range in {
		var current any = root
		tokens := append(tokenize(key), nil)
		for i := 0; i < len(tokens)-1; i++ {
			current = place(current, tokens[i], tokens[i+1], value)
		}
	}
	return root
}

func place(current, tok, next, value any) any {
	switch tok := tok.(type) {
	case mapToken:
		node := current.(map[string]any)
		switch next := next.(type) {
		case mapToken:
			child, _ := node[tok.key].(map[string]any)
			if child == nil {
				child = map[string]any{}
				node[tok.key] = child
			}
			return child
		case listToken:
			child, _ := node[tok.key].([]any)
			if next.position >= len(child) {
				grown := make([]any, next.position+1)
				copy(grown, child)
				child = grown
				node[tok.key] = child
			}
			return child
		default:
			node[tok.key] = value
		}
	case listToken:
		node := current.([]any)
		switch next := next.(type) {
		case mapToken:
			child, _ := node[tok.position].(map[string]any)
			if child == nil {
				child = map[string]any{}
				node[tok.position] = child
			}
			return child
		case listToken:
			child, _ := node[tok.position].([]any)
			if next.position >= len(child) {
				grown := make([]any, next.position+1)
				copy(grown, child)
				child = grown
				node[tok.position] = child
			}
			return child
		default:
			node[tok.position] = value
		}
	}
	return nil
}
