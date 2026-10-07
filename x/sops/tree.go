package sops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

func tryJSON(in []byte) ([]byte, bool, error) {
	trim := bytes.TrimSpace(in)
	if len(trim) == 0 || trim[0] != '{' {
		return nil, false, nil
	}
	var holder sopsFile
	if err := json.Unmarshal(in, &holder); err != nil || !holder.Sops.isSops() {
		return nil, false, nil
	}
	out, err := decryptJSON(in, holder.Sops, true)
	return out, true, err
}

func tryYAML(in []byte) ([]byte, bool, error) {
	if !textFile(in) {
		return nil, false, nil
	}
	var holder sopsFile
	if err := yaml.Unmarshal(in, &holder); err != nil || !holder.Sops.isSops() {
		return nil, false, nil
	}
	out, err := decryptYAML(in, holder.Sops)
	return out, true, err
}

func decodeJSON(in []byte, binary bool) ([]byte, error) {
	var holder sopsFile
	if err := json.Unmarshal(in, &holder); err != nil || !holder.Sops.isSops() {
		return in, nil
	}
	return decryptJSON(in, holder.Sops, binary)
}

func decodeYAML(in []byte) ([]byte, error) {
	if !textFile(in) {
		return in, nil
	}
	var holder sopsFile
	if err := yaml.Unmarshal(in, &holder); err != nil || !holder.Sops.isSops() {
		return in, nil
	}
	return decryptYAML(in, holder.Sops)
}

func decryptJSON(in []byte, meta *sopsMeta, binary bool) ([]byte, error) {
	items, err := parseJSON(in)
	if err != nil {
		return nil, err
	}
	items = dropSops(items)
	binary = binary && len(items) == 1 && items[0].key == "data"
	return finish(items, meta, binary, emitJSON)
}

func decryptYAML(in []byte, meta *sopsMeta) ([]byte, error) {
	docs, err := parseYAML(in)
	if err != nil {
		return nil, err
	}
	for i := range docs {
		docs[i] = dropSops(docs[i])
	}
	key, err := openKey(meta)
	if err != nil {
		return nil, err
	}
	out, err := decryptBranches(docs, meta, key)
	if err != nil {
		return nil, err
	}
	return emitYAML(out)
}

func finish(items []item, meta *sopsMeta, binary bool, emit func([]item) ([]byte, error)) ([]byte, error) {
	key, err := openKey(meta)
	if err != nil {
		return nil, err
	}
	out, err := decryptTree(items, meta, key)
	if err != nil {
		return nil, err
	}
	if binary {
		switch data := out[0].val.(type) {
		case string:
			return []byte(data), nil
		case []byte:
			return data, nil
		default:
			return nil, fmt.Errorf("sops: binary data has type %T", out[0].val)
		}
	}
	return emit(out)
}

func openKey(meta *sopsMeta) ([]byte, error) {
	recipients, err := meta.recipients()
	if err != nil {
		return nil, err
	}
	return dataKey(recipients)
}

func dropSops(items []item) []item {
	out := make([]item, 0, len(items))
	for _, it := range items {
		if it.key != "sops" {
			out = append(out, it)
		}
	}
	return out
}

func parseJSON(in []byte) ([]item, error) {
	dec := json.NewDecoder(bytes.NewReader(in))
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("sops: %w", err)
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '{' {
		return nil, fmt.Errorf("sops: json file must be an object")
	}
	return parseJSONObject(dec)
}

func parseJSONObject(dec *json.Decoder) ([]item, error) {
	var items []item
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("sops: json key has type %T", tok)
		}
		val, err := parseJSONValue(dec)
		if err != nil {
			return nil, err
		}
		items = append(items, item{key: key, val: val})
	}
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '}' {
		return nil, fmt.Errorf("sops: json object did not end")
	}
	return items, nil
}

func parseJSONValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return tok, nil
	}
	switch delim {
	case '{':
		return parseJSONObject(dec)
	case '[':
		return parseJSONArray(dec)
	default:
		return nil, fmt.Errorf("sops: unexpected json %s", delim)
	}
}

func parseJSONArray(dec *json.Decoder) ([]any, error) {
	var items []any
	for dec.More() {
		val, err := parseJSONValue(dec)
		if err != nil {
			return nil, err
		}
		items = append(items, val)
	}
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != ']' {
		return nil, fmt.Errorf("sops: json array did not end")
	}
	return items, nil
}

func parseYAML(in []byte) ([][]item, error) {
	dec := yaml.NewDecoder(bytes.NewReader(in))
	var docs [][]item
	for {
		var node yaml.Node
		err := dec.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("sops: %w", err)
		}
		val, err := yamlValue(&node)
		if err != nil {
			return nil, err
		}
		items, ok := val.([]item)
		if !ok {
			return nil, fmt.Errorf("sops: yaml document must be a mapping")
		}
		docs = append(docs, items)
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("sops: empty yaml")
	}
	return docs, nil
}

func yamlValue(node *yaml.Node) (any, error) {
	if node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			return []item{}, nil
		}
		return yamlValue(node.Content[0])
	case yaml.MappingNode:
		items := make([]item, 0, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			var key string
			if err := node.Content[i].Decode(&key); err != nil {
				return nil, err
			}
			val, err := yamlValue(node.Content[i+1])
			if err != nil {
				return nil, err
			}
			items = append(items, item{key: key, val: val})
		}
		return items, nil
	case yaml.SequenceNode:
		items := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			val, err := yamlValue(child)
			if err != nil {
				return nil, err
			}
			items = append(items, val)
		}
		return items, nil
	case yaml.ScalarNode:
		var val any
		if err := node.Decode(&val); err != nil {
			return nil, err
		}
		return val, nil
	default:
		return nil, fmt.Errorf("sops: yaml kind %d", node.Kind)
	}
}

func emitJSON(items []item) ([]byte, error) {
	raw, err := encodeObject(items)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "\t"); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func encodeObject(items []item) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, it := range items {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(it.key)
		if err != nil {
			return nil, err
		}
		val, err := encodeAny(it.val)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteString(": ")
		buf.Write(val)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func encodeArray(items []any) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, it := range items {
		if i > 0 {
			buf.WriteByte(',')
		}
		val, err := encodeAny(it)
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

func encodeAny(v any) ([]byte, error) {
	switch v := v.(type) {
	case []item:
		return encodeObject(v)
	case []any:
		return encodeArray(v)
	default:
		return json.Marshal(v)
	}
}

func emitYAML(docs [][]item) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(4)
	for _, doc := range docs {
		node, err := mappingNode(doc)
		if err != nil {
			return nil, err
		}
		document := yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{node}}
		if err := enc.Encode(&document); err != nil {
			return nil, err
		}
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func mappingNode(items []item) (*yaml.Node, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, it := range items {
		key := &yaml.Node{}
		if err := key.Encode(it.key); err != nil {
			return nil, err
		}
		val, err := valueNode(it.val)
		if err != nil {
			return nil, err
		}
		node.Content = append(node.Content, key, val)
	}
	return node, nil
}

func valueNode(v any) (*yaml.Node, error) {
	switch v := v.(type) {
	case []item:
		return mappingNode(v)
	case []any:
		node := &yaml.Node{Kind: yaml.SequenceNode}
		for _, it := range v {
			child, err := valueNode(it)
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, child)
		}
		return node, nil
	default:
		node := &yaml.Node{}
		if err := node.Encode(v); err != nil {
			return nil, err
		}
		return node, nil
	}
}
