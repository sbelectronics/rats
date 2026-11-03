package main

// game_sprites.go - translation of ../src/Sprites.asm (frame 0287h): draw/erase
// 1-, 2- and 4-cell sprites into the 64K map (pMap, DGROUP 1352h).  Reached only
// through rgpDraw (0658h) / rgpErase (066Ch), indexed by obj.size.
// The map address is ES:[BX+DI] with ES:DI = pMap and BX = pos (16-bit wrap);
// INC BL/DEC BL step x within the row, INC BH steps y; AL (the cell code) wraps at 8 bits.

func init() {
	RegisterFar(0x0287, 0x000C, Draw1)
	RegisterFar(0x0287, 0x0023, Draw2)
	RegisterFar(0x0287, 0x0044, Draw4)
	RegisterFar(0x0287, 0x0079, Erase1)
	RegisterFar(0x0287, 0x008A, Erase2)
	RegisterFar(0x0287, 0x00A0, Erase4)
}

// Draw1: map(pos) = ch; rgPos slot (near ptr pPos) = pos
func Draw1(ch byte, pos uint16, pPos uint16) {
	gSetMap(pos, ch)
	SetW(pPos, pos)
}

// Draw2: cells (x,y), (x,y+1); code incremented per cell
func Draw2(ch byte, pos uint16, pPos uint16) {
	gSetMap(pos, ch)
	SetW(pPos, pos)
	ch++
	pos = gSetHi(pos, gHi(pos)+1)
	pPos += 2
	gSetMap(pos, ch)
	SetW(pPos, pos)
}

// Draw4: 2x2 sprite, cells (x,y) (x+1,y) (x+1,y+1) (x,y+1)
func Draw4(ch byte, pos uint16, pPos uint16) {
	gSetMap(pos, ch)
	SetW(pPos, pos)
	ch++
	pos = gSetLo(pos, gLo(pos)+1)
	pPos += 2
	gSetMap(pos, ch)
	SetW(pPos, pos)
	ch++
	pos = gSetHi(pos, gHi(pos)+1)
	pPos += 2
	gSetMap(pos, ch)
	SetW(pPos, pos)
	ch++
	pos = gSetLo(pos, gLo(pos)-1)
	pPos += 2
	gSetMap(pos, ch)
	SetW(pPos, pos)
}

func Erase1(pos uint16) {
	gSetMap(pos, 0)
}

func Erase2(pos uint16) {
	gSetMap(pos, 0)
	pos = gSetHi(pos, gHi(pos)+1)
	gSetMap(pos, 0)
}

func Erase4(pos uint16) {
	gSetMap(pos, 0)
	pos = gSetLo(pos, gLo(pos)+1)
	gSetMap(pos, 0)
	pos = gSetHi(pos, gHi(pos)+1)
	gSetMap(pos, 0)
	pos = gSetLo(pos, gLo(pos)-1)
	gSetMap(pos, 0)
}
