# Mach-9

_Work in progress_

A tool to convert Mach-O object files into Plan 9 assembler formatted output

To use, first compile your assembly code with debug info included

```
as -g -o hello.o hello.asm
```

Then run it through the tool:

```
mach9 hello.o > hello.s
```
