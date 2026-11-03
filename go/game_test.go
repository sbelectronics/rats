package main

// game_test.go - deterministic self-tests of the game modules (no platform needed):
// Random against values computed by hand from Random.plm, maze generation invariants
// (walls only, full connectivity on the torus map), far-procedure registrations, Sleep
// bookkeeping, and a long RunTick simulation (autopilot player) checking that the
// object pool, wheel lists, counters and emulated stack stay consistent.

import (
	"fmt"
	"reflect"
	"testing"
)

type gameExit struct{}

var gtVidCfgSeg uint16

// gameStubs replaces every api.go function the game calls with a deterministic stub.
func gameStubs(t *testing.T) {
	copy(dg[:], dgroupInit)
	for i := range dg[len(dgroupInit):] {
		dg[len(dgroupInit)+i] = 0
	}
	for i := range mapSeg {
		mapSeg[i] = 0
	}
	sp = 0x3CFC
	if gtVidCfgSeg == 0 {
		gtVidCfgSeg = AllocSeg(16)
	}
	CheckErc = func(erc uint16) {
		if erc != 0 {
			t.Fatalf("CheckErc(%04X)", erc)
		}
	}
	AllocMemorySL = func(cb uint16, pp Ptr) uint16 { SetFP(pp, MkPtr(SegMap, 0)); return 0 }
	VidInit = func(pFrames, cFrames uint16, fWide, fReverse byte) {
		SetW(gCScrCols, 80)
		SetW(gCScrLines, 34)
		SetP(gPVidCfg, MkPtr(gtVidCfgSeg, 0))
	}
	GetDateTime = func(p Ptr) uint16 { SetFDW(p, 0x12345678); return 0 }
	CopySb = func(dst, src Ptr) { MOVB(src, dst, uint16(FB(src))+1) }
	AppendSb = func(dst, src Ptr) {
		n := FB(dst)
		MOVB(src.Add(1), dst.Add(uint16(n)+1), uint16(FB(src)))
		SetFB(dst, n+FB(src))
	}
	AppendCh = func(dst Ptr, ch byte) { n := FB(dst); SetFB(dst.Add(uint16(n)+1), ch); SetFB(dst, n+1) }
	SbFromNumber = func(psb Ptr, q uint32, cch uint16, chFill byte) {
		s := fmt.Sprintf("%*d", cch, q)
		SetFB(psb, byte(len(s)))
		PutBytes(psb.Add(1), []byte(s))
	}
	DoForm = func(pRec, pForm, frame, col, row, pInit uint16) byte {
		SetDW(gDwMazeNum, 1) // as if the user typed maze number 1
		return TRUE
	}
	Exit = func() { panic(gameExit{}) }
	LoadFont = func(psb uint16) {}
	GetHighScore = func(pFile uint16) uint16 { return 0 }
	EnterHighScore = func(pFile, score, row uint16) {}
	VidPutChars = func(col, row, cnt uint16, pb Ptr, attr byte) {}
	VidPutCell = func(col, row, w uint16) {}
	VidGetCell = func(col, row uint16) uint16 { return 0 }
	VidFillAttr = func(col, row, cnt uint16, attr byte) {}
	ClearRect = func(col, row, cCols, cRows uint16) {}
	SetKbdUnencodedMode = func(f uint16) uint16 { return 0 }
	ReadKbd = func(p Ptr) uint16 { SetFB(p, 0x1B); return 0 }
	ReadKbdDirect = func(mode uint16, p Ptr) uint16 { return 1 } // no key
	SetScreenVidAttr = func(i, f uint16) uint16 { return 0 }
	Beep = func() uint16 { return 0 }
	HwBeep = func() {}
	Delay = func(n uint16) uint16 { return 0 }
	RTEvents = func() uint16 { return 0 }
}

// Expected values computed by hand from Random.plm: SeedRandom(1) runs one step
// (ring(0) = 32FF + ring(20) 98FD), then ring(i) = ring(i) + ring(i+20) for i = 1..4.
func TestGameRandom(t *testing.T) {
	gameStubs(t)
	initRing := []uint16{0x32FF, 0xB29D, 0xC936, 0x43C5, 0x4980, 0xCC05, 0xB4ED, 0x98B7,
		0xE893, 0xF24B, 0x2CA4, 0x7FFA, 0xAE2C, 0x6FF9, 0x3D1C, 0x66BE,
		0xCD4D, 0x8A25, 0x3917, 0x3D94, 0x98FD, 0x1502, 0xFDE1, 0xB7B7,
		0x0DCC, 0x8D75, 0x4B65, 0xBB22, 0x22F0, 0x8839, 0x47F8, 0x38AD, 0xE0CD}
	for i, v := range initRing {
		if W(gRgwInitRg+2*uint16(i)) != v {
			t.Fatalf("rgwInitRing(%d) = %04X in dgroup.bin, want %04X", i, W(gRgwInitRg+2*uint16(i)), v)
		}
	}
	SeedRandom(1)
	if W(gRgwRing) != 0xCBFC {
		t.Fatalf("ring(0) after SeedRandom(1) = %04X", W(gRgwRing))
	}
	want := []uint16{0xC79F, 0xC717, 0xFB7C, 0x574C}
	for i, w := range want {
		if g := Random(0); g != w {
			t.Fatalf("Random(0) #%d = %04X, want %04X", i+1, g, w)
		}
	}
	// independent model for a long run, and scaling Random(n) = HIGH(w * n)
	ring := append([]uint16(nil), initRing...)
	ir := 0
	step := func() uint16 {
		lag := (ir + 33 - 13) % 33
		ring[ir] += ring[lag]
		w := ring[ir]
		ir = (ir + 1) % 33
		return w
	}
	SeedRandom(0x1234)
	for i := 0; i < 0x1234; i++ {
		step()
	}
	for i := 0; i < 5000; i++ {
		n := uint16(i % 300)
		w := step()
		exp := w
		if n != 0 {
			exp = uint16(uint32(w) * uint32(n) >> 16)
		}
		if g := Random(n); g != exp {
			t.Fatalf("Random(%d) step %d = %d, want %d", n, i, g, exp)
		}
	}
	if GetRandomCount() != 0x1234+5000 {
		t.Fatalf("dwCalls = %d", GetRandomCount())
	}
	// GetRandomSeed from date/time 12345678h: lo = 56^34 = 62, hi = 78^12 = 6A
	if s := GetRandomSeed(); s != 0x6A62 {
		t.Fatalf("GetRandomSeed = %08X", s)
	}
	if sp != 0x3CFC {
		t.Fatalf("stack not balanced: %04X", sp)
	}
}

// genMap builds the maze of InitGame for (mazeNum, density) into the map.
func genMap(mazeNum uint32, density uint16) {
	for i := range mapSeg {
		mapSeg[i] = 0
	}
	SetP(gPMap, MkPtr(SegMap, 0))
	SetB(gFPreview, 0)
	SeedRandom(mazeNum)
	GenMaze(0x10, 0x20, 2, 2, 2, 2, density*0x147, MazeCell)
}

func TestGameMaze(t *testing.T) {
	gameStubs(t)
	for _, tc := range []struct {
		n uint32
		d uint16
	}{{1, 0x55}, {12345, 0x55}, {7, 0xC8}, {99, 0}} {
		genMap(tc.n, tc.d)
		first := mapSeg
		genMap(tc.n, tc.d)
		if first != mapSeg {
			t.Fatalf("maze %d not deterministic", tc.n)
		}
		walls := 0
		for i, c := range mapSeg {
			switch c {
			case 0:
			case 5, 0x0A:
				walls++
			default:
				if c > 0x0F {
					t.Fatalf("maze %d: map(%04X) = %02X", tc.n, i, c)
				}
				walls++
			}
		}
		// flood fill the free cells (4-neighbour, the map wraps at 256 in x and y)
		seen := make([]bool, 0x10000)
		start := uint16(8 | 4<<8)
		if mapSeg[start] != 0 {
			t.Fatalf("maze %d: cell centre not free", tc.n)
		}
		stack := []uint16{start}
		seen[start] = true
		for len(stack) > 0 {
			p := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			x, y := byte(p), byte(p>>8)
			for _, q := range []uint16{gMkPos(x+1, y), gMkPos(x-1, y), gMkPos(x, y+1), gMkPos(x, y-1)} {
				if !seen[q] && mapSeg[q] == 0 {
					seen[q] = true
					stack = append(stack, q)
				}
			}
		}
		for cy := 0; cy < 32; cy++ {
			for cx := 0; cx < 16; cx++ {
				c := uint16(cx*16+8) | uint16(cy*8+4)<<8
				if !seen[c] {
					t.Fatalf("maze %d density %d: cell (%d,%d) unreachable", tc.n, tc.d, cx, cy)
				}
			}
		}
		t.Logf("maze %d density %d: %d wall cells, Random calls %d", tc.n, tc.d, walls, GetRandomCount())
	}
	if sp != 0x3CFC {
		t.Fatalf("stack not balanced: %04X", sp)
	}
}

func TestGameFarProcs(t *testing.T) {
	gameStubs(t)
	same := func(a, b any) bool { return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer() }
	handlers := []any{PlayerTick, NestTick, YoungTick, RatTick, RatTick, ShotTick, GhostTick, BlastTick}
	for i, h := range handlers {
		f := FarFn(P(gRgpHandler + 4*uint16(i)))
		if _, ok := f.(func(uint16, uint16)); !ok || !same(f, h) {
			t.Fatalf("rgpHandler(%d) wrong", i)
		}
	}
	draws := []any{nil, Draw1, Draw2, nil, Draw4}
	erases := []any{nil, Erase1, Erase2, nil, Erase4}
	for i := 0; i < 5; i++ {
		if draws[i] == nil {
			if P(gRgpDraw+4*uint16(i)) != 0 || P(gRgpErase+4*uint16(i)) != 0 {
				t.Fatalf("sprite entry %d not null", i)
			}
			continue
		}
		d := FarFn(P(gRgpDraw + 4*uint16(i)))
		e := FarFn(P(gRgpErase + 4*uint16(i)))
		if _, ok := d.(func(byte, uint16, uint16)); !ok || !same(d, draws[i]) {
			t.Fatalf("rgpDraw(%d) wrong", i)
		}
		if _, ok := e.(func(uint16)); !ok || !same(e, erases[i]) {
			t.Fatalf("rgpErase(%d) wrong", i)
		}
	}
}

func TestGameSleep(t *testing.T) {
	gameStubs(t)
	old := sleepPollQuantum
	sleepPollQuantum = 0
	defer func() { sleepPollQuantum = old }()
	// script: events arrive after a given number of polls
	var polls int
	var due []int // poll index at which an event arrives
	RTEvents = func() uint16 {
		polls++
		if len(due) > 0 && polls >= due[0] {
			due = due[1:]
			return 1
		}
		return 0
	}
	if W(gCPollsPerTick) != 100 || B(gFFirst) != 0xFF {
		t.Fatalf("Sleep initial data wrong")
	}
	// first call: OpenRTClock consumes one RTEvents call; no event within 50+1 polls
	Sleep(2)
	if polls != 1+51 || W(gCIdle) != 51 || W(gTickCount) != 0 {
		t.Fatalf("timeout: polls %d cIdle %d", polls, W(gCIdle))
	}
	// event on the 10th poll of the next call
	due = []int{polls + 10}
	Sleep(2)
	if W(gCIdle) != 0 || W(gCPollsPerTick) != (300+60+2)/4 || B(gBTenths) != 1 {
		t.Fatalf("event: cPollsPerTick %d cIdle %d bTenths %d", W(gCPollsPerTick), W(gCIdle), B(gBTenths))
	}
	for i := 0; i < 9; i++ {
		due = []int{polls + 1}
		Sleep(2)
	}
	if W(gTickCount) != 1 || B(gBTenths) != 0 {
		t.Fatalf("tickCount %d bTenths %d", W(gTickCount), B(gBTenths))
	}
}

// checkPool verifies the object pool between ticks: every record is either on the free
// list or on exactly one wheel list, and the counters match the live objects.
func checkPool(t *testing.T, tick int, checkCounts bool) {
	t.Helper()
	where := map[uint16]int{}
	valid := func(p uint16) bool { return p >= gRgObj && p < gRgPos && (p-gRgObj)%16 == 0 }
	for i := uint16(0); i < 16; i++ {
		head := gRgHead + 2*i
		tailOk := W(gRgTail+2*i) == head
		n := 0
		for p := W(head); p != head; p = W(p) {
			if !valid(p) {
				t.Fatalf("tick %d: wheel %d bad link %04X", tick, i, p)
			}
			if _, dup := where[p]; dup {
				t.Fatalf("tick %d: object %04X twice", tick, p)
			}
			where[p] = int(i)
			if B(p+obType) >= 8 {
				t.Fatalf("tick %d: live object %04X type %02X", tick, p, B(p+obType))
			}
			if W(gRgTail+2*i) == p {
				tailOk = true
			}
			if n++; n > 256 {
				t.Fatalf("tick %d: wheel %d loops", tick, i)
			}
		}
		if !tailOk {
			t.Fatalf("tick %d: rgTail(%d) = %04X not in its list", tick, i, W(gRgTail+2*i))
		}
	}
	nFree := 0
	for p := W(gPFree); p != 0; p = W(p) {
		if !valid(p) || B(p+obType) != 0xFF || B(p+obSize) != 0 {
			t.Fatalf("tick %d: bad free entry %04X", tick, p)
		}
		if _, dup := where[p]; dup {
			t.Fatalf("tick %d: free object %04X also live", tick, p)
		}
		where[p] = -1
		if nFree++; nFree > 256 {
			t.Fatalf("tick %d: free list loops", tick)
		}
	}
	if len(where) != 256 {
		t.Fatalf("tick %d: %d objects accounted for", tick, len(where))
	}
	if checkCounts {
		var rats, crazy, fact uint16
		for p, w := range where {
			if w < 0 {
				continue
			}
			switch B(p + obType) {
			case 1:
				fact++
			case 2, 3:
				rats++
			case 4:
				crazy++
			}
		}
		if rats != W(gCRatsLeft) || crazy != W(gCCrazyLeft) || fact != W(gCFactLeft) {
			t.Fatalf("tick %d: counts rats %d/%d small %d/%d nests %d/%d", tick, rats, W(gCRatsLeft),
				crazy, W(gCCrazyLeft), fact, W(gCFactLeft))
		}
	}
	if pp := W(gPPlayer); pp != 0 && (!valid(pp) || B(pp+obType) != 0) {
		t.Fatalf("tick %d: pPlayer %04X is not a player", tick, pp)
	}
	if sp != 0x3CFC {
		t.Fatalf("tick %d: stack not balanced: %04X", tick, sp)
	}
}

func TestGameRunTicks(t *testing.T) {
	gameStubs(t)
	ended := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(gameExit); !ok {
					panic(r)
				}
				ended = true
			}
		}()
		InitGame()
	}()
	if ended {
		t.Fatalf("InitGame exited")
	}
	// 5 factories (options default), lives 3 in dgroup.bin minus the first player
	if W(gCFactLeft) != 5 || W(gPPlayer) == 0 || W(gCLives) != 2 {
		t.Fatalf("InitGame: nests %d player %04X lives %d", W(gCFactLeft), W(gPPlayer), W(gCLives))
	}
	checkPool(t, 0, true)
	SetW(gCLives, 3)
	SetB(W(gPPlayer)+obFlags, B(W(gPPlayer)+obFlags)|1) // autopilot demo
	full := false
	kinds := map[byte]int{}
	tick := 0
	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(gameExit); !ok {
					panic(r)
				}
				ended = true
				sp = 0x3CFC
			}
		}()
		for tick = 1; tick <= 20000; tick++ {
			RunTick()
			if tick%20 == 0 {
				SetW(gTickCount, W(gTickCount)+1) // 2 RunTicks per 100 ms event
			}
			if W(gPFree) == 0 {
				full = true
			}
			checkPool(t, tick, !full)
			for p := gRgObj; p < gRgPos; p += 16 {
				if B(p+obType) < 8 {
					kinds[B(p+obType)]++
				}
			}
			if W(gPPlayer) != 0 {
				SetB(W(gPPlayer)+obFlags, B(W(gPPlayer)+obFlags)|1)
			}
		}
	}()
	t.Logf("ticks %d ended %v score %d rats dead %d small dead %d nests dead %d deaths %d; object-ticks by type %v",
		tick, ended, W(gScore), W(gCRatsDead), W(gCCrazyDead), W(gCFactDead), W(gCDeaths), kinds)
	if kinds[5] == 0 || kinds[2] == 0 {
		t.Fatalf("simulation did not exercise shots/rats: %v", kinds)
	}
}
