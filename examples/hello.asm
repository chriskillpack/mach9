// Assemble with
//   as hello.asm -g -o hello.o

.global main, visible
.align 2

main:
    mov X0, #0
    mov X16, #1
    svc 0

local:
    mov x0, #1

visible:
    mov x0, #2
