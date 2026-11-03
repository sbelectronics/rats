package main

// lib_formfields.go - translation of ../src/FormFields.plm (image 5286-54E4, frame
// 0528h, code at offset 06h).  Field-type procedures of the forms runtime, reached only
// through the far pointers in the form descriptors (options form DGROUP 0680h.., score
// form 0A92h..): Fmt(pOut, pValue) turns the value in the caller's record into a string
// (text fields) or a choice value (choice fields); Parse(pValue, pIn) WORD converts back
// and returns 0 or a forms erc (31002 not Yes/No, 31003 invalid number, 31004 number out
// of range, 31005 string too long).  All parameters are DGROUP offsets.

const ldgRgfBool = 0x0C7C // (2) BYTE 0, 0FFh

func ffFmtSb(pOut, pIn uint16) { strCopySb(DG(pOut), DG(pIn)) }

func ffParseSb(pOut, pIn uint16) uint16 {
	strCopySb(DG(pOut), DG(pIn))
	return 0
}

func ffFmtSb2(pOut, pIn uint16) { strCopySb(DG(pOut), DG(pIn)) }

func ffParseSb12(pOut, pIn uint16) uint16 {
	if uint16(B(pIn)) > 12 {
		return 31005
	}
	strCopySb(DG(pOut), DG(pIn))
	return 0
}

func ffFmtSb3(pOut, pIn uint16) { strCopySb(DG(pOut), DG(pIn)) }

func ffParseSb92(pOut, pIn uint16) uint16 {
	if uint16(B(pIn)) > 92 {
		return 31005
	}
	strCopySb(DG(pOut), DG(pIn))
	return 0
}

// FmtBool: BOOLEAN -> choice value 1 (Yes) / 0 (No).
func ffFmtBool(pOut, pIn uint16) {
	if Test(B(pIn)) {
		SetB(pOut, 1)
	} else {
		SetB(pOut, 0)
	}
}

func ffParseBool(pOut, pIn uint16) uint16 {
	if uint16(B(pIn)) >= 2 {
		return 31002
	}
	SetB(pOut, B(ldgRgfBool+uint16(B(pIn))))
	return 0
}

func ffFmtByte(pOut, pIn uint16) {
	strSbFromNumber(DG(pOut), uint32(B(pIn)), 0, ' ')
}

func ffParseByte(pOut, pIn uint16) uint16 {
	defer Release(Mark())
	dw := Push(4)
	strTrimSb(DG(pIn), 0xFF, 0xFF)
	if B(pIn) == 0 {
		SetB(pOut, 0)
	} else {
		if !Test(strFDwFromSb(dw, DG(pIn))) {
			return 31003
		}
		if B(dw+1) != 0 || W(dw+2) != 0 {
			return 31004
		}
		SetB(pOut, B(dw))
	}
	return 0
}

func ffFmtNumber(pOut, pIn uint16) {
	strSbFromNumber(DG(pOut), uint32(W(pIn)), 0, ' ')
}

func ffParseNumber(pOut, pIn uint16) uint16 {
	defer Release(Mark())
	dw := Push(4)
	strTrimSb(DG(pIn), 0xFF, 0xFF)
	if B(pIn) == 0 {
		SetW(pOut, 0)
	} else {
		if !Test(strFDwFromSb(dw, DG(pIn))) {
			return 31003
		}
		if W(dw+2) != 0 {
			return 31004
		}
		SetW(pOut, W(dw))
	}
	return 0
}

func ffFmtDword(pOut, pIn uint16) {
	strSbFromNumber(DG(pOut), DW(pIn), 0, ' ')
}

func ffParseDword(pOut, pIn uint16) uint16 {
	strTrimSb(DG(pIn), 0xFF, 0xFF)
	if B(pIn) == 0 {
		SetDW(pOut, 0)
	} else if !Test(strFDwFromSb(pOut, DG(pIn))) {
		return 31003
	}
	return 0
}

func init() {
	// frame 528h: module offset + 6 (listing FormFields.lst)
	RegisterFar(0x0528, 0x0006, ffFmtSb)
	RegisterFar(0x0528, 0x001C, ffParseSb)
	RegisterFar(0x0528, 0x0035, ffFmtSb2)
	RegisterFar(0x0528, 0x004B, ffParseSb12)
	RegisterFar(0x0528, 0x0075, ffFmtSb3)
	RegisterFar(0x0528, 0x008B, ffParseSb92)
	RegisterFar(0x0528, 0x00B5, ffFmtBool)
	RegisterFar(0x0528, 0x00D3, ffParseBool)
	RegisterFar(0x0528, 0x00FE, ffFmtByte)
	RegisterFar(0x0528, 0x0122, ffParseByte)
	RegisterFar(0x0528, 0x0180, ffFmtNumber)
	RegisterFar(0x0528, 0x01A2, ffParseNumber)
	RegisterFar(0x0528, 0x01FB, ffFmtDword)
	RegisterFar(0x0528, 0x021B, ffParseDword)
}
