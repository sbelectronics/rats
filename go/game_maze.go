package main

// game_maze.go - translation of ../src/Maze.plm (frame 02FB): GenMaze, Eller's
// algorithm one row at a time.  pProc(x, y, fHoriz) is called for each wall
// (the original receives it as a far procedure pointer; here it is a Go func).
// Note PL/M does not short-circuit: "Random(0) < density AND (...)" always calls Random.

func GenMaze(w, h uint16, modeLeft, modeRight, modeTop, modeBottom byte, density uint16,
	pProc func(x, y uint16, fHoriz byte)) {
	defer Release(Mark())
	set := Push(512) // set(256) WORD - its address is passed to FINDW
	setW := func(i uint16) uint16 { return W(set + 2*i) }
	setSetW := func(i, v uint16) { SetW(set+2*i, v) }
	var nextSet, y, x, sTo, sFrom, j, sOld uint16
	var fWall byte

	// top border; every cell of the first row is its own set
	x = 0
	for x < w {
		switch modeTop {
		case 0:
			fWall = 0xFF
		case 1:
			fWall = Bool(Random(0) < density)
		case 2:
			fWall = Bool(Random(0) < density)
		}
		if Test(fWall) {
			pProc(x, 0, 0xFF)
		}
		setSetW(x, x)
		x = x + 1
	}
	nextSet = w
	y = 0
	for y < h {
		switch modeLeft {
		case 0:
			fWall = 0xFF
		case 1:
			fWall = Bool(Random(0) < density)
		case 2:
			fWall = Bool(Random(0) < density)
		}
		if Test(fWall) {
			pProc(0, y, 0)
		}
		// vertical walls inside the row
		x = 0
		for x < w-1 {
			r := Random(0) < density // evaluated first and always
			if r && (y < h-1 || setW(x) == setW(x+1)) {
				pProc(x+1, y, 0)
			} else {
				// no wall: merge set(x+1) into set(x)
				sFrom = setW(x + 1)
				sTo = setW(x)
				j = 0
				for j < w {
					if setW(j) == sFrom {
						setSetW(j, sTo)
					}
					j = j + 1
				}
			}
			x = x + 1
		}
		switch modeRight {
		case 0:
			fWall = 0xFF
		case 1:
			fWall = Bool(Random(0) < density)
		case 2:
			fWall = 0
		}
		if Test(fWall) {
			pProc(w, y, 0)
		}
		// horizontal walls below the row
		if y < h-1 {
			x = 0
			for x < w {
				if Random(0) < density {
					sOld = setW(x)
					setSetW(x, nextSet)
					if FINDW(DG(set), sOld, w) == 0xFFFF {
						setSetW(x, sOld) // last cell of its set: keep open
					} else {
						pProc(x, y+1, 0xFF)
						nextSet = nextSet + 1
					}
				}
				x = x + 1
			}
		}
		y = y + 1
	}
	// bottom border
	x = 0
	for x < w {
		switch modeBottom {
		case 0:
			fWall = 0xFF
		case 1:
			fWall = Bool(Random(0) < density)
		case 2:
			fWall = 0
		}
		if Test(fWall) {
			pProc(x, h, 0xFF)
		}
		x = x + 1
	}
}
