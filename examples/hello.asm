// Assemble with
//   as hello.asm -g -o hello.o

.global party, visible
.align 2

//m9: ·party(SB),NOSPLIT,$0-32
party:
    mov X0, #0
    mov X16, #1
    svc 0

local:
    mov x0, #1

//m9: ·visible(SB),NOSPLIT,$0
visible:
    mov x0, #2
