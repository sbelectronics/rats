# Rats 9.2: Go port for Windows

This is a translation of the reconstructed PL/M/ASM source in `..\src` into Go. It keeps the original's
memory model, so it plays like the CTOS original:
- the 64K data segment sits at the original offsets and starts from the original initial data in
  `dgroup.bin`;
- 8- and 16-bit arithmetic is exact;
- the stack is emulated for locals whose address is taken;
- the original's quirks and bugs are kept.

The same maze number produces the same maze and the same rat behaviour.

## Build and run
```
go build -trimpath -o rats.exe .   # -trimpath keeps local file paths out of the executable
rats.exe                 # window, 2x scale
rats.exe -scale 1        # 800x510 (80x34 cells of 10x15)
rats.exe -phosphor green # or amber / white
```
High scores go in `%APPDATA%\Rats\Rats-<factories>-<density>.scores`. The format is the original
512-byte page. Use `-datadir` to choose a different folder.

## Keys (CTOS key → PC key)
| CTOS key | PC key | in the game |
|---|---|---|
| arrows / keypad 8 4 5 6 | arrows / numpad 8 4 5 6 | direction (N W S E) |
| MOVE | Space | move |
| CANCEL | Left Ctrl, Esc | fire |
| left / right SHIFT | left / right Shift | fire left / right (both = backwards) |
| GO | Enter, F10 | autopilot; in forms, accept |
| HELP | F1 | spawn a clone (costs a life) |
| FINISH | F12, Pause | quit the game |
| NEXT/RETURN | Tab | next field in a form |

## Layout
- **Shared core:**
  - `mem.go`: memory model, emulated stack, PL/M built-ins.
  - `api.go`: the contract between the parts.
- **`game_*.go`:** RatsA/B/C, Maze, Random, Sleep, Sprites (64 procedures).
- **`lib_*.go`:** Forms, FormFields, Scores, Strings, DateTime, ErrMsg, the heap, helpers, and the
  CTOS date/math runtime.
- **`platform_*.go` + `main.go`:** Ebiten window, the original `Sys.font` and `Rats.font`, CTOS
  video attributes, keyboard (raw and encoded modes, from the CTOS translation table), 100 ms RT
  clock, beeps, files, memory.
- **Data files:** `dgroup.bin` (the original initial DGROUP), `Rats.font` and `Sys.font` were
  extracted from the original Rats 9.2 and CTOS system files. They are embedded in the executable.
- **`PORT_GUIDE.md`:** the rules the translation followed.
- **Tests:** `go test .` (game, libraries, platform).
- **Snapshots:** `rats.exe -snapshot out.png -snapdelay 5s -snapkeys "1s:1b,2s:12:0f"` renders the
  screen to a PNG with scripted keys (hex CTOS codes; 1b = GO, 12 = MOVE, 0f = right).

## Known differences
- **Sounds:** they are 1 kHz square waves of guessed length (0.10 s / 0.15 s).
- **Errors:** CheckErc on an error exits with a message; the original trapped into the debugger.
- **Unknown request:** CTOS request 258 at start-up isn't issued. Its result was ignored anyway.
- **Hardware:** only driver A / hardware type 0 (80×34, Rats.font) is emulated, not the T1/NGEN
  variant (T1Rats.font, 29 lines).
- **Keyboard:** the CODE key and diacritics aren't supported.
- **Impossible values:** where the original would misbehave (a DO CASE out of range, more than 99
  factories), the port stops with an error instead.
