# How the Rats modules were matched to byte-exact PL/M-86

This covers the workflow and how the Intel PL/M-86 V2.3 compiler (CTOS "x10.1") generates code.
It's useful for changing a module without breaking byte-identity, or for decompiling another
program built with the same compiler. See `NOTES.md` for the run-file format, the CTOS call
encoding and the DGROUP layout.

## Workflow
```
sh tools/build.sh <Module>         # compile src/<Module>.plm -> work/build/<Module>/<Module>.obj/.lst
python3 tools/cmpx.py work/build/<Module>/<Module>.obj <start_hex> \
      --syms src/<Module>.sym [--const OFF] [--data OFF] [--hunks N]
```
`cmpx.py` resolves the object's fixups and places the module's CODE at `<start_hex>`. It then
diffs the result instruction by instruction against the original. The goal is "differing bytes 0".
- **Resolving externals:** names come from `tools/symbols.txt` and the module's `.sym` file. Each
  line is `NAME FRAME OFFSET` in hex:
  - code: frame = module_start >> 4, offset = linear address - frame*16;
  - data: frame 7A6, offset = DGROUP offset;
  - CTOS requests: their magic CS:IP.
- **Displacement-only differences** in jumps and calls are hidden. Fix the first real hunk,
  rebuild, and repeat.
- **Own CONST/DATA segments** (local data, string literals, `DATA(...)` tables): pass their DGROUP
  placement with `--const` / `--data`. You can find them from the addresses the original code
  uses. Data that another module owns is easiest declared `EXTERNAL`.
- **Listings:** the `.lst` (with the `CODE` control) shows the generated code for each statement.
- **The original:** `tools/rdis2.py <start> <end> <frame>` gives an annotated disassembly. It
  takes names from `work/names.txt`, `work/rqnames.txt` and `work/ossub.txt`.
- **Whole program:** `tools/verifyall.sh` checks every module and then the relinked `Rats.run`.

## Compiler facts (x10.1 PL/M-86 V2.3)

### Module setup
- **Header:** `$MEDIUM OPTIMIZE(n)`. The CTOS front end passed `NOTYPE` (`build.sh` does too).
- **Optimisation level:** modules differ.
  - Scores, Maze and Sleep are OPTIMIZE(1). That level keeps `jcc +3; jmp` pairs and dead jumps,
    but still tracks register and constant contents.
  - The other PL/M modules are OPTIMIZE(2).
  - OPTIMIZE(0) and (3) never matched.
- **Main module:** the main-body code comes first, at offset 0 (`mov bp,sp; sti; call Main`),
  with no HLT.

### Procedures
- Every procedure with a stack frame is `REENTRANT`, so its locals are on the stack.
- PUBLIC and EXTERNAL procedures are far. Non-PUBLIC procedures in `$MEDIUM` are near.
- Procedures that are only reached through pointer tables are still PUBLIC. A procedure whose
  address is taken with `@` must be PUBLIC; otherwise it is near and you get warning 151.
- A procedure defined later in the module needs no forward declaration.

### Locals and declaration order
- **Allocation order:** separately declared locals are allocated from BP downwards in
  declaration order: the first declared is `[bp-2]`. In a factored list
  `DECLARE (a, b) WORD` the order is **reversed**. So declare locals one by one, in frame order.
- **Small frames:** 4 bytes of locals or less give `push cx` (once or twice) instead of `sub sp,n`.
- **Operand order:** for commutative operators (`=`, `<>`, `+`, `AND` ...) the operand that was
  **declared earlier** is loaded first, whatever order you write them in. To fix operand order,
  reorder the declarations, including EXTERNAL data.
- **Reused values:** the optimiser reuses identical subexpressions and register contents. If the
  original recomputes something, the source spelled it differently. For example, RatsA uses
  `bPlayerX` and `ptPlayer.x` for the same byte.

### Expressions and data access
- `(.x - .rgObj) / 4` compiles to `div`; the original used `SHR(.x - .rgObj, 2)`.
- `.var` gives a WORD offset and `@var` a POINTER (`push ds/push off` or `les`). BASED on a WORD
  variable gives DS-relative `[bx+..]` access.
- MOVB, FINDW and the like on `@local` or `@dgroup` data generate `push ds; pop es`. They need
  POINTER (`@`) arguments.
- `CALL SETW(0, p, n)` gives `rep stosw` and `SETB` gives `rep stosb`.
- **Booleans:** `IF flag THEN` gives `rcr al,1; jae`. Comparison results used as values give
  `mov al,0FFh; jxx; inc ax`.
- `DO WHILE (x := expr) <> c;` keeps x in a register (embedded assignment).
- `a, b, c = 0;` stores in reverse order (c, b, a).
- **Widening:** `word = word OR byte` zero-extends at word width. A byte operation on the low
  byte needs a BYTE overlay (`ab(2) BYTE AT(@w)`, `ab(0) = ab(0) OR b`).
- `x = x PLUS 0` gives `adc` (32-bit sums are split into two words).
- `x STRUCTURE(off WORD, seg WORD) AT (@p)` works on reentrant locals and parameters, which allows
  pointer arithmetic.
- `LENGTH()` and `SIZE()` are WORD constants.

### Control flow
- **DO CASE:** a jump table `jmp cs:[bx+tbl]`. Identical case bodies are merged, and a case whose
  tail equals the next case falls into it. A `GOTO label` into a labelled shared case also works.
- **RETURN:** `RETURN;` in the middle of a procedure jumps to the epilogue. `RETURN 0` gives
  `mov ax,0; jmp epilogue`.
- **IF/ELSE:** `IF c THEN t = X; ELSE t = Y;` loads the THEN value first, so choose the THEN/ELSE
  arrangement to match.
- **Short-circuit:** `IF (A<d OR B<d) AND (C<e OR D<e)` compiles to short-circuit jump chains.
- **Cross-jumping:** identical call tails in IF/ELSE are merged, and constants already in
  registers are reused as arguments.
- **Loops:** `DO k = 1 TO 7` gives `cmp k,7; ja ...; inc k; jne`. `GOTO lbl`, where lbl labels a
  DO WHILE, jumps to the loop test. Labels inside a DO WHILE body are local to it.

### Calls
- **Indirect far calls** through a table: `pProc = rgp(i); CALL pProc(args);` with a POINTER
  local `pProc`.
  - The arguments of an indirect call through a POINTER are untyped, so small constants are pushed
    as bytes. Use `DOUBLE(0)` to force a word.
- **BYTE arguments** are pushed as words (`mov al,..; push ax` or `push word ptr [byte]`). A WORD
  parameter given a byte expression gets `mov ah,0`.
- **CTOS procedures:** declare them EXTERNAL with the signatures from CTOS `Ctoslib.edf`, and give
  them their magic CS:IP in the `.sym` file (e.g. `SETSCREENVIDATTR FFB8 69A`). All addresses are
  mod 10000h:
  - **Request rq:** CS = -(rq AND 0FFF8h), IP = 210h + (rq AND 0FFF8h)*10h + ((rq*2) AND 0Fh).
  - **OsSub n:** CS = 0FFFFh - 16*(n/8), IP = 230h + 100h*(n/8) + 2n - 10h*(n/8).
  - **Kernel Check:** FFFF:218.

### CONST segment layout
1. All `DATA` declarations, in textual order, including procedure-local DATA. Word-class items
   (even size) come first; then BYTE arrays and odd-size structures, in declaration order.
2. Then the inline `.( )` and `@( )` string constants, in code order.

`DATA(.x)` may reference variables declared later. Inline constants are not allowed inside DATA
lists (error 130).
