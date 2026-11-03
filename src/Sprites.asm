;  Sprites - hand-written ASM86 sprite draw/erase into the maze map
;  (image 0x287C-0x2930, frame 0287h, code starts at offset 0Ch of that frame).
;  Reached only through the far-pointer tables rgpDraw (DGROUP 0658h) and
;  rgpErase (DGROUP 066Ch), indexed by obj.size (entries 0 and 3 are null).
;  The map (pMap, DGROUP 1352h) is a byte per cell, cell index = (y SHL 8) OR x,
;  so INC BL/DEC BL step x and INC BH steps y.  No BP frame: args via DI = SP.
;  Built by tools/buildall.sh; checked by tools/verifyall.sh.
;
;  Draw1(ch BYTE, pos WORD, pPos WORD)  map(pos) = ch; rgPos slot (near ptr pPos) = pos
;  Draw2(ch BYTE, pos WORD, pPos WORD)  2-cell sprite: cells pos, pos+(0,1); the cell
;                                       code is incremented for each further cell and
;                                       each position is stored in the next pPos word
;  Draw4(ch BYTE, pos WORD, pPos WORD)  2x2 sprite: cells (x,y) (x+1,y) (x+1,y+1) (x,y+1)
;  Erase1(pos WORD)                     map(pos) = 0
;  Erase2(pos WORD)                     clears the two cells of a Draw2 sprite
;  Erase4(pos WORD)                     clears the four cells of a Draw4 sprite

NAME    Sprites

DGROUP  GROUP   DATA
DATA    SEGMENT PUBLIC 'DATA'
        EXTRN   pMap:DWORD
DATA    ENDS

Sprites_CODE    SEGMENT PUBLIC 'CODE'
        ASSUME  CS:Sprites_CODE, DS:DGROUP

        PUBLIC  Draw1, Draw2, Draw4, Erase1, Erase2, Erase4

Draw1   PROC    FAR
        MOV     DI, SP
        MOV     AL, BYTE PTR [DI+8]
        MOV     BX, WORD PTR [DI+6]
        MOV     SI, WORD PTR [DI+4]
        LES     DI, pMap
        MOV     ES:BYTE PTR [BX+DI], AL
        MOV     WORD PTR [SI], BX
        RET     6
Draw1   ENDP

Draw2   PROC    FAR
        MOV     DI, SP
        MOV     AL, BYTE PTR [DI+8]
        MOV     BX, WORD PTR [DI+6]
        MOV     SI, WORD PTR [DI+4]
        LES     DI, pMap
        MOV     ES:BYTE PTR [BX+DI], AL
        MOV     WORD PTR [SI], BX
        INC     AX
        INC     BH
        INC     SI
        INC     SI
        MOV     ES:BYTE PTR [BX+DI], AL
        MOV     WORD PTR [SI], BX
        RET     6
Draw2   ENDP

Draw4   PROC    FAR
        MOV     DI, SP
        MOV     AL, BYTE PTR [DI+8]
        MOV     BX, WORD PTR [DI+6]
        MOV     SI, WORD PTR [DI+4]
        LES     DI, pMap
        MOV     ES:BYTE PTR [BX+DI], AL
        MOV     WORD PTR [SI], BX
        INC     AX
        INC     BL
        INC     SI
        INC     SI
        MOV     ES:BYTE PTR [BX+DI], AL
        MOV     WORD PTR [SI], BX
        INC     AX
        INC     BH
        INC     SI
        INC     SI
        MOV     ES:BYTE PTR [BX+DI], AL
        MOV     WORD PTR [SI], BX
        INC     AX
        DEC     BL
        INC     SI
        INC     SI
        MOV     ES:BYTE PTR [BX+DI], AL
        MOV     WORD PTR [SI], BX
        RET     6
Draw4   ENDP

Erase1  PROC    FAR
        MOV     DI, SP
        MOV     AL, 0
        MOV     BX, WORD PTR [DI+4]
        LES     DI, pMap
        MOV     ES:BYTE PTR [BX+DI], AL
        RET     2
Erase1  ENDP

Erase2  PROC    FAR
        MOV     DI, SP
        MOV     AL, 0
        MOV     BX, WORD PTR [DI+4]
        LES     DI, pMap
        MOV     ES:BYTE PTR [BX+DI], AL
        INC     BH
        MOV     ES:BYTE PTR [BX+DI], AL
        RET     2
Erase2  ENDP

Erase4  PROC    FAR
        MOV     DI, SP
        MOV     AL, 0
        MOV     BX, WORD PTR [DI+4]
        LES     DI, pMap
        MOV     ES:BYTE PTR [BX+DI], AL
        INC     BL
        MOV     ES:BYTE PTR [BX+DI], AL
        INC     BH
        MOV     ES:BYTE PTR [BX+DI], AL
        DEC     BL
        MOV     ES:BYTE PTR [BX+DI], AL
        RET     2
Erase4  ENDP

Sprites_CODE    ENDS
        END
