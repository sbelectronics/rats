package main

// platform_mem.go - CTOS memory services (AllocMemorySL/LL, DeallocMemorySL/LL)
// and the platform's own "system" segment.
//
// The heap of Mod3396 (MemAlloc/MemFree) does PARAGRAPH arithmetic on the segments it
// gets from AllocMemorySL: it splits a block by adding cParas to the segment value and
// hands back pieces of a block with DeallocMemorySL(p, cb).  Separate 64K arrays per
// segment (mem.go AllocSeg) cannot support that, so the pool here is one linear byte
// array (the "physical" memory of paragraphs pPoolLo..pPoolHi-1) and every allocated
// paragraph s gets its own segs[s] entry that aliases pPhys[(s-pPoolLo)*16 : +64K].
// Short-lived memory is taken from the top of the pool downward, long-lived memory
// from the bottom upward, like CTOS.  The first 64K AllocMemorySL request (InitGame's
// maze map) is answered with mem.go's SegMap (1000h) while that segment is free.
//
// System segment SegSys (F800h: not part of the pool, and not a multiple of 1000h so
// that mem.go AllocSeg never collides with it) holds the CTOS structures the program
// reads through far pointers:
//   F800:0000  VCB (video control block, GetPStructure(2), pointed to by pVidCfg 2F1Eh)
//   F800:0200  ASCB (GetpAscb; user name length at +40h, text at +41h)
//   F800:0400  video line table (GetPStructure(7), pVidBlocks 2F1Ah), 256 far pointers
//   F800:1000  video map, one 100h-byte block per screen line (see platform_screen.go)
//   F800:3200  scratch line block for out-of-range rows

import "sync"

const (
	SegSys             uint16 = 0xF800
	pPoolLo            uint16 = 0x2000
	pPoolHi            uint16 = 0xF000 // exclusive
	pNParas                   = int(pPoolHi - pPoolLo)
	pErcMemoryNotAvail        = 400
	pErcBadDeallocate         = 402
)

var (
	pSysMem  [0x10000]byte
	pPhys    = make([]byte, pNParas*16+0x10000)
	pParaUse = make([]bool, pNParas)
	pLlSize  = map[uint16]int{} // long-lived allocations: segment -> paragraphs
	pMapUsed bool
	pMemMu   sync.Mutex
)

func init() {
	segs[SegSys] = pSysMem[:]
	AllocMemorySL = pAllocMemorySL
	DeallocMemorySL = pDeallocMemorySL
	AllocMemoryLL = pAllocMemoryLL
	DeallocMemoryLL = pDeallocMemoryLL
}

func pMarkParas(s uint16, n int, used bool) {
	for i := 0; i < n; i++ {
		seg := s + uint16(i)
		idx := int(seg - pPoolLo)
		pParaUse[idx] = used
		if used {
			o := idx * 16
			segs[seg] = pPhys[o : o+0x10000]
		} else {
			delete(segs, seg)
		}
	}
}

// pFindRun finds n free paragraphs; fromTop selects the highest run, else the lowest.
func pFindRun(n int, fromTop bool) (uint16, bool) {
	if n <= 0 || n > pNParas {
		return 0, false
	}
	if fromTop {
		run := 0
		for i := pNParas - 1; i >= 0; i-- {
			if pParaBusy(i) {
				run = 0
				continue
			}
			run++
			if run == n {
				return pPoolLo + uint16(i), true
			}
		}
	} else {
		run := 0
		for i := 0; i < pNParas; i++ {
			if pParaBusy(i) {
				run = 0
				continue
			}
			run++
			if run == n {
				return pPoolLo + uint16(i-n+1), true
			}
		}
	}
	return 0, false
}

func pZeroParas(s uint16, n int) {
	o := int(s-pPoolLo) * 16
	clear(pPhys[o : o+n*16])
}

func pParasFor(cb uint32) int {
	n := int((cb + 15) >> 4)
	if n == 0 {
		n = 1
	}
	return n
}

func pAllocMemorySL(cb uint16, ppRet Ptr) uint16 {
	pMemMu.Lock()
	defer pMemMu.Unlock()
	n := pParasFor(uint32(cb))
	if n == 0x1000 && !pMapUsed {
		pMapUsed = true
		clear(mapSeg[:])
		SetFP(ppRet, MkPtr(SegMap, 0))
		return 0
	}
	s, ok := pFindRun(n, true)
	if !ok {
		return pErcMemoryNotAvail
	}
	pMarkParas(s, n, true)
	pZeroParas(s, n)
	SetFP(ppRet, MkPtr(s, 0))
	return 0
}

func pDeallocMemorySL(p Ptr, cb uint16) uint16 {
	pMemMu.Lock()
	defer pMemMu.Unlock()
	if p.Seg() == SegMap {
		if !pMapUsed {
			return pErcBadDeallocate
		}
		pMapUsed = false
		return 0
	}
	return pFreeParas(uint32(p.Seg())+uint32(p.Off()>>4), pParasFor(uint32(cb)))
}

func pFreeParas(s32 uint32, n int) uint16 {
	if s32 < uint32(pPoolLo) || s32+uint32(n) > uint32(pPoolHi) {
		return pErcBadDeallocate
	}
	s := uint16(s32)
	for i := 0; i < n; i++ {
		if !pParaUse[int(s-pPoolLo)+i] {
			return pErcBadDeallocate
		}
	}
	pMarkParas(s, n, false)
	return 0
}

func pAllocMemoryLL(cb uint32, ppRet Ptr) uint16 {
	pMemMu.Lock()
	defer pMemMu.Unlock()
	n := pParasFor(cb)
	s, ok := pFindRun(n, false)
	if !ok {
		return pErcMemoryNotAvail
	}
	pMarkParas(s, n, true)
	pZeroParas(s, n)
	pLlSize[s] = n
	SetFP(ppRet, MkPtr(s, 0))
	return 0
}

func pDeallocMemoryLL(p Ptr) uint16 {
	pMemMu.Lock()
	defer pMemMu.Unlock()
	s := p.Seg() + p.Off()>>4
	n, ok := pLlSize[s]
	if !ok {
		return pErcBadDeallocate
	}
	delete(pLlSize, s)
	return pFreeParas(uint32(s), n)
}

// pParaBusy: paragraph index i is allocated by us, or its segment number is taken by
// someone else's segs entry (mem.go AllocSeg, test fakes) - never clobber those.
func pParaBusy(i int) bool {
	return pParaUse[i] || segs[pPoolLo+uint16(i)] != nil
}
