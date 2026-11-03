package main

// game_ratsc.go - translation of ../src/RatsC.plm (frame 01DF): game initialisation
// and user interface (options form, maze callback, status lines, score, end of game,
// keyboard).

func InitGame() {
	defer Release(Mark())
	pch := Push(2) // ch (ReadKbd(@ch))
	var n, i, j, pos, pos2 uint16
	var rgFact [100]uint16
	var dx, dy, tries byte
	CheckErc(AllocMemorySL(0xFFFF, DG(gPMap)))
	SETW(0, P(gPMap), 0x8000)
	InitVideo()
	GetOptions()
	SelectFont()
	SetW(gScore, 0)
	SetW(gHiScore, GetHighScore(gSbScoresFile))
	VidPutChars(0, 0, 80, DG(gRgbStatus1), 0)
	VidPutChars(0, 1, 80, DG(gRgbStatus2), 0)
	UpdateStatus()
	InitObjects()
	SeedRandom(DW(gDwMazeNum))
	GenMaze(0x10, 0x20, 2, 2, 2, 2, W(gWDensity)*0x147, MazeCell)
	// place the player and the factories, well apart from each other
	n = W(gNFactories) + 1
	if n < 33 {
		dx = B(gRgbSpacing + n)
	} else {
		dx = 0
	}
	dy = dx >> 1
	i = 0
	for i < n {
		tries = 0
	retry:
		pos = RandomCell()
		j = 0
		for j < i {
			pos2 = rgFact[j]
			if (gLo(pos)-gLo(pos2) < dx || gLo(pos2)-gLo(pos) < dx) &&
				(gHi(pos)-gHi(pos2) < dy || gHi(pos2)-gHi(pos) < dy) {
				tries = tries + 1
				if tries < 10 {
					goto retry
				}
				if pos == pos2 {
					goto retry
				}
			}
			j = j + 1
		}
		rgFact[i] = pos
		i = i + 1
	}
	NewPlayer(rgFact[0], 4, B(gBLeftHanded)&8, TRUE)
	if Test(B(gFPreview)) {
		MarkFactory(rgFact[0], 4)
	}
	i = 1
	for i < n {
		NewNest(rgFact[i], byte(i))
		if Test(B(gFPreview)) {
			MarkFactory(rgFact[i], 5)
		}
		i = i + 1
	}
	if Test(B(gFPreview)) {
		// sneak preview: wait for CANCEL (ESC) or GO, then erase it
		for {
			CheckErc(ReadKbd(DG(pch)))
			if B(pch) == 0x1B || B(pch) == 7 {
				break
			}
			Beep()
		}
		ClearRect(0x18, 2, 0x20, W(gCScrLines)-2)
	}
	UpdateStatus()
	CheckErc(SetKbdUnencodedMode(0xFF))
	SetW(gWKeysUp, 0)
	SetW(gWKeysHit, 0)
	SetW(gWKeysHeld, 0)
}

func InitVideo() {
	// erc = Rq258(9, pNil, 0): a CTOS request whose result is ignored; the screen setup
	// it did is part of the platform's VidInit in the port.
	VidInit(0x102, 1, 0, TRUE)
	SetW(gPosView, 0)
	SetW(gWScrW, W(gCScrCols))
	SetW(gWScrH, W(gCScrLines)-2)
}

// GetOptions: options form; scores file is "[!sys]<games>Rats-<factories>-<density>.scores"
func GetOptions() {
	defer Release(Mark())
	sbNum := Push(13)
	SetDW(gDwMazeNum, GetRandomSeed())
	CopySb(DG(gSbVersion), DG(gSbRatsVer))
	if !Test(DoForm(gNFactories, gOptForm, 0, 0xFF, 0xFF, gNFactories)) {
		Exit()
	}
	CopySb(DG(gSbScoresFile), DG(gSbScoresPfx))
	SbFromNumber(DG(sbNum), uint32(W(gNFactories)), 0, ' ')
	AppendSb(DG(gSbScoresFile), DG(sbNum))
	AppendCh(DG(gSbScoresFile), '-')
	SbFromNumber(DG(sbNum), uint32(W(gWDensity)), 0, ' ')
	AppendSb(DG(gSbScoresFile), DG(sbNum))
	AppendSb(DG(gSbScoresFile), DG(gSbScoresSfx))
}

func SelectFont() {
	if FB(P(gPVidCfg)) == 4 {
		LoadFont(gSbT1Font)
	} else {
		LoadFont(gSbFont)
	}
}

func InitObjects() {
	var i, p uint16
	i = 0
	for i < 16 {
		SetW(gRgHead+2*i, gRgHead+2*i)
		SetW(gRgTail+2*i, gRgHead+2*i)
		i = i + 1
	}
	SetW(gIWheel, 0)
	SetW(gPFree, 0)
	p = gRgPos
	for p > gRgObj {
		p = p - 16
		SetB(p+obType, 0xFF)
		SetB(p+obSize, 0)
		SetW(p+obLink, W(gPFree))
		SetW(gPFree, p)
	}
	SetW(gPPlayer, 0)
	SetW(gWViewW, W(gWScrW))
	SetW(gWViewH, W(gWScrH))
}

// MazeCell: GenMaze callback - draw one wall of maze cell (x,y) into the map (cells
// are 16x8); with "sneak preview" also on the screen.
func MazeCell(x, y uint16, fHoriz byte) {
	var pos, col, row, k uint16
	if Test(B(gFPreview)) {
		col = x << 1
		row = y
		if Test(fHoriz) {
			OrAttr(col, row, 2)
			col = (col + 1) & 0x1F
			OrAttr(col, row, 0x0A)
			col = (col + 1) & 0x1F
			OrAttr(col, row, 8)
		} else {
			OrAttr(col, row, 4)
			row = (row + 1) & 0x1F
			OrAttr(col, row, 1)
		}
	}
	pos = gSetLo(pos, byte(x<<4))
	pos = gSetHi(pos, byte(y<<3))
	if Test(fHoriz) {
		gSetMap(pos, gMap(pos)|2)
		pos = gSetLo(pos, gLo(pos)+1)
		SETB(0x0A, gMapPtr(pos), 15)
		pos = gSetLo(pos, gLo(pos)+15)
		gSetMap(pos, gMap(pos)|8)
	} else {
		gSetMap(pos, gMap(pos)|4)
		pos = gSetHi(pos, gHi(pos)+1)
		for k = 1; k <= 7; k++ {
			gSetMap(pos, 5)
			pos = gSetHi(pos, gHi(pos)+1)
		}
		gSetMap(pos, gMap(pos)|1)
	}
}

// OrAttr: OR bits into the character byte of a preview cell (line-drawing pieces).
func OrAttr(col, row uint16, attr byte) {
	var a uint16
	if row+2 < W(gCScrLines) {
		a = VidGetCell(col+0x18, row+2)
		a = gSetLo(a, gLo(a)|attr)
		VidPutCell(col+0x18, row+2, a)
	}
}

// MarkFactory: preview - highlight a factory position on the miniature maze.
func MarkFactory(pos uint16, attr byte) {
	var cx, cy, col, row uint16
	cx = uint16(gLo(pos) >> 4)
	cy = uint16(gHi(pos) >> 3)
	col = cx << 1
	row = cy
	if row+2 < W(gCScrLines) {
		VidFillAttr(col+0x18, row+2, 3, attr)
	}
	row = (row + 1) & 0x1F
	if row+2 < W(gCScrLines) {
		VidFillAttr(col+0x18, row+2, 3, attr)
	}
}

func AddScore(pts uint16) {
	SetW(gScore, W(gScore)+pts)
	for Test(Bool(W(gIBonus) < 5) & Bool(W(gRgwBonus+2*W(gIBonus)) <= W(gScore)) & ^B(gFForever)) {
		SetW(gCLives, W(gCLives)+1)
		Beep()
		SetW(gIBonus, W(gIBonus)+1)
	}
}

func UpdateStatus() {
	PutNum(4, 0, uint32(W(gCRatsDead)), 3)
	PutNum(4, 1, uint32(W(gCRatsLeft)), 3)
	PutNum(0x10, 0, uint32(W(gCCrazyDead)), 3)
	PutNum(0x10, 1, uint32(W(gCCrazyLeft)), 3)
	PutNum(0x1E, 0, uint32(W(gCFactDead)), 2)
	PutNum(0x1E, 1, uint32(W(gCFactLeft)), 2)
	PutNum(0x2B, 0, uint32(W(gCDeaths)), 2)
	PutNum(0x2B, 1, uint32(W(gCLives)), 2)
	PutNum(0x3C, 0, uint32(W(gScore)), 5)
	PutNum(0x3C, 1, uint32(W(gHiScore)), 5)
	PutNum(0x4A, 1, DW(gDwMazeNum), 5)
	UpdateClock()
	if W(gCFactDead) < 5 {
		SetW(gWFactValue, W(gRgwFactValue+2*W(gCFactDead)))
	} else {
		SetW(gWFactValue, W(gRgwFactValue+2*4))
	}
}

// UpdateClock: show the time; difficulty = factories killed*8K + rats killed*64 +
// time*32, 0FFFFh on overflow ("hi = hi PLUS 0" adds the carry of the addition).
func UpdateClock() {
	var hi uint16
	add := func(x uint16) {
		s := uint32(W(gWDifficulty)) + uint32(x)
		SetW(gWDifficulty, uint16(s))
		hi = hi + uint16(s>>16) // ADC hi,0
	}
	SetW(gTClock, W(gTickCount))
	PutNum(0x4A, 0, uint32(W(gTickCount)), 5)
	hi = 0
	SetW(gWDifficulty, 0)
	if W(gCFactDead) < 8 {
		add(W(gCFactDead) << 13)
	} else {
		hi = hi + 1
	}
	if W(gCRatsDead) < 0x400 {
		add(W(gCRatsDead) << 6)
	} else {
		hi = hi + 1
	}
	if W(gTickCount) < 0x800 {
		add(W(gTickCount) << 5)
	} else {
		hi = hi + 1
	}
	if hi != 0 {
		SetW(gWDifficulty, 0xFFFF)
	}
}

func PutNum(col, row uint16, dw uint32, cDigits uint16) {
	defer Release(Mark())
	sb := Push(13)
	SbFromNumber(DG(sb), dw, cDigits, ' ')
	VidPutChars(col, row, uint16(B(sb)), DG(sb+1), 0)
}

// EndLevel: reason 0 = all factories destroyed, 1 = out of lives, 2 = quit.
func EndLevel(reason byte) {
	var row, t uint16
	CheckErc(SetScreenVidAttr(0, 0))
	UpdateStatus()
	ClearRect(0, 2, W(gCScrCols), W(gCScrLines)-2)
	CenterMsg(2, W(gRgpEndMsg+2*uint16(reason)))
	row = 10
	CheckErc(Delay(row))
	ShowLine(30, row, gSbRaw, ' ', W(gScore), 0)
	UpdateStatus()
	HwBeep()
	row = row + 2
	CheckErc(SetKbdUnencodedMode(0))
	for W(gCLives) != 0 {
		CheckErc(Delay(10))
		ShowLine(30, row, gSbUnused, '+', 1000, 0)
		SetW(gScore, W(gScore)+1000)
		SetW(gCLives, W(gCLives)-1)
		UpdateStatus()
		HwBeep()
		row = row + 2
	}
	CheckErc(Delay(10))
	if W(gTickCount) <= W(gScore) {
		t = W(gTickCount)
	} else {
		t = W(gScore)
	}
	ShowLine(30, row, gSbElapsed, '-', t, 0)
	SetW(gScore, W(gScore)-t)
	UpdateStatus()
	HwBeep()
	row = row + 2
	CheckErc(Delay(10))
	ShowLine(30, row, gSbFinal, ' ', W(gScore), 8)
	UpdateStatus()
	Beep()
	row = row + 2
	CheckErc(Delay(50))
	if !Test(B(gFForever)) {
		EnterHighScore(gSbScoresFile, W(gScore), 3)
	}
	ClearRect(0, 0, W(gCScrCols), W(gCScrLines))
	Exit()
}

func CenterMsg(row, psb uint16) {
	VidPutChars((W(gCScrCols)-uint16(B(psb)))>>1, row, uint16(B(psb)), DG(psb+1), 0)
}

// ShowLine: "<label><sign><number>" line of the end-of-game tally.
func ShowLine(col, row, psb uint16, chSign byte, val uint16, attr byte) {
	defer Release(Mark())
	sbNum := Push(13)
	SbFromNumber(DG(sbNum), uint32(val), 5, ' ')
	VidPutChars(col, row, uint16(B(psb)), DG(psb+1), attr)
	VidPutCell(uint16(B(psb))+col, row, uint16(attr)<<8+uint16(chSign))
	VidPutChars(uint16(B(psb))+col+1, row, uint16(B(sbNum)), DG(sbNum+1), attr)
}

// PollKeyboard: key codes, bit 7 set = key released.
func PollKeyboard() {
	defer Release(Mark())
	pch := Push(2)
	var i, mask uint16
	var ch byte
	for {
		if ReadKbdDirect(1, DG(pch)) != 0 {
			return
		}
		ch = B(pch)
		if ch&0x7F == 4 && ch&0x80 == 0 {
			EndLevel(2)
		} else {
			i = FINDB(DG(gRgbKeys), ch&0x7F, 14)
			if i != 0xFFFF {
				mask = W(gRgwKeyMask + 2*i)
				if ch&0x80 == 0 {
					SetW(gWKeysHit, W(gWKeysHit)|mask)
					SetW(gWKeysUp, W(gWKeysUp)&^mask)
				} else {
					SetW(gWKeysUp, W(gWKeysUp)|mask)
				}
			}
		}
	}
}
