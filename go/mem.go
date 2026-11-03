package main

// mem.go - the memory model of the original program, kept so that the translated
// PL/M behaves exactly like the original (16-bit offsets, aliasing, sentinels, bugs).
//
// The original runs with DS = SS = DGROUP.  All near pointers (PL/M WORD "pointers",
// .var) are 16-bit offsets into DGROUP; we keep DGROUP as a 64K byte array `dg` and
// keep those offsets as uint16.  Far pointers (PL/M POINTER, @var) are 32-bit
// seg:off values (type Ptr).  Segments that exist:
//   SegDG  (07A6h)  DGROUP: CONST+DATA (initialised from dgroup.bin) and the stack
//   SegMap (1000h)  the 64K maze map allocated by InitGame (AllocMemorySL)
//   SegHeap(2000h+) blocks handed out by AllocMemorySL/AllocMemoryLL (see AllocSeg)
//   code frames (0000h..07A5h) only appear inside far pointers to procedures; they are
//   dispatched through the farProc registry (see api.go), never dereferenced.

import (
	_ "embed"
	"encoding/binary"
)

//go:embed dgroup.bin
var dgroupInit []byte // DGROUP 0000-2F45 exactly as rebuilt from the PL/M/ASM sources

const (
	SegDG   uint16 = 0x07A6
	SegMap  uint16 = 0x1000
	segHeap uint16 = 0x2000
)

// Ptr is a far pointer: high word = segment, low word = offset (as stored in memory).
type Ptr uint32

func MkPtr(seg, off uint16) Ptr { return Ptr(uint32(seg)<<16 | uint32(off)) }
func (p Ptr) Seg() uint16       { return uint16(p >> 16) }
func (p Ptr) Off() uint16       { return uint16(p) }
func (p Ptr) Add(n uint16) Ptr  { return MkPtr(p.Seg(), p.Off()+n) } // offset arithmetic wraps like 8086
func DG(off uint16) Ptr         { return MkPtr(SegDG, off) }         // @var for a DGROUP variable

// NilPtr is the CTOS nil pointer C000:FFFF (see Mod55FC / pNil at DGROUP 0C92h).
const NilPtr Ptr = 0xC000FFFF

var dg [0x10000]byte
var mapSeg [0x10000]byte
var segs = map[uint16][]byte{}

func init() {
	copy(dg[:], dgroupInit)
	segs[SegDG] = dg[:]
	segs[SegMap] = mapSeg[:]
}

// AllocSeg returns a new segment of n bytes (n <= 65536), zero filled, for CTOS memory requests.
func AllocSeg(n int) uint16 {
	s := segHeap
	for segs[s] != nil {
		s += 0x1000
	}
	segs[s] = make([]byte, 0x10000)
	return s
}
func FreeSeg(s uint16) {
	if s != SegDG && s != SegMap {
		delete(segs, s)
	}
}

func memOf(seg uint16) []byte {
	m := segs[seg]
	if m == nil {
		panic("access to unmapped segment")
	}
	return m
}

// ---- DGROUP near accessors (offset arithmetic wraps at 64K like the 8086) ----
func B(o uint16) byte          { return dg[o] }
func SetB(o uint16, v byte)    { dg[o] = v }
func W(o uint16) uint16        { return uint16(dg[o]) | uint16(dg[o+1])<<8 }
func SetW(o uint16, v uint16)  { dg[o] = byte(v); dg[o+1] = byte(v >> 8) }
func DW(o uint16) uint32       { return uint32(W(o)) | uint32(W(o+2))<<16 }
func SetDW(o uint16, v uint32) { SetW(o, uint16(v)); SetW(o+2, uint16(v>>16)) }
func P(o uint16) Ptr           { return Ptr(DW(o)) }
func SetP(o uint16, p Ptr)     { SetDW(o, uint32(p)) }

// ---- far accessors ----
func FB(p Ptr) byte          { return memOf(p.Seg())[p.Off()] }
func SetFB(p Ptr, v byte)    { memOf(p.Seg())[p.Off()] = v }
func FW(p Ptr) uint16        { return uint16(FB(p)) | uint16(FB(p.Add(1)))<<8 }
func SetFW(p Ptr, v uint16)  { SetFB(p, byte(v)); SetFB(p.Add(1), byte(v>>8)) }
func FDW(p Ptr) uint32       { return uint32(FW(p)) | uint32(FW(p.Add(2)))<<16 }
func SetFDW(p Ptr, v uint32) { SetFW(p, uint16(v)); SetFW(p.Add(2), uint16(v>>16)) }
func FP(p Ptr) Ptr           { return Ptr(FDW(p)) }
func SetFP(p Ptr, q Ptr)     { SetFDW(p, uint32(q)) }

// Bytes returns a copy of n bytes at p (for building Go strings, file I/O ...).
func Bytes(p Ptr, n int) []byte {
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = FB(p.Add(uint16(i)))
	}
	return out
}
func PutBytes(p Ptr, b []byte) {
	for i, c := range b {
		SetFB(p.Add(uint16(i)), c)
	}
}

// ---- the stack ----
// REENTRANT procedures keep locals on the stack.  Locals whose address is taken
// (.local / @local passed to another procedure, e.g. NewObj templates, KillObj's
// `save` record, sb buffers) must live in DGROUP so the 16-bit offsets stay valid.
// Use Push(n) at procedure entry to allocate such a block and defer Pop(mark).
// Other locals can simply be Go variables of the right width (uint8/uint16/...).
// The original SP starts at 3CFCh; the stack area is DGROUP 2F46h..3CFBh.
var sp uint16 = 0x3CFC

const stackLimit = 0x2F46

// Push allocates n bytes (rounded to even) of stack, zero-filled, returns its offset.
func Push(n int) uint16 {
	n = (n + 1) &^ 1
	if int(sp)-n < stackLimit {
		panic("stack overflow")
	}
	sp -= uint16(n)
	for i := 0; i < n; i++ {
		dg[sp+uint16(i)] = 0
	}
	return sp
}

// Mark returns the current stack pointer, for `defer Release(Mark())`.
func Mark() uint16     { return sp }
func Release(m uint16) { sp = m }

// ---- PL/M built-ins (byte-exact semantics) ----
func le16(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }

// MOVB(src, dst, n): copy n bytes ascending (overlap behaves like REP MOVSB).
func MOVB(src, dst Ptr, n uint16) {
	for i := uint16(0); i < n; i++ {
		SetFB(dst.Add(i), FB(src.Add(i)))
	}
}

// MOVRB(src, dst, n): copy n bytes descending (STD; REP MOVSB from the end).
func MOVRB(src, dst Ptr, n uint16) {
	for i := n; i > 0; i-- {
		SetFB(dst.Add(i-1), FB(src.Add(i-1)))
	}
}
func MOVW(src, dst Ptr, n uint16) { MOVB(src, dst, n*2) }
func SETB(v byte, dst Ptr, n uint16) {
	for i := uint16(0); i < n; i++ {
		SetFB(dst.Add(i), v)
	}
}
func SETW(v uint16, dst Ptr, n uint16) {
	for i := uint16(0); i < n; i++ {
		SetFW(dst.Add(2*i), v)
	}
}

// FINDB(p, v, n): index of first byte = v, or 0FFFFh.
func FINDB(p Ptr, v byte, n uint16) uint16 {
	for i := uint16(0); i < n; i++ {
		if FB(p.Add(i)) == v {
			return i
		}
	}
	return 0xFFFF
}

// FINDW(p, v, n): index of first word = v, or 0FFFFh.
func FINDW(p Ptr, v uint16, n uint16) uint16 {
	for i := uint16(0); i < n; i++ {
		if FW(p.Add(2*i)) == v {
			return i
		}
	}
	return 0xFFFF
}

// SKIPB(p, v, n): index of first byte <> v, or 0FFFFh.
func SKIPB(p Ptr, v byte, n uint16) uint16 {
	for i := uint16(0); i < n; i++ {
		if FB(p.Add(i)) != v {
			return i
		}
	}
	return 0xFFFF
}

// FINDRB / SKIPRB: search from the end; return index of the match or 0FFFFh.
func FINDRB(p Ptr, v byte, n uint16) uint16 {
	for i := n; i > 0; i-- {
		if FB(p.Add(i-1)) == v {
			return i - 1
		}
	}
	return 0xFFFF
}
func SKIPRB(p Ptr, v byte, n uint16) uint16 {
	for i := n; i > 0; i-- {
		if FB(p.Add(i-1)) != v {
			return i - 1
		}
	}
	return 0xFFFF
}

// CMPB(p1, p2, n): index of first differing byte, or 0FFFFh if equal.
func CMPB(p1, p2 Ptr, n uint16) uint16 {
	for i := uint16(0); i < n; i++ {
		if FB(p1.Add(i)) != FB(p2.Add(i)) {
			return i
		}
	}
	return 0xFFFF
}

// PL/M boolean results: TRUE = 0FFh, FALSE = 0. IF tests bit 0 (rcr al,1).
const TRUE, FALSE byte = 0xFF, 0

func Bool(c bool) byte {
	if c {
		return TRUE
	}
	return FALSE
}
func Test(b byte) bool { return b&1 != 0 }
