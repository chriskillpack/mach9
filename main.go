package main

import (
	"debug/dwarf"
	"debug/macho"
	"fmt"
	"log"
	"os"
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
	Name   string
	Offset int
	Data   []byte
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

	// Extract the opcodes
	code := make([]byte, text.Size)
	if n, err := text.ReadAt(code, 0); n < int(text.Size) || err != nil {
		if err == nil {
			log.Fatal("Failed to read all bytes in section")
		}
		log.Fatal(err)
	}

	// We need the symbols table to help identify where each function begins
	// and ends in the assembly
	if mf.Symtab == nil {
		log.Fatal("Missing symbol table, can't do anything")
	}

	var symbols []symbol

	// Symbols declared with .global directive have type 15, 14 otherwise
	// There is this ltmp0 symbol which I don't know what it is, but it can be ignored
	for _, sym := range mf.Symtab.Syms {
		if sym.Name[0] == 'l' {
			// Skip "private link labels"
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

	for i := 0; i < len(symbols)-1; i++ {
		start := &symbols[i]
		end := &symbols[i+1]

		start.Data = code[start.Offset:end.Offset]
	}

	for _, symbol := range symbols[:len(symbols)-1] {
		outtemplate.Execute(os.Stdout, symbol)
	}
}

//lint:ignore U1000 This function is incomplete and will be finished later
func printDWARF(mf *macho.File) {
	dwarfdata, err := mf.DWARF()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("DWARF info")
	reader := dwarfdata.Reader()
	for {
		entry, err := reader.Next()
		if err != nil {
			log.Fatal(err)
			break
		}
		if entry == nil {
			break
		}

		if entry.Tag == dwarf.TagCompileUnit {
			cuName, ok := entry.Val(dwarf.AttrName).(string)
			_ = cuName
			if !ok {
				continue
			}
			lr, err := dwarfdata.LineReader(entry)
			if err != nil {
				continue
			}

			var le dwarf.LineEntry
			for {
				if lr.Next(&le) != nil {
					break
				}
				fmt.Printf("entry at line %d in %s: %+v\n", le.Line, le.File.Name, le)
			}
		}
		fmt.Printf("entry %+v\n", entry)
	}
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
