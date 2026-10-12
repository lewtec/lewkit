package view

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/disasm"
	"github.com/lewtec/lewkit/x/disasm/decomp"
)

// asmToken is one piece of an assembly line. templ paints it.
type asmToken struct {
	Kind string
	Text string
	Href string
	ID   string
}

// asmListing is the assembly block, one slice per line.
type asmListing [][]asmToken

// registerWord matches an operand token that is a register.
// Memory-size words such as dword and ptr are left as plain text.
var registerWord = regexp.MustCompile(`^(?:` +
	`r(?:1[0-5]|[89])[bwd]?|` +
	`r(?:ax|bx|cx|dx|si|di|bp|sp)|` +
	`e(?:ax|bx|cx|dx|si|di|bp|sp)|` +
	`(?:ax|bx|cx|dx|si|di|bp|sp)|` +
	`[abcd][hl]|` +
	`(?:sil|dil|bpl|spl)|` +
	`[xyz]mm(?:[0-9]|1[0-5])|` +
	`(?:rip|eip|ip)|` +
	`[cdefgs]s|` +
	`[xw](?:[0-9]|[12][0-9]|30)|` +
	`(?:xzr|wzr|lr|pc|fp)|` +
	`st[0-7]` +
	`)$`)

// linker turns an address into an anchor on this page, a function page, or the address page.
type linker struct {
	artifact   *decomp.Artifact
	slug       string
	here       map[uint64]struct{}
	starts     map[uint64]struct{}
	owner      map[uint64]uint64
	covers     []addressSpan
	spans      []addressSpan
	anchored   map[uint64]struct{}
	labelLinks bool
}

// addressSpan is an instruction or a function extent.
// end is the first address past the span.
type addressSpan struct {
	start    uint64
	end      uint64
	function uint64
}

func newLinker(artifact *decomp.Artifact, functions []decomp.Function, here map[uint64]struct{}, labelLinks bool) *linker {
	if here == nil {
		here = map[uint64]struct{}{}
	}
	link := &linker{
		artifact:   artifact,
		slug:       artifact.Slug(),
		here:       here,
		starts:     make(map[uint64]struct{}, len(functions)),
		owner:      map[uint64]uint64{},
		anchored:   map[uint64]struct{}{},
		labelLinks: labelLinks,
	}
	for _, function := range functions {
		link.starts[function.Address] = struct{}{}
		if function.Size > 0 {
			link.spans = append(link.spans, addressSpan{
				start:    function.Address,
				end:      function.Address + function.Size,
				function: function.Address,
			})
		}
		for _, instruction := range function.Instructions {
			if _, ok := link.owner[instruction.Address]; ok || instruction.Size == 0 {
				continue
			}
			link.owner[instruction.Address] = function.Address
			link.covers = append(link.covers, addressSpan{
				start:    instruction.Address,
				end:      instruction.Address + uint64(instruction.Size),
				function: function.Address,
			})
		}
	}
	sort.Slice(link.covers, func(i, j int) bool { return link.covers[i].start < link.covers[j].start })
	sort.Slice(link.spans, func(i, j int) bool { return link.spans[i].start < link.spans[j].start })
	return link
}

func functionAssembly(artifact *decomp.Artifact, function decomp.Function) asmListing {
	here := map[uint64]struct{}{}
	for _, instruction := range function.Instructions {
		here[instruction.Address] = struct{}{}
	}
	link := newLinker(artifact, artifact.Functions(), here, false)
	var lines asmListing
	for _, block := range function.Blocks {
		lines = append(lines, blockLine(link, block))
		for _, instruction := range block.Instructions {
			lines = append(lines, instructionLines(link, artifact, function, instruction)...)
		}
	}
	lines = append(lines, looseNoteLines(link, artifact, function)...)
	for _, text := range function.Strings {
		lines = append(lines, []asmToken{{Kind: "comment", Text: "; string " + strconv.Quote(text)}})
	}
	return lines
}

func streamAssembly(artifact *decomp.Artifact) asmListing {
	functions := artifact.Functions()
	type row struct {
		function    decomp.Function
		instruction disasm.Instruction
	}
	rows := make([]row, 0)
	here := map[uint64]struct{}{}
	for _, function := range functions {
		for _, instruction := range function.Instructions {
			rows = append(rows, row{function: function, instruction: instruction})
			here[instruction.Address] = struct{}{}
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].instruction.Address < rows[j].instruction.Address
	})
	link := newLinker(artifact, functions, here, true)
	var lines asmListing
	for _, row := range rows {
		lines = append(lines, instructionLines(link, artifact, row.function, row.instruction)...)
	}
	return lines
}

func blockLine(link *linker, block decomp.Block) []asmToken {
	line := []asmToken{{Kind: "comment", Text: "; block "}, hexToken(link, block.Address), {Kind: "comment", Text: " " + block.Kind}}
	if block.Target != 0 {
		line = append(line, asmToken{Kind: "comment", Text: " -> "}, hexToken(link, block.Target))
	}
	if block.Fall != 0 {
		line = append(line, asmToken{Kind: "comment", Text: " fall "}, hexToken(link, block.Fall))
	}
	return line
}

func instructionLines(link *linker, artifact *decomp.Artifact, function decomp.Function, instruction disasm.Instruction) asmListing {
	var lines asmListing
	if instruction.Address == function.Address && function.Comment != "" {
		lines = append(lines, []asmToken{{Kind: "comment", Text: "; " + function.Comment}})
	}
	if label, ok := labelToken(link, instruction.Address); ok {
		lines = append(lines, []asmToken{label})
	}
	raw := hex.EncodeToString(instruction.Bytes)
	pad := 1
	if len(raw) < 16 {
		pad += 16 - len(raw)
	}
	id := ""
	if _, done := link.anchored[instruction.Address]; !done {
		link.anchored[instruction.Address] = struct{}{}
		id = formatHex(instruction.Address)
	}
	line := []asmToken{
		{Kind: "addr", Text: fmt.Sprintf("0x%08x", instruction.Address), Href: "#" + formatHex(instruction.Address), ID: id},
		{Kind: "plain", Text: "  "},
		{Kind: "bytes", Text: raw},
		{Kind: "plain", Text: strings.Repeat(" ", pad)},
		{Kind: "mnem", Text: instruction.Mnemonic},
	}
	if instruction.Operands != "" {
		line = append(line, asmToken{Kind: "plain", Text: " "})
		line = append(line, operandTokens(link, instruction.Operands)...)
	}
	lines = append(lines, line)
	lines = append(lines, instructionNoteLines(artifact, instruction)...)
	return lines
}

func instructionNoteLines(artifact *decomp.Artifact, instruction disasm.Instruction) asmListing {
	var lines asmListing
	written := map[uint64]struct{}{}
	if text, ok := artifact.Note(instruction.Address); ok {
		lines = append(lines, []asmToken{{Kind: "comment", Text: "; note " + text}})
		written[instruction.Address] = struct{}{}
	}
	for _, address := range operandAddresses(instruction.Operands) {
		if _, seen := written[address]; seen {
			continue
		}
		text, ok := artifact.Note(address)
		if !ok {
			continue
		}
		lines = append(lines, []asmToken{{Kind: "comment", Text: "; note " + text}})
		written[address] = struct{}{}
	}
	return lines
}

func looseNoteLines(link *linker, artifact *decomp.Artifact, function decomp.Function) asmListing {
	placed := map[uint64]struct{}{}
	for _, instruction := range function.Instructions {
		placed[instruction.Address] = struct{}{}
	}
	var lines asmListing
	for _, note := range artifact.Notes() {
		if _, ok := placed[note.Address]; ok {
			continue
		}
		if !containsAddress(function, note.Address) {
			continue
		}
		lines = append(lines, []asmToken{
			{Kind: "comment", Text: "; note "},
			hexToken(link, note.Address),
			{Kind: "comment", Text: " " + note.Text},
		})
	}
	return lines
}

func containsAddress(function decomp.Function, address uint64) bool {
	if function.Size == 0 {
		return function.Address == address
	}
	return address >= function.Address && address < function.Address+function.Size
}

func labelToken(link *linker, address uint64) (asmToken, bool) {
	name, ok := link.artifact.Label(address)
	if !ok {
		return asmToken{}, false
	}
	token := asmToken{Kind: "label", Text: "<" + name + ">:"}
	if _, start := link.starts[address]; start && link.labelLinks {
		token.Href = functionPath(link.slug, address)
	}
	return token, true
}

func hexToken(link *linker, address uint64) asmToken {
	text := formatHex(address)
	href, ok := link.href(address)
	if !ok {
		return asmToken{Kind: "imm", Text: text}
	}
	return asmToken{Kind: "addr", Text: text, Href: href}
}

func (link *linker) href(address uint64) (string, bool) {
	if _, ok := link.here[address]; ok {
		return "#" + formatHex(address), true
	}
	if _, ok := link.starts[address]; ok {
		return functionPath(link.slug, address), true
	}
	if owner, ok := link.owner[address]; ok {
		return link.instruction(owner, address), true
	}
	if hit, ok := link.cover(address); ok {
		if _, onPage := link.here[hit.start]; onPage {
			return "#" + formatHex(hit.start), true
		}
		return link.instruction(hit.function, hit.start), true
	}
	if function, ok := link.span(address); ok {
		if _, onPage := link.here[function]; onPage {
			return "#" + formatHex(function), true
		}
		return functionPath(link.slug, function), true
	}
	if _, inImage := link.artifact.Contains(address); inImage {
		return artifactPath(link.slug) + "/at/" + formatHex(address), true
	}
	return "", false
}

func (link *linker) instruction(function, address uint64) string {
	if _, onPage := link.here[address]; onPage {
		return "#" + formatHex(address)
	}
	return functionPath(link.slug, function) + "#" + formatHex(address)
}

// cover is the instruction that contains address when address is not the first byte.
func (link *linker) cover(address uint64) (addressSpan, bool) {
	index := sort.Search(len(link.covers), func(i int) bool { return link.covers[i].start > address }) - 1
	if index < 0 {
		return addressSpan{}, false
	}
	hit := link.covers[index]
	if address >= hit.start && address < hit.end {
		return hit, true
	}
	return addressSpan{}, false
}

// span is the tightest function extent that contains address.
func (link *linker) span(address uint64) (uint64, bool) {
	index := sort.Search(len(link.spans), func(i int) bool { return link.spans[i].start > address }) - 1
	var function uint64
	width := uint64(^uint64(0))
	found := false
	for index >= 0 {
		hit := link.spans[index]
		if address >= hit.start && address < hit.end && hit.end-hit.start < width {
			function = hit.function
			width = hit.end - hit.start
			found = true
		}
		index--
	}
	return function, found
}

func operandTokens(link *linker, operands string) []asmToken {
	var tokens []asmToken
	var plain strings.Builder
	flush := func() {
		if plain.Len() == 0 {
			return
		}
		tokens = append(tokens, asmToken{Kind: "plain", Text: plain.String()})
		plain.Reset()
	}
	for index := 0; index < len(operands); {
		if start, end, address, ok := operandHex(operands, index); ok {
			plain.WriteString(operands[index:start])
			flush()
			token := operands[start:end]
			href, linked := link.href(address)
			if linked {
				tokens = append(tokens, asmToken{Kind: "addr", Text: token, Href: href})
			} else {
				tokens = append(tokens, asmToken{Kind: "imm", Text: token})
			}
			if name, named := link.artifact.Label(address); named {
				label := asmToken{Kind: "label", Text: " <" + name + ">"}
				if linked {
					label.Href = href
				}
				tokens = append(tokens, label)
			}
			index = end
			continue
		}
		if isIdentStart(operands[index]) {
			end := index + 1
			for end < len(operands) && isIdentCont(operands[end]) {
				end++
			}
			word := operands[index:end]
			if isRegister(word) {
				flush()
				tokens = append(tokens, asmToken{Kind: "reg", Text: word})
			} else {
				plain.WriteString(word)
			}
			index = end
			continue
		}
		if operands[index] >= '0' && operands[index] <= '9' {
			end := index + 1
			for end < len(operands) && operands[end] >= '0' && operands[end] <= '9' {
				end++
			}
			flush()
			tokens = append(tokens, asmToken{Kind: "imm", Text: operands[index:end]})
			index = end
			continue
		}
		plain.WriteByte(operands[index])
		index++
	}
	flush()
	return tokens
}

func operandAddresses(operands string) []uint64 {
	var addresses []uint64
	for index := 0; index < len(operands); {
		_, end, address, ok := operandHex(operands, index)
		if !ok {
			index++
			continue
		}
		addresses = append(addresses, address)
		index = end
	}
	return addresses
}

func operandHex(text string, index int) (start, end int, address uint64, ok bool) {
	if index+2 > len(text) || text[index] != '0' || (text[index+1] != 'x' && text[index+1] != 'X') {
		return 0, 0, 0, false
	}
	if index > 0 {
		prev := text[index-1]
		if isIdentCont(prev) {
			return 0, 0, 0, false
		}
	}
	end = index + 2
	for end < len(text) && isHex(text[end]) {
		end++
	}
	if end == index+2 {
		return 0, 0, 0, false
	}
	if end < len(text) && isIdentCont(text[end]) {
		return 0, 0, 0, false
	}
	address, err := strconv.ParseUint(text[index+2:end], 16, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	return index, end, address, true
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func isIdentStart(c byte) bool {
	return c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
}

func isIdentCont(c byte) bool {
	return isIdentStart(c) || c >= '0' && c <= '9'
}

func isRegister(word string) bool {
	return registerWord.MatchString(strings.ToLower(word))
}
