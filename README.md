# Mach-9

_Work in progress. Some of the described behavior is aspirational and not yet complete._

A tool to convert Mach-O object files into Plan 9 assembler formatted output.

To use, first write some assembly and run it through your favorite assembler with debug info included

```
$ cat hello.asm
.global party, visible
.align 2

//m9: NOSPLIT,$0-32
party:
    mov X0, #0
    mov X16, #1
    svc 0

local:
    mov x0, #1

//m9:
visible:
    mov x0, #2

$ as -g -o hello.o hello.asm
```

Then pass the assembled object file to `mach9`:

```
$ mach9 hello.o | tee hello_arm.s
#include "textflag.h"

TEXT ·party,NOSPLIT,$0-32
	WORD $0x000080D2
	WORD $0x300080D2
	WORD $0x010000D4
	WORD $0x200080D2

TEXT ·visible
	WORD $0x400080D2
```

The output is a valid Plan9 assembly file that contains the assembled code as
byte sequence.

## m9 Directive

_TODO - functionality incomplete._

## Motivation

For another project I wanted to implement some Go functions in ARM assembly using the NEON instruction extensions (SIMD instruction set), which are not supported by Go Assembler's Intermediate Language. The solution is to use an assembler that does support the extensions and to convert the object code into literal sequences that the Go Assembler will accept.

## TODO

- Resolve questions about endianness
- ARM targets only support WORD, not LONG or BYTE. Using WORD pads to 4 bytes. Investigate what to do?
- Investigate why DWARF labels drop _ prefix from symbols. C thing?
- support data segment
