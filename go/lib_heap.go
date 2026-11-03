package main

// lib_heap.go - translation of ../src/Mod3396.plm (image 3396-35AB, frame 0339h):
// a small heap on top of CTOS short-lived memory.  Free blocks are paragraph aligned
// (offset 0) and kept in a list sorted by DESCENDING segment address, head in pFreeHead
// (DGROUP 2CB0h), ended by the nil pointer C000:FFFF.  Block header at offset 0 of its
// segment: +0 size in paragraphs WORD, +2 next POINTER; the user pointer is block+2.

const ldgPFreeHead = 0x2CB0

// heapSeg makes a segment value produced by paragraph arithmetic (splitting a free
// block) addressable: in the port each CTOS allocation is its own 64K array, so a
// segment inside a block is mapped as an alias of the containing block's memory.
func heapSeg(seg uint16) uint16 {
	if segs[seg] != nil {
		return seg
	}
	best, found := uint16(0), false
	for s := range segs {
		if s < seg && (!found || s > best) {
			best, found = s, true
		}
	}
	if found {
		d := int(seg-best) * 16
		if m := segs[best]; d < len(m) {
			segs[seg] = m[d:]
		}
	}
	return seg
}

// MemAlloc(ppRet, cb): first fit; splits the top part off a larger block, otherwise asks
// CTOS (AllocMemorySL) after returning the lowest free block to CTOS (MemTrim).  Returns
// an erc (400 if cb is too large).
func heapMemAlloc(ppRet Ptr, cb uint16) uint16 {
	defer Release(Mark())
	pp := Push(4) // p (its address is passed to AllocMemorySL)
	var cParas uint16
	if cb <= 0xFFEE {
		cb = cb + 2
		cParas = (cb + 15) >> 4
	} else if cb <= 0xFFFE {
		cb = 0xFFFF
		cParas = 0x1000
	} else {
		return 400
	}
	ppLink := DG(ldgPFreeHead)
	for Test(hlpNotNil(FP(ppLink))) {
		p := FP(ppLink)
		SetP(pp, p)
		if FW(p) > cParas { // split: allocate the top part
			SetFW(p, FW(p)-cParas)
			SetP(pp, MkPtr(heapSeg(p.Seg()+FW(p)), p.Off()))
			goto setSize
		} else if FW(p) == cParas {
			SetFP(ppLink, FP(p.Add(2)))
			goto found
		}
		ppLink = p.Add(2)
	}
	if Test(hlpNotNil(P(ldgPFreeHead))) {
		heapMemTrim()
	}
	if erc := AllocMemorySL(cb, DG(pp)); erc != 0 {
		return erc
	}
setSize:
	SetFW(P(pp), cParas)
found:
	SetFP(ppRet, P(pp).Add(2))
	return 0
}

// MemFree(pMem): inserts the block, coalescing with neighbours, then MemTrim.
func heapMemFree(pMem Ptr) {
	pMem = MkPtr(pMem.Seg(), pMem.Off()-2)
	ppLink := DG(ldgPFreeHead)
again:
	for Test(hlpNotNil(FP(ppLink))) {
		p := FP(ppLink)
		if pMem.Seg() < p.Seg() {
			if pMem.Seg()+FW(pMem) == p.Seg() { // block just below p: absorb p
				SetFW(pMem, FW(pMem)+FW(p))
				SetFP(ppLink, FP(p.Add(2)))
				goto again
			}
		} else if p.Seg()+FW(p) == pMem.Seg() { // block just above p: grow p
			SetFW(p, FW(p)+FW(pMem))
			goto done
		} else {
			goto insert
		}
		ppLink = p.Add(2)
	}
insert:
	SetFP(pMem.Add(2), FP(ppLink))
	SetFP(ppLink, pMem)
done:
	heapMemTrim()
}

// MemTrim: returns the last (lowest) free block to CTOS.
func heapMemTrim() {
	ppLink := DG(ldgPFreeHead)
	var p Ptr
	for {
		p = FP(ppLink)
		if Test(hlpIsNil(FP(p.Add(2)))) {
			break
		}
		ppLink = p.Add(2)
	}
	if DeallocMemorySL(p, FW(p)<<4-1) == 0 {
		SetFP(ppLink, P(ldgPNil))
	}
}

func init() {
	MemAlloc = heapMemAlloc
	MemFree = heapMemFree
}
