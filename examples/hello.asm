// Assemble with
//   as hello.asm -g -o hello.o

.global _main, _visible
.align 2

_main:
    mov X0, #0
    mov X16, #1
    svc 0

_local:
    mov x0, #1

_visible:
    mov x0, #2
