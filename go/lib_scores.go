package main

// lib_scores.go - translation of ../src/Scores.plm (image 2930-2FAF, frame 0293h):
// the high-score table.  The score file (name = DGROUP offset of a length-prefixed string,
// e.g. sbScoresFile at 2C67h) is one 512-byte page:
//    +0   cnt WORD               number of entries (0..10)
//    +2   recs(10) ScoreRec      26 bytes each, sorted by descending score
// ScoreRec: +0 score WORD, +2 date (CTOS date/time DWORD), +6 name(20) BYTE (blank padded).
// CONST at DGROUP 09EE (strings) and 0A92 (the score form descriptor).

const (
	ldgSbDateFmt    = 0x09EE // (29, '!oo!/!0d!/!0y! !hh!:!0m! !aa!')
	ldgSbAce        = 0x0A0C
	ldgSbEnterName  = 0x0A1B
	ldgSbBetterLuck = 0x0A41
	ldgSbPressGo    = 0x0A58
	ldgSTitle       = 0x0A6A // 40 chars
	ldgScoreForm    = 0x0A92

	scRecSize   = 26
	scBufSize   = 512
	scFieldSize = 16 // FieldType (16) BYTE = a forms StateType
	scNameLen   = 20

	uModeRead   = 0x6D72 // 'mr'
	uModeModify = 0x6D6D // 'mm'
)

func scGetHighScore(pFile uint16) uint16 {
	defer Release(Mark())
	buf := Push(scBufSize)
	scReadScores(buf, pFile)
	pRecs := buf + 2
	if W(buf) == 0 {
		return 0
	}
	return W(pRecs)
}

func scEnterHighScore(pFile, score, row uint16) {
	defer Release(Mark())
	buf := Push(scBufSize)
	fields := Push(10 * scFieldSize)
	ppAscb := Push(4)
	pch := Push(1)
	if row == 0x8000 {
		row = (W(ldgCScrLines) - 26) >> 1
	}
	scReadScores(buf, pFile)
	pRecs := buf + 2
	rec := func(i uint16) uint16 { return pRecs + i*scRecSize }
	i := byte(0)
	for uint16(i) < W(buf) && W(rec(uint16(i))) >= score {
		i++
	}
	if i < 10 {
		pRec := rec(uint16(i))
		if W(buf) < 10 {
			SetW(buf, W(buf)+1)
		}
		MOVRB(DG(pRec), DG(rec(uint16(i)+1)), (W(buf)-1-uint16(i))*scRecSize)
		SetW(pRec, score)
		SetDW(pRec+2, dtGetNow())
		ceCheckErc(GetpAscb(DG(ppAscb)))
		pAscb := P(ppAscb)
		cbName := FB(pAscb.Add(0x40))
		if cbName > scNameLen {
			cbName = scNameLen
		}
		MOVB(pAscb.Add(0x41), DG(pRec+6), uint16(cbName))
		SETB(' ', DG(pRec+6+uint16(cbName)), scNameLen-uint16(cbName))
	}
	scShowScores(fields, buf, row)
	if i < 10 {
		scShowMsg(0, ldgSbAce, row)
		scShowMsg(1, ldgSbEnterName, row)
		if Test(frmFormEdit(rec(uint16(i)), fields+uint16(i)*scFieldSize)) {
			scWriteScores(buf, pFile)
		}
	} else {
		scShowMsg(0, ldgSbBetterLuck, row)
		scShowMsg(1, ldgSbPressGo, row)
		for {
			ceCheckErc(ReadKbd(DG(pch)))
			ch := B(pch)
			if ch == 0x1B || ch == 7 || ch == 4 {
				break
			}
			Beep()
		}
	}
	scClearScores(fields, buf, row)
}

func scShowScores(pFields, pBuf, row uint16) {
	pRecs := pBuf + 2
	ClearRect(20, row, 40, 26)
	VidPutChars(20, row, 40, DG(ldgSTitle), 5)
	for i := uint16(0); i < W(pBuf); i++ {
		frmFormOpenShow(pFields+i*scFieldSize, ldgScoreForm, 0, 20, row+2+i*2, pRecs+i*scRecSize)
	}
	VidFillCells(20, row+2+20+3, 40, 5<<8)
}

func scShowMsg(line, pSb, row uint16) {
	cb := uint16(B(pSb))
	if cb > 40 {
		cb = 40
	}
	col := (40 - cb) >> 1
	VidPutChars(col+20, row+2+20+line, cb, DG(pSb+1), 0)
}

func scClearScores(pFields, pBuf, row uint16) {
	VidFillCells(20, row, 40, 0)
	for i := uint16(0); i < W(pBuf); i++ {
		frmFormClose(pFields + i*scFieldSize)
	}
	VidFillCells(20, row+2+20, 40, 0)
	VidFillCells(20, row+2+20+1, 40, 0)
	VidFillCells(20, row+2+20+3, 40, 0)
}

// form field procedures (referenced from scoreForm)

// FmtDate(pSbOut, pDate) WORD: its result is whatever FmtDateTime left in AX (unused).
func scFmtDate(pSbOut, pDate uint16) uint16 {
	dtFmtDateTime(DG(pSbOut), DW(pDate), DG(ldgSbDateFmt))
	return 0
}

func scParseDate(pDate, pSb uint16) uint16 { return 0 }

func scFmtName(pSbOut, pName uint16) {
	SetB(pSbOut, byte(SKIPRB(DG(pName), ' ', scNameLen)+1))
	MOVB(DG(pName), DG(pSbOut+1), uint16(B(pSbOut)))
}

func scParseName(pName, pSb uint16) uint16 {
	MOVB(DG(pSb+1), DG(pName), uint16(B(pSb)))
	SETB(' ', DG(pName+uint16(B(pSb))), scNameLen-uint16(B(pSb)))
	return 0
}

func scReadScores(pBuf, pName uint16) {
	defer Release(Mark())
	pfh := Push(2)
	pcbRead := Push(2)
	var erc uint16
	for {
		erc = OpenFile(DG(pfh), DG(pName+1), uint16(B(pName)), 0, 0, uModeRead)
		if erc == 0xD7 { // no such volume
			SetW(pBuf, 0)
			return
		} else if erc == 0xCB { // no such file
			SetW(pBuf, 0)
			return
		} else if erc == 0xDC { // file in use
			continue
		}
		ceCheckErc(erc)
		break
	}
	erc = ReadFile(W(pfh), DG(pBuf), scBufSize, 0, DG(pcbRead))
	if erc == 2 { // end of file
		SetW(pBuf, 0)
	} else {
		ceCheckErc(erc)
	}
	ceCheckErc(CloseFile(W(pfh)))
}

func scWriteScores(pBuf, pName uint16) {
	defer Release(Mark())
	pfh := Push(2)
	pcbWritten := Push(2)
	lfaSize := uint32(512)
	var erc uint16
	for {
		erc = OpenFile(DG(pfh), DG(pName+1), uint16(B(pName)), 0, 0, uModeModify)
		if erc == 0xD7 {
			return
		} else if erc == 0xCB {
			ceCheckErc(CreateFile(DG(pName+1), uint16(B(pName)), 0, 0, lfaSize))
			continue
		} else if erc == 0xDC {
			continue
		}
		ceCheckErc(erc)
		break
	}
	for {
		erc = WriteFile(W(pfh), DG(pBuf), scBufSize, 0, DG(pcbWritten))
		if erc == 2 {
			ceCheckErc(ChangeFileLength(W(pfh), lfaSize))
			continue
		}
		ceCheckErc(erc)
		break
	}
	ceCheckErc(CloseFile(W(pfh)))
}

func init() {
	GetHighScore = scGetHighScore
	EnterHighScore = scEnterHighScore
	RegisterFar(0x0293, 0x0429, scFmtDate)
	RegisterFar(0x0293, 0x0447, scParseDate)
	RegisterFar(0x0293, 0x0451, scFmtName)
	RegisterFar(0x0293, 0x0485, scParseName)
}
