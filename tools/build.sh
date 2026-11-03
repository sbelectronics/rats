#!/bin/sh
# build.sh <Module> [extra compiler controls]
# Compiles src/<Module>.plm in work/build/<Module>/ and prints any errors or warnings.
R=$(cd "$(dirname "$0")/.." && pwd)
m=$1; shift
d=$R/work/build/$m; mkdir -p "$d"; cd "$d" || exit 9
cp "$R/src/$m.plm" "$m.plm"
"$R/udi/plm86" -I "$R/src" "$m.plm" NOTYPE CODE "$@" > "$m.out" 2>&1
rc=$?
grep -E "ERROR|WARNING" "$m.lst" 2>/dev/null | grep -v "^ *0 PROGRAM" | head -30
[ $rc -ne 0 ] && tail -5 "$m.out"
echo "rc=$rc  obj=$d/$m.obj  lst=$d/$m.lst"
