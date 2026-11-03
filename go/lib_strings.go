package main

// lib_strings.go - translation of ../src/Strings.plm (image 5CD2-6BB4, frame 05CDh).
// Strings are "sb": a length byte followed by up to 255 characters.  POINTER parameters
// are Ptr, WORD ("o"/near) parameters are DGROUP offsets.  CONST at DGROUP 0D82-10DE is
// used in place (from dgroup.bin).

const (
	ldgRgwPow10    = 0x0D82 // (5) WORD 1,10,100,1000,10000
	ldgSbNull      = 0x0D8C // empty string
	ldgRgbLower    = 0x0D8D // (256) BYTE
	ldgRgbUpper    = 0x0E8D // (256) BYTE
	ldgRgbCollate  = 0x0F8D // (256) BYTE
	ldgRgchHex     = 0x108D // '0123456789ABCDEF'
	ldgRgchRoman   = 0x109D // 'iviiixlxxxcdcccmVmmmXLXXX'
	ldgMpichdich   = 0x10B6 // (5) BYTE 20,15,10,5,0
	ldgMpdigitich  = 0x10BB // (10) BYTE
	ldgMpdigitcch  = 0x10C5 // (10) BYTE
	ldgSbNumericCh = 0x10CF // (15,'+-0123456789',6,10H,'.') inline constant of FNumericSb
)

// UcbType: reserved WORD, defaultVol(13), defaultDir(13), defaultPW(13), prefix(41), defaultNode(13)
const (
	uUcbDefaultVol  = 2
	uUcbDefaultDir  = 15
	uUcbPrefix      = 41
	uUcbDefaultNode = 82
	uUcbSize        = 95
)

func uSbCch(p Ptr) uint16         { return uint16(FB(p)) }
func uSbRgch(p Ptr, i uint16) Ptr { return p.Add(1 + i) } // @sb.rgch(i)

// UlCmpB (OsSub 3): index of the first differing byte, 0FFFFh if equal.
func strUlCmpB(p1, p2 Ptr, cch uint16) uint16 { return CMPB(p1, p2, cch) }

func strAppendCh(psbDst Ptr, ch byte) {
	if FB(psbDst) < 255 {
		SetFB(uSbRgch(psbDst, uSbCch(psbDst)), ch)
		SetFB(psbDst, FB(psbDst)+1)
	}
}

func strAppendRgch(psbDst Ptr, prgch Ptr, cch uint16) {
	cchRemain := 255 - uSbCch(psbDst)
	if cch > cchRemain {
		cch = cchRemain
	}
	MOVB(prgch, uSbRgch(psbDst, uSbCch(psbDst)), cch)
	SetFB(psbDst, byte(uSbCch(psbDst)+cch))
}

func strAppendSb(psbDst, psbSrc Ptr) {
	strAppendRgch(psbDst, uSbRgch(psbSrc, 0), uSbCch(psbSrc))
}

// MakeFileSpec: build "{node}[vol]<dir>file" from its parts (nil pointers are skipped).
func strMakeFileSpec(psbRet, psbNode, psbVol, psbDir, psbFile Ptr) {
	SetFB(psbRet, 0)
	if Test(hlpNotNil(psbNode)) {
		if FB(psbNode) != 0 {
			strAppendCh(psbRet, '{')
			strAppendSb(psbRet, psbNode)
			strAppendCh(psbRet, '}')
		}
	}
	if Test(hlpNotNil(psbVol)) {
		strAppendCh(psbRet, '[')
		strAppendSb(psbRet, psbVol)
		strAppendCh(psbRet, ']')
	}
	if Test(hlpNotNil(psbDir)) {
		strAppendCh(psbRet, '<')
		strAppendSb(psbRet, psbDir)
		strAppendCh(psbRet, '>')
	}
	if Test(hlpNotNil(psbFile)) {
		strAppendSb(psbRet, psbFile)
	}
}

func strCopySb(psbDst, psbSrc Ptr) {
	MOVB(psbSrc, psbDst, uSbCch(psbSrc)+1)
}

// DeleteRgch: delete cch characters at ich.
func strDeleteRgch(psb Ptr, ich, cch uint16) {
	if uSbCch(psb) < ich {
		ich = uSbCch(psb)
	}
	if uSbCch(psb)-ich < cch {
		cch = uSbCch(psb) - ich
	}
	SetFB(psb, byte(uSbCch(psb)-cch))
	MOVB(uSbRgch(psb, cch+ich), uSbRgch(psb, ich), uSbCch(psb)-ich)
}

// SubSb: sbDst = cch characters of sbSrc starting at ich.
func strSubSb(psbDst, psbSrc Ptr, ich, cch uint16) {
	if uSbCch(psbSrc) < ich {
		ich = uSbCch(psbSrc)
	}
	if uSbCch(psbSrc)-ich < cch {
		cch = uSbCch(psbSrc) - ich
	}
	SetFB(psbDst, 0)
	strAppendRgch(psbDst, uSbRgch(psbSrc, ich), cch)
}

func strFAlphaCh(ch byte) byte {
	return Bool((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z'))
}

func strFChInSb(ch byte, psb Ptr) byte {
	return Bool(FINDB(uSbRgch(psb, 0), ch, uSbCch(psb)) != 0xFFFF)
}

func strFEqRgch(prgch1, prgch2 Ptr, cch uint16) byte {
	return Bool(strUlCmpB(prgch1, prgch2, cch) == 0xFFFF)
}

func strFEqSb(psb1, psb2 Ptr) byte {
	if FB(psb1) == FB(psb2) {
		return strFEqRgch(uSbRgch(psb1, 0), uSbRgch(psb2, 0), uSbCch(psb1))
	}
	return FALSE
}

// GetField: sbDst = TAB-separated field number iField of sbSrc.
func strGetField(psbDst, psbSrc Ptr, iField uint16) byte {
	var ichLim uint16
	ichFirst, i := uint16(0), uint16(0)
	for uSbCch(psbSrc) > ichFirst {
		dich := FINDB(uSbRgch(psbSrc, ichFirst), 9, uSbCch(psbSrc)-ichFirst)
		if dich == 0xFFFF {
			ichLim = uSbCch(psbSrc)
		} else {
			ichLim = ichFirst + dich
		}
		if iField == i {
			strSubSb(psbDst, psbSrc, ichFirst, ichLim-ichFirst)
			return TRUE
		}
		i++
		ichFirst = ichLim + 1
	}
	return FALSE
}

// FIntFromRgch: signed decimal with optional '+'/'-' sign, -7FFFh..7FFFh, into WORD at ow.
func strFIntFromRgch(ow uint16, prgch Ptr, cch uint16) byte {
	defer Release(Mark())
	wVal := Push(2)
	var ich, fNeg uint16
	if cch == 0 {
		return FALSE
	}
	if FB(prgch) == '-' {
		ich = 1
		fNeg = 0xFF
	} else if FB(prgch) == '+' {
		ich = 1
		fNeg = 0
	} else {
		fNeg, ich = 0, 0
	}
	if !Test(strFWFromRgch(wVal, prgch.Add(ich), cch-ich)) {
		return FALSE
	}
	if W(wVal) > 0x7FFF {
		return FALSE
	}
	if fNeg&1 != 0 {
		SetW(ow, -W(wVal))
	} else {
		SetW(ow, W(wVal))
	}
	return TRUE
}

func strFIntFromSb(ow uint16, psb Ptr) byte {
	return strFIntFromRgch(ow, uSbRgch(psb, 0), uSbCch(psb))
}

// FNumericCh: digits, the fraction characters 06h (1/2) and 10h (1/4), '.'.
func strFNumericCh(ch byte) byte {
	return Bool((ch >= '0' && ch <= '9') || ch == 6 || ch == 0x10 || ch == '.')
}

func strFNumericSb(psb Ptr) byte {
	if FB(psb) != 0 {
		if Test(strFChInSb(FB(uSbRgch(psb, 0)), DG(ldgSbNumericCh))) {
			return TRUE
		}
	}
	return FALSE
}

// ParseFileSpec: split "{node}[vol]<dir>file"; missing parts get the user's defaults.
func strParseFileSpec(psbNode, psbVol, psbDir, psbFile, psbSpec Ptr) byte {
	defer Release(Mark())
	ucb := Push(uUcbSize)
	ceCheckErc(GetUcb(DG(ucb), uUcbSize))
	if Test(hlpNotNil(psbNode)) {
		SetFB(psbNode, 0)
	}
	if Test(hlpNotNil(psbVol)) {
		SetFB(psbVol, 0)
	}
	if Test(hlpNotNil(psbDir)) {
		SetFB(psbDir, 0)
	}
	if Test(hlpNotNil(psbFile)) {
		SetFB(psbFile, 0)
	}
	ich := uint16(0)
	var cch uint16
	spec := func(i uint16) byte { return FB(uSbRgch(psbSpec, i)) }

	// one part: open/close delimiters, destination, default string
	part := func(chOpen, chClose byte, psbPart Ptr, oDefault uint16) bool {
		if spec(ich) == chOpen {
			cch = FINDB(uSbRgch(psbSpec, ich), chClose, uSbCch(psbSpec)-ich) + 1
			if cch == 0 || cch > 14 {
				return false
			}
			if Test(hlpNotNil(psbPart)) {
				strAppendRgch(psbPart, uSbRgch(psbSpec, ich+1), cch-2)
			}
			ich = ich + cch
		} else {
			if ich != 0 {
				return false
			}
			if Test(hlpNotNil(psbPart)) {
				strAppendSb(psbPart, DG(ucb+oDefault))
			}
		}
		return true
	}
	if !part('{', '}', psbNode, uUcbDefaultNode) ||
		!part('[', ']', psbVol, uUcbDefaultVol) ||
		!part('<', '>', psbDir, uUcbDefaultDir) {
		// Fail:
		if Test(hlpNotNil(psbFile)) {
			strAppendRgch(psbFile, uSbRgch(psbSpec, ich), uSbCch(psbSpec)-ich)
		}
		return FALSE
	}
	if Test(hlpNotNil(psbFile)) {
		if ich == 0 && B(ucb+uUcbPrefix) != 0 {
			strAppendSb(psbFile, DG(ucb+uUcbPrefix))
		}
		strAppendRgch(psbFile, uSbRgch(psbSpec, ich), uSbCch(psbSpec)-ich)
	}
	return TRUE
}

func strFPrefixSb(psb, psbPrefix Ptr) byte {
	if FB(psb) >= FB(psbPrefix) {
		return strFEqRgch(uSbRgch(psb, 0), uSbRgch(psbPrefix, 0), uSbCch(psbPrefix))
	}
	return FALSE
}

// FDwFromRgch: unsigned decimal to the DWORD at odw (no overflow check).
func strFDwFromRgch(odw uint16, prgch Ptr, cch uint16) byte {
	if cch == 0 {
		return FALSE
	}
	SetDW(odw, 0)
	for ich := uint16(0); ich < cch; ich++ {
		ch := FB(prgch.Add(ich))
		if ch < '0' || ch > '9' {
			return FALSE
		}
		SetDW(odw, rtLqDwordMul(DW(odw), 10)+uint32(ch)-'0')
	}
	return TRUE
}

func strFDwFromSb(odw uint16, psb Ptr) byte {
	return strFDwFromRgch(odw, uSbRgch(psb, 0), uSbCch(psb))
}

func strFSuffixSb(psb, psbSuffix Ptr) byte {
	if FB(psb) >= FB(psbSuffix) {
		return strFEqRgch(uSbRgch(psb, uSbCch(psb)-uSbCch(psbSuffix)), uSbRgch(psbSuffix, 0),
			uSbCch(psbSuffix))
	}
	return FALSE
}

// FFixedFromRgch: "int.frac", "int<06h>" (1/2) or "int<10h>" (1/4): WORD wInt at owInt and
// the fraction (units of 1/10000h) at owFrac.
func strFFixedFromRgch(owInt, owFrac uint16, prgch Ptr, cch uint16) byte {
	if cch == 0 {
		return FALSE
	}
	ichDot := FINDB(prgch, '.', cch)
	if ichDot == 0xFFFF {
		if FB(prgch.Add(cch-1)) == 6 {
			ichDot = cch - 1
			SetW(owFrac, 0x8000)
		} else if FB(prgch.Add(cch-1)) == 0x10 {
			ichDot = cch - 1
			SetW(owFrac, 0x4000)
		} else {
			ichDot = cch
			SetW(owFrac, 0)
		}
	} else if ichDot+1 == cch {
		if ichDot == 0 {
			return FALSE
		}
		SetW(owFrac, 0)
	} else {
		ichFrac := ichDot + 1
		cchFrac := cch - ichFrac
		if !Test(strFWFromRgch(owFrac, prgch.Add(ichFrac), cchFrac)) {
			return FALSE
		}
		if cchFrac >= 5 {
			return FALSE
		}
		SetW(owFrac, hlpFracDiv(W(owFrac), W(ldgRgwPow10+2*cchFrac)))
	}
	if ichDot == 0 {
		SetW(owInt, 0)
	} else if !Test(strFWFromRgch(owInt, prgch, ichDot)) {
		return FALSE
	}
	return TRUE
}

func strFFixedFromSb(owInt, owFrac uint16, psb Ptr) byte {
	return strFFixedFromRgch(owInt, owFrac, uSbRgch(psb, 0), uSbCch(psb))
}

func strFWFromRgch(ow uint16, prgch Ptr, cch uint16) byte {
	defer Release(Mark())
	dw := Push(4)
	if !Test(strFDwFromRgch(dw, prgch, cch)) {
		return FALSE
	}
	if W(dw+2) != 0 {
		return FALSE
	}
	SetW(ow, W(dw))
	return TRUE
}

func strFWFromSb(ow uint16, psb Ptr) byte {
	return strFWFromRgch(ow, uSbRgch(psb, 0), uSbCch(psb))
}

// InsertSb: insert sbSrc into sbDst at ich (truncating at 255).
func strInsertSb(psbDst Ptr, ich uint16, psbSrc Ptr) {
	if uSbCch(psbDst) < ich {
		ich = uSbCch(psbDst)
	}
	cchIns := uSbCch(psbSrc)
	cchTail := uSbCch(psbDst) - ich
	cchMax := 255 - ich
	if cchTail+cchIns > cchMax {
		if cchIns > cchMax {
			cchIns = cchMax
		}
		cchTail = cchMax - cchIns
	}
	MOVRB(uSbRgch(psbDst, ich), uSbRgch(psbDst, ich+cchIns), cchTail)
	MOVB(uSbRgch(psbSrc, 0), uSbRgch(psbDst, ich), cchIns)
	SetFB(psbDst, byte(ich+cchIns+cchTail))
}

// CompareRgch via rgbCollate: 0 = less, 1 = equal, 2 = greater.
func strCompareRgch(prgch1 Ptr, cch1 uint16, prgch2 Ptr, cch2 uint16) byte {
	for ich := uint16(0); ich < cch1; ich++ {
		if ich >= cch2 {
			return 2
		}
		ch1 := B(ldgRgbCollate + uint16(FB(prgch1.Add(ich))))
		ch2 := B(ldgRgbCollate + uint16(FB(prgch2.Add(ich))))
		if ch1 < ch2 {
			return 0
		}
		if ch1 > ch2 {
			return 2
		}
	}
	if cch1 == cch2 {
		return 1
	}
	return 0
}

func strCompareSb(psb1, psb2 Ptr) byte {
	return strCompareRgch(uSbRgch(psb1, 0), uSbCch(psb1), uSbRgch(psb2, 0), uSbCch(psb2))
}

func strSbFromInt(psb Ptr, n int16, cch uint16) {
	var fNeg byte
	if n >= 0 {
		fNeg = FALSE
	} else {
		n = -n
		fNeg = TRUE
	}
	strSbFromNumberA(psb, uint32(uint16(n)), cch, ' ', fNeg)
}

// SbFromNumber: decimal, right aligned in at least cch chars, filled with chFill.
func strSbFromNumber(psb Ptr, q uint32, cch uint16, chFill byte) {
	strSbFromNumberA(psb, q, cch, chFill, FALSE)
}

func strSbFromW(psb Ptr, w uint16) {
	strSbFromNumber(psb, uint32(w), 1, ' ')
}

// SbFromFixed: wInt.frac with cDec decimals (max 4 significant, rest '0').
func strSbFromFixed(psb Ptr, wInt, wFrac, cchInt, cDec uint16, chFill byte) {
	defer Release(Mark())
	sbTemp := DG(Push(256))
	var cDecSig uint16
	if cDec < 5 {
		cDecSig = cDec
	} else {
		cDecSig = 4
	}
	wDec := hlpFracMul(W(ldgRgwPow10+2*cDecSig), wFrac)
	if W(ldgRgwPow10+2*cDecSig) == wDec {
		wDec = 0
		wInt++
	}
	SetFB(psb, 0)
	if wInt != 0 || cchInt != 0 {
		strSbFromNumber(sbTemp, uint32(wInt), cchInt, chFill)
		strAppendSb(psb, sbTemp)
	}
	if wDec != 0 || cDec != 0 {
		strAppendCh(psb, '.')
		strSbFromNumber(sbTemp, uint32(wDec), cDecSig, '0')
		strAppendSb(psb, sbTemp)
		for cDec > cDecSig {
			strAppendCh(psb, '0')
			cDecSig++
		}
	}
}

// SbHexFromW: 4 hex digits followed by 'h'.
func strSbHexFromW(psb Ptr, w uint16) {
	i := uint16(0)
	for i < 4 {
		w = w<<4 | w>>12
		SetFB(uSbRgch(psb, i), B(ldgRgchHex+(w&0xF)))
		i++
	}
	SetFB(uSbRgch(psb, i), 'h')
	i++
	SetFB(psb, byte(i))
}

// SbRomanFromW: roman numeral of w (5 decimal digits; the ten-thousands digit 9 reads one
// byte past rgchRoman, as the original does).
func strSbRomanFromW(psb Ptr, w uint16) {
	defer Release(Mark())
	sbDecimal := DG(Push(13))
	strSbFromNumber(sbDecimal, uint32(w), 5, '0')
	SetFB(psb, 0)
	for ichDecimal := uint16(0); ichDecimal < uSbCch(sbDecimal); ichDecimal++ {
		digit := FB(uSbRgch(sbDecimal, ichDecimal)) - '0'
		ichRoman := uint16(B(ldgMpdigitich+uint16(digit)) + B(ldgMpichdich+ichDecimal))
		cchRoman := uint16(B(ldgMpdigitcch + uint16(digit)))
		strAppendRgch(psb, DG(ldgRgchRoman+ichRoman), cchRoman)
	}
}

// SplitSb: split sbSrc at the first ch: sbFirst (trailing blanks trimmed) and sbRest
// (leading blanks trimmed).
func strSplitSb(psbFirst Ptr, ch byte, psbRest, psbSrc Ptr) {
	ich := FINDB(uSbRgch(psbSrc, 0), ch, uSbCch(psbSrc))
	if ich == 0xFFFF {
		ich = uSbCch(psbSrc)
	}
	SetFB(psbFirst, byte(ich))
	MOVB(uSbRgch(psbSrc, 0), uSbRgch(psbFirst, 0), uSbCch(psbFirst))
	strTrimSb(psbFirst, FALSE, TRUE)
	if uSbCch(psbSrc) > ich {
		SetFB(psbRest, byte(uSbCch(psbSrc)-(ich+1)))
		MOVB(uSbRgch(psbSrc, ich+1), uSbRgch(psbRest, 0), uSbCch(psbRest))
		strTrimSb(psbRest, TRUE, FALSE)
	} else {
		SetFB(psbRest, 0)
	}
}

// TrimSb: strip leading and/or trailing blanks.
func strTrimSb(psb Ptr, fLeading, fTrailing byte) {
	if Test(fLeading) {
		ich := SKIPB(uSbRgch(psb, 0), ' ', uSbCch(psb))
		if ich == 0xFFFF {
			SetFB(psb, 0)
		} else if ich != 0 {
			SetFB(psb, byte(uSbCch(psb)-ich))
			MOVB(uSbRgch(psb, ich), uSbRgch(psb, 0), uSbCch(psb))
		}
	}
	if Test(fTrailing) {
		SetFB(psb, byte(SKIPRB(uSbRgch(psb, 0), ' ', uSbCch(psb))+1))
	}
}

func strSbFromNumberA(psb Ptr, q uint32, cch uint16, chFill byte, fNeg byte) {
	defer Release(Mark())
	rgch := Push(255)
	ichFirst := 255 - cch
	ich := uint16(255)
	for q != 0 || ich == 255 {
		ich--
		var r uint32
		q, r = rtLqDwordDiv(q, 10)
		SetB(rgch+ich, byte('0'+r))
	}
	if Test(fNeg) {
		ich--
		SetB(rgch+ich, '-')
	}
	for ich > ichFirst {
		ich--
		SetB(rgch+ich, chFill)
	}
	SetFB(psb, 0)
	strAppendRgch(psb, DG(rgch+ich), 255-ich)
}

func init() {
	CopySb = strCopySb
	AppendSb = strAppendSb
	AppendCh = strAppendCh
	SbFromNumber = strSbFromNumber
}
