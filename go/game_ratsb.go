package main

// game_ratsb.go - translation of ../src/RatsB.plm (frame 0090): object behaviours.
// Object types: 0 player, 1 nest, 2 young rat, 3 rat, 4 small rat, 5 shot, 6 dying
// object (ghost), 7 blast fragment.  See the header of RatsB.plm for the fields.
// Templates (tmpl/save) and Probe result bytes (.b) have their address taken, so they
// live on the emulated stack in DGROUP.

// newTmpl fills a 16-byte template record at t (link is left 0).
func setTmpl(t uint16, delay, ctr, typ, size, b6, dir, flags, b9 byte, wA, wC, wE uint16) {
	SetB(t+obDelay, delay)
	SetB(t+obCtr, ctr)
	SetB(t+obType, typ)
	SetB(t+obSize, size)
	SetB(t+obB6, b6)
	SetB(t+obDir, dir)
	SetB(t+obFlags, flags)
	SetB(t+obB9, b9)
	SetW(t+obWA, wA)
	SetW(t+obWC, wC)
	SetW(t+obWE, wE)
}

// NewPlayer: player (type 0); fNew = the real player (else an autopilot clone).
func NewPlayer(pos uint16, dir, flags, fNew byte) {
	defer Release(Mark())
	tmpl := Push(obLen)
	setTmpl(tmpl, 2, 0, 0, 4, 5, dir, flags|0x20, 0, 0, 0, 0)
	p := NewObj(tmpl, pos, 0)
	if p != 0 {
		if Test(fNew) {
			SetW(gPPlayer, p)
			SetW(gWKeysHeld, W(gWKeysHeld)&^W(gWKeysUp))
			SetW(gWKeysHit, W(gWKeysHit)&^W(gWKeysUp))
			SetW(gWKeysUp, 0)
		}
		SetW(gCLives, W(gCLives)-1)
		UpdateStatus()
	}
}

// NewNest: rat nest (type 1).
func NewNest(pos uint16, delay byte) {
	defer Release(Mark())
	tmpl := Push(obLen)
	setTmpl(tmpl, B(gRgbDelay+0), 0, 1, 4, 7, 2, 0x20, 0, 0, 0, 0)
	p := NewObj(tmpl, pos, delay)
	if p != 0 {
		SetW(gCFactLeft, W(gCFactLeft)+1)
		UpdateStatus()
	}
}

// NewYoung: young rat (type 2) spawned by nest pOwner with value val.
func NewYoung(pos, val, pOwner uint16) {
	defer Release(Mark())
	tmpl := Push(obLen)
	var w, v, wA, p, q, r, s, t uint16
	var fBig, delay byte
	w = W(gWDifficulty)
	v = (w>>10 + 10) * 5
	if val < v || val-v < 50 {
		w = (val/5 - 10) << 10
		v = (w>>10 + 10) * 5
	}
	fBig = Bool(Random(0) < w>>3)
	if Test(fBig) {
		q = v / 25
		r = v % 25
		s = q * 5
		t = s + r
		_ = t
		w = w >> 1
	}
	delay = B(gRgbDelay + w>>12)
	wA = w >> 2
	SetB(tmpl+obDelay, delay)
	SetB(tmpl+obCtr, 0)
	SetB(tmpl+obType, 2)
	SetB(tmpl+obSize, 2)
	SetB(tmpl+obB6, 3)
	SetB(tmpl+obDir, 2)
	SetB(tmpl+obFlags, byte(Random(0)&8))
	if Test(fBig) {
		SetB(tmpl+obFlags, B(tmpl+obFlags)|0x10)
	}
	SetB(tmpl+obB9, 0)
	SetW(tmpl+obWA, wA)
	SetW(tmpl+obWC, v)
	SetW(tmpl+obWE, pOwner)
	p = NewObj(tmpl, pos, byte(Random(uint16(B(tmpl+obDelay)))))
	if p != 0 {
		if pOwner != 0 {
			SetW(pOwner+obWA, W(pOwner+obWA)+W(tmpl+obWC))
		}
		SetW(gCRatsLeft, W(gCRatsLeft)+1)
		UpdateStatus()
	}
}

// NewRat: small rat (type 4) at pos+delta.
func NewRat(pos, delta uint16, dir, flags byte, val, pOwner uint16) {
	defer Release(Mark())
	tmpl := Push(obLen)
	var w uint16
	pos = gMkPos(gLo(pos)+gLo(delta), gHi(pos)+gHi(delta))
	if val<<1 >= 50 {
		w = ((val<<1)/5 - 10) << 10
	} else {
		w = 0
	}
	setTmpl(tmpl, B(gRgbDelay+w>>12), 0, 4, 1, 1, dir, flags&8, 0, 0, val, pOwner)
	p := NewObj(tmpl, pos, byte(Random(uint16(B(tmpl+obDelay)))))
	if p != 0 {
		SetW(gCCrazyLeft, W(gCCrazyLeft)+1)
		UpdateStatus()
	}
}

// NewShot: shot (type 5) at pos+delta flying in dir.
func NewShot(pos, delta uint16, dir byte, pOwner uint16) {
	defer Release(Mark())
	tmpl := Push(obLen)
	pb := Push(2) // b
	var pHit uint16
	pos = gMkPos(gLo(pos)+gLo(delta), gHi(pos)+gHi(delta))
	for gMap(pos) != 0 {
		pHit = FindObjAt(pos)
		if pHit != 0 {
			if pHit == pOwner {
				pos = Probe(pos, dir, 0, pb)
			} else {
				if B(pHit+obType) < 6 {
					KillObj(pHit)
				}
				return
			}
		} else {
			return
		}
	}
	setTmpl(tmpl, 1, 0, 5, 1, 0, dir, 0, 0, 0, 0, 0)
	NewObj(tmpl, pos, 0)
}

// NewGhost: dying object (type 6) replacing a killed one (called from KillObj).
func NewGhost(pos uint16, size, b6, flags, bMap byte, w14, pOwner uint16, fKind byte) {
	defer Release(Mark())
	tmpl := Push(obLen)
	setTmpl(tmpl, 2, 0, 6, size, b6, 8, flags, bMap, uint16(fKind), w14, pOwner)
	NewObj(tmpl, pos, 1)
}

// NewBlast: blast fragment (type 7) at pos+delta.
func NewBlast(pos, delta uint16, ctr, size, dir, b9 byte) {
	defer Release(Mark())
	tmpl := Push(obLen)
	var pHit uint16
	pos = gMkPos(gLo(pos)+gLo(delta), gHi(pos)+gHi(delta))
	if gMap(pos) != 0 {
		pHit = FindObjAt(pos)
		if pHit != 0 {
			if B(pHit+obType) < 6 {
				KillObj(pHit)
			}
		}
		return
	}
	setTmpl(tmpl, 2, ctr, 7, 1, size, dir, 0, b9, 0, 0, 0)
	NewObj(tmpl, pos, 1)
}

// Probe: try to step from pos in direction dir (8 = object's own direction); *pb gets
// the first non-empty map cell (0 = free).  Returns the new position.
func Probe(pos uint16, dir byte, p, pb uint16) uint16 {
	if dir == 8 {
		dir = B(p + obDir)
		if B(p+obCtr)&1 == 0 {
			dir = B(gRgbDirAlt + uint16(dir))
		}
	}
	if p != 0 {
		if B(p+obSize) > 1 {
			dir = dir + 8 + 1
		}
	}
	x, y := gLo(pos), gHi(pos)
	var b byte
	m := func() byte { b = gMap(gMkPos(x, y)); SetB(pb, b); return b }
	zero := func() { b = 0; SetB(pb, 0) }
	switch dir {
	case 0: // N
		y--
		m()
	case 1: // NE
		x++
		if m() == 0 {
			y--
			m()
		}
	case 2: // E
		x++
		m()
	case 3: // SE
		x++
		if m() == 0 {
			y++
			m()
		}
	case 4: // S
		y++
		m()
	case 5: // SW
		x--
		if m() == 0 {
			y++
			m()
		}
	case 6: // W
		x--
		m()
	case 7: // NW
		x--
		if m() == 0 {
			y--
			m()
		}
	case 8: // none
		zero()
	case 9: // N (2x2)
		x++
		y--
		if m() == 0 {
			x--
			m()
		}
	case 10: // NE (2x2)
		x += 2
		y++
		if m() == 0 {
			y--
			if m() == 0 {
				y--
				if m() == 0 {
					x--
					m()
				}
			}
		}
	case 11: // E (2x2)
		x += 2
		y++
		if m() == 0 {
			y--
			if m() == 0 {
				x--
			}
		}
	case 12: // SE (2x2)
		x += 2
		if m() == 0 {
			y++
			if m() == 0 {
				y++
				if m() == 0 {
					x--
					if m() == 0 {
						y--
					}
				}
			}
		}
	case 13: // S (2x2)
		y += 2
		x++
		if m() == 0 {
			x--
			if m() == 0 {
				y--
			}
		}
	case 14: // SW (2x2)
		y += 2
		if m() == 0 {
			x--
			if m() == 0 {
				y--
				if m() == 0 {
					y--
					if m() == 0 {
						y++
					}
				}
			}
		}
	case 15: // W (2x2)
		x--
		y++
		if m() == 0 {
			y--
			m()
		}
	case 16: // NW (2x2)
		x--
		y++
		if m() == 0 {
			y--
			if m() == 0 {
				y--
				if m() == 0 {
					x++
					if m() == 0 {
						x--
					}
				}
			}
		}
	case 17: // none (2x2)
		zero()
	}
	return gMkPos(x, y)
}

// PlayerTick: handler for type 0 (player, 2x2).  Frame 90h offset 06DDh.
func PlayerTick(pos, p uint16) {
	defer Release(Mark())
	pb := Push(2) // b
	var newPos, pTarget, tpos, r, idx uint16
	var bDir, fireDir, dirX byte
	if p == W(gPPlayer) {
		if W(gWKeysHeld)|W(gWKeysHit) != 0 {
			SetB(p+obFlags, B(p+obFlags)&0xFE)
			SetB(p+obB9, 0)
		}
	}
	if W(p+obWA) != 0 {
		SetW(p+obWA, W(p+obWA)-1)
	}
	if B(p+obFlags)&1 != 0 {
		SetB(p+obFlags, B(p+obFlags)|0x40)
		if B(p+obB9) != 0 {
			newPos = Turn(pos, p)
		} else {
			pTarget = FindTarget(pos)
			if pTarget == 0 {
				r = Random(100)
				if r == 0 {
					bDir = (B(p+obDir)+1)&7 + 8*16
				} else if r == 1 {
					bDir = (B(p+obDir)-1)&7 + 8*16
				} else {
					bDir = B(p+obDir) + 8*16
				}
			} else {
				tpos = W(gPosSlot(pTarget))
				idx = DirIndex(pos, tpos)
				bDir = B(gRgbSeekPlayer + idx)
			}
			SetB(p+obDir, bDir&7)
			newPos = Probe(pos, 8, p, pb)
			if B(pb) == 0 {
				fireDir = bDir >> 4
				if fireDir != 8 {
					if W(p+obWA) == 0 {
						Fire(p, newPos, fireDir)
						SetW(p+obWA, W(gWShotReload))
					}
				}
			} else {
				newPos = Turn(pos, p)
			}
		}
	} else {
		SetW(gWKeysHeld, W(gWKeysHeld)|W(gWKeysHit))
		bDir = B(gRgbKeyDir + W(gWKeysHeld)&0x0F)
		if bDir != 8 {
			SetB(p+obDir, bDir)
		}
		if W(gWKeysHeld)&0x40 != 0 {
			SetB(p+obFlags, B(p+obFlags)|0x40)
			if W(gWKeysHit)&0x40 != 0 {
				SetB(p+obCtr, 1)
			}
			newPos = Probe(pos, 8, p, pb)
			if B(pb) != 0 {
				goto done
			}
		} else {
			SetB(p+obFlags, B(p+obFlags)&0xBF)
			newPos = pos
		}
		if W(gWKeysHeld)&0x80 != 0 {
			switch (W(gWKeysHeld) & 0x30) >> 4 {
			case 0:
				dirX = B(p + obDir)
			case 1:
				dirX = (B(p+obDir) - 2) & 7
			case 2:
				dirX = (B(p+obDir) + 2) & 7
			case 3:
				dirX = B(p+obDir) ^ 4
			}
			if W(gWKeysHit)&0x80 != 0 || W(p+obWA) == 0 {
				Fire(p, newPos, dirX)
				SetW(p+obWA, W(gWShotReload))
			}
		}
		if W(gWKeysHeld)&0x200 != 0 {
			if W(gWKeysHit)&0x200 != 0 || W(p+obWA) == 0 {
				SpawnClone(newPos, B(p+obDir))
				SetW(p+obWA, W(gWShotReload))
			}
		}
		if W(gWKeysHeld)&0x100 != 0 {
			SetB(p+obFlags, B(p+obFlags)|1)
		}
	done:
		SetW(gWKeysHeld, W(gWKeysHeld)&^W(gWKeysUp))
		SetW(gWKeysHit, 0)
		SetW(gWKeysUp, 0)
	}
	MoveCur(newPos)
}

// Fire a shot from object p (at pos) in direction dir.
func Fire(p, pos uint16, dir byte) {
	var delta uint16
	if B(p+obFlags)&8 != 0 {
		delta = W(gRgwShotDelta + 2*uint16(dir))
	} else {
		delta = W(gRgwShotDelta2 + 2*uint16(dir))
	}
	NewShot(pos, delta, dir, p)
}

// SpawnClone: autopilot clone 16 columns / 8 rows away in direction dir.
func SpawnClone(pos uint16, dir byte) {
	x, y := gLo(pos), gHi(pos)
	switch dir {
	case 0:
		y -= 8
	case 1:
		x += 16
		y -= 8
	case 2:
		x += 16
	case 3:
		x += 16
		y += 8
	case 4:
		y += 8
	case 5:
		x -= 16
		y += 8
	case 6:
		x -= 16
	case 7:
		x -= 16
		y -= 8
	}
	pos = gMkPos(x, y)
	if W(gCLives) != 0 {
		if Test(IsFree(pos, 4)) {
			NewPlayer(pos, dir, byte(Random(0)&8|1), 0)
		}
	}
}

// NestTick: handler for type 1 (rat nest, 2x2).  Frame 90h offset 0A0Ah.
func NestTick(pos, p uint16) {
	var half, room, alive, killed, spawned, q uint16
	spawned = W(p + obWA)
	killed = W(p + obWC)
	alive = spawned - killed
	room = W(gWFactValue) - spawned
	half = W(gWFactValue) >> 1
	if half > alive && room != 0 {
		q = gMkPos(gLo(pos)+2, gHi(pos)+1)
		if gMap(q) == 0 {
			q = gSetHi(q, gHi(q)-1)
			if gMap(q) == 0 {
				NewYoung(q, room, p)
			}
		}
	}
	SetB(p+obDelay, B(gRgbDelay+W(gWDifficulty)>>12))
	MoveCur(pos)
}

// YoungTick: handler for type 2 (young rat, 1x2).  Frame 90h offset 0A9Bh.
func YoungTick(pos, p uint16) {
	defer Release(Mark())
	save := Push(obLen)
	var q uint16
	q = pos
	q = gSetLo(q, gLo(q)+1)
	if gMap(q) == 0 {
		q = gSetHi(q, gHi(q)+1)
		if gMap(q) == 0 {
			FreeObj(save, p)
			SetB(save+obType, 3)
			SetB(save+obSize, 4)
			if B(save+obFlags)&0x10 == 0 {
				SetB(save+obB6, 1)
			}
			p = NewObj(save, pos, B(save+obDelay)-1)
			return
		}
	}
	MoveCur(pos)
}

// RatTick: handler for types 3 (rat, 2x2) and 4 (small rat, 1x1).  Frame 90h offset 0B0Dh.
func RatTick(pos, p uint16) {
	defer Release(Mark())
	pb := Push(2) // b
	var pOwn, newPos, r, delta, idx, ownerPos, delta2 uint16
	var bDir, fireDir, ownerDir byte
	pOwn = W(p + obWE)
	if Test(Bool(pOwn == 0) & B(gFCrazyRats)) {
		// orphan rat: random walk, shoots at random
		r = Random(7)
		if B(p+obFlags)&8 == 0 {
			r = r + 2
		}
		SetB(p+obDir, (B(p+obDir)+B(gRgbTurn+r))&7)
		newPos = Probe(pos, 8, p, pb)
		if B(pb) == 0 {
			if B(p+obCtr)&1 == 0 {
				if Random(0) < W(p+obWA) {
					if B(p+obFlags)&8 != 0 {
						delta = W(gRgwRatDelta + 2*uint16(B(p+obDir)))
					} else {
						delta = W(gRgwRatDelta2 + 2*uint16(B(p+obDir)))
					}
					NewShot(newPos, delta, B(p+obDir), p)
				}
			}
		}
	} else if B(p+obB9) != 0 {
		newPos = Turn(pos, p)
	} else if Test(CanSee(pos, B(p+obSize))) {
		// player in view: chase him and shoot
		idx = DirIndex(pos, W(gPosPlayer))
		if B(p+obType) == 3 {
			bDir = B(gRgbSeek3 + idx)
		} else {
			bDir = B(gRgbSeek4 + idx)
		}
		SetB(p+obDir, bDir&7)
		newPos = Probe(pos, 8, p, pb)
		if B(pb) >= 0xC0 {
			goto move
		}
		if B(pb) == 0 {
			fireDir = bDir >> 4
			if fireDir != 8 {
				if B(p+obCtr)&1 == 0 {
					if Random(0) < W(p+obWA) {
						if pOwn == 0 {
							ownerDir = 8
						} else {
							ownerPos = W(gPosSlot(pOwn))
							idx = DirIndex(pos, ownerPos)
							ownerDir = B(gRgbSeek3+idx) >> 4
						}
						if fireDir != ownerDir {
							if B(p+obFlags)&8 != 0 {
								delta2 = W(gRgwRatDelta + 2*uint16(fireDir))
							} else {
								delta2 = W(gRgwRatDelta2 + 2*uint16(fireDir))
							}
							NewShot(newPos, delta2, fireDir, p)
						}
					}
				}
			}
		} else {
			newPos = Turn(pos, p)
		}
	} else {
		// wander
		idx = Random(7)
		if B(p+obFlags)&8 == 0 {
			idx = idx + 2
		}
		SetB(p+obDir, (B(p+obDir)+B(gRgbTurn+idx))&7)
		newPos = Probe(pos, 8, p, pb)
		if B(pb) != 0 {
			newPos = pos
			SetB(p+obDir, B(p+obDir)^4)
		}
	}
move:
	MoveCur(newPos)
}

// ShotTick: handler for type 5 (shot).  Frame 90h offset 0D67h.
func ShotTick(pos, p uint16) {
	defer Release(Mark())
	pb := Push(2)
	MoveCur(Probe(pos, 8, p, pb))
}

// GhostTick: handler for type 6 (dying object).  Frame 90h offset 0D8Ch.
func GhostTick(pos, p uint16) {
	defer Release(Mark())
	save := Push(obLen)
	cells := Push(8)
	var cell uint16
	var ctr, b6 byte
	ctr = B(p + obCtr)
	b6 = B(p + obB6)
	if ctr >= 3 || b6 == 0 {
		FreeObj(save, p)
		if B(save+obFlags)&0x10 != 0 {
			// pregnant rat: releases four small rats
			NewRat(pos, 0, 7, B(save+obFlags), W(save+obWC), W(save+obWE))
			NewRat(pos, 1, 1, B(save+obFlags), W(save+obWC), W(save+obWE))
			NewRat(pos, 0x101, 3, B(save+obFlags), W(save+obWC), W(save+obWE))
			NewRat(pos, 0x100, 5, B(save+obFlags), W(save+obWC), W(save+obWE))
		}
		if W(save+obWA) != 0xFF {
			for {
				cell = RandomCell()
				if !Test(^IsFree(cell, 4)) {
					break
				}
			}
			switch W(save + obWA) {
			case 0:
				NewPlayer(cell, 4, byte(Random(0)&8|1), 0)
			case 1:
				NewNest(cell, 0)
			}
		}
		if B(save+obB9) != 0 {
			callDraw(B(save+obSize), B(save+obB9), pos, cells)
		}
		return
	}
	if ctr >= b6 {
		goto done
	}
	if B(p+obSize) == 1 {
		NewBlast(pos, 1, ctr, b6, 2, 3)
		NewBlast(pos, 0xFF, ctr, b6, 6, 9)
		if ctr == 1 {
			NewBlast(pos, 0xFF00, ctr, b6, 0, 0)
			NewBlast(pos, 0x100, ctr, b6, 4, 6)
		}
	} else if B(p+obSize) == 2 {
		NewBlast(pos, 1, ctr, b6, 2, 2)
		NewBlast(pos, 0x101, ctr, b6, 2, 4)
		NewBlast(pos, 0x1FF, ctr, b6, 6, 8)
		NewBlast(pos, 0xFF, ctr, b6, 6, 0x0A)
		if ctr == 1 {
			NewBlast(pos, 0xFF00, ctr, b6, 0, 0)
			NewBlast(pos, 0x200, ctr, b6, 4, 6)
		}
	} else {
		NewBlast(pos, 2, ctr, b6, 2, 2)
		NewBlast(pos, 0x102, ctr, b6, 2, 4)
		NewBlast(pos, 0x1FF, ctr, b6, 6, 8)
		NewBlast(pos, 0xFF, ctr, b6, 6, 0x0A)
		if ctr == 1 {
			NewBlast(pos, 0xFF01, ctr, b6, 0, 1)
			NewBlast(pos, 0x201, ctr, b6, 4, 5)
			NewBlast(pos, 0x200, ctr, b6, 4, 7)
			NewBlast(pos, 0xFF00, ctr, b6, 0, 0x0B)
		}
	}
done:
	MoveCur(pos)
}

// BlastTick: handler for type 7 (blast fragment).  Frame 90h offset 1073h.
func BlastTick(pos, p uint16) {
	defer Release(Mark())
	save := Push(obLen)
	pb := Push(2)
	var ctr, b6 byte
	ctr = B(p + obCtr)
	b6 = B(p + obB6)
	if ctr >= b6 {
		FreeObj(save, p)
		return
	}
	pos = Probe(pos, 8, p, pb)
	odd := ctr&1 != 0 && ctr != 1
	if B(pb) == 0 {
		switch B(p + obB9) {
		case 0:
			NewBlast(pos, 1, ctr, b6, 1, 0x0C)
			NewBlast(pos, 0xFF, ctr, b6, 7, 0x0C)
		case 1:
			NewBlast(pos, 1, ctr, b6, 1, 0x0C)
		case 2:
			if odd {
				NewBlast(pos, 0xFF00, ctr, b6, 1, 0x0C)
			}
		case 3:
			if odd {
				NewBlast(pos, 0xFF00, ctr, b6, 1, 0x0C)
				NewBlast(pos, 0x100, ctr, b6, 3, 0x0C)
			}
		case 4:
			if odd {
				NewBlast(pos, 0x100, ctr, b6, 3, 0x0C)
			}
		case 5:
			NewBlast(pos, 1, ctr, b6, 3, 0x0C)
		case 6:
			NewBlast(pos, 1, ctr, b6, 3, 0x0C)
			NewBlast(pos, 0xFF, ctr, b6, 5, 0x0C)
		case 7:
			NewBlast(pos, 0xFF, ctr, b6, 5, 0x0C)
		case 8:
			if odd {
				NewBlast(pos, 0x100, ctr, b6, 5, 0x0C)
			}
		case 9:
			if odd {
				NewBlast(pos, 0x100, ctr, b6, 5, 0x0C)
				NewBlast(pos, 0xFF00, ctr, b6, 7, 0x0C)
			}
		case 10:
			if odd {
				NewBlast(pos, 0xFF00, ctr, b6, 7, 0x0C)
			}
		case 11:
			NewBlast(pos, 0xFF, ctr, b6, 7, 0x0C)
		case 12:
		}
	}
	MoveCur(pos)
}

// Turn: obstacle avoidance (b9 = number of 45-degree turns away from the heading).
func Turn(pos, p uint16) uint16 {
	defer Release(Mark())
	pb := Push(2)
	var step, d, n, lim byte
	if B(p+obFlags)&8 != 0 {
		step = 1
	} else {
		step = 0xFF
	}
	d = B(p + obDir)
	n = B(p + obB9)
	if n != 0 {
		for n != 0 {
			d = (d - step) & 7
			n = n - 1
			Probe(pos, d, p, pb)
			if B(pb) != 0 {
				goto away
			}
		}
		goto ok
	}
away:
	lim = n + 8
again:
	d = (step + d) & 7
	n = n + 1
	if n >= lim {
		goto fail
	}
	Probe(pos, d, p, pb)
	if B(pb) != 0 {
		goto again
	}
ok:
	SetB(p+obDir, d)
	SetB(p+obB9, n)
	return Probe(pos, 8, p, pb)
fail:
	return pos
}

// CanSee: TRUE if pos lies inside the view window (bViewX/Y, wViewW/H).
func CanSee(pos uint16, size byte) byte {
	var dx, dy byte
	if W(gPPlayer) == 0 {
		return FALSE
	}
	dx = gLo(pos) - B(gBViewX)
	dy = gHi(pos) - B(gBViewY)
	if size > 1 {
		if dx == 0xFF {
			dx = 0
		}
		if dy == 0xFF {
			dy = 0
		}
	}
	return Bool(uint16(dx) < W(gWViewW)) & Bool(uint16(dy) < W(gWViewH))
}

// DirIndex: index into the 20x16 rgbSeek tables for the vector from b to a.
func DirIndex(a, b uint16) uint16 {
	var dx, dy byte
	dx = gLo(a) - gLo(b)
	dy = gHi(a) - gHi(b)
	for (dx > 10 && dx <= 0xF6) || (dy > 8 && dy <= 0xF8) {
		dx = dx >> 1
		if dx >= 0x40 {
			dx = dx + 0x80
		}
		dy = dy >> 1
		if dy >= 0x40 {
			dy = dy + 0x80
		}
	}
	return uint16(dy+7)*20 + uint16(dx+9)
}

// FindTarget: autopilot - most attractive target (rat/nest) in view.
func FindTarget(pos uint16) uint16 {
	var pBest, best, iPos, i, q, score, dist uint16
	i, iPos, best, pBest = 0, 0, 0, 0
	q = gRgObj
	for i < 256 {
		if B(q+obType) < 8 {
			switch B(q + obType) {
			case 0:
				score = 0
			case 1:
				if Test(CanSee(W(gRgPos+2*iPos), B(q+obSize))) {
					dist = Distance(pos, W(gRgPos+2*iPos))
					score = 0x70 - dist<<4
				} else {
					score = 0
				}
			case 2, 3, 4:
				if Test(CanSee(W(gRgPos+2*iPos), B(q+obSize))) {
					dist = Distance(pos, W(gRgPos+2*iPos))
					score = uint16(B(gRgbNearScore+uint16(B(q+obDelay)))) - dist<<4
				} else {
					score = 0
				}
			case 5, 6, 7:
				score = 0
			}
			if best < score {
				pBest = q
				best = score
			}
		}
		iPos = iPos + 4
		i = i + 1
		q = q + 16
	}
	return pBest
}

// Distance class: log2(max(|dx|/2, |dy|)) + 1 (0 = same cell).
func Distance(a, b uint16) uint16 {
	var n uint16
	var dx, dy byte
	dx = gLo(a) - gLo(b)
	if dx > 0x80 {
		dx = -dx
	}
	dy = gHi(a) - gHi(b)
	if dy > 0x80 {
		dy = -dy
	}
	dx = dx >> 1
	if dx < dy {
		dx = dy
	}
	n = 0
	for dx != 0 {
		n = n + 1
		dx = dx >> 1
	}
	return n
}
