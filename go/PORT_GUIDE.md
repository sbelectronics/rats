# How the Go port is translated

The port follows the byte-exact reconstructed source in `../src` (PL/M-86 and ASM86) procedure by
procedure. The aim is to play exactly like the original:
- the same maze generation and the same object behaviour;
- the same timing, 10 game ticks per second;
- the same screen, 80 x 34 character cells in the original fonts;
- the same options form and the same high-score file format.

Read `../README.md` first; it has the module map and describes the game. The rules below explain
why the Go code looks the way it does. Keep to them when changing it.

## Layout
- Everything is one `package main`, in three parts:
  - `game_*.go`: RatsA/B/C, Maze, Random, Sleep, Sprites;
  - `lib_*.go`: the library modules (Forms, Strings, Scores, ...);
  - `main.go` and `platform_*.go`: the emulated CTOS services and the Ebiten front end.
- `mem.go` (memory model) and `api.go` (the function variables the parts call each other through)
  are shared.
- Each part assigns its `api.go` variables in `init()`. Within a part, functions call each other
  directly.

## Translating PL/M faithfully
- **Keep the memory model** (`mem.go`).
  - Game state lives in the 64K array `dg` at the original DGROUP offsets. `dgroup.bin` (the
    original initial data) initialises it.
  - The names of the offsets are in `../src/RatsData.asm`, `../tools/symbols.txt` and `../src/*.sym`.
  - Near pointers are `uint16` offsets; far pointers are `Ptr`.
  - Game variables are accessed through small named accessors or `const` offsets, e.g.
    `func iWheel() uint16 { return W(0x2B9A) }`.
  - Don't move game data into Go structs. The code depends on aliasing:
    - the timing-wheel sentinel nodes are `rgHead` entries, and their link field is at offset 0;
    - object pointers are DGROUP offsets into `rgObj` (135Ah, 16-byte records);
    - `rgPos` is indexed by `SHR(p - .rgObj, 2)`.
- **Keep integer widths.** BYTE = `uint8`, WORD = `uint16`, DWORD = `uint32`, INTEGER = `int16`.
  - Evaluate every expression at the width the PL/M compiler used: byte arithmetic wraps at 256,
    and WORD comparisons are unsigned.
  - When in doubt, check the instructions in the module's compiler listing
    (`work/build/<Module>/<Module>.lst`, written by `tools/build.sh`) or in `../tools/rdis2.py`.
- **Booleans:** TRUE = 0FFh, FALSE = 0, and `IF x` tests bit 0 (use `Test(x)`).
- **REENTRANT locals whose address is taken** (`.save`, `@sbBuf`, the NewObj templates and so on)
  live on the emulated stack in DGROUP: `defer Release(Mark()); off := Push(16)`. Other locals are
  plain Go variables.
- **Keep the original's bugs and quirks.** For example:
  - FreeObj sets the tail pointer to `obj.link`;
  - FindDw reads the wrong parameter;
  - ReadMsg drops the last character.

  Don't "fix" them: they change behaviour.
- **Indirect calls through DGROUP far-pointer tables** use `FarFn(P(off))` plus a type assertion.
  - Procedures are registered under their original frame:offset, e.g.
    `RegisterFar(0x0090, 0x06DD, PlayerTick)`.
  - **0018h:** 8 per-type handlers `func(pos, p uint16)`, frame 90h, at offsets 6DD, A0A, A9B,
    B0D, B0D, D67, D8C and 1073.
  - **0658h / 066Ch:** sprite Draw/Erase, frame 287h.
    - Draw1/2/4 at 0C/23/44: `func(ch byte, pos uint16, pPos uint16)`.
    - Erase1/2/4 at 79/8A/A0: `func(pos uint16)`.
    - Entries 0 and 3 are null.
  - **Form field procedures** in the options form (0680h..) and the score form (0A92h..). Their
    frame is 528h (FormFields) or 293h (Scores' FmtDate, ParseDate, FmtName, ParseName).
    Signatures: `func(pOut, pIn Ptr) uint16` (Parse) and `func(pOut, pIn Ptr)` (Fmt).
  - **10EAh (VidTab):** the video driver table is not needed, because the video thunks are
    `api.go` functions.

## Platform emulation
- **Screen:** VidSel picks driver type 0 for hardware type 0, so the port emulates that hardware:
  34 lines, Rats.font and driver A. `cScrCols` = 80, `cScrLines` = 34, and `pVidCfg` points at
  a byte 0.
- **Fonts:**
  - **Format:** 256 glyphs × 32 bytes: 16 rows of little-endian 16-bit words, 10 pixels wide,
    bit 9 = leftmost pixel.
  - **Cell:** row 0 is blank in every glyph and is not displayed. The cell is 10 × 15 and the
    screen 800 × 510.
  - **Which font when:** the screen starts in the CTOS system font (`Sys.font`), which the options
    form uses. LoadFont then switches to `Rats.font`.
- **Attributes:** per-cell bits; see `VidDrvA.asm` (4-bit attribute nibbles) and the CTOS attribute
  definitions (reverse video, half-bright, underline, blink).
  - `SetScreenVidAttr(0, f)` is whole-screen reverse video; the game uses it for flashing.
  - `SetScreenVidAttr(1, f)` blanks or unblanks the screen; it is used while loading a font.
- **Timing:**
  - The RT clock produces one event per 100 ms. `RTEvents()` returns how many arrived since the
    last call.
  - `Sleep(n)` (from Sleep.plm) polls RTEvents. Its calibrated busy-poll is emulated in real time:
    a call returns when the next 100 ms event arrives, or after the time cMax polls would have
    taken. It doesn't depend on CPU speed.
  - `tickCount` (2CACh) counts every 10 events, so it is in seconds. `Delay(n)` = n × 100 ms.
- **Keyboard:** CTOS key codes from CTOS `Kbd_All.asm` (rgInfoIkey).
  - Unencoded mode delivers press and release events; a release is the code OR 80h.
  - Encoded mode delivers characters with typematic repeat, taken from the "unshifted/shifted
    value" columns of rgInfoIkey with shift applied.
  - The PC key mapping is in `README.md` and `main.go`.
- **Threading:** the translated game is blocking PL/M control flow (DO FOREVER, ReadKbd waits).
  - `main.go` runs `GameMain()` (RatsA's Main) in its own goroutine.
  - Ebiten's Update and Draw only read the screen buffer under a mutex and feed the keyboard queue.
  - `Exit` ends the process.
- **Files:** CTOS file specs like `[!sys]<games>Rats-3-50.scores` map to
  `%APPDATA%\Rats\Rats-3-50.scores`: the volume and directory are dropped. The format is the
  original 512-byte page.
- **Time:** a CTOS DateTime is a DWORD:
  - low word = seconds in the half-day;
  - high word = half-days since the CTOS epoch, 1 March 1952.

  The runtime routines follow CTOS `ExpDt.Plm`, `CmpDT.Plm` and `FmtDt.Plm`.
