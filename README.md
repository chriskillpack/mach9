# Mach-9

_Work in progress_

A tool to convert Mach-O object files into Plan 9 assembler formatted output

To use, first compile your assembly code with debug info included

```
as -g -o hello.o hello.asm
```

Then run it through the tool:

```
$ mach9 hello.o
#include "textflag.h"

TEXT ·party,NOSPLIT,$0-16
	WORD $0x000080D2
	WORD $0x300080D2
	WORD $0x010000D4
	WORD $0x200080D2

TEXT ·visible,NOSPLIT,$0-16
	WORD $0x400080D2
```

## Motivation

For another project I wanted to implement some Go functions in ARM assembly using the NEON instruction extensions (SIMD instruction set), which are not supported by Go Assembler's Intermediate Language. The solution is to use an assembler that does support the extensions and to convert the object code into literal sequences that the Go Assembler will accept.