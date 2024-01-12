// To assemble:
//   as add_arm64.asm -g -o add_arm64.o

.global armadd

// func armadd(a, b int) int
//m9: ·armadd(SB),NOSPLIT,$0-24
armadd:
    ldr x0, [x29, #16]  // a
    ldr x1, [x29, #24]  // b
    add x0, x0, x1
    str x0, [x29, #32]  // result
    ret
