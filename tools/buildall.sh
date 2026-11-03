#!/bin/sh
# buildall.sh - build every Rats module from src/ into work/build/<Module>/<Module>.obj
# (PL/M-86 via tools/build.sh, ASM86 via udi/udi86.py), then relink with tools/link.py.
# Needs the Intel compiler and assembler in orig/ (see tools/getorig.sh).
R=$(cd "$(dirname "$0")/.." && pwd)
PY=$R/venv/bin/python; [ -x "$PY" ] || PY=python3
ASM86=$R/orig/asm86.86
fail=0
for m in RatsA RatsB RatsC Scores Maze Sleep Mod3396 Mod35AC DateTime Forms FormFields \
         Mod54E4 Font Random ErrMsg Strings VidSel VidInit VidThk; do
  out=$(sh "$R/tools/build.sh" $m 2>&1 | tail -1)
  case "$out" in rc=0*) ;; *) echo "PLM $m FAILED: $out"; fail=1;; esac
done
for m in RatsData RatsTables VidTab Sprites Thunk3616 CheckErc Mod55FC VidDrvA VidDrvB; do
  d=$R/work/build/$m; mkdir -p "$d"; cd "$d" || exit 9
  rm -f ./*.obj ./*.OBJ; cp "$R/src/$m.asm" .
  "$PY" "$R/udi/udi86.py" "$ASM86" "ASM86 $m.asm" > $m.out 2>&1
  # ASM86 may write the object name in upper case
  o=$(ls | grep -i "^$m.obj$")
  if [ -z "$o" ]; then echo "ASM $m FAILED"; fail=1; else [ "$o" = "$m.obj" ] || mv "$o" $m.obj; fi
done
[ $fail = 0 ] && echo "all modules built"
"$PY" "$R/tools/link.py"
