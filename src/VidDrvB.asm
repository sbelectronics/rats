;  VidDrvB - hand-written ASM86 video driver B (image 0x7200-0x7306, frame 0720h,
;  code starts at offset 0).  Driver for frame buffers of (char, attr) word
;  cells: cell at line + 2*col.  Same 8 entry points / calling convention as
;  VidDrvA (see there): called argument-less from the VidThk thunks, parameters
;  addressed through BP ([BP+0Ch] = last), RETF without a count.
;  Separate module (own segment at frame 720h, own copies of the helpers).
;  Built by tools/buildall.sh; checked by tools/verifyall.sh.
;
;  VidB0(pb, col, row, cnt)        read cnt characters into pb
;  VidB1(pb, col, row, cnt)        read cnt cells into pb as words (char, attr)
;  VidB2(col, row) -> AX           read one cell: AL = char, AH = attr
;  VidB3(col, row, cnt, attr)      fill cnt attributes with attr
;  VidB4(col, row, w)              write one cell (char LOW(w), attr HIGH(w))
;  VidB5(col, row, cnt, w)         fill cnt cells with w
;  VidB6(col, row, cnt, pb, attr)  write cnt chars from pb, all with attr
;  VidB7(col, row, cnt, pb)        write cnt (char, attr) words from pb
;  SrcPos(col, row)  (near)        DS = ES = row segment, SI -> cell
;  DstPos(col, row)  (near)        ES = row segment, DI -> cell

NAME    VidDrvB

DGROUP  GROUP   DATA
DATA    SEGMENT PUBLIC 'DATA'
        EXTRN   pVidBlocks:DWORD
DATA    ENDS

VidDrvB_CODE    SEGMENT PUBLIC 'CODE'
        ASSUME  CS:VidDrvB_CODE, DS:DGROUP

        PUBLIC  VidB0, VidB1, VidB2, VidB3, VidB4, VidB5, VidB6, VidB7

VidB0   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        PUSH    DS
        MOV     CX, WORD PTR [BP+0CH]
        JCXZ    B0Done
        PUSH    WORD PTR [BP+10H]
        PUSH    WORD PTR [BP+0EH]
        CALL    SrcPos
        LES     DI, DWORD PTR [BP+12H]
        CLD
B0Loop: LODSW
        STOSB
        LOOP    B0Loop
B0Done: POP     DS
        POP     BP
        RET
VidB0   ENDP

VidB1   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        PUSH    DS
        MOV     CX, WORD PTR [BP+0CH]
        JCXZ    B1Done
        PUSH    WORD PTR [BP+10H]
        PUSH    WORD PTR [BP+0EH]
        CALL    SrcPos
        LES     DI, DWORD PTR [BP+12H]
        CLD
        REP     MOVSW
B1Done: POP     DS
        POP     BP
        RET
VidB1   ENDP

VidB2   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        PUSH    DS
        PUSH    WORD PTR [BP+0EH]
        PUSH    WORD PTR [BP+0CH]
        CALL    SrcPos
        LODSW
        POP     DS
        POP     BP
        RET
VidB2   ENDP

VidB3   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        MOV     CX, WORD PTR [BP+0EH]
        JCXZ    B3Done
        PUSH    WORD PTR [BP+12H]
        PUSH    WORD PTR [BP+10H]
        CALL    DstPos
        MOV     AL, BYTE PTR [BP+0CH]
        CLD
B3Loop: INC     DI
        STOSB
        LOOP    B3Loop
B3Done: POP     BP
        RET
VidB3   ENDP

VidB4   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        PUSH    WORD PTR [BP+10H]
        PUSH    WORD PTR [BP+0EH]
        CALL    DstPos
        MOV     AX, WORD PTR [BP+0CH]
        STOSW
        POP     BP
        RET
VidB4   ENDP

VidB5   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        MOV     CX, WORD PTR [BP+0EH]
        JCXZ    B5Done
        PUSH    WORD PTR [BP+12H]
        PUSH    WORD PTR [BP+10H]
        CALL    DstPos
        MOV     AX, WORD PTR [BP+0CH]
        CLD
        REP     STOSW
B5Done: POP     BP
        RET
VidB5   ENDP

VidB6   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        PUSH    DS
        MOV     CX, WORD PTR [BP+12H]
        JCXZ    B6Done
        PUSH    WORD PTR [BP+16H]
        PUSH    WORD PTR [BP+14H]
        CALL    DstPos
        LDS     SI, DWORD PTR [BP+0EH]
        MOV     AH, BYTE PTR [BP+0CH]
        CLD
B6Loop: LODSB
        STOSW
        LOOP    B6Loop
B6Done: POP     DS
        POP     BP
        RET
VidB6   ENDP

VidB7   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        PUSH    DS
        MOV     CX, WORD PTR [BP+10H]
        JCXZ    B7Done
        PUSH    WORD PTR [BP+14H]
        PUSH    WORD PTR [BP+12H]
        CALL    DstPos
        LDS     SI, DWORD PTR [BP+0CH]
        CLD
        REP     MOVSW
B7Done: POP     DS
        POP     BP
        RET
VidB7   ENDP

SrcPos  PROC    NEAR
        PUSH    BP
        MOV     BP, SP
        MOV     AX, WORD PTR [BP+6]
        MOV     BX, WORD PTR [BP+4]
        LES     SI, pVidBlocks
        SHL     BX, 1
        SHL     BX, 1
        LES     SI, ES:DWORD PTR [BX+SI]
        SHL     AX, 1
        ADD     SI, AX
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
        SHL     AX, 1
        ADD     DI, AX
        POP     BP
        RET     4
DstPos  ENDP

VidDrvB_CODE    ENDS
        END
