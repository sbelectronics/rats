# Original files

`Rats.run` is the unmodified original Rats 9.2 for CTOS, the program this source reconstructs. It is
Convergent Technologies software, included for preservation and so that the reconstruction can be
checked. `sha256sum -c SHA256SUMS` checks that it is intact. It is used by:
- `tools/link.py`, which takes the run-file header and the CTOS runtime tail from it and compares
  the rebuilt `Rats.run` with it;
- `tools/cmpx.py` and `tools/rdis2.py`.

The rebuild also needs two Intel tools, which are **not** in the repository:

| file | what | used by |
|---|---|---|
| `V2.3PLM86.86` | Intel PL/M-86 compiler V2.3, CTOS UDI build ("x10.1") | `udi/plm86` |
| `asm86.86` | Intel ASM86 assembler V2.0 | `tools/buildall.sh` via `udi/udi86.py` |

Both ship on CTOS system and development disks. `tools/getorig.sh <dir>...` finds them in a copy of
such disks, checks their SHA-256 against the versions used here, and copies them into this directory.
