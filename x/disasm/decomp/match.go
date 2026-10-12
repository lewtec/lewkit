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

const systemPrompt = "You decompile one native function. Reply with one JSON object and no other text. " +
	"The object has name, signature, source, and blocks. source is a C function. " +
	"blocks must list every fact block with the same address, kind, target, and fall. " +
	"source must contain the function name, each direct callee name, and each referenced string."

// MatchConfig bounds one [Artifact.Match] call.
type MatchConfig struct {
	// Rounds is the number of completions. Zero uses 4.
	Rounds int
}

// Result is the last decompilation [Artifact.Match] considered.
type Result struct {
	Address   uint64
	Name      string
	Signature string
	Source    string
	Rounds    int
	Matched   bool
	Residuals []string
}

// Match asks the attached model to decompile the function at address.
// A round matches when the candidate blocks copy the lifted control flow
// and the source names the function, each direct callee, and each referenced string.
// The accepted source is stored on the function. An exhausted budget returns
// the last result and [ErrUnmatched].
func (artifact *Artifact) Match(ctx context.Context, address uint64, config MatchConfig) (Result, error) {
	if artifact.project == nil || artifact.project.model == nil {
		return Result{}, ErrNoModel
	}
	function := artifact.byAddress[address]
	if function == nil {
		return Result{}, fmt.Errorf("%w: 0x%x", ErrNotFound, address)
	}
	rounds := config.Rounds
	if rounds <= 0 {
		rounds = 4
	}
	var notes []string
	var result Result
	result.Address = address
	for round := 1; round <= rounds; round++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		text, err := artifact.project.model.Complete(ctx, artifact.prompt(*function, notes))
		if err != nil {
			return result, err
		}
		parsed, err := parseCandidate(text)
		result.Rounds = round
		if err != nil {
			notes = []string{err.Error()}
			result.Residuals = append([]string(nil), notes...)
			continue
		}
		notes = differences(*function, parsed, artifact)
		result.Name = parsed.Name
		result.Signature = parsed.Signature
		result.Source = parsed.Source
		result.Residuals = append([]string(nil), notes...)
		if len(notes) == 0 {
			result.Matched = true
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
				result.Matched = false
				return result, err
			}
			return result, nil
		}
	}
	return result, ErrUnmatched
}

func (artifact *Artifact) prompt(function Function, earlier []string) Prompt {
	listing, err := artifact.Listing(function.Address)
	if err != nil {
		listing = ""
	}
	facts := evidence{
		Name:      function.Name,
		Address:   formatAddress(function.Address),
		Signature: function.Signature,
		Listing:   listing,
		Strings:   function.Strings,
	}
	for _, block := range function.Blocks {
		item := factBlock{Address: formatAddress(block.Address), Kind: block.Kind}
		if block.Target != 0 {
			item.Target = formatAddress(block.Target)
		}
		if block.Fall != 0 {
			item.Fall = formatAddress(block.Fall)
		}
		facts.Blocks = append(facts.Blocks, item)
	}
	seen := map[uint64]struct{}{}
	for _, reference := range function.References {
		if reference.Kind != string(kindCall) {
			continue
		}
		if _, ok := seen[reference.To]; ok {
			continue
		}
		seen[reference.To] = struct{}{}
		facts.Callees = append(facts.Callees, factName{
			Address: formatAddress(reference.To),
			Name:    artifact.nameAt(reference.To),
		})
	}
	encoded, err := json.MarshalIndent(facts, "", "  ")
	if err != nil {
		encoded = []byte("{}")
	}
	var user strings.Builder
	user.WriteString("Match this function.\n\n")
	user.Write(encoded)
	if len(earlier) > 0 {
		user.WriteString("\n\nThe previous reply did not match:\n")
		for _, note := range earlier {
			user.WriteString("- ")
			user.WriteString(note)
			user.WriteByte('\n')
		}
	}
	return Prompt{System: systemPrompt, User: user.String()}
}

type evidence struct {
	Name      string      `json:"name"`
	Address   string      `json:"address"`
	Signature string      `json:"signature,omitempty"`
	Listing   string      `json:"listing"`
	Blocks    []factBlock `json:"blocks"`
	Callees   []factName  `json:"callees,omitempty"`
	Strings   []string    `json:"strings,omitempty"`
}

type factBlock struct {
	Address string `json:"address"`
	Kind    string `json:"kind"`
	Target  string `json:"target,omitempty"`
	Fall    string `json:"fall,omitempty"`
}

type factName struct {
	Address string `json:"address"`
	Name    string `json:"name"`
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
		return 0, fmt.Errorf("address %q: %w", text, err)
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
		return candidate{}, fmt.Errorf("candidate: %w", err)
	}
	return parsed, nil
}

func extractObject(text string) (json.RawMessage, error) {
	start := strings.IndexByte(text, '{')
	if start < 0 {
		return nil, fmt.Errorf("model response has no json object")
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
	return nil, fmt.Errorf("model response has an unfinished json object")
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

func formatAddress(address uint64) string {
	return fmt.Sprintf("0x%x", address)
}
