package main

// lib_ctosrt.go - the standard CTOS library routines linked at the end of the image
// (7306-7A5F) and their CONST at DGROUP 112A-1229.  Translated from the original
// machine code of the image (which is the pre-time-zone version of the CTOS Util
// routines; cf. CTOS CmpDT.Plm, ExpDt.Plm, FmtDt.Plm and Math32.Asm, DWord_Mf.Asm):
//   CompactDateTime  (frame 730h:06)   7306
//   ExpandDateTime   (frame 746h:0C)   746C
//   FormatDateTime   (frame 759h:02)   7592, with AppendNum 7840 and AppendText 7900
//   LQ_DWORD_DIV     (frame 79Ch:00)   79C0  (DX:AX / DI:CX -> quotient DX:AX, rem SI:DI)
//   LQ_DWORD_MUL     (frame 79Ch:59)   7A19
//   Mul32 / Div32 / Mod32 (frame 7A3h) 7A30 / 7A41 / 7A51
// CTOS date/time: low word = seconds in the half day, high word = half days since
// 1 March 1952 (a Saturday).

const (
	ldgMpMonthCumDays   = 0x112A // (12) WORD 306,337,0,31,61,...
	ldgMpMonthDays      = 0x1142 // (12) BYTE 31,28,31,...
	ldgMpIMonthSinceMar = 0x114E // (12) WORD 0,31,61,...
	ldgRgbMonths        = 0x1166 // (120) BYTE  7,'January  ',...
	ldgRgbWeekdays      = 0x11DE // (70) BYTE   6,'Sunday   ',...
	ldgRgbAmPm          = 0x1224 // (6) BYTE    2,'am',2,'pm'
)

const (
	uErcYearOutOfRange      = 2700
	uErcDayNotInMonth       = 2701
	uErcDayAndDateDisagree  = 2702
	uErcIllegalMonthOrTime  = 2703
	uErcLqDivideByZeroError = 0x1B // LQ_DWORD_DIV calls ErrorExit(27) on a zero divisor
)

// ExpDateTime record (8 bytes): year WORD, month (0-based) BYTE, day, dayOfWeek, hour,
// minute, second BYTE.

// rtCompactDateTime(pExpDateTime, pDateTimeRet) ErcType
func rtCompactDateTime(pExp, pRet Ptr) uint16 {
	year := FW(pExp)
	if year < 1952 || year >= 2042 {
		return uErcYearOutOfRange
	}
	month := FB(pExp.Add(2))
	var dYears uint16
	if month < 2 {
		dYears = year - 1953
	} else {
		dYears = year - 1952
	}
	if month > 11 {
		return uErcIllegalMonthOrTime
	}
	day := FB(pExp.Add(3))
	if day == 0 || day > B(ldgMpMonthDays+uint16(month)) {
		if month != 1 || day != 29 || year%4 != 0 {
			return uErcDayNotInMonth
		}
	}
	dDays := (dYears>>2)*1461 + (dYears&3)*365 + W(ldgMpMonthCumDays+2*uint16(month)) +
		uint16(day) - 1
	if uint16(FB(pExp.Add(4))) != (dDays+6)%7 {
		return uErcDayAndDateDisagree
	}
	hour, min, sec := FB(pExp.Add(5)), FB(pExp.Add(6)), FB(pExp.Add(7))
	if hour > 23 || min > 59 || sec > 59 {
		if hour != 24 || min != 0 || sec != 0 {
			return uErcIllegalMonthOrTime
		}
	}
	seconds := (uint16(hour)%12)*3600 + uint16(min)*60 + uint16(sec)
	SetFW(pRet, seconds)
	SetFW(pRet.Add(2), dDays<<1+uint16(hour)/12)
	return 0
}

// rtExpandDateTime(dateTime, pExpDateTimeRet) ErcType (always 0)
func rtExpandDateTime(dt uint32, p Ptr) uint16 {
	secs, dt2 := uint16(dt), uint16(dt>>16)
	cDays := dt2 / 2
	cQuadYears := cDays / 1461
	cQuadDays := cDays % 1461
	var deltaYears, iDay uint16
	if cQuadDays == 1460 {
		deltaYears, iDay = 3, 365
	} else {
		deltaYears, iDay = cQuadDays/365, cQuadDays%365
	}
	iMonth := iDay / 30
	if iMonth > 11 {
		iMonth = 11
	}
	if iDay < W(ldgMpIMonthSinceMar+2*iMonth) {
		iMonth--
	}
	SetFW(p, cQuadYears<<2+1952+deltaYears)
	SetFB(p.Add(2), byte((iMonth+2)%12))
	if FB(p.Add(2)) < 2 {
		SetFW(p, FW(p)+1)
	}
	SetFB(p.Add(3), byte(iDay-W(ldgMpIMonthSinceMar+2*iMonth)+1))
	SetFB(p.Add(4), byte((cDays+6)%7))
	SetFB(p.Add(7), byte(secs%60))
	cMins := secs / 60
	SetFB(p.Add(6), byte(cMins%60))
	SetFB(p.Add(5), byte((dt2%2)*12+cMins/60))
	return 0
}

// rtFormatDateTime(dateTime, psbTemplate, psbRet) ErcType.  Keys between '!' marks:
// y o n d w h t m s a (upper case = upper-case text), '0'/'*' fill prefixes, the key
// length gives the field width.
func rtFormatDateTime(dt uint32, psbFmt, psbDt Ptr) uint16 {
	defer Release(Mark())
	edt := Push(8)
	pEdt := DG(edt)
	if erc := rtExpandDateTime(dt, pEdt); erc != 0 {
		return erc
	}
	fmtCh := func(i uint16) Ptr { return psbFmt.Add(1 + i) } // @sbFmt.rgch(i)
	dtCh := func() Ptr { return psbDt.Add(1 + uint16(FB(psbDt))) }
	SetFB(psbDt, 0)
	cchFmt := uint16(FB(psbFmt))
	ichFmt := uint16(0)
	for ichFmt < cchFmt {
		var ichFirst, ichLim uint16
		ich := FINDB(fmtCh(ichFmt), '!', cchFmt-ichFmt)
		if ich == 0xFFFF {
			ichFirst = cchFmt
			ichLim = cchFmt
		} else {
			ichFirst = ich + ichFmt
			ich = FINDB(fmtCh(ichFirst+1), '!', cchFmt-ichFirst-1)
			if ich == 0xFFFF {
				ichLim = cchFmt
			} else {
				ichLim = ichFirst + 1 + ich
			}
		}
		if ichFirst > ichFmt {
			MOVB(fmtCh(ichFmt), dtCh(), ichFirst-ichFmt)
			SetFB(psbDt, byte(uint16(FB(psbDt))+ichFirst-ichFmt))
		}
		if ichFirst < cchFmt {
			if ichFirst+1 < ichLim {
				ichFirst++
			}
			ich = ichFirst
			chFill := byte(' ')
			fUc := byte(0)
			var ch byte
			for {
				ch = FB(fmtCh(ich))
				ich++
				if ch == '*' || ch == '0' {
					chFill = ch
					if ich < ichLim {
						continue
					}
				}
				break
			}
			var cch uint16
			if chFill == '*' {
				cch = 0
			} else {
				cch = ichLim - ichFirst
			}
			if ch >= 'A' && ch <= 'Z' {
				ch = ch + 'a' - 'A'
				fUc = 0xFF
			}
			cb := byte(cch) // AppendNum/AppendText take the width as a BYTE
			switch ch {
			case 'y':
				rtAppendNum(psbDt, FW(pEdt), cb, chFill)
			case 'o':
				rtAppendNum(psbDt, uint16(FB(pEdt.Add(2))+1), cb, chFill)
			case 'n':
				rtAppendText(psbDt, ldgRgbMonths+uint16(FB(pEdt.Add(2)))*10, cb, fUc)
			case 'd':
				rtAppendNum(psbDt, uint16(FB(pEdt.Add(3))), cb, chFill)
			case 'w':
				rtAppendText(psbDt, ldgRgbWeekdays+uint16(FB(pEdt.Add(4)))*10, cb, fUc)
			case 'h':
				hour := FB(pEdt.Add(5))
				if hour == 0 {
					hour = 12
				} else if hour > 12 {
					hour -= 12
				}
				rtAppendNum(psbDt, uint16(hour), cb, chFill)
			case 't':
				rtAppendNum(psbDt, uint16(FB(pEdt.Add(5))), cb, chFill)
			case 'm':
				rtAppendNum(psbDt, uint16(FB(pEdt.Add(6))), cb, chFill)
			case 's':
				rtAppendNum(psbDt, uint16(FB(pEdt.Add(7))), cb, chFill)
			case 'a':
				var iAmPm byte
				if FB(pEdt.Add(5)) < 12 {
					iAmPm = 0
				} else {
					iAmPm = 1
				}
				rtAppendText(psbDt, ldgRgbAmPm+uint16(iAmPm)*3, cb, fUc)
			default:
				MOVB(fmtCh(ichFirst), dtCh(), ichLim-ichFirst)
				SetFB(psbDt, byte(uint16(FB(psbDt))+ichLim-ichFirst))
			}
		}
		ichFmt = ichLim + 1
	}
	return 0
}

// AppendNum(psb, w, cch, chFill): decimal w right-aligned in cch characters (cch 0 = as
// many as needed; fewer than needed keeps the last cch digits).
func rtAppendNum(psb Ptr, w uint16, cch, chFill byte) {
	defer Release(Mark())
	rgch := Push(5)
	cchUsed := byte(0)
	for w != 0 || cchUsed == 0 {
		cchUsed++
		SetB(rgch+uint16(5-cchUsed), byte('0'+w%10))
		w /= 10
	}
	if cch == 0 {
		cch = cchUsed
	} else if cch > cchUsed {
		SETB(chFill, psb.Add(1+uint16(FB(psb))), uint16(cch-cchUsed))
		SetFB(psb, cch+FB(psb)-cchUsed)
		cch = cchUsed
	}
	MOVB(DG(rgch+uint16(5-cch)), psb.Add(1+uint16(FB(psb))), uint16(cch))
	SetFB(psb, FB(psb)+cch)
}

// AppendText(psb, osbSrc, cch, fUc): copy the (DGROUP) string osbSrc, padded with blanks
// or cut to cch characters (cch 0 = its length), upper-cased if fUc.
func rtAppendText(psb Ptr, osbSrc uint16, cch, fUc byte) {
	if cch == 0 {
		cch = B(osbSrc)
	}
	cchMove := cch
	if cchMove > B(osbSrc) {
		cchMove = B(osbSrc)
	}
	MOVB(DG(osbSrc+1), psb.Add(1+uint16(FB(psb))), uint16(cchMove))
	if Test(fUc) {
		ich := FB(psb)
		ichLim := FB(psb) + cchMove
		for ich < ichLim {
			p := psb.Add(1 + uint16(ich))
			ch := FB(p)
			if ch >= 'a' && ch <= 'z' {
				SetFB(p, ch+'A'-'a')
			}
			ich++
		}
	}
	SetFB(psb, FB(psb)+cchMove)
	cch -= cchMove
	if cch > 0 {
		SETB(' ', psb.Add(1+uint16(FB(psb))), uint16(cch))
		SetFB(psb, FB(psb)+cch)
	}
}

// LQ_DWORD_DIV: unsigned 32-bit quotient and remainder; ErrorExit(27) on a zero divisor.
func rtLqDwordDiv(a, b uint32) (q, r uint32) {
	if b == 0 {
		ErrorExit(uErcLqDivideByZeroError)
		panic("LQ_DWORD_DIV: divide by zero")
	}
	return a / b, a % b
}

// LQ_DWORD_MUL: low 32 bits of the product.
func rtLqDwordMul(a, b uint32) uint32 { return a * b }

// Mul32(w1, w2) DWORD
func rtMul32(a, b uint16) uint32 { return uint32(a) * uint32(b) }

// Div32(dw, w) WORD / Mod32(dw, w) WORD: a quotient that does not fit is a divide error.
func rtDiv32(dw uint32, w uint16) uint16 {
	if w == 0 || dw/uint32(w) > 0xFFFF {
		panic("Div32: divide error")
	}
	return uint16(dw / uint32(w))
}
func rtMod32(dw uint32, w uint16) uint16 {
	if w == 0 || dw/uint32(w) > 0xFFFF {
		panic("Mod32: divide error")
	}
	return uint16(dw % uint32(w))
}

func init() {
	Mul32 = rtMul32
}
