package main

// lib_helpers.go - translation of the small hand-written ASM86 helpers:
//   Mod55FC.asm   (frame 055Fh) IsNil, NotNil, FEqPtr, FillDw, MoveDw, MoveDwDown, FindDw,
//                 SkipDw, CmpDw, GetDS, FracDiv, FracMul; CONST pNil at DGROUP 0C92h
//   CheckErc.asm  (frame 03CBh) Breakpoint, CheckErc, SetErcHandler; CONST 0AF4h..0AFDh
//   Thunk3616.asm (frame 0361h) CallProc; DATA pCallTarget at 2CB4h
//   Mod54E4.plm   (frame 054Eh) FLedOn, SetLed (keyboard LEDs)

const (
	ldgPNil           = 0x0C92 // pNil  DW 0FFFFh, 0C000h
	ldgDwNil          = 0x0C96 // dwNil DD 0FFFFFFFFh (Random's CONST, used by ErrMsg)
	ldgFBreakOnErc    = 0x0AF4 // DB 0FFh
	ldgFErcContinue   = 0x0AF5 // DB 0
	ldgPErcHandler    = 0x0AF6 // DD ErrorExit (FFEF:0334)
	ldgPDefErcHandler = 0x0AFA // DD ErrorExit
	ldgPCallTarget    = 0x2CB4 // Thunk3616 pCallTarget
)

// far address of the CTOS OS subroutine ErrorExit (OsSub 10) as stored in pErcHandler.
const (
	uFrameErrorExit uint16 = 0xFFEF
	uOffErrorExit   uint16 = 0x0334
)

// ---------------- Mod55FC ----------------

func hlpIsNil(p Ptr) byte {
	if p.Seg() == 0xC000 && p.Off() == 0xFFFF {
		return 0xFF
	}
	return 0
}

func hlpNotNil(p Ptr) byte {
	if p.Seg() == 0xC000 && p.Off() == 0xFFFF {
		return 0
	}
	return 0xFF
}

func hlpFEqPtr(p1, p2 Ptr) byte {
	if p1.Seg() == p2.Seg() && p1.Off() == p2.Off() {
		return 0xFF
	}
	return 0
}

// FillDw(dw, pDst, cdw): STOSW low, STOSW high, cdw times.
func hlpFillDw(dw uint32, pDst Ptr, cdw uint16) {
	di := pDst.Off()
	for cx := cdw; cx != 0; cx-- {
		SetFW(MkPtr(pDst.Seg(), di), uint16(dw))
		di += 2
		SetFW(MkPtr(pDst.Seg(), di), uint16(dw>>16))
		di += 2
	}
}

// MoveDw(pSrc, pDst, cdw): REP MOVSW of 2*cdw words, ascending (CX = cdw SHL 1, 16 bits).
func hlpMoveDw(pSrc, pDst Ptr, cdw uint16) {
	if cdw == 0 {
		return
	}
	cx := cdw << 1
	si, di := pSrc.Off(), pDst.Off()
	for ; cx != 0; cx-- {
		SetFW(MkPtr(pDst.Seg(), di), FW(MkPtr(pSrc.Seg(), si)))
		si += 2
		di += 2
	}
}

// MoveDwDown(pSrc, pDst, cdw): STD; REP MOVSW from the last word down.
func hlpMoveDwDown(pSrc, pDst Ptr, cdw uint16) {
	if cdw == 0 {
		return
	}
	cx := cdw << 1
	ax := (cx - 1) << 1
	si, di := pSrc.Off()+ax, pDst.Off()+ax
	for ; cx != 0; cx-- {
		SetFW(MkPtr(pDst.Seg(), di), FW(MkPtr(pSrc.Seg(), si)))
		si -= 2
		di -= 2
	}
}

// uRepScasw emulates REPNE SCASW (ne=true) / REPE SCASW (ne=false) on ES:DI with CX words.
// zf is the zero flag before the instruction (left unchanged when CX = 0).
func uRepScasw(seg uint16, di, cx *uint16, ax uint16, ne bool, zf bool) bool {
	for *cx != 0 {
		*cx--
		w := FW(MkPtr(seg, *di))
		*di += 2
		zf = w == ax
		if ne && zf || !ne && !zf {
			break
		}
	}
	return zf
}

// FindDw(dw, p, cdw): index of dw in p(0..cdw-1) or 0FFFFh.
// Original bug kept: the value searched for is taken from the pointer parameter
// (AX = [BP+8] = offset of p, DX = [BP+0Ah] = segment of p) instead of dw.
func hlpFindDw(dw uint32, p Ptr, cdw uint16) uint16 {
	_ = dw
	if cdw == 0 {
		return 0xFFFF
	}
	cx := cdw << 1
	zf := cx == 0 // flags of SHL CX,1
	seg, di := p.Seg(), p.Off()
	ax, dx := p.Off(), p.Seg()
	for {
		zf = uRepScasw(seg, &di, &cx, ax, true, zf)
		if !zf {
			return 0xFFFF
		}
		bx := di - p.Off()
		zf = bx&2 == 0 // TEST BX,2
		if zf {
			continue
		}
		zf = dx == FW(MkPtr(seg, di)) // CMP DX, ES:[DI]
		if !zf {
			continue
		}
		return bx >> 2
	}
}

// SkipDw(p, dw, cdw): index of the first entry <> dw, or 0FFFFh (logic exactly as the asm).
func hlpSkipDw(p Ptr, dw uint32, cdw uint16) uint16 {
	if cdw == 0 {
		return 0xFFFF
	}
	cx := cdw << 1
	zf := cx == 0
	seg, di := p.Seg(), p.Off()
	ax, dx := uint16(dw), uint16(dw>>16)
	for {
		zf = uRepScasw(seg, &di, &cx, ax, false, zf)
		if zf {
			return 0xFFFF
		}
		bx := di - p.Off()
		zf = bx&2 == 0
		if zf {
			continue
		}
		zf = dx == FW(MkPtr(seg, di))
		if zf {
			continue
		}
		return bx >> 2
	}
}

// CmpDw(a, b): 0 if a<b, 1 if a=b, 2 if a>b (unsigned).
func hlpCmpDw(a, b uint32) uint16 {
	switch {
	case a < b:
		return 0
	case a > b:
		return 2
	}
	return 1
}

func hlpGetDS() uint16 { return SegDG }

// FracDiv(a, b) = (a*65536 + b/2) / b, 0FFFFh if a = b.  A quotient that does not fit
// 16 bits is a divide error (INT 0) on the 8086.
func hlpFracDiv(a, b uint16) uint16 {
	if a == b {
		return 0xFFFF
	}
	if b == 0 {
		panic("FracDiv: divide error")
	}
	n := uint32(a)<<16 | uint32(b>>1)
	q := n / uint32(b)
	if q > 0xFFFF {
		panic("FracDiv: divide overflow")
	}
	return uint16(q)
}

// FracMul(a, b): MUL; RCL AX,1; ADC DX,0 -> high word rounded by bit 15 of the low word.
func hlpFracMul(a, b uint16) uint16 {
	prod := uint32(a) * uint32(b)
	return uint16(prod>>16) + uint16(prod>>15)&1
}

// ---------------- CheckErc.asm ----------------

// Breakpoint executes INT 3.  The port has no debugger.
func ceBreakpoint() {}

// CheckErc(erc): 0 -> return.  fBreakOnErc (0FFh in the image) would trap to the debugger
// repeatedly until fErcContinue is set; without a debugger the port terminates through
// ErrorExit(erc) instead (see api.go).  Otherwise pErcHandler(erc) is called with
// pErcHandler = pDefErcHandler while it runs, then restored.
func ceCheckErc(erc uint16) {
	if erc == 0 {
		return
	}
	if B(ldgFBreakOnErc) == 0xFF {
		// No debugger: the trap loop can never be left; behave like the default handler.
		ceBreakpoint()
		ErrorExit(erc)
		return
	}
	saved := P(ldgPErcHandler)
	SetP(ldgPErcHandler, P(ldgPDefErcHandler))
	switch f := FarFn(saved).(type) {
	case func(uint16):
		f(erc)
	case func(uint16) uint16:
		f(erc)
	default:
		panic("CheckErc: bad handler type")
	}
	SetP(ldgPErcHandler, saved)
}

// SetErcHandler(pProc)
func ceSetErcHandler(pProc Ptr) { SetP(ldgPErcHandler, pProc) }

// ---------------- Thunk3616.asm ----------------

// CallProc(a, b, pProc) WORD: jumps to pProc as if called as pProc(a, b).
func thkCallProc(a, b uint16, pProc Ptr) uint16 {
	SetP(ldgPCallTarget, pProc)
	switch f := FarFn(pProc).(type) {
	case func(uint16, uint16) uint16:
		return f(a, b)
	case func(uint16, uint16):
		f(a, b)
		return 0 // AX would be whatever the procedure left in it
	default:
		panic("CallProc: bad procedure type")
	}
}

// ---------------- Mod54E4.plm (KbdLed) ----------------
// The PC keyboard LEDs are not driven; the CTOS LED bitmap (KbdReadLEDs / KbdSetLED) is
// emulated here so that e.g. the overtype LED (7) set by the forms runtime is remembered.

var ledBits byte

func uKbdReadLEDs(pbLEDsRet Ptr) uint16 { SetFB(pbLEDsRet, ledBits); return 0 }
func uKbdSetLED(iLED, fOn uint16) uint16 {
	if iLED < 8 {
		if fOn&1 != 0 {
			ledBits |= 1 << iLED
		} else {
			ledBits &^= 1 << iLED
		}
	}
	return 0
}

func ledFLedOn(iLed byte) byte {
	defer Release(Mark())
	pb := Push(1) // bLeds
	ceCheckErc(uKbdReadLEDs(DG(pb)))
	return Bool((B(pb)>>iLed)&1 != 0)
}

func ledSetLed(iLed, fOn byte) {
	ceCheckErc(uKbdSetLED(uint16(iLed), uint16(fOn)))
}

func init() {
	CheckErc = ceCheckErc
	// the default error handler stored in DGROUP 0AF6h/0AFAh
	RegisterFar(uFrameErrorExit, uOffErrorExit, func(erc uint16) { ErrorExit(erc) })
	// register the helper entry points under their original far addresses (frame 055Fh)
	RegisterFar(0x055F, 0x000C, hlpIsNil)
	RegisterFar(0x055F, 0x0027, hlpNotNil)
	RegisterFar(0x055F, 0x015A, hlpFracDiv)
	RegisterFar(0x055F, 0x0177, hlpFracMul)
}
