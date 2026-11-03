package main

// lib_forms.go - translation of ../src/Forms.plm (image 3D1C-5286, frame 03D1h):
// the small full-screen forms (dialog) runtime used for the options form (DGROUP 0680h)
// and the score table (0A92h).  All descriptors, records and states are DGROUP offsets.
//
// FormType : cols +0, rows +1, msgCol +2, msgRow +3, msgWidth +4, msgAttr +5, msgAlign +6,
//            b7 +7, cbText WORD +8, cbFields WORD +0Ah, rgb +0Ch (text lines, then fields)
// StateType: pForm +0, frame +2, col +4, row +6, absCol +8, absRow +0Ah, pBuf POINTER +0Ch
// FieldType: pFmt POINTER +0, pParse POINTER +4, off WORD +8, col +0Ah, row +0Bh, width +0Ch,
//            fEdit +0Dh, fDefault +0Eh, kind +0Fh, align +10h, cb +11h, rgb +12h
// ChoiceType (5): col, width, cbGap, key, value
// LineType : attr, cb, rgb (attr 0FFh: cb blanks, no chars; cb 0: new line)
// vid (BASED pVidCfg, the CTOS video structure): cFrames +0Ah, frames (20 bytes each) +20h;
// FrameType: row0 +0, col0 +1, rows +2, cols +3, curRow +0Ah, curCol +0Bh.

const (
	ldgRgbKeyClass = 0x0AFE // (256) BYTE keyboard code -> edit action
	ldgSbErc31000  = 0x0BFE
	ldgSbErc31001  = 0x0C13
	ldgSbErc31002  = 0x0C29
	ldgSbErc31003  = 0x0C48
	ldgSbErc31004  = 0x0C57
	ldgSbErc31005  = 0x0C6B
	ldgPVidCfg     = 0x2F1E
	ldgCScrCols    = 0x2F22
	ldgCScrLines   = 0x2F24
)

// form / state / field / choice field offsets
const (
	uFmCols, uFmRows, uFmMsgCol, uFmMsgRow, uFmMsgWidth, uFmMsgAttr, uFmMsgAlign = 0, 1, 2, 3, 4, 5, 6
	uFmCbText, uFmCbFields, uFmRgb                                               = 8, 0x0A, 0x0C

	uStPForm, uStFrame, uStCol, uStRow, uStAbsCol, uStAbsRow, uStPBuf = 0, 2, 4, 6, 8, 0x0A, 0x0C
	uStSize                                                           = 16

	uFdPFmt, uFdPParse, uFdOff, uFdCol, uFdRow, uFdWidth = 0, 4, 8, 0x0A, 0x0B, 0x0C
	uFdFEdit, uFdFDefault, uFdKind, uFdAlign, uFdCb      = 0x0D, 0x0E, 0x0F, 0x10, 0x11
	uFdRgb                                               = 0x12

	uChCol, uChWidth, uChCbGap, uChKey, uChValue = 0, 1, 2, 3, 4

	uFrRow0, uFrCol0, uFrRows, uFrCols, uFrCurRow, uFrCurCol = 0, 1, 2, 3, 0x0A, 0x0B
	uVidCFrames, uVidRgbFrames                               = 0x0A, 0x20
)

// CTOS LockVideo / UnLockVideo: the port's video calls are atomic already.
func frmLockVideo()   {}
func frmUnLockVideo() {}

// CTOS PosFrameCursor (platform).
func frmPosFrameCursor(iFrame, iCol, iLine uint16) uint16 {
	if PosFrameCursor == nil {
		return 0
	}
	return PosFrameCursor(iFrame, iCol, iLine)
}

func frmVid() Ptr { return P(ldgPVidCfg) }
func frmFrame(frame uint16) Ptr {
	return frmVid().Add(uVidRgbFrames + frame*20) // @vid.rgbFrames(frame * 20)
}

// call fld.pFmt(pOut, pIn) (direct call through the far pointer)
func frmCallFmt(pProc Ptr, pOut, pIn uint16) {
	switch f := FarFn(pProc).(type) {
	case func(uint16, uint16):
		f(pOut, pIn)
	case func(uint16, uint16) uint16:
		f(pOut, pIn)
	default:
		panic("form field format proc: bad type")
	}
}

// ------------------------------------------------------------------

// DoFormEx(pForm, frame, col, row, pValidate, pRec, pInit): open, show pInit, edit until GO
// and pValidate(.state, .rec) returns 0 (else show its error and edit again), close.
func frmDoFormEx(pForm, frame, col, row uint16, pValidate Ptr, pRec, pInit uint16) {
	defer Release(Mark())
	state := Push(uStSize)
	frmFormOpen(state, pForm, frame, col, row)
	if pInit != 0xFFFF {
		frmFormShowValues(state, pInit)
	}
	for Test(frmFormEdit(pRec, state)) {
		erc := thkCallProc(state, pRec, pValidate)
		if erc == 0 {
			break
		}
		frmFormShowErc(state, erc, B(pForm+uFmMsgAttr))
	}
	frmFormClose(state)
}

// DoForm(pRec, pForm, frame, col, row, pInit): open, show pInit, edit once; TRUE if GO.
func frmDoForm(pRec, pForm, frame, col, row, pInit uint16) byte {
	defer Release(Mark())
	state := Push(uStSize)
	frmFormOpen(state, pForm, frame, col, row)
	if pInit != 0xFFFF {
		frmFormShowValues(state, pInit)
	}
	fOk := frmFormEdit(pRec, state)
	frmFormClose(state)
	return fOk
}

func frmFormOpenShow(pState, pForm, frame, col, row, pRec uint16) {
	frmFormOpen(pState, pForm, frame, col, row)
	frmFormShowValues(pState, pRec)
}

// FormOpen(pState, pForm, frame, col, row): centre (0FFh), save the screen under the form,
// draw the static text.
func frmFormOpen(pState, pForm, frame, col, row uint16) {
	defer Release(Mark())
	ppBuf := Push(4) // pBuf (address passed to MemAlloc)
	var x0, y0, cxMax, cyMax uint16
	if frame == 0xFFFF {
		y0, x0 = 0, 0
		cxMax = W(ldgCScrCols)
		cyMax = W(ldgCScrLines)
	} else {
		if uint16(FB(frmVid().Add(uVidCFrames))) <= frame {
			ceCheckErc(500)
		}
		pFrame := frmFrame(frame)
		x0 = uint16(FB(pFrame.Add(uFrCol0)))
		y0 = uint16(FB(pFrame.Add(uFrRow0)))
		cxMax = uint16(FB(pFrame.Add(uFrCols)))
		cyMax = uint16(FB(pFrame.Add(uFrRows)))
	}
	cols, rows := uint16(B(pForm+uFmCols)), uint16(B(pForm+uFmRows))
	if col == 0xFF {
		col = (cxMax - cols) >> 1
	}
	if row == 0xFF {
		row = (cyMax - rows) >> 1
	}
	if int16(col) < 0 || col+cols > cxMax || int16(row) < 0 || row+rows > cyMax {
		ceCheckErc(500)
	}
	absCol := col + x0
	absRow := row + y0
	ceCheckErc(heapMemAlloc(DG(ppBuf), (cols*rows)<<1))
	pBuf := P(ppBuf)
	SetW(pState+uStPForm, pForm)
	SetW(pState+uStFrame, frame)
	SetW(pState+uStCol, col)
	SetW(pState+uStRow, row)
	SetW(pState+uStAbsCol, absCol)
	SetW(pState+uStAbsRow, absRow)
	SetP(pState+uStPBuf, pBuf)
	frmLockVideo()
	var i, rowT, colT uint16
	for i < W(pForm+uFmCbText) {
		pLine := pForm + uFmRgb + i
		cb := B(pLine + 1)
		cbStep := uint16(cb + 2) // BYTE arithmetic
		if cb != 0 {
			VidReadCells(pBuf.Add(2*(cols*rowT+colT)), absCol+colT, absRow+rowT, uint16(cb))
			if B(pLine) == 0xFF {
				VidFillCells(absCol+colT, absRow+rowT, uint16(cb), 0)
				cbStep = 2
			} else {
				VidPutChars(absCol+colT, absRow+rowT, uint16(cb), DG(pLine+2), B(pLine))
			}
			colT = colT + uint16(cb)
		} else {
			colT = 0
			rowT = rowT + 1
		}
		i = i + cbStep
	}
	frmUnLockVideo()
}

// FormShowValues(pState, pRec): format every field from the record.
func frmFormShowValues(pState, pRec uint16) {
	pForm := W(pState + uStPForm)
	i := uint16(0)
	for i < W(pForm+uFmCbFields) {
		pFld := pForm + uFmRgb + W(pForm+uFmCbText) + i
		frmFieldShow(pState, pFld, pRec)
		i = i + frmFieldSize(pFld)
	}
}

// FormEdit(pRec, pState): edit + parse all fields (until no error); TRUE if GO.
func frmFormEdit(pRec, pState uint16) byte {
	defer Release(Mark())
	ppFld := Push(2)
	SetW(ppFld, 0xFFFF)
	erc := uint16(0)
	for {
		if !Test(frmFormEditLoop(pState, W(ppFld), erc)) {
			return FALSE
		}
		if erc = frmFormParseAll(pRec, pState, ppFld); erc == 0 {
			return TRUE
		}
	}
}

// FormShowErc(pState, erc, attr): error message in the message line.
func frmFormShowErc(pState, erc uint16, attr byte) {
	defer Release(Mark())
	sb := Push(256)
	frmGetErcText(DG(sb), erc)
	frmFormShowMsg(pState, sb, attr)
	if erc != 0 {
		Beep()
	}
}

// FormShowMsg(pState, psb, attr): message line (attr 0FFh = form's, psb 0FFFFh = empty).
func frmFormShowMsg(pState, psb uint16, attr byte) {
	defer Release(Mark())
	Push(10) // unused (5) WORD
	pForm := W(pState + uStPForm)
	if attr == 0xFF {
		attr = B(pForm + uFmMsgAttr)
	}
	if psb == 0xFFFF {
		psb = ldgSbNull
	}
	frmDrawAligned(uint16(B(pForm+uFmMsgCol))+W(pState+uStAbsCol),
		uint16(B(pForm+uFmMsgRow))+W(pState+uStAbsRow),
		uint16(B(pForm+uFmMsgWidth)), B(pForm+uFmMsgAlign), psb, attr, 0xFFFF)
}

// FormClose(pState): restore the screen, free the buffer.
func frmFormClose(pState uint16) {
	pForm := W(pState + uStPForm)
	absCol := W(pState + uStAbsCol)
	absRow := W(pState + uStAbsRow)
	pBuf := P(pState + uStPBuf)
	cols := uint16(B(pForm + uFmCols))
	frmLockVideo()
	var i, rowT, colT uint16
	for i < W(pForm+uFmCbText) {
		pLine := pForm + uFmRgb + i
		cb := B(pLine + 1)
		cbStep := uint16(cb + 2) // BYTE arithmetic
		if cb != 0 {
			if B(pLine) == 0xFF {
				cbStep = 2
			}
			VidPutCells(absCol+colT, absRow+rowT, uint16(cb), pBuf.Add(2*(cols*rowT+colT)))
			colT = colT + uint16(cb)
		} else {
			colT = 0
			rowT = rowT + 1
		}
		i = i + cbStep
	}
	frmUnLockVideo()
	heapMemFree(P(pState + uStPBuf))
}

// FormParseAll(pRec, pState, ppFldRet): parse all fields into pRec; on an error the field
// is stored at ppFldRet and the erc returned.
func frmFormParseAll(pRec, pState, ppFldRet uint16) uint16 {
	pForm := W(pState + uStPForm)
	i := uint16(0)
	for i < W(pForm+uFmCbFields) {
		pFld := pForm + uFmRgb + W(pForm+uFmCbText) + i
		if erc := frmFieldParse(pRec, pState, pFld); erc != 0 {
			SetW(ppFldRet, pFld)
			return erc
		}
		i = i + frmFieldSize(pFld)
	}
	return 0
}

// FormEditLoop(pState, pFld, erc): keyboard loop; pFld = current field (0FFFFh: first
// editable field).  TRUE if left with GO.
func frmFormEditLoop(pState, pFld, erc uint16) byte {
	defer Release(Mark())
	// locals whose address is taken
	px := Push(2)
	py := Push(2)
	sb := Push(256)
	ppad := Push(2)
	pi1 := Push(2)
	pi2 := Push(2)
	pi3 := Push(2)
	pi4 := Push(2)
	pi5 := Push(2)
	pch := Push(1)
	var frCol, frRow, pNew uint16
	var fRedraw, fCursor, fFind, fOver byte

	x := func() uint16 { return W(px) }
	y := func() uint16 { return W(py) }
	pad := func() uint16 { return W(ppad) }
	fcol := func() uint16 { return uint16(B(pFld + uFdCol)) }
	kind := func() byte { return B(pFld + uFdKind) }

	pForm := W(pState + uStPForm)
	if erc != 0 {
		frmFormShowErc(pState, erc, B(pForm+uFmMsgAttr))
	}
	frame := W(pState + uStFrame)
	if frame != 0xFFFF {
		pFrame := frmFrame(frame)
		frCol = uint16(FB(pFrame.Add(uFrCurCol)))
		frRow = uint16(FB(pFrame.Add(uFrCurRow)))
	}
	if pFld == 0xFFFF {
		pFld = frmNextField(pForm, 0, 0)
		if pFld == 0xFFFF {
			return FALSE
		}
	}
	frmFieldHome(px, py, pState, pFld)
	if W(pState+uStFrame) != 0xFFFF {
		ceCheckErc(frmPosFrameCursor(W(pState+uStFrame), x()+W(pState+uStCol), y()+W(pState+uStRow)))
	}
	frmFieldHighlight(pState, pFld, TRUE)
	if kind() == 0 {
		frmFieldReadScreen(sb, pState, pFld, ppad)
	}
	fFind, fCursor, fRedraw = FALSE, FALSE, FALSE
	fOver = ledFLedOn(7)
	formCols := uint16(B(pForm + uFmCols))
	formRows := uint16(B(pForm + uFmRows))
	for {
		ceCheckErc(ReadKbd(DG(pch)))
		ch := B(pch)
		if erc != 0 {
			frmFormShowErc(pState, 0, 0)
			erc = 0
		}
		beep := false
		switch B(ldgRgbKeyClass + uint16(ch)) {
		case 0:
			beep = true
		case 1: // left
			if x() != 0 {
				SetW(px, x()-1)
				fFind, fCursor = TRUE, TRUE
			} else {
				beep = true
			}
		case 2: // right
			if x() != uint16(byte(formCols-1)) {
				SetW(px, x()+1)
				fFind, fCursor = TRUE, TRUE
			} else {
				beep = true
			}
		case 3: // up
			if y() != 0 {
				SetW(py, y()-1)
				fFind, fCursor = TRUE, TRUE
			} else {
				beep = true
			}
		case 4: // down
			if y() != uint16(byte(formRows-1)) {
				SetW(py, y()+1)
				fFind, fCursor = TRUE, TRUE
			} else {
				beep = true
			}
		case 5: // character
			if pFld == 0xFFFF {
				beep = true
				break
			}
			switch kind() {
			case 0:
				SetW(pi1, x()-(fcol()+pad()))
				if !Test(frmSbInsertCh(sb, pi1, uint16(B(pFld+uFdWidth)), ch, fOver)) {
					beep = true
					break
				}
				SetW(px, fcol()+pad()+W(pi1))
				fRedraw, fCursor = TRUE, TRUE
			case 1:
				if !Test(frmChoiceSelect(pState, pFld, ch, TRUE)) {
					beep = true
				}
			}
		case 6: // backspace
			if pFld == 0xFFFF {
				beep = true
				break
			}
			switch kind() {
			case 0:
				SetW(pi2, x()-(fcol()+pad()))
				if !Test(frmSbDeleteLeft(sb, pi2, fOver)) {
					beep = true
					break
				}
				SetW(px, fcol()+pad()+W(pi2))
				fRedraw, fCursor = TRUE, TRUE
			case 1:
				beep = true
			}
		case 7: // delete word
			if pFld == 0xFFFF {
				beep = true
				break
			}
			switch kind() {
			case 0:
				SetW(pi3, x()-(fcol()+pad()))
				if !Test(frmSbDeleteWord(sb, pi3, fOver)) {
					beep = true
					break
				}
				SetW(px, fcol()+pad()+W(pi3))
				fRedraw, fCursor = TRUE, TRUE
			case 1:
				beep = true
			}
		case 8: // delete
			if pFld == 0xFFFF {
				beep = true
				break
			}
			switch kind() {
			case 0:
				SetW(pi4, x()-(fcol()+pad()))
				if !Test(frmSbDeleteCh(sb, pi4, fOver)) {
					beep = true
					break
				}
				SetW(px, fcol()+pad()+W(pi4))
				fRedraw, fCursor = TRUE, TRUE
			case 1:
				beep = true
			}
		case 9: // overtype toggle
			fOver = ^fOver
			ledSetLed(7, fOver)
		case 10: // next field
			frmFieldHome(px, py, pState, frmNextField(pForm, x(), y()))
			fFind, fCursor = TRUE, TRUE
		case 11, 12: // GO / CANCEL
			goto done
		case 13: // start of field
			if pFld == 0xFFFF {
				SetW(px, 0)
				fFind = TRUE
			} else if kind() == 0 {
				SetW(px, fcol()+pad())
			} else {
				SetW(px, fcol())
			}
			fCursor = TRUE
		case 14: // end of field
			if pFld == 0xFFFF {
				SetW(px, uint16(byte(formCols-1)))
				fFind = TRUE
			} else if kind() == 0 {
				SetW(px, fcol()+pad()+uint16(B(sb)))
			} else {
				SetW(px, uint16(B(pFld+uFdCol)+B(pFld+uFdWidth))) // BYTE arithmetic
			}
			fCursor = TRUE
		case 15: // top
			SetW(py, 0)
			fFind, fCursor = TRUE, TRUE
		case 16: // bottom
			SetW(py, uint16(byte(formRows-1)))
			fFind, fCursor = TRUE, TRUE
		}
		if beep {
			Beep()
			continue
		}
		if Test(fRedraw) {
			SetW(pi5, x()-(fcol()+pad()))
			frmFieldDrawText(pState, pFld, sb, TRUE, ppad)
			SetW(px, fcol()+pad()+W(pi5))
			fRedraw = FALSE
		}
		if Test(fCursor) {
			if W(pState+uStFrame) != 0xFFFF {
				ceCheckErc(frmPosFrameCursor(W(pState+uStFrame), x()+W(pState+uStCol), y()+W(pState+uStRow)))
			}
			fCursor = FALSE
		}
		if Test(fFind) {
			pNew = frmFieldAt(pForm, x(), y())
			if pFld != pNew {
				if pFld != 0xFFFF {
					frmFieldHighlight(pState, pFld, FALSE)
				}
				pFld = pNew
				if pFld != 0xFFFF {
					frmFieldHighlight(pState, pFld, TRUE)
					if kind() == 0 {
						frmFieldReadScreen(sb, pState, pFld, ppad)
					}
				}
			}
			fFind = FALSE
		}
	}
done:
	if pFld != 0xFFFF {
		frmFieldHighlight(pState, pFld, FALSE)
	}
	if W(pState+uStFrame) != 0xFFFF {
		ceCheckErc(frmPosFrameCursor(W(pState+uStFrame), frCol, frRow))
	}
	return Bool(B(pch) == 0x1B)
}

// FieldAt(pForm, x, y): editable field containing (x, y), or 0FFFFh.
func frmFieldAt(pForm, x, y uint16) uint16 {
	i := uint16(0)
	for i < W(pForm+uFmCbFields) {
		pFld := pForm + uFmRgb + W(pForm+uFmCbText) + i
		col := B(pFld + uFdCol)
		if uint16(col) <= x && uint16(col+B(pFld+uFdWidth)) >= x && uint16(B(pFld+uFdRow)) == y &&
			Test(B(pFld+uFdFEdit)) {
			return pFld
		}
		i = i + frmFieldSize(pFld)
	}
	return 0xFFFF
}

// NextField(pForm, x, y): first editable field after (x, y), wrapping around; 0FFFFh if none.
func frmNextField(pForm, x, y uint16) uint16 {
	for {
		i := uint16(0)
		for i < W(pForm+uFmCbFields) {
			pFld := pForm + uFmRgb + W(pForm+uFmCbText) + i
			if Test(B(pFld + uFdFEdit)) {
				row := uint16(B(pFld + uFdRow))
				if row > y || (row == y && uint16(B(pFld+uFdCol)) > x) {
					return pFld
				}
			}
			i = i + frmFieldSize(pFld)
		}
		if x == 0 && y == 0 {
			return 0xFFFF
		}
		y, x = 0, 0
	}
}

// SbInsertCh(psb, pi, width, ch, fOver)
func frmSbInsertCh(psb, pi, width uint16, ch, fOver byte) byte {
	i := func() uint16 { return W(pi) }
	cb := func() uint16 { return uint16(B(psb)) }
	if int16(i()) < 0 {
		SetW(pi, 0)
		fOver = FALSE
	} else if cb() <= i() {
		SetW(pi, cb())
		fOver = FALSE
	}
	if !Test(fOver) {
		if cb() >= width {
			return FALSE
		}
		MOVRB(DG(psb+i()+1), DG(psb+i()+2), cb()-i())
		SetB(psb, B(psb)+1)
	}
	SetB(psb+i()+1, ch)
	SetW(pi, i()+1)
	return TRUE
}

// SbDeleteLeft(psb, pi, fOver): backspace.
func frmSbDeleteLeft(psb, pi uint16, fOver byte) byte {
	if int16(B(psb)) < int16(W(pi)) {
		SetW(pi, uint16(B(psb)))
	}
	if int16(W(pi)) <= 0 {
		return FALSE
	}
	SetW(pi, W(pi)-1)
	if !Test(fOver) {
		SetB(psb, B(psb)-1)
		MOVB(DG(psb+W(pi)+2), DG(psb+W(pi)+1), uint16(B(psb))-W(pi))
	}
	return TRUE
}

// SbDeleteWord(psb, pi, fOver)
func frmSbDeleteWord(psb, pi uint16, fOver byte) byte {
	if int16(B(psb)) < int16(W(pi)) {
		SetW(pi, uint16(B(psb)))
	}
	if int16(W(pi)) <= 0 {
		return FALSE
	}
	j := SKIPRB(DG(psb+1), ' ', W(pi)) + 1
	j = FINDRB(DG(psb+1), ' ', j) + 1
	if !Test(fOver) {
		i := W(pi)
		MOVB(DG(psb+i+1), DG(psb+j+1), uint16(B(psb))-i)
		SetB(psb, byte(uint16(B(psb))-(i-j)))
	}
	SetW(pi, j)
	return TRUE
}

// SbDeleteCh(psb, pi, fOver)
func frmSbDeleteCh(psb, pi uint16, fOver byte) byte {
	_ = fOver
	i := W(pi)
	if int16(i) < 0 || uint16(B(psb)) <= i {
		return FALSE
	}
	SetB(psb, B(psb)-1)
	MOVB(DG(psb+i+2), DG(psb+i+1), uint16(B(psb))-i)
	return TRUE
}

// FieldShow(pState, pFld, pRec)
func frmFieldShow(pState, pFld, pRec uint16) {
	defer Release(Mark())
	sb := Push(256)
	b := Push(1)
	switch B(pFld + uFdKind) {
	case 0:
		frmCallFmt(P(pFld+uFdPFmt), sb, pRec+W(pFld+uFdOff))
		frmFieldDrawText(pState, pFld, sb, FALSE, 0xFFFF)
	case 1:
		frmCallFmt(P(pFld+uFdPFmt), b, pRec+W(pFld+uFdOff))
		frmChoiceShow(pState, pFld, B(b), FALSE)
	}
}

// FieldParse(pRec, pState, pFld): erc of the field's parse proc (31000/31001 if empty).
func frmFieldParse(pRec, pState, pFld uint16) uint16 {
	defer Release(Mark())
	sb := Push(256)
	b := Push(1)
	var pb uint16
	switch B(pFld + uFdKind) {
	case 0:
		frmFieldReadScreen(sb, pState, pFld, 0xFFFF)
		if B(sb) == 0 {
			if Test(B(pFld + uFdFDefault)) {
				strCopySb(DG(sb), DG(pFld+uFdCb))
			} else {
				return 31000
			}
		}
		pb = sb
	case 1:
		SetB(b, frmChoiceGetValue(pState, pFld))
		if B(b) == 0xFF {
			if Test(B(pFld + uFdFDefault)) {
				SetB(b, B(pFld+uFdAlign))
			} else {
				return 31001
			}
		}
		pb = b
	}
	return thkCallProc(pRec+W(pFld+uFdOff), pb, P(pFld+uFdPParse))
}

// FieldHome(pX, pY, pState, pFld): cursor position at the start of the field's contents.
func frmFieldHome(pX, pY, pState, pFld uint16) {
	defer Release(Mark())
	sb := Push(256)
	ppad := Push(2)
	switch B(pFld + uFdKind) {
	case 0:
		frmFieldReadScreen(sb, pState, pFld, ppad)
		SetW(pX, uint16(B(pFld+uFdCol))+W(ppad)+uint16(B(sb)))
	case 1:
		SetW(pX, uint16(B(pFld+uFdCol)))
	}
	SetW(pY, uint16(B(pFld+uFdRow)))
}

// FieldHighlight(pState, pFld, fOn)
func frmFieldHighlight(pState, pFld uint16, fOn byte) {
	var attr byte
	frmLockVideo()
	switch B(pFld + uFdKind) {
	case 0:
		col := uint16(B(pFld+uFdCol)) + W(pState+uStAbsCol)
		row := uint16(B(pFld+uFdRow)) + W(pState+uStAbsRow)
		cnt := uint16(B(pFld + uFdWidth))
		if Test(fOn) {
			attr = 5
		} else {
			attr = 0
		}
		VidFillAttr(col, row, cnt, attr)
	case 1:
		for i := uint16(0); i < uint16(B(pFld+uFdCb)); i++ {
			pChc := pFld + uFdRgb + i*5
			col := uint16(B(pChc+uChCol)) + W(pState+uStAbsCol)
			row := uint16(B(pFld+uFdRow)) + W(pState+uStAbsRow)
			cnt := uint16(B(pChc + uChWidth))
			wAttr := VidGetCell(col, row)
			if byte(wAttr>>8)&1 == 0 {
				if Test(fOn) {
					attr = 4
				} else {
					attr = 0
				}
			} else {
				if Test(fOn) {
					attr = 5
				} else {
					attr = 1
				}
			}
			VidFillAttr(col, row, cnt, attr)
			if Test(fOn) {
				attr = 5
			} else {
				attr = 0
			}
			VidFillAttr(col+cnt, row, uint16(B(pChc+uChCbGap)), attr)
		}
	}
	frmUnLockVideo()
}

// FieldSize(pFld): size of the field descriptor.
func frmFieldSize(pFld uint16) uint16 {
	switch B(pFld + uFdKind) {
	case 0:
		return uint16(B(pFld+uFdCb) + 0x12) // BYTE arithmetic
	case 1:
		return uint16(B(pFld+uFdCb))*5 + 0x12
	}
	panic("FieldSize: bad field kind")
}

// FieldReadScreen(psb, pState, pFld, pPadRet): read the field's text from the screen into
// sb, strip leading/trailing empty (0) cells; *pPadRet = number of leading empty cells.
func frmFieldReadScreen(psb, pState, pFld, pPadRet uint16) {
	col := uint16(B(pFld+uFdCol)) + W(pState+uStAbsCol)
	row := uint16(B(pFld+uFdRow)) + W(pState+uStAbsRow)
	cnt := uint16(B(pFld + uFdWidth))
	frmLockVideo()
	VidReadChars(DG(psb+1), col, row, cnt)
	frmUnLockVideo()
	SetB(psb, byte(cnt))
	k := SKIPB(DG(psb+1), 0, uint16(B(psb)))
	if k == 0xFFFF {
		k = uint16(B(psb))
	}
	if k != 0 {
		SetB(psb, byte(uint16(B(psb))-k))
		MOVB(DG(psb+k+1), DG(psb+1), uint16(B(psb)))
	}
	if pPadRet != 0xFFFF {
		SetW(pPadRet, k)
	}
	SetB(psb, byte(SKIPRB(DG(psb+1), 0, uint16(B(psb)))+1))
}

// FieldDrawText(pState, pFld, psb, fHi, pPadRet)
func frmFieldDrawText(pState, pFld, psb uint16, fHi byte, pPadRet uint16) {
	var attr byte
	if Test(fHi) {
		attr = 5
	} else {
		attr = 0
	}
	frmDrawAligned(uint16(B(pFld+uFdCol))+W(pState+uStAbsCol), uint16(B(pFld+uFdRow))+W(pState+uStAbsRow),
		uint16(B(pFld+uFdWidth)), B(pFld+uFdAlign), psb, attr, pPadRet)
}

// DrawAligned(col, row, width, align, psb, attr, pPadRet): draw sb in a width-cell area:
// align 0 left, 1 right, 2 centred.
func frmDrawAligned(col, row, width uint16, align byte, psb uint16, attr byte, pPadRet uint16) {
	var padL, padR uint16
	cb := uint16(B(psb))
	if width < cb {
		cb = width
	}
	cbExtra := width - cb
	switch align {
	case 0:
		padL, padR = 0, cbExtra
	case 1:
		padL, padR = cbExtra, 0
	case 2:
		padL = cbExtra >> 1
		padR = cbExtra - padL
	default:
		panic("DrawAligned: bad alignment")
	}
	frmLockVideo()
	if padL != 0 {
		VidFillCells(col, row, padL, uint16(attr)<<8)
	}
	VidPutChars(col+padL, row, cb, DG(psb+1), attr)
	if padR != 0 {
		VidFillCells(col+padL+cb, row, padR, uint16(attr)<<8)
	}
	frmUnLockVideo()
	if pPadRet != 0xFFFF {
		SetW(pPadRet, padL)
	}
}

// ChoiceGetValue(pState, pFld): value of the choice shown selected, 0FFh if none.
func frmChoiceGetValue(pState, pFld uint16) byte {
	b := byte(0xFF)
	frmLockVideo()
	for i := uint16(0); i < uint16(B(pFld+uFdCb)); i++ {
		pChc := pFld + uFdRgb + i*5
		col := uint16(B(pChc+uChCol)) + W(pState+uStAbsCol)
		row := uint16(B(pFld+uFdRow)) + W(pState+uStAbsRow)
		wAttr := VidGetCell(col, row)
		if byte(wAttr>>8)&1 == 0 {
			b = B(pChc + uChValue)
			break
		}
	}
	frmUnLockVideo()
	return b
}

// ChoiceSelect(pState, pFld, ch, fOn): select the choice whose key is ch; FALSE if none.
func frmChoiceSelect(pState, pFld uint16, ch, fOn byte) byte {
	var attr byte
	frmLockVideo()
	fFound := FALSE
	pSel := uint16(0xFFFF)
	for i := uint16(0); i < uint16(B(pFld+uFdCb)); i++ {
		pChc := pFld + uFdRgb + i*5
		col := uint16(B(pChc+uChCol)) + W(pState+uStAbsCol)
		row := uint16(B(pFld+uFdRow)) + W(pState+uStAbsRow)
		cnt := uint16(B(pChc + uChWidth))
		if B(ldgRgbUpper+uint16(B(pChc+uChKey))) == B(ldgRgbUpper+uint16(ch)) {
			if Test(fOn) {
				attr = 4
			} else {
				attr = 0
			}
			fFound = TRUE
		} else {
			if Test(fOn) {
				attr = 5
			} else {
				attr = 1
			}
			wAttr := VidGetCell(col, row)
			if byte(wAttr>>8)&1 == 0 {
				pSel = pChc
			}
		}
		VidFillAttr(col, row, cnt, attr)
	}
	if !Test(fFound) && pSel != 0xFFFF {
		col := uint16(B(pSel+uChCol)) + W(pState+uStAbsCol)
		row := uint16(B(pFld+uFdRow)) + W(pState+uStAbsRow)
		cnt := uint16(B(pSel + uChWidth))
		if Test(fOn) {
			attr = 4
		} else {
			attr = 0
		}
		VidFillAttr(col, row, cnt, attr)
	}
	frmUnLockVideo()
	return fFound
}

// ChoiceShow(pState, pFld, val, fOn): show the choices with value val selected.
func frmChoiceShow(pState, pFld uint16, val, fOn byte) {
	var attr byte
	frmLockVideo()
	for i := uint16(0); i < uint16(B(pFld+uFdCb)); i++ {
		pChc := pFld + uFdRgb + i*5
		col := uint16(B(pChc+uChCol)) + W(pState+uStAbsCol)
		row := uint16(B(pFld+uFdRow)) + W(pState+uStAbsRow)
		cnt := uint16(B(pChc + uChWidth))
		if B(pChc+uChValue) == val {
			if Test(fOn) {
				attr = 4
			} else {
				attr = 0
			}
		} else {
			if Test(fOn) {
				attr = 5
			} else {
				attr = 1
			}
		}
		VidFillAttr(col, row, cnt, attr)
	}
	frmUnLockVideo()
}

// GetErcText(psb, erc)
func frmGetErcText(psb Ptr, erc uint16) {
	switch erc {
	case 0:
		SetFB(psb, 0)
	case 31000:
		strCopySb(psb, DG(ldgSbErc31000))
	case 31001:
		strCopySb(psb, DG(ldgSbErc31001))
	case 31002:
		strCopySb(psb, DG(ldgSbErc31002))
	case 31003:
		strCopySb(psb, DG(ldgSbErc31003))
	case 31004:
		strCopySb(psb, DG(ldgSbErc31004))
	case 31005:
		strCopySb(psb, DG(ldgSbErc31005))
	default:
		emGetErcMsg(psb, erc)
	}
}

func init() {
	DoForm = frmDoForm
}
