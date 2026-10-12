package decomp

import (
	"context"
	"fmt"
	"sort"

	"github.com/lewtec/lewkit/x/disasm"
)

func lift(ctx context.Context, object disasm.Object, format string) (*Artifact, error) {
	architecture, mode, err := object.Architecture()
	if err != nil {
		return nil, err
	}
	sections, err := object.Sections()
	if err != nil {
		return nil, err
	}
	symbols := rebaseSymbols(object, sections)
	var executable bool
	for _, section := range sections {
		if section.Exec && len(section.Bytes) > 0 {
			executable = true
			break
		}
	}
	if !executable {
		return nil, ErrNoCode
	}
	options := []disasm.Option{}
	if architecture == disasm.ArchitectureX86 {
		options = append(options, disasm.WithSyntax(disasm.SyntaxIntel))
	}
	engine, err := disasm.Open(ctx, architecture, mode, options...)
	if err != nil {
		return nil, err
	}
	defer engine.Close()

	decoded := map[uint64]disasm.Instruction{}
	step := instructionStep(architecture, mode)
	for _, section := range sections {
		if !section.Exec || len(section.Bytes) == 0 {
			continue
		}
		if err := decodeSection(ctx, engine, section, step, decoded); err != nil {
			return nil, err
		}
	}

	names := map[uint64]string{}
	for _, symbol := range symbols {
		if symbol.Name == "" {
			continue
		}
		if _, exists := names[symbol.Address]; exists {
			continue
		}
		names[symbol.Address] = symbol.Name
	}
	imports := imageImports(object, names)
	exports := imageExports(object, names, decoded)
	starts := functionStarts(object, symbols, sections, decoded, imports, names, exports)
	texts := scanStrings(sections)
	functions := recoverFunctions(starts, decoded, names, sections, texts)
	artifact := &Artifact{
		format:       format,
		architecture: architecture,
		mode:         mode,
		entry:        object.Entry(),
		sections:     sections,
		symbols:      names,
		notes:        map[uint64]string{},
		imageStrings: texts,
		functions:    functions,
		byAddress:    map[uint64]*Function{},
	}
	indexArtifact(artifact)
	return artifact, nil
}

func imageImports(object disasm.Object, names map[uint64]string) map[uint64]string {
	importer, ok := object.(interface {
		Imports() ([]disasm.Import, error)
	})
	if !ok {
		return nil
	}
	imported, err := importer.Imports()
	if err != nil || len(imported) == 0 {
		return nil
	}
	imports := map[uint64]string{}
	for _, item := range imported {
		if item.Name == "" {
			continue
		}
		label := item.Name
		if item.Library != "" {
			label = item.Library + "!" + item.Name
		}
		if _, exists := imports[item.Address]; exists {
			continue
		}
		imports[item.Address] = label
		if _, exists := names[item.Address]; !exists {
			names[item.Address] = label
		}
	}
	return imports
}

func imageExports(object disasm.Object, names map[uint64]string, decoded map[uint64]disasm.Instruction) map[uint64]struct{} {
	exporter, ok := object.(interface {
		Exports() ([]disasm.Export, error)
	})
	if !ok {
		return nil
	}
	exported, err := exporter.Exports()
	if err != nil {
		return nil
	}
	starts := map[uint64]struct{}{}
	for _, item := range exported {
		if item.Name == "" {
			continue
		}
		if _, exists := names[item.Address]; !exists {
			names[item.Address] = item.Name
		}
		if _, ok := decoded[item.Address]; ok {
			starts[item.Address] = struct{}{}
		}
	}
	return starts
}

func functionStarts(object disasm.Object, symbols disasm.Symbols, sections []disasm.Section, decoded map[uint64]disasm.Instruction, imports, names map[uint64]string, exports map[uint64]struct{}) []uint64 {
	starts := map[uint64]struct{}{}
	if _, ok := decoded[object.Entry()]; ok {
		starts[object.Entry()] = struct{}{}
	}
	for _, symbol := range symbols {
		if !symbol.Code {
			continue
		}
		if _, ok := decoded[symbol.Address]; ok {
			starts[symbol.Address] = struct{}{}
		}
	}
	for address := range exports {
		starts[address] = struct{}{}
	}
	for _, section := range sections {
		if !section.Exec {
			continue
		}
		if address, ok := firstInstruction(section, decoded); ok {
			starts[address] = struct{}{}
		}
	}
	for _, instruction := range decoded {
		flow := inspect(instruction.Mnemonic, instruction.Operands)
		if flow.kind == kindCall && flow.direct {
			if _, ok := decoded[flow.target]; ok {
				starts[flow.target] = struct{}{}
			}
		}
		if flow.kind != kindJump || len(imports) == 0 {
			continue
		}
		next := instruction.Address + uint64(instruction.Size)
		for _, address := range referencedAddresses(instruction.Operands, next) {
			name := imports[address]
			if name == "" {
				continue
			}
			starts[instruction.Address] = struct{}{}
			if _, exists := names[instruction.Address]; !exists {
				names[instruction.Address] = name
			}
		}
	}
	ordered := make([]uint64, 0, len(starts))
	for address := range starts {
		ordered = append(ordered, address)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	return ordered
}

func recoverFunctions(starts []uint64, decoded map[uint64]disasm.Instruction, names map[uint64]string, sections []disasm.Section, texts []cstring) []Function {
	claimed := map[uint64]struct{}{}
	boundaries := map[uint64]struct{}{}
	for _, start := range starts {
		boundaries[start] = struct{}{}
	}
	functions := make([]Function, 0, len(starts))
	for _, start := range starts {
		if _, taken := claimed[start]; taken {
			continue
		}
		instructions := walkFunction(start, decoded, claimed, boundaries)
		if len(instructions) == 0 {
			continue
		}
		functions = append(functions, buildFunction(start, instructions, sections, texts, names))
	}
	var rest []uint64
	for address := range decoded {
		if _, taken := claimed[address]; taken {
			continue
		}
		rest = append(rest, address)
	}
	sort.Slice(rest, func(i, j int) bool { return rest[i] < rest[j] })
	for _, start := range rest {
		if _, taken := claimed[start]; taken {
			continue
		}
		instructions := walkFunction(start, decoded, claimed, boundaries)
		if len(instructions) == 0 {
			continue
		}
		functions = append(functions, buildFunction(start, instructions, sections, texts, names))
	}
	sort.Slice(functions, func(i, j int) bool { return functions[i].Address < functions[j].Address })
	return functions
}

func walkFunction(start uint64, decoded map[uint64]disasm.Instruction, claimed, boundaries map[uint64]struct{}) []disasm.Instruction {
	if _, ok := decoded[start]; !ok {
		return nil
	}
	queue := []uint64{start}
	var instructions []disasm.Instruction
	for len(queue) > 0 {
		address := queue[0]
		queue = queue[1:]
		if _, taken := claimed[address]; taken {
			continue
		}
		// Another function start, including an import thunk, stays that function.
		if address != start {
			if _, boundary := boundaries[address]; boundary {
				continue
			}
		}
		instruction, ok := decoded[address]
		if !ok {
			continue
		}
		claimed[address] = struct{}{}
		instructions = append(instructions, instruction)
		flow := inspect(instruction.Mnemonic, instruction.Operands)
		if flow.direct && (flow.kind == kindJump || flow.kind == kindBranch) {
			queue = append(queue, flow.target)
		}
		switch flow.kind {
		case kindPlain, kindBranch, kindCall:
			queue = append(queue, instruction.Address+uint64(instruction.Size))
		}
	}
	sort.Slice(instructions, func(i, j int) bool { return instructions[i].Address < instructions[j].Address })
	return instructions
}

func indexArtifact(artifact *Artifact) {
	artifact.byAddress = map[uint64]*Function{}
	for index := range artifact.functions {
		artifact.byAddress[artifact.functions[index].Address] = &artifact.functions[index]
	}
}

func rebaseSymbols(object disasm.Object, sections []disasm.Section) disasm.Symbols {
	symbols := object.Symbols()
	var base uint64
	for _, section := range sections {
		address, _, _, err := object.TextSection(section.Name)
		if err != nil || section.Address < address {
			continue
		}
		base = section.Address - address
		break
	}
	if base == 0 {
		return symbols
	}
	shifted := append(disasm.Symbols(nil), symbols...)
	for index := range shifted {
		shifted[index].Address += base
	}
	return shifted
}

func decodeSection(ctx context.Context, engine *disasm.Engine, section disasm.Section, step int, decoded map[uint64]disasm.Instruction) error {
	offset := 0
	for offset < len(section.Bytes) {
		if err := ctx.Err(); err != nil {
			return err
		}
		address := section.Address + uint64(offset)
		if instruction, ok := decoded[address]; ok {
			if instruction.Size == 0 {
				offset += step
				continue
			}
			offset += int(instruction.Size)
			continue
		}
		instructions, err := engine.Disassemble(ctx, section.Bytes[offset:], address)
		if err != nil {
			return err
		}
		if len(instructions) == 0 {
			offset += step
			continue
		}
		var last disasm.Instruction
		for _, instruction := range instructions {
			if _, exists := decoded[instruction.Address]; exists || instruction.Size == 0 {
				break
			}
			decoded[instruction.Address] = instruction
			last = instruction
		}
		if last.Size == 0 {
			offset += step
			continue
		}
		next := last.Address + uint64(last.Size)
		if next <= address {
			offset += step
			continue
		}
		offset = int(next - section.Address)
	}
	return nil
}

func instructionStep(architecture disasm.Architecture, mode disasm.Mode) int {
	if architecture == disasm.ArchitectureARM && mode&disasm.ModeThumb != 0 {
		return 2
	}
	switch architecture {
	case disasm.ArchitectureAArch64, disasm.ArchitectureARM, disasm.ArchitectureMIPS, disasm.ArchitectureRISCV, disasm.ArchitecturePowerPC:
		return 4
	default:
		return 1
	}
}

func firstInstruction(section disasm.Section, decoded map[uint64]disasm.Instruction) (uint64, bool) {
	var found uint64
	var ok bool
	limit := section.Address + section.Size
	if section.Size < uint64(len(section.Bytes)) {
		limit = section.Address + uint64(len(section.Bytes))
	}
	for address := range decoded {
		if address < section.Address || address >= limit {
			continue
		}
		if !ok || address < found {
			found = address
			ok = true
		}
	}
	return found, ok
}

func buildFunction(start uint64, instructions []disasm.Instruction, sections []disasm.Section, strings []cstring, names map[uint64]string) Function {
	function := Function{Address: start, Name: names[start]}
	if function.Name == "" {
		function.Name = fmt.Sprintf("sub_%x", start)
	}
	function.Instructions = instructions
	if len(instructions) > 0 {
		last := instructions[len(instructions)-1]
		function.Size = last.Address + uint64(last.Size) - start
	}
	function.Blocks = buildBlocks(instructions)
	seen := map[string]struct{}{}
	for _, instruction := range instructions {
		for _, reference := range referencesOf(instruction, sections) {
			key := fmt.Sprintf("%d:%d:%s", reference.From, reference.To, reference.Kind)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			function.References = append(function.References, reference)
			if reference.Kind != "data" {
				continue
			}
			text, ok := stringAt(strings, reference.To)
			if ok && !containsString(function.Strings, text) {
				function.Strings = append(function.Strings, text)
			}
		}
	}
	return function
}

func buildBlocks(instructions []disasm.Instruction) []Block {
	if len(instructions) == 0 {
		return nil
	}
	present := map[uint64]bool{}
	for _, instruction := range instructions {
		present[instruction.Address] = true
	}
	leaders := map[uint64]bool{instructions[0].Address: true}
	for index, instruction := range instructions {
		flow := inspect(instruction.Mnemonic, instruction.Operands)
		if flow.direct && present[flow.target] {
			leaders[flow.target] = true
		}
		if index+1 >= len(instructions) {
			continue
		}
		switch flow.kind {
		case kindBranch, kindCall, kindJump, kindReturn:
			leaders[instructions[index+1].Address] = true
		}
	}
	var blocks []Block
	for _, instruction := range instructions {
		if len(blocks) == 0 || leaders[instruction.Address] {
			blocks = append(blocks, Block{Address: instruction.Address})
		}
		last := len(blocks) - 1
		blocks[last].Instructions = append(blocks[last].Instructions, instruction)
	}
	for index := range blocks {
		owned := blocks[index].Instructions
		last := owned[len(owned)-1]
		next := last.Address + uint64(last.Size)
		flow := inspect(last.Mnemonic, last.Operands)
		blocks[index].Kind = string(flow.kind)
		if flow.direct {
			blocks[index].Target = flow.target
		}
		switch flow.kind {
		case kindPlain, kindBranch, kindCall:
			if present[next] {
				blocks[index].Fall = next
			}
		}
	}
	return blocks
}

func referencesOf(instruction disasm.Instruction, sections []disasm.Section) []Reference {
	next := instruction.Address + uint64(instruction.Size)
	flow := inspect(instruction.Mnemonic, instruction.Operands)
	var references []Reference
	if flow.direct && flow.target != 0 && containsAddress(sections, flow.target) {
		references = append(references, Reference{From: instruction.Address, To: flow.target, Kind: string(flow.kind)})
	}
	for _, address := range referencedAddresses(instruction.Operands, next) {
		if flow.direct && address == flow.target {
			continue
		}
		if containsAddress(sections, address) {
			references = append(references, Reference{From: instruction.Address, To: address, Kind: "data"})
		}
	}
	return references
}

func containsAddress(sections []disasm.Section, address uint64) bool {
	for _, section := range sections {
		if address >= section.Address && address < section.Address+section.Size {
			return true
		}
	}
	return false
}

type cstring struct {
	Address uint64
	Text    string
}

func scanStrings(sections []disasm.Section) []cstring {
	var found []cstring
	for _, section := range sections {
		if section.Exec || len(section.Bytes) == 0 {
			continue
		}
		found = append(found, scanASCII(section.Bytes, section.Address)...)
		found = append(found, scanUTF16(section.Bytes, section.Address)...)
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].Address == found[j].Address {
			return len(found[i].Text) > len(found[j].Text)
		}
		return found[i].Address < found[j].Address
	})
	if len(found) == 0 {
		return nil
	}
	unique := found[:1]
	for _, item := range found[1:] {
		if item.Address == unique[len(unique)-1].Address {
			continue
		}
		unique = append(unique, item)
	}
	return unique
}

func scanASCII(data []byte, base uint64) []cstring {
	var found []cstring
	index := 0
	for index < len(data) {
		if !printable(data[index]) {
			index++
			continue
		}
		end := index
		for end < len(data) && printable(data[end]) {
			end++
		}
		if end-index >= 4 && end < len(data) && data[end] == 0 {
			found = append(found, cstring{
				Address: base + uint64(index),
				Text:    string(data[index:end]),
			})
		}
		index = end
	}
	return found
}

func scanUTF16(data []byte, base uint64) []cstring {
	var found []cstring
	index := 0
	for index+1 < len(data) {
		if !utf16Printable(data, index) {
			index++
			continue
		}
		end := index
		for end+1 < len(data) && utf16Printable(data, end) {
			end += 2
		}
		if (end-index)/2 >= 4 && end+1 < len(data) && data[end] == 0 && data[end+1] == 0 {
			chars := make([]byte, 0, (end-index)/2)
			for cursor := index; cursor < end; cursor += 2 {
				chars = append(chars, data[cursor])
			}
			found = append(found, cstring{
				Address: base + uint64(index),
				Text:    string(chars),
			})
			index = end + 2
			continue
		}
		index++
	}
	return found
}

func utf16Printable(data []byte, index int) bool {
	return index+1 < len(data) && data[index+1] == 0 && printable(data[index])
}

func stringAt(strings []cstring, address uint64) (string, bool) {
	for _, item := range strings {
		if address >= item.Address && address < item.Address+uint64(len(item.Text)) {
			return item.Text, true
		}
	}
	return "", false
}

func containsString(values []string, text string) bool {
	for _, value := range values {
		if value == text {
			return true
		}
	}
	return false
}

func printable(value byte) bool {
	return value >= 0x20 && value < 0x7f
}
