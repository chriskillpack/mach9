```
$ objdump -s hello.o

hello.o:	file format mach-o arm64

Contents of section __text:
 0000 000080d2 300080d2 010000d4 200080d2  ....0....... ...
 0010 400080d2                             @...
Contents of section __debug_info:
 0014 ba000000 04000000 00000801 00000000  ................
 0024 00000000 00000000 14000000 00000000  ................
 0034 68656c6c 6f2e7300 2f557365 72732f63  hello.s./Users/c
 0044 68726973 2f636f64 652f7372 632f6172  hris/code/src/ar
 0054 6d746573 74004170 706c6520 636c616e  mtest.Apple clan
 0064 67207665 7273696f 6e203132 2e302e35  g version 12.0.5
 0074 2028636c 616e672d 31323035 2e302e32   (clang-1205.0.2
 0084 322e3929 00018002 6d61696e 00010000  2.9)....main....
 0094 00080000 00000000 00000000 00026c6f  ..............lo
 00a4 63616c00 01000000 0d000000 0c000000  cal.............
 00b4 00000000 02766973 69626c65 00010000  .....visible....
 00c4 00100000 00100000 00000000 0000      ..............
Contents of section __debug_abbrev:
 00d2 01110110 17110112 0103081b 08250813  .............%..
 00e2 05000002 0a000308 3a063b06 11010000  ........:.;.....
 00f2 00                                   .
Contents of section __debug_aranges:
 00f3 2c000000 02000000 00000800 00000000  ,...............
 0103 00000000 00000000 14000000 00000000  ................
 0113 00000000 00000000 00000000 00000000  ................
Contents of section __debug_line:
 0123 3a000000 04001f00 00000101 01fb0e0d  :...............
 0133 00010101 01000000 01000001 0068656c  .............hel
 0143 6c6f2e73 00000000 00000902 00000000  lo.s............
 0153 00000000 1a4b4b4d 4d020400 0101      .....KKMM.....
```

```
$ objdump -t hello.o

hello.o:	file format mach-o arm64

SYMBOL TABLE:
0000000000000000 l     F __TEXT,__text ltmp0
000000000000000c l     F __TEXT,__text _local
00000000000000f3 l     O __DWARF,__debug_aranges ltmp1
0000000000000000 g     F __TEXT,__text _main
0000000000000010 g     F __TEXT,__text _visible
```

Output of symbol table as read using debug/macho
```
Symbol 0 = {Name:ltmp0 Type:14 Sect:1 Desc:0 Value:0}
Symbol 1 = {Name:_local Type:14 Sect:1 Desc:0 Value:12}
Symbol 2 = {Name:ltmp1 Type:14 Sect:4 Desc:0 Value:243}
Symbol 3 = {Name:_main Type:15 Sect:1 Desc:0 Value:0}
Symbol 4 = {Name:_visible Type:15 Sect:1 Desc:0 Value:16}
```

Notes on the ltmp? symbols. According to https://zhuanlan.zhihu.com/p/638457098
the l is a prefix and means LinkerPrivateGlobalPrefix on macOS platforms. Seemingly confirmed in the LLVM code docs, https://llvm.org/doxygen/classllvm_1_1MCAsmInfo.html#a952dc5034e99e797c493900cf8c9f299.
Feel like it should be okay to ignore symbols that start with an l.

Some notes on Go's handling of DWARF debug https://www.grant.pizza/blog/dwarf/

### Literal directives

Go's assembler has 3 directives for embedded literal byte sequences `BYTE`, `WORD` and `DWORD` but support varies by target architectures. By looking at source for each architecture https://github.com/golang/go/blob/master/src/cmd/internal/obj/arm/anames.go

| Platform | `BYTE` | `WORD` | `DWORD` |
| -------- | ------ | ------ | ------- |
| ARM      |        |    X   |    X    |
| ARM64    |        |    X   |    X    |
| Loong64  |        |    X   |         |
| MIPS     |        |    X   |         |
| PPC64    |        |    X   |    X    |
| RiscV    |        |    X   |         |
| s390x    |    X   |    X   |    X    |
| WASM     |        |    X   |         |
| x86      |    X   |    X   |         |

Next step is to find how they are implemented.

### Little Endian

We have assumed that the assembler processes the literals as little endian. x86 and Apple Silicon (ARM) are both little endian. Other platforms have not been tested.