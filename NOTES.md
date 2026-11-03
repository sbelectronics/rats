# Rats.run: file format and memory map

The binary this source reconstructs is Rats 9.2 ("Ver Rats 9.2", "Linker x9.2-6/13-15:55"),
49,664 bytes. Other builds exist in CTOS software collections and are not covered here:
- an 8.0 build, stored as a compressed "GWRz" run file;
- an older 42,423-byte build (`T1rats.Run` and `rats.run`).

## Run file
- V4 run file. The header is 50h bytes, laid out as in CTOS `runfmt.edf` (`tools/runinfo.py` prints it).
- 588 relocations start at 50h, as (ra, sa) pairs.
- The load image starts at A00h and is B800h bytes long.
- Entry point 0000:0000. SS = DS = DGROUP = paragraph 7A6h (image offset 7A60h), SP = 3CFCh.
- Code occupies image 0000-7A5F. Modules are word-aligned, and each module's frame is its start >> 4
  (e.g. the module at 0908h runs as 0090:0008).

## CTOS calls
CTOS services are far calls to magic CS:IP addresses (linear 200h-22Fh):
- **Requests** (RqLabl.mdf): CS = -(rq AND 0FFF8h), IP = 210h + (rq AND 0FFF8h)*10h + ((rq*2) AND 0Fh).
- **OS subroutines** (OsSub.Mdf): CS = 0FFFFh - 16*(n/8), IP = 230h + 100h*(n/8) + 2n - 10h*(n/8).
- **Kernel primitive Check:** `FFFF:0218` (linear 208h).

All results are mod 10000h. Request names come from CTOS RC.Edf and OsSub names from
ScInterface_All.Asm. `work/rqnames.txt` and `work/ossub.txt` list them for `tools/rdis2.py`. The
PL/M declarations of the CTOS procedural interface are in CTOS Ctoslib.edf and CtosTypes.edf.

## DGROUP
- **0000-1233:** CONST segments of all modules.
- **1234-2C99:** game variables (`src/RatsData.asm`).
- **After that:** library data.
- Points of interest:
  - **0018h:** 8 far pointers to the per-type object handlers in RatsB, indexed by `obj.type`.
  - **0658h / 066Ch:** 5 far pointers each, to the sprite Draw/Erase routines, indexed by `obj.size`.
    Entries 0 and 3 are null.
  - **1352h:** `pMap`, a far pointer to the 64K map segment. There is one byte per cell, indexed by
    the position word (low byte x, high byte y).
  - **135Ah:** `rgObj(256)`, 16-byte object records. The fields are described in the
    `src/RatsB.plm` header.
  - **235Ah:** `rgPos(1024)` WORD, 4 cell positions per object.
  - **2B5Ah / 2B7Ah:** `rgWheelHead/Tail(16)`, the 16-slot timing wheel. Then 2B9Ah `iWheel`,
    2B9Ch the free list, and 2B9Eh `pPlayer`.

`tools/symbols.txt` and `src/*.sym` give the name of every DGROUP address that the code references.
