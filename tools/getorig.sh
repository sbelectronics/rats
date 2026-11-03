#!/bin/sh
# getorig.sh <dir> [<dir> ...]
# Finds the two Intel tools the rebuild needs under the given directories (for example a copy of
# a CTOS system or development disk) and copies them into orig/. They are not in the repository.
# Each file is accepted only if its SHA-256 matches the version this reconstruction was made with;
# a different build of the compiler may not produce byte-identical code.
#   V2.3PLM86.86   Intel PL/M-86 V2.3 compiler, CTOS UDI build "x10.1" (run by udi/plm86)
#   asm86.86       Intel ASM86 assembler V2.0 (run by udi/udi86.py)
R=$(cd "$(dirname "$0")/.." && pwd)
[ $# -gt 0 ] || { echo "usage: $0 <dir> [<dir> ...]" >&2; exit 2; }
mkdir -p "$R/orig"
fail=0
get() { name=$1; want=$2; shift 2
  f=$(find "$@" -type f -iname "$name" 2>/dev/null | while IFS= read -r f; do
        [ "$(sha256sum "$f" | cut -d' ' -f1)" = "$want" ] && { echo "$f"; break; }
      done)
  if [ -n "$f" ]; then cp "$f" "$R/orig/$name" && echo "orig/$name <- $f"
  else echo "orig/$name: no file with SHA-256 $want found" >&2; fail=1; fi
}
get V2.3PLM86.86 7682738db9f3432f541913510edb80aa7ecca3aaa1d139ee3c7021cfc12ee0e9 "$@"
get asm86.86     8efaf522ce7ba858cc735ceb8b32c284de0c1d730d6bf5406465e598e3accf6d "$@"
exit $fail
