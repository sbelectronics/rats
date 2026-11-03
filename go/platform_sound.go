package main

// platform_sound.go - Beep (CTOS Beep request) and HwBeep (Mod35AC: speaker bit
// held on for a busy-wait of 2000/3000 loop counts).  Both are square waves played
// through Ebiten audio; the durations are plausible values, not measured ones.

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const (
	pSampleRate = 48000
	pBeepHz     = 1000
)

var (
	pAudioCtx  *audio.Context
	pAudioMu   sync.Mutex
	pPlayers   []*audio.Player
	pBeepPCM   []byte
	pHwBeepPCM []byte
)

func init() {
	Beep = pBeep
	HwBeep = pHwBeep
}

// pAudioInit is called by main.go when a window is used.
func pAudioInit() {
	pAudioCtx = audio.NewContext(pSampleRate)
	pBeepPCM = pSquare(pBeepHz, 0.10)
	pHwBeepPCM = pSquare(pBeepHz, 0.15)
}

// pSquare returns 16-bit little-endian stereo PCM of a square wave.
func pSquare(hz int, seconds float64) []byte {
	n := int(seconds * pSampleRate)
	b := make([]byte, n*4)
	half := pSampleRate / hz / 2
	const amp = 6000
	for i := 0; i < n; i++ {
		v := int16(amp)
		if (i/half)&1 != 0 {
			v = -amp
		}
		b[i*4], b[i*4+1], b[i*4+2], b[i*4+3] = byte(v), byte(uint16(v)>>8), byte(v), byte(uint16(v)>>8)
	}
	return b
}

func pPlay(pcm []byte) {
	if pAudioCtx == nil {
		return
	}
	pAudioMu.Lock()
	defer pAudioMu.Unlock()
	keep := pPlayers[:0]
	for _, p := range pPlayers {
		if p.IsPlaying() {
			keep = append(keep, p)
		} else {
			p.Close()
		}
	}
	pPlayers = keep
	p := pAudioCtx.NewPlayerFromBytes(pcm)
	p.Play()
	pPlayers = append(pPlayers, p)
}

func pBeep() uint16 { pPlay(pBeepPCM); return 0 }
func pHwBeep() {
	pGetVidType() // Mod35AC selects the beeper port by GetVidType
	pPlay(pHwBeepPCM)
}
