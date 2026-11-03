package main

// game_data.go - DGROUP layout used by the game modules (RatsA/B/C, Maze, Random,
// Sleep, Sprites).  Names and offsets from ../src/RatsData.asm, RatsTables.asm,
// ../src/*.sym and ../tools/symbols.txt.  All game state stays in dg at these offsets;
// the constants carry a "g" prefix so they cannot collide with library or platform names.

// ---- CONST (RatsData / RatsB / RatsC constants) ----
const (
	gRgbDir     uint16 = 0x0004 // rgbDir(8): sprite row by heading
	gRgbAnim    uint16 = 0x000C // rgbAnim(4)
	gRgbKind    uint16 = 0x0010 // rgbKind(8): blast fragment chars
	gRgpHandler uint16 = 0x0018 // rgpHandler(8) far procs, frame 90h

	// RatsB CONST (WORD DATA first, then BYTE DATA)
	gRgwShotDelta  uint16 = 0x0038
	gRgwShotDelta2 uint16 = 0x0048
	gRgwRatDelta   uint16 = 0x0058
	gRgwRatDelta2  uint16 = 0x0068
	gRgbDelay      uint16 = 0x0078
	gRgbDirAlt     uint16 = 0x0088
	gRgbKeyDir     uint16 = 0x0090 // (64) BYTE, 16 initialised
	gRgbTurn       uint16 = 0x00D0
	gRgbNearScore  uint16 = 0x00D9

	// RatsC CONST
	gWShotReload  uint16 = 0x00EA
	gTRespawn     uint16 = 0x00EC
	gRgwFactValue uint16 = 0x00EE
	gRgwBonus     uint16 = 0x00F8
	gRgpEndMsg    uint16 = 0x0106
	gRgwKeyMask   uint16 = 0x010C
	gSbFont       uint16 = 0x0128
	gSbT1Font     uint16 = 0x013E
	gRgbSpacing   uint16 = 0x0156
	gSbRaw        uint16 = 0x0232
	gSbUnused     uint16 = 0x0241
	gSbElapsed    uint16 = 0x0250
	gSbFinal      uint16 = 0x025F
	gRgbKeys      uint16 = 0x026E
	gSbScoresPfx  uint16 = 0x027C // (18, '[!sys]<games>Rats-')
	gSbScoresSfx  uint16 = 0x028F // (7, '.scores')

	// RatsTables
	gRgbSeekPlayer uint16 = 0x0298
	gRgbSeek3      uint16 = 0x03D8
	gRgbSeek4      uint16 = 0x0518
	gRgpDraw       uint16 = 0x0658 // rgpDraw(5) far procs, frame 287h
	gRgpErase      uint16 = 0x066C // rgpErase(5)
	gOptForm       uint16 = 0x0680

	gPNil      uint16 = 0x0C92
	gDwNil     uint16 = 0x0C96 // Random CONST
	gRgwInitRg uint16 = 0x0C9A // rgwInitRing(33)
	gSbRatsVer uint16 = 0x122A
)

// ---- DATA ----
const (
	gTLastPlayer  uint16 = 0x1234
	gPCur         uint16 = 0x1236
	gPosCur       uint16 = 0x1238
	gBPlayerB9    uint16 = 0x123A
	gBPlayerFlags uint16 = 0x123B
	gNFactories   uint16 = 0x123C
	gWDensity     uint16 = 0x123E
	gWDemolition  uint16 = 0x1240
	gDwMazeNum    uint16 = 0x1242
	gFCrazyRats   uint16 = 0x1246
	gFRandomize   uint16 = 0x1247 // fRandomizeOnDeath
	gFLeaveGhosts uint16 = 0x1248
	gFPreview     uint16 = 0x1249
	gBLeftHanded  uint16 = 0x124A
	gFForever     uint16 = 0x124B
	gSbVersion    uint16 = 0x124C
	gWViewW       uint16 = 0x134C
	gWViewH       uint16 = 0x134E
	gIBonus       uint16 = 0x1350
	gPMap         uint16 = 0x1352
	gPosView      uint16 = 0x1356 // = rgbView(0..1) = bView2X, bView2Y
	gRgbView      uint16 = 0x1356
	gBViewX       uint16 = 0x1358
	gBViewY       uint16 = 0x1359
	gRgObj        uint16 = 0x135A // 256 x 16 bytes
	gRgPos        uint16 = 0x235A // 1024 words
	gRgHead       uint16 = 0x2B5A // 16 words (sentinel nodes)
	gRgTail       uint16 = 0x2B7A
	gIWheel       uint16 = 0x2B9A
	gPFree        uint16 = 0x2B9C
	gPPlayer      uint16 = 0x2B9E
	gPosPlayer    uint16 = 0x2BA0 // = bPlayerX, ptPlayer.x
	gBPlayerX     uint16 = 0x2BA0
	gBPlayerY     uint16 = 0x2BA1
	gCRatsDead    uint16 = 0x2BA2
	gCRatsLeft    uint16 = 0x2BA4
	gCCrazyDead   uint16 = 0x2BA6
	gCCrazyLeft   uint16 = 0x2BA8
	gCFactDead    uint16 = 0x2BAA
	gCFactLeft    uint16 = 0x2BAC
	gCDeaths      uint16 = 0x2BAE
	gCLives       uint16 = 0x2BB0
	gScore        uint16 = 0x2BB2
	gHiScore      uint16 = 0x2BB4
	gWDifficulty  uint16 = 0x2BB6
	gWFactValue   uint16 = 0x2BB8
	gWKeysHeld    uint16 = 0x2BBA
	gWKeysHit     uint16 = 0x2BBC
	gWKeysUp      uint16 = 0x2BBE
	gTClock       uint16 = 0x2BC0
	gWScrW        uint16 = 0x2BC2
	gWScrH        uint16 = 0x2BC4
	gBFlash       uint16 = 0x2BC6
	gRgbStatus1   uint16 = 0x2BC7
	gRgbStatus2   uint16 = 0x2C17
	gSbScoresFile uint16 = 0x2C67

	// Sleep DATA
	gExchRtc       uint16 = 0x2C9A
	gRtc           uint16 = 0x2C9C // counter, counterReload, cEvents, exchResp, ercRet, rqCode
	gRtcCEvents    uint16 = 0x2CA0
	gCPollsPerTick uint16 = 0x2CA8
	gCIdle         uint16 = 0x2CAA
	gTickCount     uint16 = 0x2CAC
	gBTenths       uint16 = 0x2CAE
	gFFirst        uint16 = 0x2CAF

	// Random DATA
	gRgwRing uint16 = 0x2CB8 // 33 words
	gIRing   uint16 = 0x2CFA
	gDwCalls uint16 = 0x2CFC
	gFSeeded uint16 = 0x2D00

	// platform / video
	gPVidCfg   uint16 = 0x2F1E
	gCScrCols  uint16 = 0x2F22
	gCScrLines uint16 = 0x2F24
)

// ObjType field offsets: STRUCTURE (link WORD, delay BYTE, ctr BYTE, type BYTE, size BYTE,
// b6 BYTE, dir BYTE, flags BYTE, b9 BYTE, wA WORD, wC WORD, wE WORD) = 16 bytes.
const (
	obLink  uint16 = 0
	obDelay uint16 = 2
	obCtr   uint16 = 3
	obType  uint16 = 4
	obSize  uint16 = 5
	obB6    uint16 = 6
	obDir   uint16 = 7
	obFlags uint16 = 8
	obB9    uint16 = 9
	obWA    uint16 = 0xA
	obWC    uint16 = 0xC
	obWE    uint16 = 0xE
	obLen          = 16
)

// map BASED pMap (1) BYTE: map(pos) and @map(pos)
func gMapPtr(pos uint16) Ptr     { return P(gPMap).Add(pos) }
func gMap(pos uint16) byte       { return FB(gMapPtr(pos)) }
func gSetMap(pos uint16, v byte) { SetFB(gMapPtr(pos), v) }

// rgPos(SHR(p - .rgObj, 2)) - address of the first position slot of object p
func gPosSlot(p uint16) uint16 { return gRgPos + 2*((p-gRgObj)>>2) }

// little byte helpers for the "bp(2) BYTE AT (@pos)" overlays
func gLo(w uint16) byte              { return byte(w) }
func gHi(w uint16) byte              { return byte(w >> 8) }
func gSetLo(w uint16, b byte) uint16 { return w&0xFF00 | uint16(b) }
func gSetHi(w uint16, b byte) uint16 { return w&0x00FF | uint16(b)<<8 }
func gMkPos(x, y byte) uint16        { return uint16(x) | uint16(y)<<8 }
