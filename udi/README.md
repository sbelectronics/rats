# udi86 / plm86 - Intel PL/M-86 V2.3 on Linux

This runs the original Intel `V2.3PLM86.86` compiler, the one CTOS ran through
`udi.run` ("x10.1 (17May88)"), in a unicorn 8086 emulator. It produces `.obj`
files that are byte-identical to the ones the original toolchain built.

## Usage

    cd <any scratch dir>
    udi/plm86 [options] path/to/source.plm [PL/M controls...]

* Output: `<name>.obj` and `<name>.lst` in the **current directory**. The
  compiler builds `<name>` from the file name on its command line, which is the
  basename of SOURCE as you typed it (or `--name`).
* `-F f0=DIR`, `-F f1=DIR` ...: map ISIS drives `:F0:`..`:F9:`. You can repeat
  `-F` for one drive to search several dirs in order. A drive you don't map
  falls back to the plain search path.
* `-I DIR`: search path for names without a drive prefix. It is searched after
  the cwd and the source's own directory.
* File lookup ignores case (`$INCLUDE(:f0:CtosTypes.edf)` finds `ctostypes.edf`).
* `--date "MM/DD/YY HH:MM:SS"`: the time the compiler sees. It goes into the obj
  COMENT record and the listing header. Without it you get the current time.
* `--heap-kb N`: memory for the compiler's dictionary. The default is 157, which
  reproduces "157KB MEMORY AVAILABLE" from the original listings.
* `--trace`: log every UDI call to stderr.
* Options and controls may come in any order. Controls are the usual PL/M ones,
  e.g. `NOLIST NOTYPE CODE XREF OPTIMIZE(3) LARGE`.

To match CTOS-built objects, pass **`NOTYPE`**. The CTOS front end added it,
and without it the compiler writes type records into the obj. `CODE`, `XREF`
and `NOLIST` do not change the obj.

Example (this reproduces `Linker/bndutl.obj` byte for byte):

    L=<CTOS source>/Linker
    E=<CTOS source>/edf
    plm86 -F f1=$L -F f0=$E --date "04/02/91 09:23:27" $L/BndUtl.Plm --name bndutl.plm NOLIST NOTYPE CODE

Exit status: 0 = OK, 2 = compile errors (the compiler then deletes the obj),
3 = fatal (for example an include file it can't find).

`udi86.py` is the generic runner for any UDI `.86` program:

    udi86.py [--trace] [--ff-to-lf] [-D F1=dir] [-I dir] prog.86 "PROGNAME args..."

It runs `Link86.86` (8086 LINKER V2.5) and `asm86.86` (ASM86 V2.0) as well.

## How it works

* `omf86.py` parses the LTL (load-time-locatable) OMF-86 module. That covers
  RHEADR, LNAMES, LTL SEGDEF/GRPDEF, OVLDEF, REGINT, REDATA/RIDATA + FIXUPP,
  and ENDREC per overlay.
* `udi86.py` places the groups at fixed paragraphs: CGROUP at 0100h, DGROUP at
  1100h, ??OGROUP00 at 2100h, heap from 2200h. It loads the ROOT overlay, applies
  the segment-base fixups (frame = group base + group offset/16), and sets
  CS:IP/SS:SP/DS from REGINT. Then it runs in unicorn (16-bit real mode).
* **UDI call mechanism** (found in the compiler's stub library and in
  `udi.run`): `INT 0C3h` with `AX` = function number. `udi.run` hooks `INT 0C1h`
  to the same dispatcher, and asm86 uses C1h. The PL/M stub turns its near
  return into a far one (`pop di; push cs; push di`), makes near pointers far
  (`push ds`), and does `INT 0C3h; RET n`. At the INT the stack holds the far
  return address, then the PL/M arguments, with the last argument at the lowest
  address. Results come back in AX. Every call's last argument is a far pointer
  to the exception word.
* `DQ$OVERLAY` loads the named overlay's records (PHASE20, PHASE25, PHASE40,
  PHASE41, PHASE42, PHASE50 for PL/M). This happens outside `emu_start`, and then
  the translation cache is flushed. Otherwise unicorn keeps running the old
  overlay's stale translated code.
* The command line reaches the program through `DQ$GET$ARGUMENT` and
  `DQ$SWITCH$BUFFER`, using udi.run's parsing rules: delimiters `,()=!$%\~+-&|[]<>;`,
  upper-casing, and `'quoted'` text kept as is. plm86 sends
  `V2.3PLM86.86 'name.plm' controls`. The quotes keep the name's case, as the
  CTOS front end did.
* The memory allocator copies udi.run's: an 80-entry table, best fit on freed
  entries, trailing entries returned to the OS, memory not cleared, and the
  groups registered so `DQ$GET$SIZE(DS)` works.
* Connections use udi.run's numbering, the lowest free slot.
* For PL/M, form feeds read from files are turned into LF, as udi.run does for
  PLM86/ASM86. Without that, every FF gives "WARNING 29 UNPRINTABLE CHARACTER".

### Function numbers (from udi.run's dispatch table at DGROUP:0C5Ch)

| AX | call | | AX | call |
|---|---|---|---|---|
| 0 | DQ$ALLOCATE(size,ex) | | 14 | DQ$CLOSE(conn,ex) |
| 1 | DQ$FREE(seg,ex) | | 15 | DQ$DETACH(conn,ex) |
| 2 | DQ$GET$SIZE(seg,ex) | | 16 | DQ$READ(conn,buf,cnt,ex) |
| 3,4,5,10,19,27 | not implemented in udi.run either | | 17 | DQ$WRITE(conn,buf,cnt,ex) |
| 6 | DQ$DECODE$EXCEPTION(code,buf,ex) | | 18 | DQ$SEEK(conn,mode,hi,lo,ex) |
| 7 | DQ$GET$SYSTEM$ID(buf,ex) -> "x10.1 (17May88)" | | 20 | DQ$GET$CONNECTION$STATUS(conn,info,ex) |
| 8 | DQ$GET$TIME(buf,ex) "MM/DD/YYHH:MM:SS" | | 21 | DQ$CHANGE$EXTENSION(path,ext,ex) |
| 9 | DQ$DELETE(path,ex) | | 22 | DQ$OVERLAY(name,ex) |
| 11 | DQ$ATTACH(path,ex) | | 23 | DQ$TRAP$CC (no-op) |
| 12 | DQ$CREATE(path,ex) | | 24 | DQ$GET$ARGUMENT(buf,ex) -> delimiter |
| 13 | DQ$OPEN(conn,access,nbuf,ex) | | 25 | DQ$SWITCH$BUFFER(buf,ex) |
| | | | 26 | DQ$EXIT(code) |
| | | | 28 | DQ$RESERVE$IO$MEMORY(n,m,ex) (no-op) |

All of these are implemented. Devices: `:CI:` (the command line), `:CO:`/`:TO:`/`:LP:`
(stdout) and `:BB:` (null). `:WORK:` files go in a temporary
`.udi86_work_*` dir in the cwd, which is deleted on exit. The PL/M compiler
keeps everything in memory ("0KB DISK SPACE USED").

Writes and deletes are allowed only inside the cwd and the work dir. Anything
else is refused with an exception (and a message on stderr). Reads may come
from anywhere on the search path.

## Test results

Run `test_linker.py` in an empty directory, with `CTOS_LINKER` and `CTOS_EDF` set. It rebuilds every
CTOS Linker `*.plm` that has a shipped `.obj`/`.lst`, taking the command
line and date from the shipped listing, and compares:

* **26/26 objs byte-identical** (including the timestamp in the COMENT record).
  24 of them build with the older CTOS `.edf` include set. `bndcfg` and `bndcmd` need
  `codeSetJapan`, so they need a newer include set.
* The listings match except for "LINES READ", which is short by a constant 314
  lines for most modules, and "MEMORY USED" (off by 1KB). The `CtosTypes.edf`/
  `CtosLib.edf` used in 1991 has not survived: no combination of the known
  copies gives the original line counts. `bndsfe.lst`, which includes
  neither, is fully identical.
* `CODE` works (tried on `BndUtl` and `CheckErc`), and the obj stays identical
  with `CODE`. `CheckErc.Plm` compiles with `-F f0=<CTOS edf dir>`.
* Speed: about 1 s per module.

## Known limitations

* Not implemented: the DQ calls udi.run doesn't implement either (MALLOCATE,
  RENAME, TRUNCATE, FILE$INFO, ...), and DQ$TRAP$EXCEPTION semantics.
  Any other software interrupt stops the program with an error.
* `DQ$CREATE` truncates an existing file. That is the UDI semantics; CTOS opened
  it in modify mode.
* `DQ$GET$CONNECTION$STATUS` access/seek bytes follow udi.run's table.
* `:CI:` only returns the command line. There is no interactive console input.
* Link86/asm86 are only smoke-tested. The CTOS sources were built with the CTOS
  Linker/Assembler (`.run`), not these.

## Files

* `plm86`, `plm86.py`: the compiler front end
* `udi86.py`: UDI emulator / .86 loader
* `omf86.py`: LTL OMF-86 parser (`python omf86.py file.86` dumps it)
* `test_linker.py`: regression test against the Linker sources
