package decompview

import (
	"encoding/hex"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/decomp"
	"github.com/lewtec/lewkit/x/disasm"
)

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

func functionAssembly(artifact *decomp.Artifact, function decomp.Function) string {
	here := map[uint64]struct{}{}
	for _, instruction := range function.Instructions {
		here[instruction.Address] = struct{}{}
	}
	link := newLinker(artifact, artifact.Functions(), here, false)
	var body strings.Builder
	for _, block := range function.Blocks {
		writeBlock(&body, link, block)
		for _, instruction := range block.Instructions {
			writeInstruction(&body, link, artifact, function, instruction)
		}
	}
	writeLooseNotes(&body, link, artifact, function)
	for _, text := range function.Strings {
		fmt.Fprintf(&body, `<span class="asm-comment">; string %s</span>`+"\n", html.EscapeString(strconv.Quote(text)))
	}
	return body.String()
}

func streamAssembly(artifact *decomp.Artifact) string {
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
	var body strings.Builder
	for _, row := range rows {
		writeInstruction(&body, link, artifact, row.function, row.instruction)
	}
	return body.String()
}

func writeBlock(body *strings.Builder, link *linker, block decomp.Block) {
	body.WriteString(`<span class="asm-comment">; block `)
	link.writeHex(body, block.Address)
	body.WriteByte(' ')
	body.WriteString(html.EscapeString(block.Kind))
	if block.Target != 0 {
		body.WriteString(" -&gt; ")
		link.writeHex(body, block.Target)
	}
	if block.Fall != 0 {
		body.WriteString(" fall ")
		link.writeHex(body, block.Fall)
	}
	body.WriteString("</span>\n")
}

func writeInstruction(body *strings.Builder, link *linker, artifact *decomp.Artifact, function decomp.Function, instruction disasm.Instruction) {
	if instruction.Address == function.Address && function.Comment != "" {
		fmt.Fprintf(body, `<span class="asm-comment">; %s</span>`+"\n", html.EscapeString(function.Comment))
	}
	link.writeLabel(body, instruction.Address)
	raw := hex.EncodeToString(instruction.Bytes)
	pad := 1
	if len(raw) < 16 {
		pad += 16 - len(raw)
	}
	id := ""
	if _, done := link.anchored[instruction.Address]; !done {
		link.anchored[instruction.Address] = struct{}{}
		id = ` id="` + formatHex(instruction.Address) + `"`
	}
	fmt.Fprintf(body, `<a class="asm-addr"%s href="#%s">0x%08x</a>  <span class="asm-bytes">%s</span>%s<span class="asm-mnem">%s</span>`,
		id, formatHex(instruction.Address), instruction.Address, raw, strings.Repeat(" ", pad), html.EscapeString(instruction.Mnemonic))
	if instruction.Operands != "" {
		body.WriteByte(' ')
		writeOperands(body, link, instruction.Operands)
	}
	body.WriteByte('\n')
	writeInstructionNotes(body, artifact, instruction)
}

func writeInstructionNotes(body *strings.Builder, artifact *decomp.Artifact, instruction disasm.Instruction) {
	written := map[uint64]struct{}{}
	if text, ok := artifact.Note(instruction.Address); ok {
		fmt.Fprintf(body, `<span class="asm-comment">; note %s</span>`+"\n", html.EscapeString(text))
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
		fmt.Fprintf(body, `<span class="asm-comment">; note %s</span>`+"\n", html.EscapeString(text))
		written[address] = struct{}{}
	}
}

func writeLooseNotes(body *strings.Builder, link *linker, artifact *decomp.Artifact, function decomp.Function) {
	placed := map[uint64]struct{}{}
	for _, instruction := range function.Instructions {
		placed[instruction.Address] = struct{}{}
	}
	for _, note := range artifact.Notes() {
		if _, ok := placed[note.Address]; ok {
			continue
		}
		if !containsAddress(function, note.Address) {
			continue
		}
		body.WriteString(`<span class="asm-comment">; note `)
		link.writeHex(body, note.Address)
		body.WriteByte(' ')
		body.WriteString(html.EscapeString(note.Text))
		body.WriteString("</span>\n")
	}
}

func containsAddress(function decomp.Function, address uint64) bool {
	if function.Size == 0 {
		return function.Address == address
	}
	return address >= function.Address && address < function.Address+function.Size
}

func (link *linker) writeLabel(body *strings.Builder, address uint64) {
	name, ok := link.artifact.Label(address)
	if !ok {
		return
	}
	escaped := html.EscapeString(name)
	if _, start := link.starts[address]; start && link.labelLinks {
		fmt.Fprintf(body, `<a class="asm-label" href="%s">&lt;%s&gt;:</a>`+"\n", html.EscapeString(functionPath(link.slug, address)), escaped)
		return
	}
	fmt.Fprintf(body, `<span class="asm-label">&lt;%s&gt;:</span>`+"\n", escaped)
}

func (link *linker) writeHex(body *strings.Builder, address uint64) {
	text := formatHex(address)
	href, ok := link.href(address)
	if !ok {
		fmt.Fprintf(body, `<span class="asm-imm">%s</span>`, text)
		return
	}
	fmt.Fprintf(body, `<a class="asm-addr" href="%s">%s</a>`, html.EscapeString(href), text)
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

func writeOperands(body *strings.Builder, link *linker, operands string) {
	for index := 0; index < len(operands); {
		if start, end, address, ok := operandHex(operands, index); ok {
			body.WriteString(html.EscapeString(operands[index:start]))
			token := operands[start:end]
			href, linked := link.href(address)
			if linked {
				fmt.Fprintf(body, `<a class="asm-addr" href="%s">%s</a>`, html.EscapeString(href), html.EscapeString(token))
			} else {
				fmt.Fprintf(body, `<span class="asm-imm">%s</span>`, html.EscapeString(token))
			}
			if name, named := link.artifact.Label(address); named {
				escaped := html.EscapeString(name)
				if linked {
					fmt.Fprintf(body, ` <a class="asm-label" href="%s">&lt;%s&gt;</a>`, html.EscapeString(href), escaped)
				} else {
					fmt.Fprintf(body, ` <span class="asm-label">&lt;%s&gt;</span>`, escaped)
				}
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
				fmt.Fprintf(body, `<span class="asm-reg">%s</span>`, html.EscapeString(word))
			} else {
				body.WriteString(html.EscapeString(word))
			}
			index = end
			continue
		}
		if operands[index] >= '0' && operands[index] <= '9' {
			end := index + 1
			for end < len(operands) && operands[end] >= '0' && operands[end] <= '9' {
				end++
			}
			fmt.Fprintf(body, `<span class="asm-imm">%s</span>`, operands[index:end])
			index = end
			continue
		}
		body.WriteString(html.EscapeString(operands[index : index+1]))
		index++
	}
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
