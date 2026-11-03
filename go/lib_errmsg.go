package main

// lib_errmsg.go - translation of ../src/ErrMsg.plm (image 591A-5CD2, frame 0591h).
// Error code -> message text.  Messages can be stored in the scratch file
// [scr]<$>Erc.tmp in 16-byte paragraphs (pseudo erc 8000h OR paragraph); the file is
// accessed through one cached 512-byte sector buffer.  CONST = DGROUP 0CDC-0D7F,
// DATA = DGROUP 2D02-2F0E.  (Nothing in Rats calls StoreMsg, so the file is never used;
// the module is translated completely anyway.)

const (
	ldgRgbFileSpecErc = 0x0CDC // '[scr]<$>Erc.tmp' (15 bytes)
	ldgSbErcEq        = 0x0CEB // (6, 'erc = ')
	ldgRgbErcTab      = 0x0CF2 // entries: erc WORD, sb; terminated by erc 0
	ldgEmFh           = 0x2D02 // WORD 0FFFFh = not open
	ldgEmLfaFileEnd   = 0x2D04 // DWORD
	ldgEmRgbSector    = 0x2D08 // (200h) BYTE
	ldgEmLfaSector    = 0x2F08 // DWORD
	ldgEmParaNext     = 0x2F0C // WORD
	ldgEmFDirty       = 0x2F0E // BYTE
	emCbSector        = 0x200
)

// StoreMsg(psb): store sb in the file, return 8000h + paragraph (0 if empty).
func emStoreMsg(psb uint16) uint16 {
	if B(psb) == 0 {
		return 0
	}
	if W(ldgEmFh) == 0xFFFF {
		emOpenMsgFile()
	}
	para := W(ldgEmParaNext)
	SetW(ldgEmParaNext, W(ldgEmParaNext)+uint16((B(psb)+1+0x10-1)>>4)) // BYTE arithmetic
	emWriteMsg(DG(psb), para)
	return para | 0x8000
}

// GetErcMsg(psbRet, erc): 0 -> empty; 8000h + para -> stored text; else built-in text
// or 'erc = <n>'.
func emGetErcMsg(psbRet Ptr, erc uint16) {
	if erc == 0 {
		SetFB(psbRet, 0)
	} else if erc&0x8000 == 0 {
		emGetStdErcMsg(psbRet, erc)
	} else {
		emReadMsg(psbRet, erc&0x7FFF)
	}
}

func emGetStdErcMsg(psbRet Ptr, erc uint16) {
	defer Release(Mark())
	sbNum := DG(Push(13))
	pEntry := uint16(ldgRgbErcTab)
	for W(pEntry) != 0 {
		if W(pEntry) == erc {
			strCopySb(psbRet, DG(pEntry+2))
			return
		}
		pEntry = pEntry + 3 + uint16(B(pEntry+2))
	}
	SetFB(psbRet, 0)
	strAppendSb(psbRet, DG(ldgSbErcEq))
	strSbFromNumber(sbNum, uint32(erc), 1, 0)
	strAppendSb(psbRet, sbNum)
}

func emOpenMsgFile() {
	erc := CreateFile(DG(ldgRgbFileSpecErc), 15, P(ldgPNil), 0, 0)
	if erc != 0xE0 { // 224: file already exists
		ceCheckErc(erc)
	}
	ceCheckErc(OpenFile(DG(ldgEmFh), DG(ldgRgbFileSpecErc), 15, P(ldgPNil), 0, 0x6D6D))
	ceCheckErc(GetFileStatus(W(ldgEmFh), 0, DG(ldgEmLfaFileEnd), 4))
	SetDW(ldgEmLfaSector, DW(ldgDwNil))
	SetB(ldgEmFDirty, FALSE)
	SetW(ldgEmParaNext, 0)
}

// WriteMsg(pb, para): copy sb pb into the file at para*16.
func emWriteMsg(pb Ptr, para uint16) {
	lfa := rtMul32(para, 16)
	iOff := uint16(lfa) & 0x1FF
	lfa &^= 0x1FF            // awLfa(0) AND 0FE00h
	cb := uint16(FB(pb) + 1) // BYTE arithmetic
	i := uint16(0)
	for i < cb {
		emLoadSector(lfa)
		cbChunk := cb - i
		if emCbSector-iOff < cbChunk {
			cbChunk = emCbSector - iOff
		}
		MOVB(pb.Add(i), DG(ldgEmRgbSector+iOff), cbChunk)
		SetB(ldgEmFDirty, TRUE)
		lfa += emCbSector
		iOff = 0
		i += cbChunk
	}
}

// ReadMsg(psbRet, para): read sb from the file at para*16.  Original bug kept: the loop
// starts at i = 1 with cb = length, so the last character is never copied.
func emReadMsg(psbRet Ptr, para uint16) {
	lfa := rtMul32(para, 16)
	iOff := uint16(lfa) & 0x1FF
	lfa &^= 0x1FF
	emLoadSector(lfa)
	SetFB(psbRet, B(ldgEmRgbSector+iOff))
	iOff++
	cb := uint16(FB(psbRet))
	i := uint16(1)
	for i < cb {
		emLoadSector(lfa)
		cbChunk := cb - i
		if emCbSector-iOff < cbChunk {
			cbChunk = emCbSector - iOff
		}
		MOVB(DG(ldgEmRgbSector+iOff), psbRet.Add(i), cbChunk)
		lfa += emCbSector
		iOff = 0
		i += cbChunk
	}
}

// LoadSector(lfa): make the buffer hold the sector at lfa (write back a dirty buffer
// first, extending the file).
func emLoadSector(lfa uint32) {
	defer Release(Mark())
	cbRet := Push(2)
	if lfa == DW(ldgEmLfaSector) {
		return
	}
	if Test(B(ldgEmFDirty)) {
		if DW(ldgEmLfaFileEnd) <= DW(ldgEmLfaSector) {
			SetDW(ldgEmLfaFileEnd, DW(ldgEmLfaSector)+0xA00)
			ceCheckErc(ChangeFileLength(W(ldgEmFh), DW(ldgEmLfaFileEnd)))
		}
		ceCheckErc(WriteFile(W(ldgEmFh), DG(ldgEmRgbSector), emCbSector, DW(ldgEmLfaSector), DG(cbRet)))
		SetB(ldgEmFDirty, FALSE)
	}
	if lfa < DW(ldgEmLfaFileEnd) {
		ceCheckErc(ReadFile(W(ldgEmFh), DG(ldgEmRgbSector), emCbSector, lfa, DG(cbRet)))
	} else {
		SETB(0, DG(ldgEmRgbSector), emCbSector)
	}
	SetDW(ldgEmLfaSector, lfa)
}
