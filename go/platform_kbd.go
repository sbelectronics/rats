package main

// platform_kbd.go - CTOS keyboard: ReadKbd, ReadKbdDirect, SetKbdUnencodedMode.
//
// Ebiten's Update (main goroutine) feeds a goroutine-safe queue (pKbdPress /
// pKbdRelease / pKbdTypematic).  Unencoded mode queues raw key codes (press = code,
// release = code OR 80h, no repeat).  Encoded mode queues characters: the
// unshifted/shifted value of rgInfoIkey (platform_kbdtab.go), SHIFT applied, LOCK
// (Caps Lock) applied to keys with attribute 80h, typematic repeat per the key's
// attribute bits.  PC punctuation keys deliver their US-layout characters in
// encoded mode (the CTOS keyboard has a different punctuation layout).

import (
	"sync"
	"time"
)

const (
	pErcNoCharactersInKeyboard = 602 // erc.edf
	pKbdQueueMax               = 128
	pRawNone                   = 0xFF
)

// pKeyDef describes one PC key: its CTOS raw code (pRawNone = no CTOS key, encoded
// only) and, for printable PC keys, explicit encoded characters (0,0 = use rgInfoIkey).
type pKeyDef struct {
	raw          byte
	chUn, chSh   byte
	explicitChar bool
}

var (
	pKbdMu      sync.Mutex
	pKbdCond    = sync.NewCond(&pKbdMu)
	pKbdQueue   []byte
	pKbdRawMode bool
	pKbdShift   int  // number of shift keys held
	pKbdLock    bool // CTOS LOCK state (toggled by Caps Lock)
	pKbdHeld    [256]int
	// typematic state (encoded mode)
	pRepKey    int = -1 // PC key id being repeated
	pRepRaw    byte
	pRepNext   time.Time
	pRepPeriod time.Duration
)

func init() {
	ReadKbd = pReadKbd
	ReadKbdDirect = pReadKbdDirect
	SetKbdUnencodedMode = pSetKbdUnencodedMode
}

func pKbdPush(b byte) {
	if len(pKbdQueue) < pKbdQueueMax {
		pKbdQueue = append(pKbdQueue, b)
		pKbdCond.Broadcast()
	}
}

func pKbdWait() byte {
	pKbdMu.Lock()
	defer pKbdMu.Unlock()
	for len(pKbdQueue) == 0 {
		pKbdCond.Wait()
	}
	b := pKbdQueue[0]
	pKbdQueue = pKbdQueue[1:]
	return b
}

func pReadKbd(pCh Ptr) uint16 {
	SetFB(pCh, pKbdWait())
	return 0
}

func pReadKbdDirect(mode uint16, pCh Ptr) uint16 {
	if mode == 0 {
		SetFB(pCh, pKbdWait())
		return 0
	}
	pKbdMu.Lock()
	defer pKbdMu.Unlock()
	if len(pKbdQueue) == 0 {
		return pErcNoCharactersInKeyboard
	}
	SetFB(pCh, pKbdQueue[0])
	pKbdQueue = pKbdQueue[1:]
	return 0
}

// pSetKbdUnencodedMode switches mode (PL/M boolean, bit 0) and discards type-ahead.
func pSetKbdUnencodedMode(f uint16) uint16 {
	pKbdMu.Lock()
	defer pKbdMu.Unlock()
	pKbdRawMode = f&1 != 0
	pKbdQueue = pKbdQueue[:0]
	pRepKey = -1
	return 0
}

// pEncodedChar returns the character a key produces in encoded mode.
func pEncodedChar(d pKeyDef) (byte, byte, bool) {
	shifted := pKbdShift > 0
	if d.explicitChar {
		if shifted {
			return d.chSh, pKAttrRepeat, true
		}
		return d.chUn, pKAttrRepeat, true
	}
	if d.raw >= 0x80 {
		return 0, 0, false
	}
	un, sh, attr := pRgInfoIkey[int(d.raw)*3], pRgInfoIkey[int(d.raw)*3+1], pRgInfoIkey[int(d.raw)*3+2]
	if un == 0 && sh == 0 && d.raw != 0 {
		return 0, 0, false // shift, lock, code ... produce no character
	}
	if attr&0x80 != 0 && pKbdLock {
		shifted = !shifted
	}
	if shifted {
		return sh, attr, true
	}
	return un, attr, true
}

func pTypematic(attr byte) (delay, period time.Duration, ok bool) {
	switch attr & 0x60 {
	case 0x00:
		return 0, 0, false
	case 0x20:
		delay = 200 * time.Millisecond
	case 0x40:
		delay = 400 * time.Millisecond
	default:
		delay = 700 * time.Millisecond
	}
	switch attr & 0x18 {
	case 0x00:
		period = time.Second
	case 0x08:
		period = 100 * time.Millisecond
	case 0x10:
		period = 50 * time.Millisecond
	default:
		period = time.Second / 30
	}
	return delay, period, true
}

// pKbdPress handles a PC key going down (key = PC key id, d = its definition).
func pKbdPress(key int, d pKeyDef, now time.Time) {
	pKbdMu.Lock()
	defer pKbdMu.Unlock()
	if d.raw == 0x48 || d.raw == 0x49 {
		pKbdShift++
	}
	if d.raw == 0x45 {
		pKbdLock = !pKbdLock
	}
	if pKbdRawMode {
		if d.raw == pRawNone {
			return
		}
		pKbdHeld[d.raw]++
		if pKbdHeld[d.raw] == 1 {
			pKbdPush(d.raw)
		}
		return
	}
	if d.raw != pRawNone {
		pKbdHeld[d.raw]++
	}
	ch, attr, ok := pEncodedChar(d)
	if !ok {
		return
	}
	pKbdPush(ch)
	if delay, period, rep := pTypematic(attr); rep {
		pRepKey, pRepRaw, pRepNext, pRepPeriod = key, d.raw, now.Add(delay), period
	} else {
		pRepKey = -1
	}
}

// pKbdRelease handles a PC key going up.
func pKbdRelease(key int, d pKeyDef) {
	pKbdMu.Lock()
	defer pKbdMu.Unlock()
	if (d.raw == 0x48 || d.raw == 0x49) && pKbdShift > 0 {
		pKbdShift--
	}
	if key == pRepKey {
		pRepKey = -1
	}
	if d.raw == pRawNone || pKbdHeld[d.raw] == 0 {
		return
	}
	pKbdHeld[d.raw]--
	if pKbdRawMode && pKbdHeld[d.raw] == 0 {
		pKbdPush(d.raw | 0x80)
	}
}

// pKbdTypematic is called every Update with the definition of the repeating key.
func pKbdTypematic(now time.Time, lookup func(key int) (pKeyDef, bool)) {
	pKbdMu.Lock()
	defer pKbdMu.Unlock()
	if pKbdRawMode || pRepKey < 0 {
		return
	}
	d, held := lookup(pRepKey)
	if !held {
		pRepKey = -1
		return
	}
	for !now.Before(pRepNext) {
		if ch, _, ok := pEncodedChar(d); ok {
			pKbdPush(ch)
		}
		pRepNext = pRepNext.Add(pRepPeriod)
	}
}
