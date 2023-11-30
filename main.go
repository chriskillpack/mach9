package main

// TODO - investigate why DWARF labels drop _ prefix from symbols. C thing?

import (
	"bufio"
	"bytes"
	"debug/dwarf"
	"debug/macho"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

var (
	funcmap = template.FuncMap{
		"EmitOpcodes": emitOpcodes,
	}

	outtemplate = template.Must(template.New("out").Funcs(funcmap).Parse(templ))
)

const templ = `TEXT ·{{.Name}},NOSPLIT,$0-16
{{EmitOpcodes .Data}}
`

type symbol struct {
	// These fields come from the Mach-O symbol table
	Name   string
	Offset int
	Data   []byte

	// This comes from the DWARF debugging info
	DeclLine int64
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("mach9: ")

	if len(os.Args) < 2 {
		log.Fatal("No object file provided")
	}

	mf, err := macho.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer mf.Close()

	// Find the __text section, this contains the assembly opcodes
	text := mf.Section("__text")
	if text == nil {
		log.Fatal("No __text section")
	}

	// We need the symbols table to help identify where each function begins
	// and ends in the assembly
	if mf.Symtab == nil {
		log.Fatal("Missing symbol table, can't do anything")
	}

	// Extract the assembled opcodes
	code := make([]byte, text.Size)
	if n, err := text.ReadAt(code, 0); n < int(text.Size) || err != nil {
		if err == nil {
			log.Fatal("Failed to read all bytes in section")
		}
		log.Fatal(err)
	}

	var symbols []symbol

	for _, sym := range mf.Symtab.Syms {
		// See notes.md: symbols which start with l are private link labels and
		// can be ignored. I'm not confident so documenting here but not
		// implementing.

		// The bottom bit of Type is set if the symbol is an external symbol,
		// one that can be referenced by the linker and other programs. See
		// page 44 https://github.com/aidansteele/osx-abi-macho-file-format-reference/blob/master/Mach-O_File_Format.pdf
		if sym.Type&1 == 0 {
			continue
		}

		symbols = append(symbols, symbol{
			Name: sym.Name, Offset: int(sym.Value),
		})
	}
	// Add a sentinel that represents the end of the code buffer to simplify the
	// next loop.
	symbols = append(symbols, symbol{Name: "", Offset: len(code)})

	// Sort the symbols in order how they appear in the file
	slices.SortFunc(symbols, func(a, b symbol) int {
		return int(a.Offset - b.Offset)
	})

	symmap := make(map[string]*symbol)

	for i := 0; i < len(symbols)-1; i++ {
		start := &symbols[i]
		end := &symbols[i+1]

		start.Data = code[start.Offset:end.Offset]
		symmap[start.Name] = start
	}

	var scanner *bufio.Scanner
	cu, dwarfOK := parseDWARF(mf, symmap)
	if dwarfOK {
		if data, err := os.ReadFile(cu); err == nil {
			scanner = bufio.NewScanner(bytes.NewBuffer(data))
		}
	}
	_ = scanner

	for _, symbol := range symbols[:len(symbols)-1] {
		outtemplate.Execute(os.Stdout, symbol)
	}
}

// Returns the path to the source file for the compile unit and whether DWARF
// info was available. It also updates the symbols in the symmap with the
// declaration line of the symbol.
func parseDWARF(mf *macho.File, symmap map[string]*symbol) (string, bool) {
	dwarfdata, err := mf.DWARF()
	if err != nil {
		return "", false
	}

	var cuPath string

	reader := dwarfdata.Reader()
	for {
		entry, err := reader.Next()
		if err != nil {
			return "", false
		}
		if entry == nil {
			break
		}

		switch entry.Tag {
		case dwarf.TagCompileUnit:
			cuName, ok := entry.Val(dwarf.AttrName).(string)
			if !ok {
				continue
			}
			cuCompDir, ok := entry.Val(dwarf.AttrCompDir).(string)
			if !ok {
				continue
			}
			cuPath = filepath.Join(cuCompDir, cuName)

		case dwarf.TagLabel:
			labelName, ok := entry.Val(dwarf.AttrName).(string)
			if !ok {
				continue
			}

			declLine, ok := entry.Val(dwarf.AttrDeclLine).(int64)
			if !ok {
				continue
			}

			if sym, ok := symmap[labelName]; ok {
				sym.DeclLine = declLine
			}
		}
	}

	return cuPath, true
}

// Builds a string that contains the opcodes as literal bytes in Plan9 assembler format
func emitOpcodes(code []byte) string {
	builder := strings.Builder{}

	n := len(code)
	off := 0
	s := n / 4
	if s > 0 {
		for i := 0; i < s; i++ {
			opcodes := code[off : off+4]
			builder.WriteString(fmt.Sprintf("\tLONG $0x%02X%02X%02X%02X\n", opcodes[0], opcodes[1], opcodes[2], opcodes[3]))
			off += 4
		}
		n -= s * 4
	}
	s = n / 2
	if s > 0 {
		for i := 0; i < s; i++ {
			opcodes := code[off : off+2]
			builder.WriteString(fmt.Sprintf("\tWORD $0x%02X%02X\n", opcodes[0], opcodes[1]))
			off += 2
		}
		n -= s * 2
	}

	for i := 0; i < n; i++ {
		opcode := code[off]
		builder.WriteString(fmt.Sprintf("\tBYTE $0x%02X\n", opcode))
		off++
	}

	return builder.String()
}
