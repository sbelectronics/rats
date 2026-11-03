# Reversing Rats

This repository contains reverse-engineered source code for the classic Convergent
Technologies game "Rats", which ran on Convergent workstations such as the AWS, IWS
and NGEN in the 1980s.

As far as I know, the source code for Rats has been lost to time, but the binary
survived. With the help of Claude Code, I set out to see whether PL/M and assembly
source could be reconstructed from the binary. This repository contains that
reconstructed source. Built with the original Intel compiler and assembler, it
reproduces the original binary byte for byte.

This repo also contains a port of Rats to the Go programming language. It runs on
Windows, so that people with Windows computers can play Rats!

It should be possible to port the source code to other languages too.

## Rats 9.2 for Convergent CTOS — reconstructed source

Version 9.2 ("Ver Rats 9.2 Linker x9.2-6/13-15:55"); the binaries carry Convergent
Technologies copyrights from 1980 to 1983.

**Result:** the source rebuilds to a `Rats.run` that is **byte-for-byte identical** to the
original file (49,664 bytes). That covers the header, all 588 relocation entries in their
original order, and the checksum. The tools are the original Intel/Convergent PL/M-86 compiler
(x10.1 V2.3) and ASM86 assembler, run under emulation. The Go port is in [`go/`](go/README.md).

## Building the CTOS original
The original game binary, `Rats.run`, is in [`orig/`](orig/README.md). The rebuild also needs
two Intel tools, which are Intel's software and are not included here:
- `V2.3PLM86.86`, the PL/M-86 compiler;
- `asm86.86`, the ASM86 assembler.

Both ship on CTOS system and development disks. `tools/getorig.sh` finds them in a copy of such
disks, checks their SHA-256 against the exact versions used here, and copies them into `orig/`.
A different build of the compiler may not produce byte-identical code. The rebuild runs the tools
under emulation and compares the result with `Rats.run`:
```
python3 -m venv venv && venv/bin/pip install unicorn capstone    # Linux or WSL
sh tools/getorig.sh /path/to/ctos/disks
sh tools/verifyall.sh      # build all 28 modules (19 PL/M, 9 ASM86), compare each, relink, compare Rats.run
```
That writes `work/Rats.run` and reports, among other things,
`relocations: 588 ... same order: True` and `identical to original Rats.run: True`.

## What the game is
You steer a "player" through a randomly generated maze full of rat **factories** (nests). The
code calls these "nests"; the options form calls them factories.
- Nests spawn young rats. The young grow into adult rats, and pregnant rats release four small
  rats when they die.
- You shoot them. Rats chase you when they can see you, using 20x16 seek tables, and shoot back.
- Destroyed objects turn into ghosts or blast fragments. A "clone" command spawns an autopilot
  copy of the player.
- The options form covers:
  - number of factories and maze density;
  - crazy rats, randomize on death, leave ghosts;
  - sneak preview of the maze, left-handed controls, play forever;
  - demolition factor and maze number.
- Scores: bonus lives at 15,000 / 25,000 / ... points. The end-of-game tally is raw score, plus
  1,000 per unused life, minus elapsed time.
- The top 10 scores are kept per setting in `[!sys]<games>Rats-<factories>-<density>.scores`.
  Each file is one 512-byte page of `{score, date, name(20)}` records.

## Source files (`src/`), in link order
| image range | module | lang | contents |
|---|---|---|---|
| 0000-0907 | RatsA.plm | PL/M O2 | main program, scheduler (16-slot timing wheel), object core: NewObj/FreeObj/KillObj/MoveCur/DrawObj, player respawn, view scrolling |
| — | RatsData.asm | data | CONST 0000 (per-type handler table, small tables) and DATA 1234-2C99: all game state (256 object records, positions, counters, options, status lines) |
| 0908-1DF7 | RatsB.plm | PL/M O2 | object behaviour: player, nests, young/adult/small rats, shots, ghosts, blasts; rat AI, map probing, autopilot |
| 1DF8-287B | RatsC.plm | PL/M O2 | init and UI: options form, font, maze generation callback, status line, score and bonus lives, end-of-game tally, keyboard |
| 287C-292F | Sprites.asm | ASM86 | draw/erase 1-, 2- and 4-cell sprites into the 64K map |
| — | RatsTables.asm | data | CONST 0298: rat seek tables, sprite pointer tables, the options form description |
| 2930-2FAF | Scores.plm | PL/M O1 | high-score file and table display/entry |
| 2FB0-32A5 | Maze.plm | PL/M O1 | maze generator (Eller's algorithm, wall callback) |
| 32A6-3395 | Sleep.plm | PL/M O1 | game timer: RT clock and Check polling, tickCount |
| 3396-35AB | Mod3396.plm | PL/M O2 | small heap on CTOS short-lived memory (MemAlloc/MemFree) |
| 35AC-3615 | Mod35AC.plm | PL/M O2 | HwBeep: speaker via I/O ports |
| 3616-3629 | Thunk3616.asm | ASM86 | CallProc: typed indirect call helper |
| 362A-3CB7 | DateTime.plm | PL/M O2 | date/time parse, format and arithmetic |
| 3CB8-3D1B | CheckErc.asm | ASM86 | CheckErc / breakpoint / error-handler hook |
| 3D1C-5285 | Forms.plm | PL/M O2 | forms/dialog runtime (field editing, choices, messages) |
| 5286-54E3 | FormFields.plm | PL/M O2 | field format/parse procedures (string, bool, byte, word, dword) |
| 54E4-552D | Mod54E4.plm | PL/M O2 | keyboard LEDs |
| 552E-55FB | Font.plm | PL/M O2 | font loading (LoadFontRam) |
| 55FC-577B | Mod55FC.asm | ASM86 | pointer, dword and fraction helpers (nil pointer constant pNil) |
| 577C-5919 | Random.plm | PL/M O2 | lagged-Fibonacci random generator |
| 591A-5CD1 | ErrMsg.plm | PL/M O2 | error-code texts plus an `[scr]<$>Erc.tmp` message store |
| 5CD2-6BB3 | Strings.plm | PL/M O2 | length-prefixed string library (parse/format numbers, roman numerals, file specs) |
| 6BB4-6C0D | VidSel.plm | PL/M O2 | detect video hardware type |
| — | VidTab.asm | data | CONST 10E6: video driver entry table |
| 6C0E-6F27 | VidInit.plm | PL/M O2 | screen and frame setup, select driver |
| 6F28-6F7F | VidThk.plm | PL/M O2 | 8 video entry thunks (call through driver table) |
| 6F80-71FF | VidDrvA.asm | ASM86 | video driver, 4-bit attribute screens |
| 7200-7305 | VidDrvB.asm | ASM86 | video driver, (char, attr) word screens |
| 7306-7A5F | (runtime) | — | standard CTOS library: CompactDateTime, ExpandDateTime, FormatDateTime, LQ dword mul/div, Mul32 |

"O1"/"O2" is the PL/M `OPTIMIZE` level each module was compiled with.

Each `.plm`/`.asm` has a header comment. `src/*.sym` give `NAME FRAME OFFSET` for externals
(CTOS requests appear as their magic CS:IP). These are only needed for per-module comparison
with `tools/cmpx.py`.

## What is *not* rebuilt from source
- **The CTOS runtime tail** (image 7306-7A5F, plus its CONST at DGROUP 112A-1233, which includes
  the "Rats 9.2" string at 122A) is copied from the original. It is library code, not part of the
  game. It comes from the CTOS library sources `CmpDT.Plm`, `ExpDt.Plm`, `FmtDt.Plm`,
  `Math32.Asm` and `DWord_Mf.Asm`. Those can't be rebuilt here, because they need the include
  files `Util.erf` and `CTOS.erf`, which have not survived.
- **The run-file header fields** (stack segment/size, entry point, version string) are copied
  from the original. `tools/link.py` builds the image, the relocation table and the checksum.
- **The CTOS linker itself** is replaced by `tools/link.py`, which reproduces its layout rules:
  - code is word-aligned, in module order;
  - DGROUP is all CONST segments, then all DATA segments, in module order;
  - relocations are emitted per module, code first, then data.

## Caveats about the reconstruction
- **The generated code is exact; some source details are reconstruction choices.** Code identity
  pins down a lot: declaration order (operand order), local variable order, statement structure,
  even where labels were. It does not pin down:
  - identifier names, which were chosen during the reconstruction;
  - which module owned some data. The three data modules (`RatsData`, `RatsTables`, `VidTab`) are
    placed where the original's segment layout and relocation order require them, but the original
    may have declared that data inside a PL/M module instead.
- **A few spots needed odd-looking source forms to match**, such as an unreferenced label or a
  `GOTO` into a shared DO CASE arm. They are commented in the source.
- **Original bugs are reproduced as they are.** Examples: `FindDw` reads the search value from the
  wrong parameter, and `ReadMsg` drops the last character.

## Toolchain (Linux or WSL, Python 3 with `unicorn` and `capstone`)
The scripts use `venv/bin/python` if it exists, otherwise `python3`.
- **`udi/`:** a UDI emulator (Python + unicorn) that runs the Intel `.86` tools.
  - `udi/plm86` runs `orig/V2.3PLM86.86`.
  - `udi/udi86.py <tool.86> "<command line>"` runs any `.86` tool, e.g. `orig/asm86.86`.
  - It was validated by recompiling all 26 modules of the CTOS Linker to byte-identical `.obj`
    files (`udi/test_linker.py`). See `udi/README.md`.
- **Building:**
  - `tools/getorig.sh <dir>...` fetches the Intel compiler and assembler into `orig/`.
  - `tools/build.sh <Mod>` compiles one PL/M module.
  - `tools/buildall.sh` builds everything (PL/M + ASM86).
- **Checking:**
  - `tools/cmpx.py <obj> <start> --syms src/<Mod>.sym [--const X --data Y]` resolves one module's
    fixups and diffs it against the original.
  - `tools/link.py` relinks everything and writes `work/relinked.img` and `work/Rats.run`.
  - `tools/verifyall.sh` runs all of the above.
- **Analysis helpers:**
  - `tools/rdis2.py <start> <end> <frame>`: annotated disassembly of the original;
  - `tools/runinfo.py`: run-file header and relocations;
  - `tools/names_report.py`: checks that external names agree between modules;
  - `tools/rename.py`: renames identifiers across a module's `.plm`/`.asm` and `.sym`;
  - `tools/gen_data.py`: generates a data-only ASM86 module from the original DGROUP.
- **Docs:**
  - `DECOMPILE_GUIDE.md`: how this PL/M-86 compiler generates code, and the matching workflow.
  - `NOTES.md`: run-file format, CTOS call encoding, memory map.
