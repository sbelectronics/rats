package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPlatformFontOrientation(t *testing.T) {
	if len(pRatsFont) != 256*32 || len(pSysFont) != 256*32 {
		t.Fatalf("font sizes %d %d", len(pRatsFont), len(pSysFont))
	}
	saved := pFont
	pFont = pRatsFont
	defer func() { pFont = saved }()
	// row 0 is never displayed: it is blank in every glyph of both fonts
	for _, f := range [][]byte{pRatsFont, pSysFont} {
		for g := 0; g < 256; g++ {
			if f[g*32] != 0 || f[g*32+1] != 0 {
				t.Fatalf("glyph %02X has pixels in font row 0", g)
			}
		}
	}
	for y := 0; y < pCellH; y++ {
		if pGlyphRow(0, y) != 0 {
			t.Fatalf("glyph 0 must be blank (ClearRect fills with 0)")
		}
	}
	// 'R': the stem is on the left, so the leftmost used column has more pixels
	// than the rightmost one in the lower half (leg).
	left, right := 0, 0
	for y := 0; y < pCellH; y++ {
		if pGlyphPixel('R', 1, y) {
			left++
		}
		if pGlyphPixel('R', 8, y) {
			right++
		}
	}
	if !pGlyphPixel('R', 3, 3-pFirstRow) || pGlyphPixel('R', 8, 3-pFirstRow) {
		t.Fatalf("'R' top bar not where expected: left=%d right=%d", left, right)
	}
}

func TestPlatformVidDriver(t *testing.T) {
	pPlatformInit()
	pVidInit(0x102, 1, 0, TRUE)
	if W(dgCScrCols) != 80 || W(dgCScrLines) != 34 || P(dgPVidCfg) != MkPtr(SegSys, 0) {
		t.Fatalf("DGROUP screen vars %d %d %08X", W(dgCScrCols), W(dgCScrLines), P(dgPVidCfg))
	}
	if FB(P(dgPVidCfg)) != 0 {
		t.Fatalf("pVidCfg must point at hardware type 0")
	}
	// odd start, odd count
	pVidA5(3, 5, 5, 0x0941)
	for c := uint16(2); c <= 8; c++ {
		w := pVidA2(c, 5)
		want := uint16(0)
		if c >= 3 && c <= 7 {
			want = 0x0941
		}
		if w != want {
			t.Fatalf("col %d: %04X want %04X", c, w, want)
		}
	}
	// VidPutCells / VidReadCells round trip at odd and even starts
	buf := Push(16)
	defer Release(buf + 16)
	for start := uint16(10); start < 12; start++ {
		for i := uint16(0); i < 5; i++ {
			SetW(buf+2*i, uint16(i+1)<<8|uint16('a'+i))
		}
		pVidA7(start, 7, 5, DG(buf))
		SETB(0, DG(buf), 10)
		pVidA1(DG(buf), start, 7, 5)
		for i := uint16(0); i < 5; i++ {
			if W(buf+2*i) != uint16(i+1)<<8|uint16('a'+i) {
				t.Fatalf("start %d cell %d = %04X", start, i, W(buf+2*i))
			}
		}
		if pVidA2(start-1, 7)>>8 != 0 || pVidA2(start+5, 7)>>8 != 0 {
			t.Fatalf("neighbour attribute changed")
		}
		pClearRect(0, 7, 80, 1)
	}
	pVidA3(0, 9, 3, 0x14) // attribute > 0Fh: corrupts like the original
	if pVidA2(1, 9)>>8 != 0x05 {
		t.Fatalf("attr quirk: %04X", pVidA2(1, 9))
	}
}

func TestPlatformMemory(t *testing.T) {
	p := Push(4)
	defer Release(p + 4)
	if pAllocMemorySL(0xFFFF, DG(p)) != 0 || P(p).Seg() != SegMap {
		t.Fatalf("first 64K -> SegMap")
	}
	if pAllocMemorySL(0x100, DG(p)) != 0 {
		t.Fatal("alloc")
	}
	s := P(p).Seg()
	SetFB(MkPtr(s, 0x20), 0x5A)
	if FB(MkPtr(s+2, 0)) != 0x5A { // paragraph aliasing (Mod3396 splits blocks)
		t.Fatal("paragraph alias")
	}
	if pDeallocMemorySL(MkPtr(s+8, 0), 0x7F) != 0 { // free the top half
		t.Fatal("partial dealloc")
	}
	if pDeallocMemorySL(MkPtr(s, 0), 0x7F) != 0 {
		t.Fatal("dealloc")
	}
	if pDeallocMemorySL(MkPtr(s, 0), 0x7F) != pErcBadDeallocate {
		t.Fatal("double dealloc")
	}
}

func TestPlatformFiles(t *testing.T) {
	pDataDir = t.TempDir()
	defer func() { pDataDir = "" }()
	path, ok := pMapFileSpec("[!sys]<games>Rats-3-50.scores")
	if !ok || path != filepath.Join(pDataDir, "Rats-3-50.scores") {
		t.Fatalf("map %q", path)
	}
	name := Push(32)
	defer Release(name + 32)
	spec := "[!sys]<games>Rats-3-50.scores"
	PutBytes(DG(name), []byte(spec))
	fh := Push(2)
	cb := Push(2)
	buf := Push(512)
	defer Release(fh)
	n := uint16(len(spec))
	if e := pOpenFile(DG(fh), DG(name), n, NilPtr, 0, pModeRead); e != pErcNoSuchFile {
		t.Fatalf("open missing: %d", e)
	}
	if e := pCreateFile(DG(name), n, NilPtr, 0, 0); e != 0 {
		t.Fatalf("create %d", e)
	}
	if e := pCreateFile(DG(name), n, NilPtr, 0, 0); e != pErcFileAlreadyExists {
		t.Fatalf("create again %d", e)
	}
	if e := pOpenFile(DG(fh), DG(name), n, NilPtr, 0, pModeModify); e != 0 {
		t.Fatalf("open %d", e)
	}
	h := W(fh)
	if e := pWriteFile(h, DG(buf), 512, 0, DG(cb)); e != pErcEOM {
		t.Fatalf("write past end %d", e)
	}
	if e := pChangeFileLength(h, 512); e != 0 {
		t.Fatal(e)
	}
	SetB(buf, 0x77)
	if e := pWriteFile(h, DG(buf), 512, 0, DG(cb)); e != 0 || W(cb) != 512 {
		t.Fatalf("write %d", e)
	}
	SetB(buf, 0)
	if e := pReadFile(h, DG(buf), 512, 0, DG(cb)); e != 0 || B(buf) != 0x77 {
		t.Fatalf("read %d", e)
	}
	if e := pReadFile(h, DG(buf), 512, 512, DG(cb)); e != pErcEOM {
		t.Fatalf("read past end %d", e)
	}
	if e := pGetFileStatus(h, 0, DG(cb-2), 4); e != 0 || DW(cb-2) != 512 {
		t.Fatalf("status %d", DW(cb-2))
	}
	if pCloseFile(h) != 0 || pCloseFile(h) != pErcBadFh {
		t.Fatal("close")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestPlatformDateTime(t *testing.T) {
	// 1 March 1952 00:00 = 0; 1 March 1952 13:00:05 = half day 1, 3605 s.
	if d := pCtosDateTime(time.Date(1952, 3, 1, 0, 0, 0, 0, time.Local)); d != 0 {
		t.Fatalf("epoch %08X", d)
	}
	if d := pCtosDateTime(time.Date(1952, 3, 1, 13, 0, 5, 0, time.Local)); d != 1<<16|3605 {
		t.Fatalf("%08X", d)
	}
	// 1 March 1956 = 4 years incl. one leap day = 1461 days
	if d := pCtosDateTime(time.Date(1956, 3, 1, 0, 0, 0, 0, time.Local)); d != 1461*2<<16 {
		t.Fatalf("%08X", d)
	}
}

func TestPlatformKbd(t *testing.T) {
	pSetKbdUnencodedMode(0xFF)
	now := time.Now()
	pKbdPress(1, pKeyDef{raw: 0x7C}, now)
	pKbdRelease(1, pKeyDef{raw: 0x7C})
	p := Push(2)
	defer Release(p + 2)
	if pReadKbdDirect(1, DG(p)) != 0 || B(p) != 0x7C {
		t.Fatal("press")
	}
	if pReadKbdDirect(1, DG(p)) != 0 || B(p) != 0xFC {
		t.Fatal("release")
	}
	if pReadKbdDirect(1, DG(p)) != pErcNoCharactersInKeyboard {
		t.Fatal("empty")
	}
	pSetKbdUnencodedMode(0)
	pKbdPress(2, pKeyDef{raw: 0x48}, now)
	pKbdPress(3, pKeyDef{raw: 0x0F}, now) // shifted right arrow -> D2h
	pKbdRelease(3, pKeyDef{raw: 0x0F})
	pKbdRelease(2, pKeyDef{raw: 0x48})
	pKbdPress(4, pKeyDef{raw: 0x61}, now)
	pKbdRelease(4, pKeyDef{raw: 0x61})
	if pReadKbd(DG(p)); B(p) != 0xD2 {
		t.Fatalf("shift right %02X", B(p))
	}
	if pReadKbd(DG(p)); B(p) != 'a' {
		t.Fatalf("a %02X", B(p))
	}
}
