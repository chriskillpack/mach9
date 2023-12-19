package main

import (
	"bufio"
	"bytes"
	"debug/dwarf"
	"debug/macho"
	"encoding/binary"
	"fmt"
	"io"
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

	functemplate = template.Must(template.New("functempl").Funcs(funcmap).Parse(funcTempl))
)

const (
	m9prefix  = "m9:"
	funcTempl = `TEXT {{.Markup}}
{{EmitOpcodes .Data}}
`
)

type symbol struct {
	// These fields come from the Mach-O symbol table
	Name   string
	Offset int
	Data   []byte

	// This comes from the DWARF debugging info
	DeclLine int

	// This comes from the source file
	Markup string
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

	cu, dwarfOK := parseDWARF(mf, symmap)
	if dwarfOK {
		if data, err := os.ReadFile(cu); err == nil {
			extractDecl(data, symmap)
		}
	}

	// Every symbol requires m9 markup
	invalid := false
	for _, v := range symmap {
		if v.Markup == "" {
			fmt.Fprintf(os.Stderr, "Symbol %q missing m9 declaration\n", v.Name)
			invalid = true
		}
	}
	if invalid {
		os.Exit(1)
	}

	generateOutput(os.Stdout, symbols[:len(symbols)-1])
}

func extractDecl(src []byte, symmap map[string]*symbol) {
	// Sort the symbols in declaration order
	syms := make([]*symbol, 0, len(symmap))
	for _, v := range symmap {
		syms = append(syms, v)
	}
	slices.SortFunc(syms, func(a, b *symbol) int {
		return int(a.DeclLine - b.DeclLine)
	})

	// Walk through the source front to back looking for declaration markers
	// on the line immediately preceeding the symbol declarations
	scanner := bufio.NewScanner(bytes.NewBuffer(src))
	ln := 1
	i := 0
	for scanner.Scan() && i < len(syms) {
		if ln == syms[i].DeclLine-1 {
			// Declaration lines contain "m9:"
			line := scanner.Text()
			idx := strings.Index(line, m9prefix)
			if idx != -1 {
				idx += len(m9prefix)
				syms[i].Markup = strings.TrimLeft(line[idx:], " \t")
			}
			i++
		}
		ln++
	}
}

func generateOutput(w io.Writer, symbols []symbol) error {
	fmt.Fprint(os.Stdout, `#include "textflag.h"

`)
	for _, symbol := range symbols {
		if err := functemplate.Execute(w, symbol); err != nil {
			return err
		}
	}

	return nil
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
				sym.DeclLine = int(declLine)
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
			builder.WriteString(fmt.Sprintf("\tWORD $%#8x\n", binary.LittleEndian.Uint32(opcodes)))
			off += 4
		}
		n -= s * 4
	}
	s = n / 2
	if s > 0 {
		for i := 0; i < s; i++ {
			opcodes := code[off : off+2]
			builder.WriteString(fmt.Sprintf("\tWORD $%#4x\n", binary.LittleEndian.Uint16(opcodes)))
			off += 2
		}
		n -= s * 2
	}

	for i := 0; i < n; i++ {
		opcode := code[off]
		builder.WriteString(fmt.Sprintf("\tWORD $%#2x\n", opcode))
		off++
	}

	return builder.String()
}
