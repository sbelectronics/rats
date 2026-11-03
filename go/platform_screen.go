package main

// platform_screen.go - the CTOS video system as Rats sees it.
//
// * VCB (GetPStructure(2), DGROUP pVidCfg 2F1Eh -> F800:0000), laid out as CTOS
//   VcbType (CTOS ctostypes.edf): +0 version (= hardware type for
//   VidSel, 0), +2 fReverse, +3 fHalfBright, +4 pMap, +8 sMap, +0Ah cFramesMax,
//   +0Bh cColsMax, +0Ch cLinesMax, +0Dh sLine, +0Eh ibToAttrs, +0Fh ibToChars,
//   +10h bSpace, +1Bh iFrameCursor, +20h rgFrame(8) of 20-byte FrameType
//   (+0 iLineStart, +1 iColStart, +2 cLines, +3 cCols, +0Ah iLineCursor,
//   +0Bh iColCursor).
// * Video map: one block per screen line (GetPStructure(7) line table -> far
//   pointers), characters at +ibToChars+col, 4-bit attributes packed two per byte
//   at +ibToAttrs+col/2 (even column = low nibble), exactly what driver A
//   (../src/VidDrvA.asm) expects.  The eight Vid* entry points below are literal
//   translations of VidA0..VidA7, byte operations included (so attribute values
//   > 0Fh corrupt the neighbour nibble just like the original).  The driver uses
//   SCREEN rows (the line table), like the original; frames only matter for the
//   forms runtime and the cursor.
// * Attribute bits (CTOS character "looks", ctostypes.edf looksIntensify=1,
//   looksUnderline=2, looksReverse=4, looksBlink=8; VidNGen.Edf vidHalf=100h,
//   vidUnder=200h, vidReverse=400h, vidBlink=800h): 1 = half-bright,
//   2 = underline, 4 = reverse video, 8 = blink.
// * SetScreenVidAttr (ctostypes.edf screenReverse=0, screenRefresh=1,
//   screenHalfBright=2): 0 = whole-screen reverse video (black on white),
//   1 = refresh on/off (off = blank screen), 2 = global half-bright.

import (
	"sync"
	"time"
)

const (
	pVcbOff      uint16 = 0x0000
	pAscbOff     uint16 = 0x0200
	pLineTabOff  uint16 = 0x0400
	pVidMapOff   uint16 = 0x1000
	pLineStride  uint16 = 0x0100
	pScratchLine uint16 = 0x3200
	pMaxLines           = 34
	pErcBadFrame        = 500

	dgBVidType   = 0x2F10
	dgSMap       = 0x2F14
	dgWVid16     = 0x2F16
	dgWVid18     = 0x2F18
	dgPVidBlocks = 0x2F1A
	dgPVidCfg    = 0x2F1E
	dgCScrCols   = 0x2F22
	dgCScrLines  = 0x2F24
	dgRgpVid     = 0x2F26
	dgRgwLines   = 0x10E6
	dgRgpDrv     = 0x10EA
)

var (
	pVidMu      sync.Mutex
	pScrReverse bool
	pScrRefresh = true
	pScrHalf    bool
	pVidSerial  uint64 // bumped on every change, lets Draw skip unchanged frames
)

func init() {
	VidReadChars = pVidA0
	VidReadCells = pVidA1
	VidGetCell = pVidA2
	VidFillAttr = pVidA3
	VidPutCell = pVidA4
	VidFillCells = pVidA5
	VidPutChars = pVidA6
	VidPutCells = pVidA7
	VidInit = pVidInit
	ClearRect = pClearRect
	LoadFont = pLoadFont
	SetScreenVidAttr = pSetScreenVidAttr
	PosFrameCursor = pPosFrameCursor
}

// ---- system segment byte access (no locking; callers hold pVidMu) ----
func pSm(o uint16) byte          { return pSysMem[o] }
func pSetSm(o uint16, v byte)    { pSysMem[o] = v }
func pSmW(o uint16) uint16       { return uint16(pSysMem[o]) | uint16(pSysMem[o+1])<<8 }
func pSetSmW(o uint16, v uint16) { pSysMem[o] = byte(v); pSysMem[o+1] = byte(v >> 8) }
func pRor4(b byte) byte          { return b>>4 | b<<4 }

// pResetVideo is CTOS ResetVideo(nCols, nLines, fAttr, bSpace, @sMap): sets the VCB
// geometry, clears all frames and builds the line table.
func pResetVideo(nCols, nLines uint16, bSpace byte) uint16 {
	if nLines > pMaxLines || (nCols != 80 && nCols != 132) {
		return pErcBadFrame
	}
	sLine := byte(nCols + nCols/2) // chars then packed attributes: 120 or 198 bytes
	v := pVcbOff
	pSetSm(v+0, 0) // version / hardware type 0 (VidSel -> driver A, 34 lines)
	pSetSm(v+1, 0xFF)
	pSetSmW(v+4, pVidMapOff)
	pSetSmW(v+6, SegSys)
	pSetSmW(v+8, nLines*pLineStride)
	pSetSm(v+0x0A, 8)
	pSetSm(v+0x0B, byte(nCols))
	pSetSm(v+0x0C, byte(nLines))
	pSetSm(v+0x0D, sLine)
	pSetSm(v+0x0E, byte(nCols)) // ibToAttrs
	pSetSm(v+0x0F, 0)           // ibToChars
	pSetSm(v+0x10, bSpace)
	pSetSm(v+0x1B, 0)
	for i := uint16(0); i < 160; i++ {
		pSetSm(v+0x20+i, 0)
	}
	for f := uint16(0); f < 8; f++ {
		pSetSm(v+0x20+f*20+0x0A, 0xFF)
		pSetSm(v+0x20+f*20+0x0B, 0xFF)
	}
	for r := uint16(0); r < 256; r++ {
		off := pScratchLine
		if r < nLines {
			off = pVidMapOff + r*pLineStride
		}
		pSetSmW(pLineTabOff+r*4, off)
		pSetSmW(pLineTabOff+r*4+2, SegSys)
	}
	return 0
}

// pInitVidFrame is CTOS InitVidFrame(iFrame, iColStart, iLineStart, nCols, nLines, ...).
func pInitVidFrame(iFrame, col, row, nCols, nLines uint16) uint16 {
	if iFrame >= uint16(pSm(pVcbOff+0x0A)) || col+nCols > uint16(pSm(pVcbOff+0x0B)) ||
		row+nLines > uint16(pSm(pVcbOff+0x0C)) {
		return pErcBadFrame
	}
	f := pVcbOff + 0x20 + iFrame*20
	for i := uint16(0); i < 20; i++ {
		pSetSm(f+i, 0)
	}
	pSetSm(f+0, byte(row))
	pSetSm(f+1, byte(col))
	pSetSm(f+2, byte(nLines))
	pSetSm(f+3, byte(nCols))
	pSetSm(f+0x0A, 0xFF)
	pSetSm(f+0x0B, 0xFF)
	return 0
}

// pInitVidMap is CTOS InitVidMap: fill the whole map with bSpace characters, attribute 0.
func pInitVidMap() {
	n := uint16(pSm(pVcbOff + 0x0C))
	cols := uint16(pSm(pVcbOff + 0x0B))
	bSpace := pSm(pVcbOff + 0x10)
	for r := uint16(0); r < n; r++ {
		b := pVidMapOff + r*pLineStride
		for c := uint16(0); c < cols; c++ {
			pSetSm(b+c, bSpace)
		}
		for c := uint16(0); c < cols/2; c++ {
			pSetSm(b+cols+c, 0)
		}
	}
}

// pPlatformVideoBoot: the state the executive leaves behind (80 x 34, frame 0 = screen).
func pPlatformVideoBoot() {
	pVidMu.Lock()
	defer pVidMu.Unlock()
	pResetVideo(80, 34, 0)
	pInitVidFrame(0, 0, 0, 80, 34)
	pInitVidMap()
	SetW(dgCScrCols, 80)
	SetW(dgCScrLines, 34)
	SetP(dgPVidCfg, MkPtr(SegSys, pVcbOff))
	pVidSerial++
}

// pGetVidType is VidSel.GetVidType: driver index from the VCB's first byte, cached
// in bVidType (2F10h).
func pGetVidType() byte {
	if B(dgBVidType) == 0xFF {
		b := FB(MkPtr(SegSys, pVcbOff))
		if b >= 5 {
			pCheckErcP(500)
		}
		t := []byte{0, 0, 0xFF, 0xFF, 1}[b]
		SetB(dgBVidType, t)
		if t == 0xFF {
			pCheckErcP(500)
		}
	}
	return B(dgBVidType)
}

func pCheckErcP(erc uint16) {
	if erc != 0 {
		if CheckErc != nil {
			CheckErc(erc)
		} else {
			ErrorExit(erc)
		}
	}
}

// pVidInit translates VidInit.plm.
func pVidInit(pFrames, cFrames uint16, fWide, fReverse byte) {
	bType := pGetVidType()
	SetP(dgPVidCfg, MkPtr(SegSys, pVcbOff)) // GetPStructure(2)
	pVidMu.Lock()
	if pFrames == 0xFFFF {
		SetW(dgSMap, pSmW(pVcbOff+8))
		SetW(dgCScrCols, uint16(pSm(pVcbOff+0x0B)))
		SetW(dgCScrLines, uint16(pSm(pVcbOff+0x0C)))
	} else {
		if Test(fWide) {
			SetW(dgCScrCols, 132)
		} else {
			SetW(dgCScrCols, 80)
		}
		cScrCols := W(dgCScrCols)
		SetW(dgCScrLines, W(dgRgwLines+2*uint16(bType)))
		cScrLines := W(dgCScrLines)
		erc := pResetVideo(cScrCols, cScrLines, 0)
		if erc == 0 {
			SetW(dgSMap, pSmW(pVcbOff+8))
		}
		pVidMu.Unlock()
		pCheckErcP(erc)
		pVidMu.Lock()
		for iFrame := uint16(0); iFrame < cFrames; iFrame++ {
			fr := pFrames + iFrame*4
			rel := func(b byte, full uint16, zeroFull bool) uint16 {
				if zeroFull && b == 0 {
					return full
				}
				if b < 0x80 {
					return uint16(b)
				}
				return uint16(b) + full + 0xFF00
			}
			col := rel(B(fr), cScrCols, false)
			row := rel(B(fr+1), cScrLines, false)
			cCols := rel(B(fr+2), cScrCols, true)
			cLines := rel(B(fr+3), cScrLines, true)
			erc := pInitVidFrame(iFrame, col, row, cCols, cLines)
			pVidMu.Unlock()
			pCheckErcP(erc)
			pVidMu.Lock()
		}
		pInitVidMap()
		pVidSerial++
		if Test(fReverse) {
			pVidMu.Unlock()
			pCheckErcP(pSetScreenVidAttr(1, 0xFF))
			pVidMu.Lock()
		}
	}
	SetW(0x2F12, uint16(pSm(pVcbOff+0x0D)))
	SetW(dgWVid16, uint16(pSm(pVcbOff+0x0E)))
	SetW(dgWVid18, uint16(pSm(pVcbOff+0x0F)))
	pVidMu.Unlock()
	SetP(dgPVidBlocks, MkPtr(SegSys, pLineTabOff)) // GetPStructure(7)
	for i := uint16(0); i < 8; i++ {
		SetP(dgRgpVid+4*i, P(dgRgpDrv+4*(2*i+uint16(bType))))
	}
}

// pClearRect translates VidInit.ClearRect.
func pClearRect(col, row, cCols, cLines uint16) {
	for cLines != 0 {
		pVidA5(col, row, cCols, 0)
		row++
		cLines--
	}
}

// pLoadFont(psb): load the font named by the sb string at DGROUP psb into the font RAM,
// with LoadFontFile's screen off / on around it.  Fonts are built in (platform_font.go);
// an unknown name leaves the current font.
func pLoadFont(psb uint16) {
	name := string(Bytes(DG(psb+1), int(B(psb))))
	pCheckErcP(pSetScreenVidAttr(1, 0))
	if f := pFontByName(name); f != nil {
		pVidMu.Lock()
		pFont = f
		pVidMu.Unlock()
	}
	time.Sleep(50 * time.Millisecond)
	pCheckErcP(pSetScreenVidAttr(1, 0xFF))
}

func pSetScreenVidAttr(iAttr, fOn uint16) uint16 {
	pVidMu.Lock()
	defer pVidMu.Unlock()
	on := fOn&1 != 0
	switch iAttr {
	case 0:
		pScrReverse = on
		pSetSm(pVcbOff+2, byte(fOn))
	case 1:
		pScrRefresh = on
	case 2:
		pScrHalf = on
		pSetSm(pVcbOff+3, byte(fOn))
	default:
		return 501
	}
	pVidSerial++
	return 0
}

func pPosFrameCursor(iFrame, iCol, iLine uint16) uint16 {
	pVidMu.Lock()
	defer pVidMu.Unlock()
	if iFrame >= uint16(pSm(pVcbOff+0x0A)) {
		return pErcBadFrame
	}
	f := pVcbOff + 0x20 + iFrame*20
	if iCol != 0xFF && iCol >= uint16(pSm(f+3)) || iLine != 0xFF && iLine >= uint16(pSm(f+2)) {
		return pErcBadFrame
	}
	pSetSm(pVcbOff+0x1B, byte(iFrame))
	pSetSm(f+0x0A, byte(iLine))
	pSetSm(f+0x0B, byte(iCol))
	pVidSerial++
	return 0
}

// ---- driver A (VidDrvA.asm), literal translation ----

// srcPos/dstPos: SI/DI -> character, BX -> attribute byte (offsets in SegSys).
func pVidPos(col, row uint16) (ch, at uint16) {
	line := pSmW(pLineTabOff + row<<2)
	ch = line + W(dgWVid18) + col
	at = line + W(dgWVid16) + col>>1
	return
}

func pVidA0(pb Ptr, col, row, cnt uint16) {
	pVidMu.Lock()
	defer pVidMu.Unlock()
	si, _ := pVidPos(col, row)
	for i := uint16(0); i < cnt; i++ {
		SetFB(pb.Add(i), pSm(si+i))
	}
}

func pVidA1(pb Ptr, col, row, cnt uint16) {
	pVidMu.Lock()
	defer pVidMu.Unlock()
	if cnt == 0 {
		return
	}
	si, bx := pVidPos(col, row)
	di := pb
	cx := cnt
	var dh byte
	stos := func(al, ah byte) { SetFB(di, al); SetFB(di.Add(1), ah); di = di.Add(2) }
	odd := col&1 != 0
	if odd {
		dh = pSm(bx)
		bx++
	}
	for {
		if !odd {
			dh = pSm(bx)
			bx++
			stos(pSm(si), dh&0x0F)
			si++
			cx--
			if cx == 0 {
				return
			}
		}
		odd = false
		stos(pSm(si), dh>>4)
		si++
		cx--
		if cx == 0 {
			return
		}
	}
}

func pVidA2(col, row uint16) uint16 {
	pVidMu.Lock()
	defer pVidMu.Unlock()
	si, bx := pVidPos(col, row)
	al, ah := pSm(si), pSm(bx)
	if col&1 != 0 {
		ah >>= 4
	} else {
		ah &= 0x0F
	}
	return uint16(ah)<<8 | uint16(al)
}

// pFillAttrs is the shared attribute-fill tail of VidA3/A5/A6.
func pFillAttrs(col, bx, cx uint16, ah byte) {
	di := bx
	if col&1 != 0 {
		pSetSm(di, pRor4(pSm(di)<<4|ah))
		di++
		cx--
	}
	carry := cx&1 != 0
	cx >>= 1
	if carry {
		pSetSm(di+cx, pSm(di+cx)&0xF0|ah)
	}
	al := ah<<4 | ah
	for ; cx != 0; cx-- {
		pSetSm(di, al)
		di++
	}
}

func pVidA3(col, row, cnt uint16, attr byte) {
	pVidMu.Lock()
	defer pVidMu.Unlock()
	if cnt == 0 {
		return
	}
	_, bx := pVidPos(col, row)
	pFillAttrs(col, bx, cnt, attr)
	pVidSerial++
}

func pVidA4(col, row, w uint16) {
	pVidMu.Lock()
	defer pVidMu.Unlock()
	di, bx := pVidPos(col, row)
	al, ah := byte(w), byte(w>>8)
	pSetSm(di, al)
	al = pSm(bx)
	if col&1 == 0 {
		al = al&0xF0 | ah
	} else {
		al = pRor4(al<<4 | ah)
	}
	pSetSm(bx, al)
	pVidSerial++
}

func pVidA5(col, row, cnt, w uint16) {
	pVidMu.Lock()
	defer pVidMu.Unlock()
	if cnt == 0 {
		return
	}
	di, bx := pVidPos(col, row)
	for i := uint16(0); i < cnt; i++ {
		pSetSm(di+i, byte(w))
	}
	pFillAttrs(col, bx, cnt, byte(w>>8))
	pVidSerial++
}

func pVidA6(col, row, cnt uint16, pb Ptr, attr byte) {
	pVidMu.Lock()
	defer pVidMu.Unlock()
	if cnt == 0 {
		return
	}
	di, bx := pVidPos(col, row)
	for i := uint16(0); i < cnt; i++ {
		pSetSm(di+i, FB(pb.Add(i)))
	}
	pFillAttrs(col, bx, cnt, attr)
	pVidSerial++
}

func pVidA7(col, row, cnt uint16, pb Ptr) {
	pVidMu.Lock()
	defer pVidMu.Unlock()
	if cnt == 0 {
		return
	}
	di, bx := pVidPos(col, row)
	si := pb
	cx := cnt
	var dh byte
	lodsw := func() (byte, byte) { al, ah := FB(si), FB(si.Add(1)); si = si.Add(2); return al, ah }
	odd := col&1 != 0
	if odd {
		dh = pSm(bx) & 0x0F
	}
	for {
		if !odd {
			al, ah := lodsw()
			pSetSm(di, al)
			di++
			dh = ah
			cx--
			if cx == 0 { // A7Last
				pSetSm(bx, pSm(bx)&0xF0|dh)
				break
			}
		}
		odd = false
		al, ah := lodsw()
		pSetSm(di, al)
		di++
		pSetSm(bx, ah<<4|dh)
		bx++
		cx--
		if cx == 0 {
			break
		}
	}
	pVidSerial++
}

// ---- rendering ----

type pRgb struct{ r, g, b byte }

var (
	pColNormal = pRgb{0xE0, 0xE0, 0xE0}
	pColHalf   = pRgb{0x80, 0x80, 0x80}
	pColBack   = pRgb{0x00, 0x00, 0x00}
	pUlRow     = pCellH - 1 // scan line used for the underline attribute (last displayed row)
)

func pSetPhosphor(name string) {
	switch name {
	case "green":
		pColNormal, pColHalf = pRgb{0x40, 0xFF, 0x60}, pRgb{0x20, 0x90, 0x30}
	case "amber":
		pColNormal, pColHalf = pRgb{0xFF, 0xB0, 0x00}, pRgb{0x90, 0x60, 0x00}
	}
}

// pScreenState is a consistent copy of everything needed to draw one frame.
type pScreenState struct {
	cols, lines   int
	ch, at        [pMaxLines * 132]byte
	reverse, half bool
	refresh       bool
	curCol        int // -1 = no cursor
	curRow        int
	serial        uint64
}

func pSnapScreen(st *pScreenState) {
	pVidMu.Lock()
	defer pVidMu.Unlock()
	st.cols = int(pSm(pVcbOff + 0x0B))
	st.lines = int(pSm(pVcbOff + 0x0C))
	if st.lines > pMaxLines {
		st.lines = pMaxLines
	}
	ibA, ibC := uint16(pSm(pVcbOff+0x0E)), uint16(pSm(pVcbOff+0x0F))
	for r := 0; r < st.lines; r++ {
		line := pSmW(pLineTabOff + uint16(r)*4)
		for c := 0; c < st.cols; c++ {
			i := r*st.cols + c
			st.ch[i] = pSm(line + ibC + uint16(c))
			a := pSm(line + ibA + uint16(c>>1))
			if c&1 != 0 {
				a >>= 4
			}
			st.at[i] = a & 0x0F
		}
	}
	st.reverse, st.half, st.refresh = pScrReverse, pScrHalf, pScrRefresh
	st.curCol = -1
	fi := uint16(pSm(pVcbOff + 0x1B))
	if fi < 8 {
		f := pVcbOff + 0x20 + fi*20
		l, c := pSm(f+0x0A), pSm(f+0x0B)
		if l != 0xFF && c != 0xFF {
			st.curRow = int(pSm(f+0)) + int(l)
			st.curCol = int(pSm(f+1)) + int(c)
		}
	}
	st.serial = pVidSerial
}

// pRenderScreen draws st into pix (RGBA, width 80*pCellW, height lines*pCellH).
// blinkOn = visible phase of blinking characters and the cursor.
func pRenderScreen(st *pScreenState, pix []byte, width int, blinkOn bool) {
	clear(pix)
	for i := 3; i < len(pix); i += 4 {
		pix[i] = 0xFF
	}
	if !st.refresh {
		return
	}
	cols := st.cols
	if cols*pCellW > width {
		cols = width / pCellW
	}
	for r := 0; r < st.lines; r++ {
		for c := 0; c < cols; c++ {
			i := r*st.cols + c
			ch, a := st.ch[i], st.at[i]
			fg := pColNormal
			if a&1 != 0 && !st.half {
				fg = pColHalf
			}
			bg := pColBack
			rev := (a&4 != 0) != st.reverse
			if r == st.curRow && c == st.curCol && blinkOn {
				rev = !rev
			}
			if rev {
				fg, bg = bg, fg
			}
			showGlyph := a&8 == 0 || blinkOn
			for y := 0; y < pCellH; y++ {
				bits := uint16(0)
				if showGlyph {
					bits = pGlyphRow(ch, y)
					if a&2 != 0 && y == pUlRow {
						bits = 0x3FF
					}
				}
				o := ((r*pCellH+y)*width + c*pCellW) * 4
				for x := 0; x < pCellW; x++ {
					col := bg
					if bits>>uint(pCellW-1-x)&1 != 0 {
						col = fg
					}
					pix[o], pix[o+1], pix[o+2] = col.r, col.g, col.b
					o += 4
				}
			}
		}
	}
}
