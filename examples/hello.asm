// Assemble with
//   as hello.asm -g -o hello.o

.global party, visible
.align 2

//m9: ·party(SB),NOSPLIT,$0-32
party:
    ldr x0, [sp, #8]    ; this is a comment
    ldr x1, [sp, #16]   // also a comment
    add x0, x0, x1
    ret

local:
    mov x0, #1

//m9: ·visible(SB),NOSPLIT,$0
visible:
    mov x0, #2
