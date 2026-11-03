package main

// platform_time.go - RTEvents, Delay, GetDateTime, Exit, ErrorExit.
//
// CTOS DateTime (CTOS ExpDt.Plm, CmpDT.Plm): a DWORD, low word =
// seconds since the start of the half day (0..43199), high word = number of half
// days since the epoch 1 March 1952 00:00 (local time; "Mar 1, '52 was a Saturday").

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	pRtMu      sync.Mutex
	pRtStart   time.Time
	pRtStarted bool
	pRtLast    int64
	pDtEpoch   = time.Date(1952, 3, 1, 0, 0, 0, 0, time.UTC)
	pWindowUp  bool // a window is showing the screen (ErrorExit waits for the user)
)

func init() {
	RTEvents = pRTEvents
	Delay = pDelay
	GetDateTime = pGetDateTime
	Exit = pExit
	ErrorExit = pErrorExit
}

// pRTEvents: number of 100 ms real-time clock events since the previous call.  The
// clock starts with the first call (OpenRTClock), which returns 0.
func pRTEvents() uint16 {
	pRtMu.Lock()
	defer pRtMu.Unlock()
	now := time.Now()
	if !pRtStarted {
		pRtStarted, pRtStart, pRtLast = true, now, 0
		return 0
	}
	n := int64(now.Sub(pRtStart) / (100 * time.Millisecond))
	d := n - pRtLast
	pRtLast = n
	return uint16(d)
}

func pDelay(tenths uint16) uint16 {
	time.Sleep(time.Duration(tenths) * 100 * time.Millisecond)
	return 0
}

// pCtosDateTime converts a local wall-clock time to the CTOS DateTime DWORD.
func pCtosDateTime(t time.Time) uint32 {
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	days := int64(day.Sub(pDtEpoch) / (24 * time.Hour))
	h := t.Hour()
	secs := (h%12)*3600 + t.Minute()*60 + t.Second()
	half := days*2 + int64(h/12)
	return uint32(uint16(half))<<16 | uint32(secs)
}

func pGetDateTime(pDt Ptr) uint16 {
	SetFDW(pDt, pCtosDateTime(time.Now()))
	return 0
}

func pExit() { os.Exit(0) }

// pErrorExit shows the erc on the bottom screen line (and on stderr), waits for a
// key (at most 15 s) when a window is open, then terminates.
func pErrorExit(erc uint16) {
	msg := fmt.Sprintf("Rats: error exit, erc = %d (%04Xh)", erc, erc)
	fmt.Fprintln(os.Stderr, msg)
	if pWindowUp {
		text := msg + "  - press any key"
		pVidMu.Lock()
		lines := uint16(pSm(pVcbOff + 0x0C))
		cols := uint16(pSm(pVcbOff + 0x0B))
		if lines > 0 {
			line := pSmW(pLineTabOff + (lines-1)*4)
			ibA, ibC := uint16(pSm(pVcbOff+0x0E)), uint16(pSm(pVcbOff+0x0F))
			for c := uint16(0); c < cols; c++ {
				ch := byte(' ')
				if int(c) < len(text) {
					ch = text[c]
				}
				pSetSm(line+ibC+c, ch)
			}
			for c := uint16(0); c < cols/2; c++ {
				pSetSm(line+ibA+c, 0x44) // reverse video
			}
		}
		pScrRefresh, pScrReverse = true, false
		pVidSerial++
		pVidMu.Unlock()
		pKbdMu.Lock()
		pKbdQueue = pKbdQueue[:0]
		pKbdMu.Unlock()
		done := make(chan struct{})
		go func() { pKbdWait(); close(done) }()
		select {
		case <-done:
		case <-time.After(15 * time.Second):
		}
	}
	os.Exit(1)
}
