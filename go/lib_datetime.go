package main

// lib_datetime.go - translation of ../src/DateTime.plm (image 362A-3CB8, frame 0362h,
// code at offset 0Ah).  A CTOS date/time is a DWORD: low word = seconds in the half day
// (0..43199), high word = dayTimes2 (half days since 1 March 1952).
// ExpDt (8 bytes): year WORD, month (0-based), day, dayOfWeek, hour, minute, second BYTE.
//
// Not translated: SetNow(dt) - it only calls the CTOS request SetDateTime, which the port
// does not provide (no part of Rats calls SetNow).

const dtHalfDay = 43200

func dtSecs(dt uint32) uint16  { return uint16(dt) }
func dtDays2(dt uint32) uint16 { return uint16(dt >> 16) }
func dtMake(secs, days2 uint16) uint32 {
	return uint32(days2)<<16 | uint32(secs)
}

// DtDiff(dt1, dt2): dt1 - dt2 in seconds (dt1 >= dt2).
func dtDtDiff(dt1, dt2 uint32) uint32 {
	s1, d1 := dtSecs(dt1), dtDays2(dt1)
	s2, d2 := dtSecs(dt2), dtDays2(dt2)
	var dwSec uint32
	if s1 >= s2 {
		dwSec = uint32(s1 - s2)
	} else {
		dwSec = uint32(s1 + dtHalfDay - s2) // WORD arithmetic
		d1--
	}
	return rtMul32(d1-d2, dtHalfDay) + dwSec
}

// DtSeconds(dt): dt as a seconds count.
func dtDtSeconds(dt uint32) uint32 {
	return rtMul32(dtDays2(dt), dtHalfDay) + uint32(dtSecs(dt))
}

// DtAddSeconds(dt, n)
func dtDtAddSeconds(dt uint32, n uint16) uint32 {
	d := dtDays2(dt)
	dw := uint32(dtSecs(dt)) + uint32(n)
	for dw >= dtHalfDay {
		dw -= dtHalfDay
		d++
	}
	return dtMake(uint16(dw), d)
}

// DtMidnight(dt): start of dt's day.
func dtDtMidnight(dt uint32) uint32 {
	return dtMake(0, dtDays2(dt)&0xFFFE)
}

// CompactDt(pExp): CompactDateTime; dayOfWeek 0FFh = unknown (tries 0, 1, ... until the
// day agrees).
func dtCompactDt(pExp Ptr) uint32 {
	defer Release(Mark())
	dt := Push(4)
	var erc uint16
	if FB(pExp.Add(4)) == 0xFF {
		SetFB(pExp.Add(4), 0)
		for {
			erc = rtCompactDateTime(pExp, DG(dt))
			if erc != uErcDayAndDateDisagree {
				break
			}
			SetFB(pExp.Add(4), FB(pExp.Add(4))+1)
		}
	} else {
		erc = rtCompactDateTime(pExp, DG(dt))
	}
	ceCheckErc(erc)
	return DW(dt)
}

// GetNow: GetDateTime.
func dtGetNow() uint32 {
	defer Release(Mark())
	dt := Push(4)
	ceCheckErc(GetDateTime(DG(dt)))
	return DW(dt)
}

// DtSubSeconds(dt, n)
func dtDtSubSeconds(dt uint32, n uint16) uint32 {
	d := dtDays2(dt)
	dw := uint32(dtSecs(dt))
	for uint32(n) > dw {
		dw += dtHalfDay
		d--
	}
	return dtMake(uint16(dw-uint32(n)), d)
}

func dtExpandDt(pExpRet Ptr, dt uint32) {
	ceCheckErc(rtExpandDateTime(dt, pExpRet))
}

// ParseDateTime(pDtRet, pb, cb): parse "mm/dd[/yy] [hh:mm[:ss]]"; the missing date part is
// today, a missing time part 00:00:00.  0 on a syntax error, else 0FFh.
func dtParseDateTime(pDtRet, pb Ptr, cb uint16) byte {
	defer Release(Mark())
	expNow := Push(8)
	exp := Push(8)
	pi := Push(2)
	bYear := Push(1)
	const (
		eYear, eMonth, eDay, eDow, eHour, eMin, eSec = 0, 2, 3, 4, 5, 6, 7
	)
	dtExpandDt(DG(expNow), dtGetNow())
	cbDate := FINDB(pb, ' ', cb)
	if cbDate == 0xFFFF {
		if FINDB(pb, '/', cb) == 0xFFFF {
			cbDate = 0
		} else {
			cbDate = cb
		}
	}
	SetW(pi, 0)
	if cbDate > W(pi) {
		if !Test(dtParseField(exp+eMonth, pb, cbDate, pi, '/', 0, 1, 13)) {
			return 0
		}
		SetB(exp+eMonth, B(exp+eMonth)-1)
		if !Test(dtParseField(exp+eDay, pb, cbDate, pi, '/', 0xFF, 1, 32)) {
			return 0
		}
		if cbDate > W(pi) {
			if !Test(dtParseField(bYear, pb, cbDate, pi, 0, 0xFF, 0, 100)) {
				return 0
			}
			SetW(exp+eYear, uint16(B(bYear))+1900)
		} else {
			SetW(exp+eYear, W(expNow+eYear))
		}
		if cb > cbDate {
			SetW(pi, W(pi)+1)
		}
	} else {
		SetB(exp+eMonth, B(expNow+eMonth))
		SetB(exp+eDay, B(expNow+eDay))
		SetW(exp+eYear, W(expNow+eYear))
	}
	if cb > W(pi) {
		if !Test(dtParseField(exp+eHour, pb, cb, pi, ':', 0, 0, 24)) {
			return 0
		}
		if !Test(dtParseField(exp+eMin, pb, cb, pi, ':', 0xFF, 0, 60)) {
			return 0
		}
		if cb > W(pi) {
			if !Test(dtParseField(exp+eSec, pb, cb, pi, 0, 0xFF, 0, 60)) {
				return 0
			}
		} else {
			SetB(exp+eSec, 0)
		}
	} else {
		SetB(exp+eHour, 0)
		SetB(exp+eMin, 0)
		SetB(exp+eSec, 0)
	}
	SetB(exp+eDow, 0xFF)
	SetFDW(pDtRet, dtCompactDt(DG(exp)))
	return 0xFF
}

func dtParseDateTimeSb(pDtRet, psb Ptr) byte {
	return dtParseDateTime(pDtRet, psb.Add(1), uint16(FB(psb)))
}

// FmtDateTime(psbRet, dt, psbTemplate): FormatDateTime.
func dtFmtDateTime(psbRet Ptr, dt uint32, psbTemplate Ptr) {
	ceCheckErc(rtFormatDateTime(dt, psbTemplate, psbRet))
}

// DtAddHours(dt, h): dt + h hours for h < 12, else dt - (24-h) hours.
func dtDtAddHours(dt uint32, h uint16) uint32 {
	if h < 12 {
		return dtDtAddSeconds(dt, h*3600)
	}
	return dtDtSubSeconds(dt, (24-h)*3600)
}

// FmtDate / FmtDateAndTime / FmtTime: "mm/dd/yy", "mm/dd/yy hh:mm:ss", "hh:mm:ss".
func dtFmtDate(psbRet Ptr, dt uint32)        { dtFmtDt(psbRet, dt, 0xFF, 0) }
func dtFmtDateAndTime(psbRet Ptr, dt uint32) { dtFmtDt(psbRet, dt, 0xFF, 0xFF) }
func dtFmtTime(psbRet Ptr, dt uint32)        { dtFmtDt(psbRet, dt, 0, 0xFF) }

// ParseField(pbRet, pb, cb, pi, chSep, fLast, min, limit): parse the number at pb(i) up to
// chSep (or the end if chSep = 0, or if fLast and there is no chSep); must be
// min <= n < limit; stores it as a BYTE at pbRet and advances the WORD i at pi.
func dtParseField(pbRet uint16, pb Ptr, cb uint16, pi uint16, chSep, fLast, min, limit byte) byte {
	defer Release(Mark())
	w := Push(2)
	var cbField uint16
	if chSep == 0 {
		cbField = cb - W(pi)
	} else {
		cbField = FINDB(pb.Add(W(pi)), chSep, cb-W(pi))
		if cbField == 0xFFFF {
			if Test(fLast) {
				chSep = 0
				cbField = cb - W(pi)
			} else {
				return 0
			}
		}
	}
	if !Test(strFWFromRgch(w, pb.Add(W(pi)), cbField)) {
		return 0
	}
	if W(w) < uint16(min) || W(w) >= uint16(limit) {
		return 0
	}
	SetB(pbRet, byte(W(w)))
	SetW(pi, W(pi)+cbField)
	if chSep != 0 {
		SetW(pi, W(pi)+1)
	}
	return 0xFF
}

// FmtDt(psbRet, dt, fDate, fTime): the formatter behind FmtDate...
func dtFmtDt(psbRet Ptr, dt uint32, fDate, fTime byte) {
	defer Release(Mark())
	exp := Push(8)
	sbNum := DG(Push(256))
	dtExpandDt(DG(exp), dt)
	SetFB(psbRet, 0)
	if Test(fDate) {
		strSbFromNumber(sbNum, uint32(B(exp+2)+1), 2, '0')
		strAppendSb(psbRet, sbNum)
		strAppendCh(psbRet, '/')
		strSbFromNumber(sbNum, uint32(B(exp+3)), 2, '0')
		strAppendSb(psbRet, sbNum)
		strAppendCh(psbRet, '/')
		strSbFromNumber(sbNum, uint32(W(exp)-1900), 2, '0')
		strAppendSb(psbRet, sbNum)
		if Test(fTime) {
			strAppendCh(psbRet, ' ')
		}
	}
	if Test(fTime) {
		strSbFromNumber(sbNum, uint32(B(exp+5)), 2, '0')
		strAppendSb(psbRet, sbNum)
		strAppendCh(psbRet, ':')
		strSbFromNumber(sbNum, uint32(B(exp+6)), 2, '0')
		strAppendSb(psbRet, sbNum)
		strAppendCh(psbRet, ':')
		strSbFromNumber(sbNum, uint32(B(exp+7)), 2, '0')
		strAppendSb(psbRet, sbNum)
	}
}

func init() {
	GetNow = dtGetNow
}
