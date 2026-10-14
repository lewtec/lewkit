package decomp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Result is one comparison of a caller-supplied candidate.
type Result struct {
	Address   uint64
	Name      string
	Signature string
	Source    string
	Matched   bool
	Residuals []string
}

// Check compares a candidate the caller already wrote with the function at address.
// The candidate is one JSON object with name, signature, source, and blocks.
// A match stores that name, signature, and source. A mismatch returns [ErrUnmatched]
// and leaves the function unchanged.
func (artifact *Artifact) Check(ctx context.Context, address uint64, text string) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	function := artifact.byAddress[address]
	if function == nil {
		return Result{}, fmt.Errorf("%w: 0x%x", ErrNotFound, address)
	}
	result := Result{Address: address}
	parsed, err := parseCandidate(text)
	if err != nil {
		result.Residuals = []string{err.Error()}
		return result, ErrUnmatched
	}
	notes := differences(*function, parsed, artifact)
	result.Name = parsed.Name
	result.Signature = parsed.Signature
	result.Source = parsed.Source
	result.Residuals = append([]string(nil), notes...)
	if len(notes) != 0 {
		return result, ErrUnmatched
	}
	previousName := function.Name
	previousSignature := function.Signature
	previousSource := function.Source
	function.Name = parsed.Name
	function.Signature = parsed.Signature
	function.Source = parsed.Source
	if err := artifact.saveFunction(ctx, function); err != nil {
		function.Name = previousName
		function.Signature = previousSignature
		function.Source = previousSource
		return result, err
	}
	result.Matched = true
	return result, nil
}

type candidate struct {
	Name      string       `json:"name"`
	Signature string       `json:"signature"`
	Source    string       `json:"source"`
	Blocks    []blockClaim `json:"blocks"`
}

type blockClaim struct {
	Address hexAddress `json:"address"`
	Kind    string     `json:"kind"`
	Target  hexAddress `json:"target"`
	Fall    hexAddress `json:"fall"`
}

type hexAddress uint64

func (address *hexAddress) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if data[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		value, err := parseAddress(text)
		if err != nil {
			return err
		}
		*address = hexAddress(value)
		return nil
	}
	var value uint64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*address = hexAddress(value)
	return nil
}

func parseAddress(text string) (uint64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, nil
	}
	value, err := strconv.ParseUint(text, 0, 64)
	if err != nil {
		return 0, fmt.Errorf("%w %q: %w", ErrParseAddressAddress, text, err)
	}
	return value, nil
}

func parseCandidate(text string) (candidate, error) {
	raw, err := extractObject(text)
	if err != nil {
		return candidate{}, err
	}
	var parsed candidate
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return candidate{}, fmt.Errorf("%w: %w", ErrParseCandidateCandidate, err)
	}
	return parsed, nil
}

func extractObject(text string) (json.RawMessage, error) {
	start := strings.IndexByte(text, '{')
	if start < 0 {
		return nil, ErrCandidateHasNoJsonObject
	}
	depth := 0
	inString := false
	escape := false
	for index := start; index < len(text); index++ {
		char := text[index]
		if inString {
			if escape {
				escape = false
				continue
			}
			switch char {
			case '\\':
				escape = true
			case '"':
				inString = false
			}
			continue
		}
		switch char {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return json.RawMessage(text[start : index+1]), nil
			}
		}
	}
	return nil, ErrCandidateHasAnUnfinishedJsonObject
}

func differences(function Function, parsed candidate, artifact *Artifact) []string {
	var notes []string
	if parsed.Name == "" {
		notes = append(notes, "name is empty")
	}
	if !strings.Contains(parsed.Source, "{") || !strings.Contains(parsed.Source, "}") {
		notes = append(notes, "source has no function body")
	}
	if parsed.Name != "" && !strings.Contains(parsed.Source, parsed.Name) {
		notes = append(notes, "source does not mention "+parsed.Name)
	}
	for _, text := range function.Strings {
		if !strings.Contains(parsed.Source, text) {
			notes = append(notes, "source does not mention string "+text)
		}
	}
	seenCallee := map[uint64]struct{}{}
	for _, reference := range function.References {
		if reference.Kind != string(kindCall) {
			continue
		}
		if _, ok := seenCallee[reference.To]; ok {
			continue
		}
		seenCallee[reference.To] = struct{}{}
		name := artifact.nameAt(reference.To)
		if !strings.Contains(parsed.Source, name) {
			notes = append(notes, "source does not mention callee "+name)
		}
	}
	claimed := map[uint64]blockClaim{}
	for _, block := range parsed.Blocks {
		address := uint64(block.Address)
		if _, exists := claimed[address]; exists {
			notes = append(notes, fmt.Sprintf("block 0x%x claimed twice", address))
		}
		claimed[address] = block
	}
	expected := map[uint64]struct{}{}
	for _, block := range function.Blocks {
		expected[block.Address] = struct{}{}
		claim, ok := claimed[block.Address]
		if !ok {
			notes = append(notes, fmt.Sprintf("block 0x%x missing", block.Address))
			continue
		}
		if claim.Kind != block.Kind {
			notes = append(notes, fmt.Sprintf("block 0x%x kind is %s, candidate says %s", block.Address, block.Kind, claim.Kind))
		}
		if uint64(claim.Target) != block.Target {
			notes = append(notes, fmt.Sprintf("block 0x%x target is 0x%x, candidate says 0x%x", block.Address, block.Target, uint64(claim.Target)))
		}
		if uint64(claim.Fall) != block.Fall {
			notes = append(notes, fmt.Sprintf("block 0x%x fall is 0x%x, candidate says 0x%x", block.Address, block.Fall, uint64(claim.Fall)))
		}
	}
	var extra []uint64
	for address := range claimed {
		if _, ok := expected[address]; !ok {
			extra = append(extra, address)
		}
	}
	sort.Slice(extra, func(i, j int) bool { return extra[i] < extra[j] })
	for _, address := range extra {
		notes = append(notes, fmt.Sprintf("block 0x%x is not in the function", address))
	}
	return notes
}

// FormatAddress prints address as 0x and lowercase hex.
func FormatAddress(address uint64) string {
	return fmt.Sprintf("0x%x", address)
}
