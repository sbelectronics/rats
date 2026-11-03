package main

// lib_test.go - tests of the library translation that do not need the platform:
// string formatting/parsing, the CTOS date/time runtime against Go's calendar, and the
// score file / forms runtime with a fake screen, keyboard and file system.

import (
	"bytes"
	"testing"
	"time"
)

// ---- helpers ----

// tmpSb puts s as an sb into scratch DGROUP memory above the stack (4000h..FFFFh, unused
// by the program).
var tmpNext uint16 = 0x4000

func tmpSb(s string) Ptr {
	if tmpNext > 0xF000 {
		tmpNext = 0x4000
	}
	o := tmpNext
	tmpNext += 256
	SetB(o, byte(len(s)))
	PutBytes(DG(o+1), []byte(s))
	return DG(o)
}
func sbStr(p Ptr) string { return string(Bytes(p.Add(1), int(FB(p)))) }

func TestSbFromNumber(t *testing.T) {
	defer Release(Mark())
	sb := tmpSb("")
	cases := []struct {
		q      uint32
		cch    uint16
		chFill byte
		want   string
	}{
		{0, 0, ' ', "0"}, {12345, 0, ' ', "12345"}, {7, 3, '0', "007"}, {42, 5, ' ', "   42"},
		{4294967295, 0, ' ', "4294967295"}, {123456, 2, ' ', "123456"}, {9, 1, ' ', "9"},
	}
	for _, c := range cases {
		strSbFromNumber(sb, c.q, c.cch, c.chFill)
		if got := sbStr(sb); got != c.want {
			t.Errorf("SbFromNumber(%d,%d,%q) = %q, want %q", c.q, c.cch, c.chFill, got, c.want)
		}
	}
	ints := []struct {
		n    int16
		cch  uint16
		want string
	}{{-5, 4, "  -5"}, {-32768, 0, "-32768"}, {32767, 0, "32767"}, {0, 0, "0"}, {-1, 1, "-1"}}
	for _, c := range ints {
		strSbFromInt(sb, c.n, c.cch)
		if got := sbStr(sb); got != c.want {
			t.Errorf("SbFromInt(%d,%d) = %q, want %q", c.n, c.cch, got, c.want)
		}
	}
	strSbHexFromW(sb, 0xBEEF)
	if got := sbStr(sb); got != "BEEFh" {
		t.Errorf("SbHexFromW = %q", got)
	}
	strSbFromFixed(sb, 3, 0x8000, 0, 2, ' ')
	if got := sbStr(sb); got != "3.50" {
		t.Errorf("SbFromFixed(3, 1/2, 2 dec) = %q", got)
	}
	strSbFromFixed(sb, 1, 0xFFFF, 0, 1, ' ') // rounds up to the next integer
	if got := sbStr(sb); got != "2.0" {
		t.Errorf("SbFromFixed(1, 0FFFFh, 1 dec) = %q", got)
	}
}

func TestRoman(t *testing.T) {
	defer Release(Mark())
	sb := tmpSb("")
	cases := map[uint16]string{
		0: "", 1: "i", 4: "iv", 9: "ix", 14: "xiv", 49: "xlix", 1987: "mcmlxxxvii",
		3999: "mmmcmxcix", 4000: "mV", 5000: "V", 10000: "X", 40000: "XL",
	}
	for w, want := range cases {
		strSbRomanFromW(sb, w)
		if got := sbStr(sb); got != want {
			t.Errorf("SbRomanFromW(%d) = %q, want %q", w, got, want)
		}
	}
	strSbRomanFromW(sb, 65535) // 6 5 5 3 5 -> ten-thousands digit 6 = "LX"
	if got := sbStr(sb); got != "LXVdxxxv" {
		t.Errorf("SbRomanFromW(65535) = %q", got)
	}
	// a ten-thousands digit 9 (not reachable with a WORD) would read the byte after rgchRoman
	if B(ldgRgchRoman+25) != 0x14 {
		t.Errorf("byte after rgchRoman = %02x", B(ldgRgchRoman+25))
	}
}

func TestParseNumbers(t *testing.T) {
	defer Release(Mark())
	dw := Push(4)
	w := Push(2)
	type c struct {
		s    string
		ok   bool
		want uint32
	}
	for _, x := range []c{{"0", true, 0}, {"123", true, 123}, {"", false, 0}, {"12a", false, 0},
		{"4294967295", true, 4294967295}, {"4294967296", true, 0}, {" 1", false, 0}} {
		ok := Test(strFDwFromSb(dw, tmpSb(x.s)))
		if ok != x.ok || ok && DW(dw) != x.want {
			t.Errorf("FDwFromSb(%q) = %v %d", x.s, ok, DW(dw))
		}
	}
	for _, x := range []c{{"65535", true, 65535}, {"65536", false, 0}, {"7", true, 7}} {
		ok := Test(strFWFromSb(w, tmpSb(x.s)))
		if ok != x.ok || ok && uint32(W(w)) != x.want {
			t.Errorf("FWFromSb(%q) = %v %d", x.s, ok, W(w))
		}
	}
	for _, x := range []struct {
		s    string
		ok   bool
		want int16
	}{{"-12", true, -12}, {"+5", true, 5}, {"32767", true, 32767}, {"32768", false, 0}, {"-", false, 0}} {
		ok := Test(strFIntFromSb(w, tmpSb(x.s)))
		if ok != x.ok || ok && int16(W(w)) != x.want {
			t.Errorf("FIntFromSb(%q) = %v %d", x.s, ok, int16(W(w)))
		}
	}
	sb := tmpSb("   ab c  ")
	strTrimSb(sb, TRUE, TRUE)
	if sbStr(sb) != "ab c" {
		t.Errorf("TrimSb = %q", sbStr(sb))
	}
	sb = tmpSb("    ")
	strTrimSb(sb, TRUE, TRUE)
	if sbStr(sb) != "" {
		t.Errorf("TrimSb(blanks) = %q", sbStr(sb))
	}
	// FormFields parse procs
	pOut := Push(4)
	in := tmpSb("  300 ")
	if erc := ffParseByte(pOut, in.Off()); erc != 31004 {
		t.Errorf("ParseByte(300) = %d", erc)
	}
	in = tmpSb(" 25")
	if erc := ffParseByte(pOut, in.Off()); erc != 0 || B(pOut) != 25 {
		t.Errorf("ParseByte(25) = %d %d", erc, B(pOut))
	}
	in = tmpSb("x")
	if erc := ffParseNumber(pOut, in.Off()); erc != 31003 {
		t.Errorf("ParseNumber(x) = %d", erc)
	}
	in = tmpSb("")
	if erc := ffParseDword(pOut, in.Off()); erc != 0 || DW(pOut) != 0 {
		t.Errorf("ParseDword('') = %d", erc)
	}
	SetB(pOut+2, 2)
	if erc := ffParseBool(pOut, pOut+2); erc != 31002 {
		t.Errorf("ParseBool(2) = %d", erc)
	}
	SetB(pOut+2, 1)
	if erc := ffParseBool(pOut, pOut+2); erc != 0 || B(pOut) != 0xFF {
		t.Errorf("ParseBool(1) = %d %x", erc, B(pOut))
	}
	// AppendRgch truncates at 255
	sb = tmpSb("")
	for i := 0; i < 300; i++ {
		strAppendCh(sb, 'x')
	}
	if FB(sb) != 255 {
		t.Errorf("AppendCh past 255: %d", FB(sb))
	}
	// CompareSb (collating, case-insensitive)
	if strCompareSb(tmpSb("abc"), tmpSb("ABC")) != 1 || strCompareSb(tmpSb("ab"), tmpSb("abc")) != 0 ||
		strCompareSb(tmpSb("b"), tmpSb("A")) != 2 {
		t.Errorf("CompareSb")
	}
}

// ---- date/time ----

var ctosEpoch = time.Date(1952, 3, 1, 0, 0, 0, 0, time.UTC)

func ctosDt(tm time.Time) uint32 {
	days := uint32(tm.Sub(ctosEpoch).Hours() / 24)
	s := tm.Hour()*3600 + tm.Minute()*60 + tm.Second()
	return (days*2+uint32(s/43200))<<16 | uint32(s%43200)
}

func TestDateTimeRoundTrip(t *testing.T) {
	defer Release(Mark())
	exp := DG(Push(8))
	ret := DG(Push(4))
	for tm := ctosEpoch; tm.Before(time.Date(2041, 11, 16, 0, 0, 0, 0, time.UTC)); tm = tm.Add(37*time.Hour + 17*time.Minute + 3*time.Second) {
		dt := ctosDt(tm)
		if erc := rtExpandDateTime(dt, exp); erc != 0 {
			t.Fatal(erc)
		}
		got := time.Date(int(FW(exp)), time.Month(FB(exp.Add(2))+1), int(FB(exp.Add(3))),
			int(FB(exp.Add(5))), int(FB(exp.Add(6))), int(FB(exp.Add(7))), 0, time.UTC)
		if !got.Equal(tm) || int(FB(exp.Add(4))) != int(tm.Weekday()) {
			t.Fatalf("ExpandDateTime(%08x) = %v dow %d, want %v dow %d", dt, got, FB(exp.Add(4)), tm, tm.Weekday())
		}
		if erc := rtCompactDateTime(exp, ret); erc != 0 || FDW(ret) != dt {
			t.Fatalf("CompactDateTime(%v) = %d %08x, want %08x", tm, erc, FDW(ret), dt)
		}
	}
	// 1 Mar 1952 00:00 = 0, a Saturday
	rtExpandDateTime(0, exp)
	if FW(exp) != 1952 || FB(exp.Add(2)) != 2 || FB(exp.Add(3)) != 1 || FB(exp.Add(4)) != 6 {
		t.Errorf("epoch expands wrong")
	}
	// errors
	set := func(y uint16, m, d, dow, h, mi, s byte) {
		SetFW(exp, y)
		PutBytes(exp.Add(2), []byte{m, d, dow, h, mi, s})
	}
	set(1951, 0, 1, 0, 0, 0, 0)
	if e := rtCompactDateTime(exp, ret); e != 2700 {
		t.Errorf("year 1951: %d", e)
	}
	set(2000, 1, 30, 0, 0, 0, 0)
	if e := rtCompactDateTime(exp, ret); e != 2701 {
		t.Errorf("Feb 30: %d", e)
	}
	set(2000, 1, 29, 2, 0, 0, 0) // 29 Feb 2000 was a Tuesday
	if e := rtCompactDateTime(exp, ret); e != 0 {
		t.Errorf("Feb 29 2000: %d", e)
	}
	set(2000, 1, 29, 3, 0, 0, 0)
	if e := rtCompactDateTime(exp, ret); e != 2702 {
		t.Errorf("wrong weekday: %d", e)
	}
	set(2000, 12, 1, 0, 0, 0, 0)
	if e := rtCompactDateTime(exp, ret); e != 2703 {
		t.Errorf("month 12: %d", e)
	}
	set(2000, 1, 29, 2, 24, 0, 0) // 24:00 is accepted
	if e := rtCompactDateTime(exp, ret); e != 0 {
		t.Errorf("24:00: %d", e)
	}
	// CompactDt finds the weekday itself
	set(1987, 4, 7, 0xFF, 15, 4, 9)
	want := ctosDt(time.Date(1987, 5, 7, 15, 4, 9, 0, time.UTC))
	if got := dtCompactDt(exp); got != want {
		t.Errorf("CompactDt = %08x want %08x", got, want)
	}
}

func TestFormatDateTime(t *testing.T) {
	defer Release(Mark())
	sb := tmpSb("")
	dt := ctosDt(time.Date(1987, 5, 7, 15, 4, 9, 0, time.UTC))
	cases := map[string]string{
		"!oo!/!0d!/!0y! !hh!:!0m! !aa!": " 5/07/87  3:04 pm", // the score table format
		"!*w! !*n! !d!, !yyyy!":         "Thursday May 7, 1987",
		"!WWW! !NNN!":                   "THU MAY",
		"!w!/!nn!":                      "T/Ma", // the key length is the field width
		"!tt!:!mm!:!ss!":                "15: 4: 9",
		"a!!b":                          "a!b",
		"!*y!":                          "1987",
		"no keys":                       "no keys",
		"!x!":                           "x",
	}
	for f, want := range cases {
		if erc := rtFormatDateTime(dt, tmpSb(f), sb); erc != 0 || sbStr(sb) != want {
			t.Errorf("FormatDateTime(%q) = %q, want %q", f, sbStr(sb), want)
		}
	}
	dtFmtDateAndTime(sb, ctosDt(time.Date(2026, 11, 2, 0, 5, 0, 0, time.UTC)))
	if sbStr(sb) != "11/02/126 00:05:00" { // year - 1900, as the original
		t.Errorf("FmtDateAndTime = %q", sbStr(sb))
	}
	// midnight shows as 12 am
	rtFormatDateTime(ctosDt(time.Date(2001, 1, 1, 0, 30, 0, 0, time.UTC)), tmpSb("!hh!:!0m! !aa!"), sb)
	if sbStr(sb) != "12:30 am" {
		t.Errorf("midnight = %q", sbStr(sb))
	}
	// DateTime arithmetic
	d1 := ctosDt(time.Date(2000, 1, 1, 23, 0, 0, 0, time.UTC))
	d2 := ctosDt(time.Date(1999, 12, 31, 1, 0, 0, 0, time.UTC))
	if dtDtDiff(d1, d2) != 46*3600 {
		t.Errorf("DtDiff = %d", dtDtDiff(d1, d2))
	}
	if dtDtAddSeconds(d2, 46*3600%65536) != ctosDt(time.Date(1999, 12, 31, 1, 0, 0, 0, time.UTC).Add(time.Duration(46*3600%65536)*time.Second)) {
		t.Errorf("DtAddSeconds")
	}
	if dtDtSubSeconds(d1, 30000) != ctosDt(time.Date(2000, 1, 1, 23, 0, 0, 0, time.UTC).Add(-30000*time.Second)) {
		t.Errorf("DtSubSeconds")
	}
}

func TestHelpers(t *testing.T) {
	if hlpFracMul(10000, 0x8000) != 5000 || hlpFracDiv(1, 2) != 0x8000 || hlpFracDiv(5, 5) != 0xFFFF {
		t.Errorf("FracMul/FracDiv")
	}
	if hlpIsNil(NilPtr) != 0xFF || hlpNotNil(NilPtr) != 0 || hlpNotNil(DG(0)) != 0xFF {
		t.Errorf("IsNil/NotNil")
	}
	defer Release(Mark())
	a := Push(16)
	for i := uint16(0); i < 4; i++ {
		SetDW(a+4*i, 0x11110000+uint32(i))
	}
	// FindDw bug: searches for the pointer value itself, not dw
	if hlpFindDw(0x11110002, DG(a), 4) != 0xFFFF {
		t.Errorf("FindDw should not find dw (original bug)")
	}
	SetP(a+8, DG(a))
	if hlpFindDw(0, DG(a), 4) != 2 {
		t.Errorf("FindDw finds the pointer value")
	}
	if hlpCmpDw(1, 2) != 0 || hlpCmpDw(2, 2) != 1 || hlpCmpDw(0x10000, 2) != 2 {
		t.Errorf("CmpDw")
	}
	if rtMul32(0xFFFF, 0xFFFF) != 0xFFFE0001 {
		t.Errorf("Mul32")
	}
}

// ---- fake platform for forms and scores ----

type fakeScreen struct{ cells [34 * 80]uint16 }

func (s *fakeScreen) at(col, row uint16) *uint16 { return &s.cells[int(row)*80+int(col)] }
func (s *fakeScreen) text(row uint16, col, n int) string {
	b := make([]byte, n)
	for i := range b {
		c := byte(*s.at(uint16(col+i), row))
		if c == 0 {
			c = ' '
		}
		b[i] = c
	}
	return string(b)
}

func installFakes(t *testing.T, keys []byte, files map[string][]byte) *fakeScreen {
	scr := &fakeScreen{}
	// save every api variable we replace
	sReadChars, sReadCells, sGetCell, sFillAttr := VidReadChars, VidReadCells, VidGetCell, VidFillAttr
	sFillCells, sPutChars, sPutCells, sClear := VidFillCells, VidPutChars, VidPutCells, ClearRect
	sReadKbd, sBeep, sAlloc, sDealloc := ReadKbd, Beep, AllocMemorySL, DeallocMemorySL
	sOpen, sCreate, sRead, sWrite, sClose, sChg := OpenFile, CreateFile, ReadFile, WriteFile, CloseFile, ChangeFileLength
	sAscb, sGetDT, sPos, sErrExit := GetpAscb, GetDateTime, PosFrameCursor, ErrorExit
	sVidCfg, sCols, sLines := P(ldgPVidCfg), W(ldgCScrCols), W(ldgCScrLines)
	t.Cleanup(func() {
		VidReadChars, VidReadCells, VidGetCell, VidFillAttr = sReadChars, sReadCells, sGetCell, sFillAttr
		VidFillCells, VidPutChars, VidPutCells, ClearRect = sFillCells, sPutChars, sPutCells, sClear
		ReadKbd, Beep, AllocMemorySL, DeallocMemorySL = sReadKbd, sBeep, sAlloc, sDealloc
		OpenFile, CreateFile, ReadFile, WriteFile, CloseFile, ChangeFileLength = sOpen, sCreate, sRead, sWrite, sClose, sChg
		GetpAscb, GetDateTime, PosFrameCursor, ErrorExit = sAscb, sGetDT, sPos, sErrExit
		SetP(ldgPVidCfg, sVidCfg)
		SetW(ldgCScrCols, sCols)
		SetW(ldgCScrLines, sLines)
	})

	VidReadChars = func(pb Ptr, col, row, cnt uint16) {
		for i := uint16(0); i < cnt; i++ {
			SetFB(pb.Add(i), byte(*scr.at(col+i, row)))
		}
	}
	VidReadCells = func(pb Ptr, col, row, cnt uint16) {
		for i := uint16(0); i < cnt; i++ {
			SetFW(pb.Add(2*i), *scr.at(col+i, row))
		}
	}
	VidGetCell = func(col, row uint16) uint16 { return *scr.at(col, row) }
	VidFillAttr = func(col, row, cnt uint16, attr byte) {
		for i := uint16(0); i < cnt; i++ {
			c := scr.at(col+i, row)
			*c = *c&0xFF | uint16(attr)<<8
		}
	}
	VidFillCells = func(col, row, cnt, w uint16) {
		for i := uint16(0); i < cnt; i++ {
			*scr.at(col+i, row) = w
		}
	}
	VidPutChars = func(col, row, cnt uint16, pb Ptr, attr byte) {
		for i := uint16(0); i < cnt; i++ {
			*scr.at(col+i, row) = uint16(FB(pb.Add(i))) | uint16(attr)<<8
		}
	}
	VidPutCells = func(col, row, cnt uint16, pb Ptr) {
		for i := uint16(0); i < cnt; i++ {
			*scr.at(col+i, row) = FW(pb.Add(2 * i))
		}
	}
	ClearRect = func(col, row, cCols, cRows uint16) {
		for r := uint16(0); r < cRows; r++ {
			VidFillCells(col, row+r, cCols, 0)
		}
	}
	ReadKbd = func(pCh Ptr) uint16 {
		if len(keys) == 0 {
			t.Fatal("keyboard script exhausted")
		}
		SetFB(pCh, keys[0])
		keys = keys[1:]
		return 0
	}
	Beep = func() uint16 { return 0 }
	PosFrameCursor = func(f, c, l uint16) uint16 { return 0 }
	ErrorExit = func(erc uint16) { t.Fatalf("ErrorExit(%d)", erc) }
	AllocMemorySL = func(cb uint16, pp Ptr) uint16 { SetFP(pp, MkPtr(AllocSeg(int(cb)), 0)); return 0 }
	DeallocMemorySL = func(p Ptr, cb uint16) uint16 { FreeSeg(p.Seg()); return 0 }
	GetDateTime = func(p Ptr) uint16 {
		SetFDW(p, ctosDt(time.Date(1987, 5, 7, 15, 4, 9, 0, time.UTC)))
		return 0
	}
	ascb := AllocSeg(0x100)
	SetFB(MkPtr(ascb, 0x40), 3)
	PutBytes(MkPtr(ascb, 0x41), []byte("Bob"))
	GetpAscb = func(pp Ptr) uint16 { SetFP(pp, MkPtr(ascb, 0)); return 0 }

	// file system: fh = index into names
	var names []string
	OpenFile = func(pFh, pbName Ptr, cbName uint16, pbPw Ptr, cbPw, mode uint16) uint16 {
		n := string(Bytes(pbName, int(cbName)))
		if _, ok := files[n]; !ok {
			return 0xCB
		}
		names = append(names, n)
		SetFW(pFh, uint16(len(names)-1))
		return 0
	}
	CreateFile = func(pbName Ptr, cbName uint16, pbPw Ptr, cbPw uint16, lfa uint32) uint16 {
		files[string(Bytes(pbName, int(cbName)))] = make([]byte, lfa)
		return 0
	}
	ReadFile = func(fh uint16, pBuf Ptr, cb uint16, lfa uint32, pcb Ptr) uint16 {
		d := files[names[fh]]
		if int(lfa)+int(cb) > len(d) {
			return 2
		}
		PutBytes(pBuf, d[lfa:int(lfa)+int(cb)])
		SetFW(pcb, cb)
		return 0
	}
	WriteFile = func(fh uint16, pBuf Ptr, cb uint16, lfa uint32, pcb Ptr) uint16 {
		d := files[names[fh]]
		if int(lfa)+int(cb) > len(d) {
			return 2
		}
		copy(d[lfa:], Bytes(pBuf, int(cb)))
		SetFW(pcb, cb)
		return 0
	}
	ChangeFileLength = func(fh uint16, lfa uint32) uint16 {
		d := files[names[fh]]
		nd := make([]byte, lfa)
		copy(nd, d)
		files[names[fh]] = nd
		return 0
	}
	CloseFile = func(fh uint16) uint16 { return 0 }

	// CTOS video structure: 1 frame = the whole 80 x 34 screen
	vs := AllocSeg(0x100)
	SetFB(MkPtr(vs, uVidCFrames), 1)
	PutBytes(MkPtr(vs, uVidRgbFrames), []byte{0, 0, 34, 80})
	SetP(ldgPVidCfg, MkPtr(vs, 0))
	SetW(ldgCScrCols, 80)
	SetW(ldgCScrLines, 34)
	return scr
}

func scoreRec(score uint16, dt uint32, name string) []byte {
	r := make([]byte, 26)
	r[0], r[1] = byte(score), byte(score>>8)
	r[2], r[3], r[4], r[5] = byte(dt), byte(dt>>8), byte(dt>>16), byte(dt>>24)
	copy(r[6:], []byte(name + "                    ")[:20])
	return r
}

const testScoresFile = "[!sys]<games>Rats-3-50.scores"

func putFileName() uint16 {
	o := Push(64)
	SetB(o, byte(len(testScoresFile)))
	PutBytes(DG(o+1), []byte(testScoresFile))
	return o
}

func TestScores(t *testing.T) {
	defer Release(Mark())
	old := ctosDt(time.Date(1986, 1, 2, 9, 30, 0, 0, time.UTC))
	page := make([]byte, 512)
	page[0] = 2
	copy(page[2:], scoreRec(5000, old, "Alice"))
	copy(page[28:], scoreRec(1000, old, "Carol"))
	files := map[string][]byte{testScoresFile: page}
	// keys: 3 x BACKSPACE, 'A', 'l', GO
	scr := installFakes(t, []byte{8, 8, 8, 'A', 'l', 0x1B}, files)
	pFile := putFileName()
	spStart := Mark()

	if hs := scGetHighScore(pFile); hs != 5000 {
		t.Fatalf("GetHighScore = %d", hs)
	}
	scEnterHighScore(pFile, 2500, 3)
	got := files[testScoresFile]
	want := make([]byte, 512)
	want[0] = 3
	now := ctosDt(time.Date(1987, 5, 7, 15, 4, 9, 0, time.UTC))
	copy(want[2:], scoreRec(5000, old, "Alice"))
	copy(want[28:], scoreRec(2500, now, "Al"))
	copy(want[54:], scoreRec(1000, old, "Carol"))
	if !bytes.Equal(got, want) {
		t.Errorf("score page:\n got %q\nwant %q", got[:80], want[:80])
	}
	// the table was cleared again
	for r := uint16(3); r < 3+26; r++ {
		if s := scr.text(r, 20, 40); s != "                                        " {
			t.Errorf("row %d not cleared: %q", r, s)
		}
	}
	if P(ldgPFreeHead) != NilPtr {
		t.Errorf("heap free list not empty: %08x", P(ldgPFreeHead))
	}
	if Mark() != spStart {
		t.Errorf("stack not balanced")
	}
}

func TestScoresShown(t *testing.T) {
	defer Release(Mark())
	old := ctosDt(time.Date(1986, 1, 2, 21, 30, 0, 0, time.UTC))
	page := make([]byte, 512)
	page[0] = 10
	for i := 0; i < 10; i++ {
		copy(page[2+26*i:], scoreRec(uint16(9000-i*500), old, "Zed"))
	}
	files := map[string][]byte{testScoresFile: page}
	var scr *fakeScreen
	var rows []string
	keys := []byte{'x', 0x1B} // not in the top 10: any key but GO/CANCEL/FINISH beeps
	scr = installFakes(t, keys, files)
	pFile := putFileName()
	// capture the screen while waiting for the key
	rk := ReadKbd
	ReadKbd = func(p Ptr) uint16 {
		if rows == nil {
			for r := uint16(3); r < 3+26; r++ {
				rows = append(rows, scr.text(r, 20, 40))
			}
		}
		return rk(p)
	}
	scEnterHighScore(pFile, 10, 3)
	if !bytes.Equal(files[testScoresFile], page) {
		t.Errorf("file changed")
	}
	if rows[0] != "              HIGH  SCORES              " {
		t.Errorf("title %q", rows[0])
	}
	if rows[2] != "      9000   1/02/86  9:30 pm  Zed      " {
		t.Errorf("row 1 %q", rows[2])
	}
	if rows[22] != "         Better luck next time.         " || rows[23] != "           Press GO to exit.            " {
		t.Errorf("messages %q / %q", rows[22], rows[23])
	}
}

func TestOptionsForm(t *testing.T) {
	defer Release(Mark())
	files := map[string][]byte{}
	// TAB to the 2nd field (maze density), type over it: delete-to-start via BACKSPACE x3,
	// "75", then GO
	scr := installFakes(t, []byte{0x0A, 8, 8, 8, '7', '5', 0x1B}, files)
	// a record laid out like the options variables: offsets per the form (DGROUP 0680h)
	rec := Push(32)
	SetW(rec+0, 3)    // nFactories (off 0)
	SetW(rec+2, 50)   // density   (off 2)
	SetB(rec+0x10, 7) // version string at off 10h: sb
	PutBytes(DG(rec+0x11), []byte("Rats9.2"))
	if f := frmDoForm(rec, 0x0680, 0, 0xFF, 0xFF, rec); !Test(f) {
		t.Fatalf("DoForm returned FALSE")
	}
	if W(rec) != 3 || W(rec+2) != 75 {
		t.Errorf("parsed factories %d density %d", W(rec), W(rec+2))
	}
	_ = scr
}
