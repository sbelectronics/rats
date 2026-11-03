;  Mod55FC - hand-written ASM86 helper library (image 0x55FC-0x577C, frame 055Fh,
;  code starts at offset 0Ch of that frame).  PL/M-callable (far, Pascal order,
;  callee pops).  Only IsNil, NotNil, FracDiv and FracMul are referenced by Rats.
;  Built by tools/buildall.sh; checked by tools/verifyall.sh.
;  ASM86 notes: forward JMPs come out as EB xx 90 (3 bytes reserved); use -1, not
;  0FFFFH, to get the sign-extended 83 /7 ib form of CMP WORD PTR [..], imm.
;
;  IsNil(p POINTER) BYTE            TRUE if p = 0C000h:0FFFFh (the "nil" pointer)
;  NotNil(p POINTER) BYTE           complement of IsNil
;  FEqPtr(p1, p2 POINTER) BYTE       TRUE if p1 = p2 (both words compared)
;  FillDw(dw DWORD, pDst POINTER, cdw WORD)          store dw cdw times at pDst
;  MoveDw(pSrc POINTER, pDst POINTER, cdw WORD)      copy cdw dwords, ascending
;  MoveDwDown(pSrc POINTER, pDst POINTER, cdw WORD)  copy cdw dwords, descending
;                                                    (for overlapping moves up)
;  FindDw(dw DWORD, p POINTER, cdw WORD) WORD   index of dw in p(0..cdw-1) or 0FFFFh.
;        NB: original bug - the value searched for is loaded from the pointer
;        parameter ([bp+8]) instead of dw ([bp+0Ch]); reproduced as is.
;  SkipDw(p POINTER, dw DWORD, cdw WORD) WORD   index of first entry <> dw, or 0FFFFh
;  CmpDw(a DWORD, b DWORD) WORD      0 if a<b, 1 if a=b, 2 if a>b (unsigned)
;  GetDS WORD                        returns DS (DGROUP)
;  FracDiv(a, b WORD) WORD           (a*65536 + b/2) / b, 0FFFFh if a = b
;                                    (fraction a/b scaled to 16 bits, needs a<b)
;  FracMul(a, b WORD) WORD           (a*b + 8000h) / 65536 (rounded 16x16 product)

NAME    Mod55FC

DGROUP  GROUP   CONST
CONST   SEGMENT WORD PUBLIC 'CONST'
        PUBLIC  pNil
pNil    DW      0FFFFH, 0C000H          ; the CTOS nil pointer C000:FFFF (DGROUP 0C92h)
CONST   ENDS

Mod55FC_CODE    SEGMENT PUBLIC 'CODE'
        ASSUME  CS:Mod55FC_CODE

        PUBLIC  IsNil, NotNil, FEqPtr, FillDw, MoveDw, MoveDwDown
        PUBLIC  FindDw, SkipDw, CmpDw, GetDS, FracDiv, FracMul

IsNil  PROC    FAR
        PUSH    BP
        MOV     BP, SP
        CMP     WORD PTR [BP+8], 0C000H
        JNE     IsNil1
        CMP     WORD PTR [BP+6], -1
        JNE     IsNil1
        MOV     AL, 0FFH
        JMP     IsNil2
IsNil1: MOV     AL, 0
IsNil2: POP     BP
        RET     4
IsNil  ENDP

NotNil PROC    FAR
        PUSH    BP
        MOV     BP, SP
        CMP     WORD PTR [BP+8], 0C000H
        JNE     NotNil1
        CMP     WORD PTR [BP+6], -1
        JNE     NotNil1
        MOV     AL, 0
        JMP     NotNil2
NotNil1: MOV    AL, 0FFH
NotNil2: POP    BP
        RET     4
NotNil ENDP

FEqPtr  PROC    FAR
        PUSH    BP
        MOV     BP, SP
        MOV     AX, WORD PTR [BP+0CH]
        CMP     AX, WORD PTR [BP+8]
        JNE     EqPtr1
        MOV     AX, WORD PTR [BP+0AH]
        CMP     AX, WORD PTR [BP+6]
        JNE     EqPtr1
        MOV     AL, 0FFH
        JMP     EqPtr2
EqPtr1: MOV     AL, 0
EqPtr2: POP     BP
        RET     8
FEqPtr  ENDP

FillDw  PROC    FAR
        PUSH    BP
        MOV     BP, SP
        MOV     CX, WORD PTR [BP+6]
        JCXZ    Fill2
        LES     DI, DWORD PTR [BP+8]
        MOV     AX, WORD PTR [BP+0CH]
        MOV     DX, WORD PTR [BP+0EH]
        CLD
Fill1:  STOSW
        XCHG    AX, DX
        STOSW
        XCHG    AX, DX
        LOOP    Fill1
Fill2:  POP     BP
        RET     0AH
FillDw  ENDP

MoveDw  PROC    FAR
        PUSH    BP
        MOV     BP, SP
        MOV     CX, WORD PTR [BP+6]
        JCXZ    Move1
        PUSH    DS
        SHL     CX, 1
        LES     DI, DWORD PTR [BP+8]
        LDS     SI, DWORD PTR [BP+0CH]
        CLD
        REP     MOVSW
        POP     DS
Move1:  POP     BP
        RET     0AH
MoveDw  ENDP

MoveDwDown PROC FAR
        PUSH    BP
        MOV     BP, SP
        MOV     CX, WORD PTR [BP+6]
        JCXZ    MoveD1
        PUSH    DS
        SHL     CX, 1
        MOV     AX, CX
        DEC     AX
        SHL     AX, 1
        LES     DI, DWORD PTR [BP+8]
        ADD     DI, AX
        LDS     SI, DWORD PTR [BP+0CH]
        ADD     SI, AX
        STD
        REP     MOVSW
        POP     DS
MoveD1: POP     BP
        RET     0AH
MoveDwDown ENDP

FindDw  PROC    FAR
        PUSH    BP
        MOV     BP, SP
        MOV     CX, WORD PTR [BP+6]
        JCXZ    Find2
        SHL     CX, 1
        LES     DI, DWORD PTR [BP+8]
        MOV     AX, WORD PTR [BP+8]
        MOV     DX, WORD PTR [BP+0AH]
        CLD
Find1:  REPNE   SCASW
        JNE     Find2
        MOV     BX, DI
        SUB     BX, WORD PTR [BP+8]
        TEST    BX, 2
        JE      Find1
        CMP     DX, ES:WORD PTR [DI]
        JNE     Find1
        MOV     AX, BX
        SHR     AX, 1
        SHR     AX, 1
        JMP     Find3
Find2:  MOV     AX, 0FFFFH
Find3:  POP     BP
        RET     0AH
FindDw  ENDP

SkipDw  PROC    FAR
        PUSH    BP
        MOV     BP, SP
        MOV     CX, WORD PTR [BP+6]
        JCXZ    Skip2
        SHL     CX, 1
        LES     DI, DWORD PTR [BP+0CH]
        MOV     AX, WORD PTR [BP+8]
        MOV     DX, WORD PTR [BP+0AH]
        CLD
Skip1:  REPE    SCASW
        JE      Skip2
        MOV     BX, DI
        SUB     BX, WORD PTR [BP+0CH]
        TEST    BX, 2
        JE      Skip1
        CMP     DX, ES:WORD PTR [DI]
        JE      Skip1
        MOV     AX, BX
        SHR     AX, 1
        SHR     AX, 1
        JMP     Skip3
Skip2:  MOV     AX, 0FFFFH
Skip3:  POP     BP
        RET     0AH
SkipDw  ENDP

CmpDw   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        MOV     DX, WORD PTR [BP+0CH]
        CMP     DX, WORD PTR [BP+8]
        JB      Cmp1
        JA      Cmp2
        MOV     AX, WORD PTR [BP+0AH]
        CMP     AX, WORD PTR [BP+6]
        JB      Cmp1
        JA      Cmp2
        MOV     AX, 1
        JMP     Cmp3
Cmp1:   MOV     AX, 0
        JMP     Cmp3
Cmp2:   MOV     AX, 2
Cmp3:   POP     BP
        RET     8
CmpDw   ENDP

GetDS   PROC    FAR
        PUSH    BP
        MOV     BP, SP
        MOV     AX, DS
        POP     BP
        RET     0
GetDS   ENDP

FracDiv PROC    FAR
        PUSH    BP
        MOV     BP, SP
        MOV     DX, WORD PTR [BP+8]
        MOV     CX, WORD PTR [BP+6]
        CMP     DX, CX
        JNE     FDiv1
        MOV     AX, 0FFFFH
        JMP     FDiv2
FDiv1:  MOV     AX, CX
        SHR     AX, 1
        DIV     CX
FDiv2:  POP     BP
        RET     4
FracDiv ENDP

FracMul PROC    FAR
        PUSH    BP
        MOV     BP, SP
        MOV     AX, WORD PTR [BP+8]
        MUL     WORD PTR [BP+6]
        RCL     AX, 1
        ADC     DX, 0
        MOV     AX, DX
        POP     BP
        RET     4
FracMul ENDP

Mod55FC_CODE    ENDS
        END
