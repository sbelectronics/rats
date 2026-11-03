#!/bin/sh
# verifyall.sh - rebuild all modules (buildall.sh), compare each module's code with the original
# (cmpx.py with its own placement and symbols), then compare the whole relinked image (link.py).
R=$(cd "$(dirname "$0")/.." && pwd)
PY=$R/venv/bin/python; [ -x "$PY" ] || PY=python3
mkdir -p "$R/work"; sh "$R/tools/buildall.sh" > "$R/work/buildall.out" 2>&1
grep -E "FAILED|all modules built" "$R/work/buildall.out"
c() { m=$1; s=$2; shift 2
      "$PY" "$R/tools/cmpx.py" "$R/work/build/$m/$m.obj" $s --syms "$R/src/$m.sym" --quiet "$@" 2>&1 | tr '\n' ' '; echo; }
c RatsA 0
c RatsB 908 --const 38
c RatsC 1df8 --const ea
c Sprites 287c
c Scores 2930 --const 9ee
c Maze 2fb0
c Sleep 32a6 --data 2c9a
c Mod3396 3396 --data 2cb0
c Mod35AC 35ac
c Thunk3616 3616 --data 2cb4
c DateTime 362a
c CheckErc 3cb8 --const af4
c Forms 3d1c --const afe
c FormFields 5286 --const c7c
c Mod54E4 54e4
c Font 552e --const c7e
c Mod55FC 55fc --const c92
c Random 577c --const c96 --data 2cb8
c ErrMsg 591a --const cdc --data 2d02
c Strings 5cd2 --const d82
c VidSel 6bb4 --const 10e0 --data 2f10
c VidInit 6c0e --const 10e6 --data 2f12
c VidThk 6f28
c VidDrvA 6f80
c VidDrvB 7200
echo "--- full relink"
"$PY" "$R/tools/link.py" | grep -v "^!!"
