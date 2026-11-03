package main

// game_sleep.go - translation of ../src/Sleep.plm (frame 032A): game timer.
//
// The original opens a CTOS real-time clock request (counter = reload = 1: one event
// per 100 ms RT tick, counted in rtc.cEvents) and busy-polls its exchange with Check,
// counting idle polls in cIdle.  cPollsPerTick is a running estimate of the idle polls
// per RT tick; Sleep(n) gives up after cPollsPerTick/n polls.  So with the main loop's
// Sleep(2), RunTick runs once at each RT event and once more about half a tick later.
//
// Port: "Check" is emulated by RTEvents() (events since the last call are added to
// rtc.cEvents and the "message" is present when any arrived), and one idle poll costs
// sleepPollQuantum of real time.  The calibration then works exactly as in the
// original: cPollsPerTick converges to (100 ms / quantum), and the timeout of
// cPollsPerTick/n polls is about 100/n ms of real time, independent of CPU speed.
// All bookkeeping (exchRtc, rtc, cPollsPerTick, cIdle, tickCount, bTenths, fFirst) is
// kept in DGROUP at 2C9Ah..2CAFh.

import "time"

// one idle Check poll; 1 ms matches the original's initial estimate of 100 polls/tick.
var sleepPollQuantum = time.Millisecond

const (
	gRtcCounter       = gRtc + 0
	gRtcCounterReload = gRtc + 2
	gRtcExchResp      = gRtc + 6
	gRtcErcRet        = gRtc + 8
	gRtcRqCode        = gRtc + 10
)

// sleepCheck emulates Check(exchRtc, @pMsg): 0 if the RT clock request was returned
// to the exchange (at least one event since it was last taken), else erc "no message".
func sleepCheck() uint16 {
	if k := RTEvents(); k != 0 {
		SetW(gRtcCEvents, W(gRtcCEvents)+k)
		return 0
	}
	return 1
}

func Sleep(n byte) {
	var cMax, i uint16
	if Test(B(gFFirst)) {
		SetW(gExchRtc, 1) // AllocExch: any exchange number
		SetW(gRtcCounter, 1)
		SetW(gRtcCounterReload, 1)
		SetW(gRtcCEvents, 0)
		SetW(gRtcExchResp, W(gExchRtc))
		SetW(gRtcErcRet, 0)
		SetW(gRtcRqCode, 0xFFFF)
		RTEvents() // OpenRTClock: start counting from now
		SetB(gFFirst, 0)
	}
	cMax = W(gCPollsPerTick) / uint16(n)
	i = 0
	for i <= cMax {
		if sleepCheck() == 0 {
			if W(gRtcCEvents) != 0 {
				SetW(gCPollsPerTick, (W(gCPollsPerTick)<<1+W(gCPollsPerTick)+W(gCIdle)+2)>>2)
				SetW(gCIdle, 0)
				SetB(gBTenths, B(gBTenths)+byte(W(gRtcCEvents)))
				for B(gBTenths) >= 10 {
					SetW(gTickCount, W(gTickCount)+1)
					SetB(gBTenths, B(gBTenths)-10)
				}
				SetW(gRtcCEvents, 0)
				return
			}
		}
		SetW(gCIdle, W(gCIdle)+1)
		i = i + 1
		if sleepPollQuantum > 0 {
			time.Sleep(sleepPollQuantum)
		}
	}
}
