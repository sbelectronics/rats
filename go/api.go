package main

// api.go - the interface between the three parts of the port: platform
// (platform_*.go), game (game_*.go) and UI/libraries (lib_*.go).  Every function that
// one part calls in another part is a function VARIABLE declared here and assigned
// in an init() in the implementing part's files.  Calling an unassigned one panics
// with a nil-function error.
//
// Names follow the reconstructed PL/M sources in ../src (see ../README.md).
// Unless noted, WORD = uint16, BYTE = uint8, POINTER = Ptr, near pointers (.x) = uint16
// DGROUP offsets.  "erc" = CTOS error code (0 = ok).

// ---------------- CTOS services (platform) ----------------
var (
	// Keyboard.  Encoded mode: ReadKbd blocks for a character (CTOS character codes,
	// e.g. 1Bh GO, 07h CANCEL, 0Ah RETURN/NEXT, 08h BACKSPACE, 09h TAB, arrows 01h/0Bh/0Eh/12h,
	// see CTOS Kbd_All.asm rgInfoIkey).
	ReadKbd func(pCh Ptr) uint16
	// Unencoded mode (after SetKbdUnencodedMode(TRUE)): ReadKbdDirect(1, pCh) returns erc<>0 when
	// no key is queued, else the raw key code (bit 7 = key released), mode 0 waits.
	ReadKbdDirect       func(mode uint16, pCh Ptr) uint16
	SetKbdUnencodedMode func(f uint16) uint16
	Beep                func() uint16              // CTOS Beep (short tone)
	Delay               func(tenths uint16) uint16 // wait n * 100 ms
	Exit                func()                     // terminate the program normally
	ErrorExit           func(erc uint16)           // terminate with error
	SetScreenVidAttr    func(iAttr, fOn uint16) uint16
	GetDateTime         func(pDt Ptr) uint16 // CTOS date/time DWORD at pDt
	// Real-time clock: number of 100 ms RT clock events since the last call (Sleep uses it).
	RTEvents func() uint16

	// Memory.  AllocMemorySL/LL(cb, ppRet) return a new segment in *ppRet (offset 0).
	AllocMemorySL   func(cb uint16, ppRet Ptr) uint16
	DeallocMemorySL func(p Ptr, cb uint16) uint16
	AllocMemoryLL   func(cb uint32, ppRet Ptr) uint16
	DeallocMemoryLL func(p Ptr) uint16

	// Files ([sys]<games>Rats-x-y.scores etc. are mapped to a local directory).
	// Same argument order as Ctoslib.edf.
	OpenFile         func(pFh Ptr, pbName Ptr, cbName uint16, pbPw Ptr, cbPw uint16, mode uint16) uint16
	CreateFile       func(pbName Ptr, cbName uint16, pbPw Ptr, cbPw uint16, lfa uint32) uint16
	ReadFile         func(fh uint16, pBuf Ptr, cb uint16, lfa uint32, pcbRet Ptr) uint16 // CTOS Read
	WriteFile        func(fh uint16, pBuf Ptr, cb uint16, lfa uint32, pcbRet Ptr) uint16 // CTOS Write
	CloseFile        func(fh uint16) uint16
	ChangeFileLength func(fh uint16, lfa uint32) uint16
	GetFileStatus    func(fh uint16, code uint16, pRet Ptr, cbRet uint16) uint16

	// Misc CTOS
	GetpAscb func(ppRet Ptr) uint16 // user name for the score table lives in the ASCB (+40h len, +41h text)
	GetUcb   func(pUcb Ptr, cb uint16) uint16

	// Hardware beeper (Mod35AC.HwBeep) - a longer tone than Beep.
	HwBeep func()
)

// ---------------- video (platform; VidThk entry points; frame-relative col/row) ----------------
// Semantics: see ../src/VidDrvA.asm / VidDrvB.asm headers. attr = character attribute bits.
var (
	VidReadChars func(pb Ptr, col, row, cnt uint16)            // read cnt chars into pb
	VidReadCells func(pb Ptr, col, row, cnt uint16)            // read cnt (char,attr) words
	VidGetCell   func(col, row uint16) uint16                  // AL=char, AH=attr
	VidFillAttr  func(col, row, cnt uint16, attr byte)         // set attr of cnt cells
	VidPutCell   func(col, row, w uint16)                      // write one (char | attr<<8)
	VidFillCells func(col, row, cnt, w uint16)                 // fill cnt cells with w
	VidPutChars  func(col, row, cnt uint16, pb Ptr, attr byte) // chars from pb, all with attr
	VidPutCells  func(col, row, cnt uint16, pb Ptr)            // (char,attr) words from pb
	// VidInit/ClearRect as in ../src/VidInit.plm; platform also sets DGROUP cScrCols (2F22h),
	// cScrLines (2F24h) and pVidCfg (2F1Eh -> a byte = hardware type 0).
	VidInit   func(pFrames, cFrames uint16, fWide, fReverse byte)
	ClearRect func(col, row, cCols, cRows uint16)
	LoadFont  func(psb uint16) // font is built in (Rats.font); keep the screen-off/on effect
)

// ---------------- libraries used by the game (lib_*.go) ----------------
var (
	DoForm         func(pRec, pForm, frame, col, row, pInit uint16) byte // TRUE if GO
	GetHighScore   func(pFile uint16) uint16
	EnterHighScore func(pFile, score, row uint16)
	CopySb         func(dst, src Ptr)
	AppendSb       func(dst, src Ptr)
	AppendCh       func(dst Ptr, ch byte)
	SbFromNumber   func(psb Ptr, q uint32, cch uint16, chFill byte)
	CheckErc       func(erc uint16) // erc<>0 -> ErrorExit(erc) (no debugger)
	MemAlloc       func(ppRet Ptr, cb uint16) uint16
	MemFree        func(p Ptr)
	GetNow         func() uint32
)

// ---------------- far-procedure registry ----------------
// Far pointers to procedures stored in DGROUP tables (handler table at 0018h, sprite
// draw/erase tables at 0658h/066Ch, form field Fmt/Parse procs in the forms at 0680h
// and 0A92h, video driver table) hold the ORIGINAL code addresses (frame:offset), because
// dgroup.bin is the original image.  Each procedure is registered under that
// address by the file that implements it; callers look it up and type-assert to the signature they expect.
var farProc = map[Ptr]any{}

func RegisterFar(frame, off uint16, fn any) { farProc[MkPtr(frame, off)] = fn }
func FarFn(p Ptr) any {
	f, ok := farProc[p]
	if !ok {
		panic("call through unregistered far pointer")
	}
	return f
}

// ---------------- further services ----------------
var (
	// Mul32(a, b) = a * b as a DWORD (CTOS Math32 runtime, frame 7A3h); used by Random.
	// Implemented in lib_*.go.
	Mul32 func(a, b uint16) uint32
	// CTOS PosFrameCursor(iFrame, iCol, iLine): place the visible cursor in a frame (frame-relative
	// coordinates); used by the forms runtime while editing.  Implemented in platform_*.go.
	PosFrameCursor func(iFrame, iCol, iLine uint16) uint16
)
