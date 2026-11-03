;  Thunk3616 - hand-written ASM86 indirect-call thunk (image 0x3616-0x362A,
;  frame 0361h, code starts at offset 06h; DATA pCallTarget at DGROUP 2CB4h).
;  Called from the forms runtime (0x3D72, 0x4BD9) as
;    CallProc(a WORD, b WORD, pProc POINTER) WORD
;  It removes pProc from the stack, keeps the caller's return address on top and
;  jumps to pProc, so pProc runs as if called directly as pProc(a, b): pProc
;  pops a/b itself (RET 4) and returns straight to the caller with its result.
;  Not reentrant (pProc is parked in a static).  The trailing RET is never reached.
;  Built by tools/buildall.sh; checked by tools/verifyall.sh.

NAME    Thunk3616

DGROUP  GROUP   DATA
DATA    SEGMENT PUBLIC 'DATA'
pCallTarget     DD      ?
DATA    ENDS

Thunk3616_CODE  SEGMENT PUBLIC 'CODE'
        ASSUME  CS:Thunk3616_CODE, DS:DGROUP

        PUBLIC  CallProc

CallProc PROC   FAR
        PUSH    BP
        POP     BP
        POP     BX
        POP     ES
        POP     WORD PTR pCallTarget
        POP     WORD PTR pCallTarget+2
        PUSH    ES
        PUSH    BX
        JMP     pCallTarget
        RET
CallProc ENDP

Thunk3616_CODE  ENDS
        END
