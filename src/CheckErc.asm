;  CheckErc - hand-written ASM86 error check with a replaceable handler
;  (image 0x3CB8-0x3D1C, frame 03CBh, code starts at offset 08h; its CONST
;  segment is DGROUP 0AF4h-0AFDh).  ASM, not PL/M: forward JMP = EB xx 90 and
;  the call through the saved handler on the stack (CALL [BP-4]).
;  Built by tools/buildall.sh; checked by tools/verifyall.sh.
;
;  Breakpoint                 executes INT 3 (debugger trap)
;  CheckErc(erc WORD)         erc = 0: return.  fBreakOnErc: trap repeatedly until
;                             fErcContinue is set (from the debugger), clear it and
;                             return.  Otherwise call pErcHandler(erc), with
;                             pErcHandler reset to pDefErcHandler while it runs
;                             (so an error inside the handler goes to the default
;                             handler); then the old handler is restored.
;  SetErcHandler(pProc POINTER)   install pProc as pErcHandler (unused by Rats)
;  Default handler: OS subroutine ErrorExit (FFEF:0334, OsSub 10).

NAME    CheckErc

        EXTRN   ErrorExit:FAR

DGROUP  GROUP   CONST
CONST   SEGMENT PUBLIC 'CONST'
fBreakOnErc     DB      0FFH
fErcContinue    DB      0
pErcHandler     DD      ErrorExit
pDefErcHandler  DD      ErrorExit
CONST   ENDS

CheckErc_CODE   SEGMENT PUBLIC 'CODE'
        ASSUME  CS:CheckErc_CODE, DS:DGROUP

        PUBLIC  Breakpoint, CheckErc, SetErcHandler

Breakpoint PROC FAR
        PUSH    BP
        MOV     BP, SP
        INT     3
        POP     BP
        RET     0
Breakpoint ENDP

CheckErc PROC   FAR
        PUSH    BP
        MOV     BP, SP
        CMP     WORD PTR [BP+6], 0
        JE      Chk3
        CMP     fBreakOnErc, 0FFH
        JNE     Chk2
Chk1:   CALL    Breakpoint
        CMP     fErcContinue, 0FFH
        JNE     Chk1
        MOV     fErcContinue, 0
        JMP     Chk3
Chk2:   PUSH    WORD PTR pErcHandler+2
        PUSH    WORD PTR pErcHandler
        LES     BX, pDefErcHandler
        MOV     WORD PTR pErcHandler, BX
        MOV     WORD PTR pErcHandler+2, ES
        PUSH    WORD PTR [BP+6]
        CALL    DWORD PTR [BP-4]
        POP     WORD PTR pErcHandler
        POP     WORD PTR pErcHandler+2
Chk3:   POP     BP
        RET     2
CheckErc ENDP

SetErcHandler PROC FAR
        PUSH    BP
        MOV     BP, SP
        LES     BX, DWORD PTR [BP+6]
        MOV     WORD PTR pErcHandler, BX
        MOV     WORD PTR pErcHandler+2, ES
        POP     BP
        RET     4
SetErcHandler ENDP

CheckErc_CODE   ENDS
        END
