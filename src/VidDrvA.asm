;  VidDrvA - hand-written ASM86 video driver A (image 0x6F80-0x71FE, frame 06F8h,
;  code starts at offset 0).  Driver for frame buffers with a separate attribute
;  area of 4-bit attributes packed two per byte (even column = low nibble, odd
;  column = high nibble).  The other driver (VidDrvB, 0x7200) is a separate
;  module with the same 8 entry points for char/attr word cells.
;  VidInit installs VidA0..VidA7 or VidB0..VidB7 (via rgpDrv, DGROUP 10EAh) into
;  rgpVid(0..7) (DGROUP 2F26h); they are called argument-less from the PL/M
;  thunks in VidThk (0x6F28) and address the thunk's own parameters through
;  BP: [BP+0Ch] = last parameter, ...  RETF without a count (the thunk pops).
;  Screen geometry from VidInit: pVidBlocks (2F1Ah) -> table of far pointers,
;  one per row; chars at line + wVid18 (2F18h) + col, attributes at
;  line + wVid16 (2F16h) + col/2.
;  Built by tools/buildall.sh; checked by tools/verifyall.sh.
;
;  VidA0(pb, col, row, cnt)        read cnt characters into pb
;  VidA1(pb, col, row, cnt)        read cnt cells into pb as words (char, attr)
;  VidA2(col, row) -> AX           read one cell: AL = char, AH = attr
;  VidA3(col, row, cnt, attr)      fill cnt attributes with attr
;  VidA4(col, row, w)              write one cell: char LOW(w), attr HIGH(w)
;  VidA5(col, row, cnt, w)         fill cnt cells with char LOW(w), attr HIGH(w)
;  VidA6(col, row, cnt, pb, attr)  write cnt chars from pb, all with attr
;  VidA7(col, row, cnt, pb)        write cnt (char, attr) words from pb
;  SrcPos(col, row)  (near)        DS = ES = row segment, SI -> char, BX -> attr byte
;  DstPos(col, row)  (near)        ES = row segment, DI -> char, BX -> attr byte

NAME    VidDrvA

DGROUP  GROUP   DATA
DATA    SEGMENT PUBLIC 'DATA'
        EXTRN   wVid16:WORD, wVid18:WORD, pVidBlocks:DWORD
DATA    ENDS

VidDrvA_CODE    SEGMENT PUBLIC 'CODE'
        ASSUME  CS:VidDrvA_CODE, DS:DGROUP

        PUBLIC  VidA0, VidA1, VidA2, VidA3, VidA4, VidA5, VidA6, VidA7

VidA0   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        PUSH    DS
        MOV     CX, WORD PTR [BP+0CH]
        JCXZ    A0Done
        PUSH    WORD PTR [BP+10H]
        PUSH    WORD PTR [BP+0EH]
        CALL    SrcPos
        LES     DI, DWORD PTR [BP+12H]
        CLD
        REP     MOVSB
A0Done: POP     DS
        POP     BP
        RET
VidA0   ENDP

VidA1   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        PUSH    DS
        MOV     CX, WORD PTR [BP+0CH]
        JCXZ    A1Done
        PUSH    WORD PTR [BP+10H]
        PUSH    WORD PTR [BP+0EH]
        CALL    SrcPos
        LES     DI, DWORD PTR [BP+12H]
        CLD
        TEST    WORD PTR [BP+10H], 1
        JE      A1Even
        MOV     DH, BYTE PTR [BX]
        INC     BX
        JMP     A1Odd
A1Even: MOV     DH, BYTE PTR [BX]
        INC     BX
        LODSB
        MOV     AH, DH
        AND     AH, 0FH
        STOSW
        LOOP    A1Odd
        JMP     A1Done
A1Odd:  LODSB
        MOV     AH, DH
        SHR     AH, 1
        SHR     AH, 1
        SHR     AH, 1
        SHR     AH, 1
        STOSW
        LOOP    A1Even
A1Done: POP     DS
        POP     BP
        RET
VidA1   ENDP

VidA2   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        PUSH    DS
        PUSH    WORD PTR [BP+0EH]
        PUSH    WORD PTR [BP+0CH]
        CALL    SrcPos
        LODSB
        MOV     AH, BYTE PTR [BX]
        TEST    WORD PTR [BP+0EH], 1
        JNE     A2Odd
        AND     AH, 0FH
        JMP     A2Done
A2Odd:  SHR     AH, 1
        SHR     AH, 1
        SHR     AH, 1
        SHR     AH, 1
A2Done: POP     DS
        POP     BP
        RET
VidA2   ENDP

VidA3   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        MOV     CX, WORD PTR [BP+0EH]
        JCXZ    A3Done
        PUSH    WORD PTR [BP+12H]
        PUSH    WORD PTR [BP+10H]
        CALL    DstPos
        MOV     AH, BYTE PTR [BP+0CH]
        CLD
        MOV     DI, BX
        TEST    WORD PTR [BP+12H], 1
        JE      A3Even
        MOV     AL, ES:BYTE PTR [DI]
        SHL     AL, 1
        SHL     AL, 1
        SHL     AL, 1
        SHL     AL, 1
        OR      AL, AH
        ROR     AL, 1
        ROR     AL, 1
        ROR     AL, 1
        ROR     AL, 1
        STOSB
        DEC     CX
A3Even: SHR     CX, 1
        JAE     A3Full
        ADD     DI, CX
        MOV     AL, ES:BYTE PTR [DI]
        AND     AL, 0F0H
        OR      AL, AH
        MOV     ES:BYTE PTR [DI], AL
        SUB     DI, CX
A3Full: JCXZ    A3Done
        MOV     AL, AH
        SHL     AL, 1
        SHL     AL, 1
        SHL     AL, 1
        SHL     AL, 1
        OR      AL, AH
        REP     STOSB
A3Done: POP     BP
        RET
VidA3   ENDP

VidA4   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        PUSH    WORD PTR [BP+10H]
        PUSH    WORD PTR [BP+0EH]
        CALL    DstPos
        MOV     AX, WORD PTR [BP+0CH]
        STOSB
        MOV     DI, BX
        MOV     AL, ES:BYTE PTR [DI]
        TEST    WORD PTR [BP+10H], 1
        JNE     A4Odd
        AND     AL, 0F0H
        OR      AL, AH
        JMP     A4Store
A4Odd:  SHL     AL, 1
        SHL     AL, 1
        SHL     AL, 1
        SHL     AL, 1
        OR      AL, AH
        ROR     AL, 1
        ROR     AL, 1
        ROR     AL, 1
        ROR     AL, 1
A4Store: STOSB
        POP     BP
        RET
VidA4   ENDP

VidA5   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        MOV     CX, WORD PTR [BP+0EH]
        JCXZ    A5Done
        PUSH    WORD PTR [BP+12H]
        PUSH    WORD PTR [BP+10H]
        CALL    DstPos
        MOV     AX, WORD PTR [BP+0CH]
        CLD
        MOV     DX, CX
        REP     STOSB
        MOV     CX, DX
        MOV     DI, BX
        TEST    WORD PTR [BP+12H], 1
        JE      A5Even
        MOV     AL, ES:BYTE PTR [DI]
        SHL     AL, 1
        SHL     AL, 1
        SHL     AL, 1
        SHL     AL, 1
        OR      AL, AH
        ROR     AL, 1
        ROR     AL, 1
        ROR     AL, 1
        ROR     AL, 1
        STOSB
        DEC     CX
A5Even: SHR     CX, 1
        JAE     A5Full
        ADD     DI, CX
        MOV     AL, ES:BYTE PTR [DI]
        AND     AL, 0F0H
        OR      AL, AH
        MOV     ES:BYTE PTR [DI], AL
        SUB     DI, CX
A5Full: JCXZ    A5Done
        MOV     AL, AH
        SHL     AL, 1
        SHL     AL, 1
        SHL     AL, 1
        SHL     AL, 1
        OR      AL, AH
        REP     STOSB
A5Done: POP     BP
        RET
VidA5   ENDP

VidA6   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        PUSH    DS
        MOV     CX, WORD PTR [BP+12H]
        JCXZ    A6Done
        PUSH    WORD PTR [BP+16H]
        PUSH    WORD PTR [BP+14H]
        CALL    DstPos
        LDS     SI, DWORD PTR [BP+0EH]
        MOV     AH, BYTE PTR [BP+0CH]
        CLD
        MOV     DX, CX
        REP     MOVSB
        MOV     CX, DX
        MOV     DI, BX
        TEST    WORD PTR [BP+16H], 1
        JE      A6Even
        MOV     AL, ES:BYTE PTR [DI]
        SHL     AL, 1
        SHL     AL, 1
        SHL     AL, 1
        SHL     AL, 1
        OR      AL, AH
        ROR     AL, 1
        ROR     AL, 1
        ROR     AL, 1
        ROR     AL, 1
        STOSB
        DEC     CX
A6Even: SHR     CX, 1
        JAE     A6Full
        ADD     DI, CX
        MOV     AL, ES:BYTE PTR [DI]
        AND     AL, 0F0H
        OR      AL, AH
        MOV     ES:BYTE PTR [DI], AL
        SUB     DI, CX
A6Full: JCXZ    A6Done
        MOV     AL, AH
        SHL     AL, 1
        SHL     AL, 1
        SHL     AL, 1
        SHL     AL, 1
        OR      AL, AH
        REP     STOSB
A6Done: POP     DS
        POP     BP
        RET
VidA6   ENDP

VidA7   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        PUSH    DS
        MOV     CX, WORD PTR [BP+10H]
        JCXZ    A7Done
        PUSH    WORD PTR [BP+14H]
        PUSH    WORD PTR [BP+12H]
        CALL    DstPos
        LDS     SI, DWORD PTR [BP+0CH]
        CLD
        TEST    WORD PTR [BP+14H], 1
        JE      A7Even
        MOV     DH, ES:BYTE PTR [BX]
        AND     DH, 0FH
        JMP     A7Odd
A7Even: LODSW
        STOSB
        MOV     DH, AH
        LOOP    A7Odd
        JMP     A7Last
A7Odd:  LODSW
        STOSB
        SHL     AH, 1
        SHL     AH, 1
        SHL     AH, 1
        SHL     AH, 1
        OR      AH, DH
        MOV     ES:BYTE PTR [BX], AH
        INC     BX
        LOOP    A7Even
        JMP     A7Done
A7Last: MOV     AH, ES:BYTE PTR [BX]
        AND     AH, 0F0H
        OR      AH, DH
        MOV     ES:BYTE PTR [BX], AH
A7Done: POP     DS
        POP     BP
        RET
VidA7   ENDP

SrcPos  PROC    NEAR
        PUSH    BP
        MOV     BP, SP
        MOV     AX, WORD PTR [BP+6]
        MOV     BX, WORD PTR [BP+4]
        LES     SI, pVidBlocks
        SHL     BX, 1
        SHL     BX, 1
        LES     SI, ES:DWORD PTR [BX+SI]
        MOV     BX, SI
        ADD     SI, wVid18
        ADD     SI, AX
        ADD     BX, wVid16
        SHR     AX, 1
        ADD     BX, AX
        MOV     AX, ES
        MOV     DS, AX
        POP     BP
        RET     4
SrcPos  ENDP

DstPos  PROC    NEAR
        PUSH    BP
        MOV     BP, SP
        MOV     AX, WORD PTR [BP+6]
        MOV     BX, WORD PTR [BP+4]
        LES     DI, pVidBlocks
        SHL     BX, 1
        SHL     BX, 1
        LES     DI, ES:DWORD PTR [BX+DI]
        MOV     BX, DI
        ADD     DI, wVid18
        ADD     DI, AX
        ADD     BX, wVid16
        SHR     AX, 1
        ADD     BX, AX
        POP     BP
        RET     4
DstPos  ENDP

VidDrvA_CODE    ENDS
        END
