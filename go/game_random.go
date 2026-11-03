package main

// game_random.go - translation of ../src/Random.plm (frame 0577): additive lagged
// Fibonacci generator x(i) = x(i) + x(i-13) over a ring of 33 words.
// CONST: dwNil 0C96h, rgwInitRing 0C9Ah.  DATA: rgwRing 2CB8h, iRing 2CFAh,
// dwCalls 2CFCh, fSeeded 2D00h.

const (
	gCRing = 33
	gCLag  = 13
)

// GetRandomSeed: DWORD seed derived from GetDateTime.
func GetRandomSeed() uint32 {
	defer Release(Mark())
	dt := Push(4) // dt DWORD, address passed to GetDateTime
	var abSeed [4]byte
	CheckErc(GetDateTime(DG(dt)))
	abSeed[0] = B(dt+1) ^ B(dt+2)
	abSeed[1] = B(dt+0) ^ B(dt+3)
	abSeed[3], abSeed[2] = 0, 0
	return uint32(abSeed[0]) | uint32(abSeed[1])<<8 | uint32(abSeed[2])<<16 | uint32(abSeed[3])<<24
}

func GetRandomCount() uint32 { return DW(gDwCalls) }

// SeedRandom: reset the ring to the initial table and run it dwSeed times
// (dwSeed = 0FFFFFFFFh: use GetRandomSeed).
func SeedRandom(dwSeed uint32) {
	var wDummy uint16
	if dwSeed == DW(gDwNil) {
		dwSeed = GetRandomSeed()
	}
	MOVW(DG(gRgwInitRg), DG(gRgwRing), gCRing)
	SetW(gIRing, 0)
	SetB(gFSeeded, TRUE)
	SetDW(gDwCalls, 0)
	for dwSeed != 0 {
		wDummy = Random(0)
		dwSeed = dwSeed - 1
	}
	_ = wDummy
}

// Random(n): next number scaled to 0..n-1 (n = 0: raw 16-bit value).
func Random(n uint16) uint16 {
	var iLag, w uint16
	if !Test(B(gFSeeded)) {
		SeedRandom(GetRandomSeed())
	}
	iRing := W(gIRing)
	if iRing >= gCLag {
		iLag = iRing - gCLag
	} else {
		iLag = iRing + gCRing - gCLag
	}
	w = W(gRgwRing+2*iRing) + W(gRgwRing+2*iLag)
	SetW(gRgwRing+2*iRing, w)
	SetW(gIRing, iRing+1)
	if W(gIRing) == gCRing {
		SetW(gIRing, 0)
	}
	SetDW(gDwCalls, DW(gDwCalls)+1)
	if n == 0 {
		return w
	}
	dw := uint32(w) * uint32(n) // Mul32 (unsigned MUL)
	return uint16(dw >> 16)
}

// RandomWeighted(pw, n): random index 0..n-1 weighted by the words pw(i).
func RandomWeighted(pw, n uint16) uint16 {
	var wSum, i, r uint16
	i, wSum = 0, 0
	for i < n {
		wSum = wSum + W(pw+2*i)
		i = i + 1
	}
	r = Random(wSum)
	i = 0
	for {
		if W(pw+2*i) > r {
			return i
		}
		r = r - W(pw+2*i)
		i = i + 1
	}
}
