package main

// game_ratsa.go - translation of ../src/RatsA.plm (frame 0000): main program,
// scheduler (16-slot timing wheel), object core.
//
// Objects are 16-byte ObjType records in rgObj (DGROUP 135Ah); object "pointers" are
// DGROUP offsets.  The wheel lists are circular through their sentinel rgHead(i)
// (whose link field is rgHead(i) itself); an object being run has link = 0.

func init() {
	GameMainFn = GameMain // platform_hooks.go: main.go runs it in a goroutine
	// rgpHandler (DGROUP 0018h) entries, frame 90h (RatsB)
	RegisterFar(0x0090, 0x06DD, PlayerTick)
	RegisterFar(0x0090, 0x0A0A, NestTick)
	RegisterFar(0x0090, 0x0A9B, YoungTick)
	RegisterFar(0x0090, 0x0B0D, RatTick)
	RegisterFar(0x0090, 0x0D67, ShotTick)
	RegisterFar(0x0090, 0x0D8C, GhostTick)
	RegisterFar(0x0090, 0x1073, BlastTick)
}

// GameMain is RatsA's Main (the module body is CALL Main).
func GameMain() {
	InitGame()
	for {
		RunTick()
		Sleep(2)
	}
}

// helpers for the far-pointer tables
func callHandler(t byte, pos, p uint16) {
	FarFn(P(gRgpHandler+4*uint16(t))).(func(uint16, uint16))(pos, p)
}
func callDraw(size byte, ch byte, pos, pPos uint16) {
	FarFn(P(gRgpDraw+4*uint16(size))).(func(byte, uint16, uint16))(ch, pos, pPos)
}
func callErase(size byte, pos uint16) {
	FarFn(P(gRgpErase + 4*uint16(size))).(func(uint16))(pos)
}

func RunTick() {
	UpdatePlayer()
	ScrollView()
	PollKeyboard()
	for {
		head := gRgHead + 2*W(gIWheel)
		SetW(gPCur, W(head))
		if W(gPCur) == head {
			break
		}
		pCur := W(gPCur)
		SetW(head, W(pCur+obLink))
		if W(gRgTail+2*W(gIWheel)) == pCur {
			SetW(gRgTail+2*W(gIWheel), W(pCur+obLink))
		}
		SetW(pCur+obLink, 0)
		SetW(gPosCur, W(gPosSlot(pCur)))
		callHandler(B(pCur+obType), W(gPosCur), pCur)
	}
	SetW(gIWheel, (W(gIWheel)+1)&0x0F)
	if W(gTClock) != W(gTickCount) {
		UpdateClock()
	}
}

func MoveCur(pos uint16) {
	if W(W(gPCur)+obLink) != 0 {
		CheckErc(0x577)
	}
	if pos != W(gPosCur) {
		if gMap(pos) != 0 {
			pHit := FindObjAt(pos)
			if pHit != W(gPCur) {
				KillObj(W(gPCur))
				if pHit == 0 {
					if W(gWDemolition) != 0 {
						if Random(0) < W(gWDemolition) {
							gSetMap(pos, 0)
						}
					}
				} else if B(pHit+obType) < 6 {
					KillObj(pHit)
				}
				return
			}
		}
		callErase(B(W(gPCur)+obSize), W(gPosCur))
		SetW(gPosCur, pos)
	}
	pCur := W(gPCur)
	DrawObj(W(gPosCur), pCur)
	w := (uint16(B(pCur+obDelay)) + W(gIWheel)) & 0x0F
	pTail := W(gRgTail + 2*w)
	SetW(pCur+obLink, W(pTail+obLink))
	SetW(pTail+obLink, pCur)
	SetW(gRgTail+2*w, pCur)
}

func DrawObj(pos, p uint16) {
	var ch, b byte
	ctr := func() byte { return B(p + obCtr) }
	dir := func() uint16 { return uint16(B(p + obDir)) }
	switch B(p + obType) {
	case 0:
		if B(p+obFlags)&0x40 != 0 {
			b = B(gRgbAnim + uint16(ctr()&3))
		} else {
			b = 8
		}
		ch = B(gRgbDir+dir())<<1 + 0xC0 + b
	case 1:
		ch = (ctr()&1)<<2 + 0x80
	case 2:
		ch = B(gRgbDir+dir()) + 0xA0 + 1
	case 3:
		ch = B(gRgbDir+dir()) + 0xA0 + (ctr()&1)<<2
	case 4:
		ch = byte(dir())<<1 + 0x90 + ctr()&1
	case 5:
		if ctr() < 0xF0 {
			SetB(gBFlash, 0x10)
		}
		ch = byte(dir()) + 0x88
	case 6:
		SetB(gBFlash, 0x10)
		if B(p+obFlags)&0x20 != 0 {
			var f uint16
			if ctr()&1 != 0 {
				f = 0x00FF
			}
			CheckErc(SetScreenVidAttr(0, f))
		}
		if B(p+obSize) == 1 {
			ch = ctr()&1 + 0x18
		} else if B(p+obSize) == 2 {
			ch = (ctr()&1)<<2 + 0x11
		} else {
			ch = (ctr()&1)<<2 + 0x10
		}
	case 7:
		SetB(gBFlash, 0x10)
		ch = (ctr()&1)<<2 + B(gRgbKind+dir())
	}
	callDraw(B(p+obSize), ch, pos, gPosSlot(p))
	SetB(p+obCtr, B(p+obCtr)+1)
	if B(p+obCtr) == 0 {
		SetB(p+obCtr, 0xF0)
	}
}

func KillObj(p uint16) {
	defer Release(Mark())
	save := Push(obLen)
	var w14, s, w1E, w20, w22, w1C uint16
	var bMap, fKind byte
	pos := W(gPosSlot(p))
	bMap = 0
	if B(p+obType) == 0 {
		if Test(B(gFRandomize)) {
			if Test(B(gFLeaveGhosts)) {
				bMap = gMap(pos)
			}
		}
	}
	FreeObj(save, p)
	fKind = TRUE
	w14 = 0
	switch B(save + obType) {
	case 0:
		SetW(gCDeaths, W(gCDeaths)+1)
		if Test(B(gFForever)) {
			SetW(gCLives, W(gCLives)+1)
		}
		if p == W(gPPlayer) {
			SetW(gPPlayer, 0)
			SetB(gBPlayerB9, B(save+obDir))
			SetB(gBPlayerFlags, B(save+obFlags))
			SetW(gTLastPlayer, W(gTickCount))
		} else if Test(B(gFForever)) {
			fKind = FALSE
		}
	case 1:
		AddScore(W(gWFactValue) - W(save+obWC))
		SetW(gCFactDead, W(gCFactDead)+1)
		SetW(gCFactLeft, W(gCFactLeft)-1)
		w16 := gRgObj
		for i := uint16(0); i < 256; i++ {
			if W(w16+obWE) == p {
				SetW(w16+obWE, 0)
			}
			w16 += 16
		}
		if Test(B(gFForever)) {
			fKind = 1
		}
	case 2:
		SetB(save+obFlags, B(save+obFlags)&0xEF)
		fallthrough
	case 3:
		fallthrough
	case 4: // case34
		if B(save+obFlags)&0x10 != 0 {
			w1E = W(save+obWC) / 25
			w20 = W(save+obWC) % 25
			w22 = w1E * 5
			s = w20 + w22
			w14 = w22
		} else {
			s = W(save + obWC)
		}
		w1C = W(save + obWE)
		if w1C != 0 {
			SetW(w1C+obWC, W(w1C+obWC)+s)
		} else if Test(B(gFCrazyRats)) {
			s = s >> 1
		}
		AddScore(s)
		if B(save+obType) == 4 {
			SetW(gCCrazyDead, W(gCCrazyDead)+1)
			SetW(gCCrazyLeft, W(gCCrazyLeft)-1)
		} else {
			SetW(gCRatsDead, W(gCRatsDead)+1)
			SetW(gCRatsLeft, W(gCRatsLeft)-1)
		}
	case 5:
	case 6, 7:
		return
	}
	UpdateStatus()
	NewGhost(pos, B(save+obSize), B(save+obB6), B(save+obFlags)&0x30, bMap, w14,
		W(save+obWE), fKind)
}

func FindObjAt(pos uint16) uint16 {
	var i, k, p uint16
	i = 0
	for {
		k = FINDW(DG(gRgPos+2*i), pos, 0x400-i)
		if k == 0xFFFF {
			break
		}
		i = i + k
		p = (i&0xFFFC)<<2 + gRgObj
		if B(p+obSize) > byte(i&3) {
			return p
		}
		i = i + 1
	}
	return 0
}

func UpdatePlayer() {
	var pos, p, i uint16
	if B(gBFlash) != 0 {
		SetB(gBFlash, B(gBFlash)-1)
	}
	if W(gCFactLeft) == 0 && W(gCRatsLeft) == 0 && W(gCCrazyLeft) == 0 && W(gCFactDead) != 0 &&
		B(gBFlash) == 0 {
		EndLevel(0)
	}
	if W(gPPlayer) == 0 {
		if W(gTickCount)-W(gTLastPlayer) >= W(gTRespawn) {
			if W(gCLives) != 0 {
				if Test(B(gFRandomize)) {
					pos = RandomCell()
				} else {
					pos = W(gPosPlayer)
				}
				if Test(IsFree(pos, 4)) {
					NewPlayer(pos, B(gBPlayerB9), B(gBPlayerFlags)&0x29, TRUE)
				}
			} else {
				p = gRgObj
				i = 0
				for i < 256 {
					if B(p+obType) == 0 {
						SetW(gPPlayer, p)
						SetB(p+obFlags, B(p+obFlags)&0xF7|B(gBLeftHanded)&8)
						goto found
					}
					p = p + 16
					i = i + 1
				}
				if B(gBFlash) == 0 {
					EndLevel(1)
				}
			}
		}
	}
found:
	if W(gPPlayer) != 0 {
		SetW(gPosPlayer, W(gPosSlot(W(gPPlayer))))
		SetB(gRgbView+2, byte(uint16(B(gBPlayerX))+1-W(gWViewW)>>1))
		SetB(gRgbView+3, byte(uint16(B(gBPlayerY))+1-W(gWViewH)>>1))
		SetB(gRgbView+0, byte(uint16(B(gBPlayerX))+1-W(gWScrW)>>1))
		SetB(gRgbView+1, byte(uint16(B(gBPlayerY))+1-W(gWScrH)>>1))
	}
}

func ScrollView() {
	var pos, i uint16
	var b byte
	pos = W(gPosView)
	b = byte(256 - uint16(gLo(pos)))
	if uint16(b) >= W(gWScrW) {
		i = 2
		for i < W(gCScrLines) {
			VidPutChars(0, i, W(gWScrW), gMapPtr(pos), 0)
			pos = gSetHi(pos, gHi(pos)+1)
			i = i + 1
		}
	} else {
		i = 2
		for i < W(gCScrLines) {
			VidPutChars(0, i, uint16(b), gMapPtr(pos), 0)
			VidPutChars(uint16(b), i, W(gWScrW)-uint16(b), gMapPtr(pos&0xFF00), 0)
			pos = gSetHi(pos, gHi(pos)+1)
			i = i + 1
		}
	}
}

func RandomCell() uint16 {
	var pos uint16
	pos = gSetLo(pos, byte(Random(0)&0xF0)+7)
	pos = gSetHi(pos, byte(Random(0)&0xF8)+4)
	return pos
}

func IsFree(pos uint16, size byte) byte {
	var b byte
	b = gMap(pos)
	if size > 1 {
		pos = gSetLo(pos, gLo(pos)+1)
		b = b | gMap(pos)
		pos = gSetHi(pos, gHi(pos)+1)
		b = b | gMap(pos)
		pos = gSetLo(pos, gLo(pos)-1)
		b = b | gMap(pos)
		pos = gSetHi(pos, gHi(pos)-1)
	}
	return Bool(b == 0)
}

func NewObj(pTmpl, pos uint16, delay byte) uint16 {
	var p, w, pTail uint16
	if W(gPFree) == 0 {
		return 0
	}
	p = W(gPFree)
	SetW(gPFree, W(p+obLink))
	MOVB(DG(pTmpl), DG(p), 16)
	DrawObj(pos, p)
	for B(p+obDelay) <= delay {
		delay = delay - B(p+obDelay)
	}
	w = (uint16(delay) + W(gIWheel) + 1) & 0x0F
	pTail = W(gRgTail + 2*w)
	SetW(p+obLink, W(pTail+obLink))
	SetW(pTail+obLink, p)
	SetW(gRgTail+2*w, p)
	return p
}

func FreeObj(pDest, p uint16) {
	var q, k uint16
	MOVB(DG(p), DG(pDest), 16)
	callErase(B(p+obSize), W(gPosSlot(p)))
	if W(p+obLink) != 0 {
		q = p
		for W(q+obLink) != p {
			q = W(q + obLink)
		}
		SetW(q+obLink, W(p+obLink))
		k = FINDW(DG(gRgTail), p, 16)
		if k != 0xFFFF {
			SetW(gRgTail+2*k, W(p+obLink)) // original bug: tail := obj.link, not q
		}
		SetW(p+obLink, 0)
	}
	SetB(p+obType, 0xFF)
	SetB(p+obSize, 0)
	SetW(p+obLink, W(gPFree))
	SetW(gPFree, p)
}
