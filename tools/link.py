#!/usr/bin/env python3
"""link.py - mini CTOS-style linker for the Rats reconstruction.

Lays out modules (in link order) like the CTOS linker did: CODE segments word-aligned in module
order from image offset 0; DGROUP (paragraph 7A6h) = all CONST segments (module order), then all
DATA segments (module order).  Pieces not yet reconstructed are 'blobs': original bytes for a code
range and/or DGROUP ranges.  Resolves all fixups from the objects' PUBDEFs (falling back to names in
tools/symbols.txt and src/*.sym for blob-defined symbols), builds the load image and compares it
with the original Rats.run image.

usage: link.py            (writes work/relinked.img)
"""
import sys, os, struct, collections
T = os.path.dirname(os.path.abspath(__file__)); sys.path.insert(0, T)
import omf2
R = T + '/..'; B = R + '/work/build'
d = open(R + '/orig/Rats.run', 'rb').read(); h = struct.unpack_from('<16H', d, 0)
orig = d[h[4] * 16:]
DGF = 0x7A6; DG = DGF * 16

# (name, obj path or None, blob spec).  Blob spec keys: code=(s,e) image range,
# constpre/const/data=(s,e) DGROUP ranges ('pre' = before the module's own segment).
M = [

 ('RatsA',     B+'/RatsA/RatsA.obj',      {}),
 ('RatsData',  B+'/RatsData/RatsData.obj', {}),     # data only: CONST 0000, DATA 1234 (game state)
 ('RatsB',     B+'/RatsB/RatsB.obj',      {}),
 ('RatsC',     B+'/RatsC/RatsC.obj',      {}),
 ('Sprites',   B+'/Sprites/Sprites.obj',  {}),
 ('RatsTables',B+'/RatsTables/RatsTables.obj', {}), # data only: CONST 0298 (seek tables, sprite ptrs, options form)
 ('Scores',    B+'/Scores/Scores.obj',    {}),
 ('Maze',      B+'/Maze/Maze.obj',        {}),
 ('Sleep',     B+'/Sleep/Sleep.obj',      {}),
 ('Mod3396',   B+'/Mod3396/Mod3396.obj',  {}),
 ('Mod35AC',   B+'/Mod35AC/Mod35AC.obj',  {}),
 ('Thunk3616', B+'/Thunk3616/Thunk3616.obj', {}),
 ('DateTime',  B+'/DateTime/DateTime.obj', {}),
 ('CheckErc',  B+'/CheckErc/CheckErc.obj', {}),
 ('Forms',     B+'/Forms/Forms.obj',      {}),
 ('FormFields',B+'/FormFields/FormFields.obj', {}),
 ('Mod54E4',   B+'/Mod54E4/Mod54E4.obj',  {}),
 ('Font',      B+'/Font/Font.obj',        {}),
 ('Mod55FC',   B+'/Mod55FC/Mod55FC.obj',  {}),
 ('Random',    B+'/Random/Random.obj',    {}),
 ('ErrMsg',    B+'/ErrMsg/ErrMsg.obj',    {}),
 ('Strings',   B+'/Strings/Strings.obj',  {}),
 ('VidSel',    B+'/VidSel/VidSel.obj',    {}),
 ('VidTab',    B+'/VidTab/VidTab.obj',    {}),     # data only: CONST 10E6 (rgwLines, rgpDrv)
 ('VidInit',   B+'/VidInit/VidInit.obj',  {}),
 ('VidThk',    B+'/VidThk/VidThk.obj',    {}),
 ('VidDrvA',   B+'/VidDrvA/VidDrvA.obj',  {}),
 ('VidDrvB',   B+'/VidDrvB/VidDrvB.obj',  {}),
 ('Runtime',   None,                      dict(code=(0x7306, 0x7A60), const=(0x112A, 0x1234))),
]

def al2(x): return (x + 1) & ~1

img = bytearray(len(orig)); own = bytearray(len(orig))   # own: 1 = from source, 2 = blob
mods = []
for name, path, bl in M:
    if path and not os.path.exists(path): sys.exit('missing ' + path)
    mods.append((name, omf2.parse(open(path, 'rb').read()) if path else None, bl))

def blob(name, kind, s, e, cur, dg):
    base = DG if dg else 0
    if cur != s: print(f'!! {name}: {kind} expected at {s:#x}, layout gives {cur:#x}')
    img[base+s:base+e] = orig[base+s:base+e]; own[base+s:base+e] = b'\2' * (e - s)
    return e

place = {}
pc = 0
for name, o, bl in mods:
    if 'code' in bl: pc = blob(name, 'code', *bl['code'], pc, False)
    if o:
        for i, sg in enumerate(o.segs):
            if sg and sg.cls == 'CODE':
                pc = al2(pc); place[(name, i)] = (pc >> 4, pc & 15, pc)
                img[pc:pc+sg.length] = sg.data; own[pc:pc+sg.length] = b'\1' * sg.length; pc += sg.length
print(f'CODE ends at {pc:#x}')
dp = 0
for cls in ('CONST', 'DATA'):
    key = cls.lower(); dp = al2(dp)
    for name, o, bl in mods:
        if key + 'pre' in bl: dp = blob(name, key + 'pre', *bl[key + 'pre'], al2(dp), True)
        if o:
            for i, sg in enumerate(o.segs):
                if sg and sg.name == cls and sg.length:
                    dp = al2(dp); place[(name, i)] = (DGF, dp, DG + dp)
                    img[DG+dp:DG+dp+sg.length] = sg.data; own[DG+dp:DG+dp+sg.length] = b'\1' * sg.length
                    dp += sg.length
        if key in bl: dp = blob(name, key, *bl[key], al2(dp), True)
    print(f'{cls} ends at DGROUP {dp:#x}')

# symbols
sym = {}
for name, o, bl in mods:
    if not o: continue
    for n, si, off in o.pubs:
        if (name, si) in place:
            f, base, lin = place[(name, si)]; sym[n.upper()] = (f, base + off)
extra = collections.defaultdict(set)
for sf in [T + '/symbols.txt'] + [os.path.join(R, 'src', x) for x in sorted(os.listdir(R + '/src')) if x.endswith('.sym')]:
    for L in open(sf):
        L = L.split('#')[0].split()
        if len(L) >= 3 and L[0].upper() not in sym:
            extra[L[0].upper()].add((int(L[1], 16), int(L[2], 16)))

unres = collections.Counter(); conflict = set(); fromextra = set()
relocs = []        # (module, holder frame, offset) in link processing order
origrel = [struct.unpack_from('<2H', d, h[12] + 4 * i) for i in range(h[3])]
for name, o, bl in mods:
    if not o:      # blob module: take its original relocation entries (those inside its ranges)
        rng = [(s, e) for k, (s, e) in bl.items() if k == 'code'] + \
              [(DG + s, DG + e) for k, (s, e) in bl.items() if k != 'code']
        for ra, sa in origrel:
            if any(s <= sa * 16 + ra < e for s, e in rng): relocs.append((name, sa, ra))
        continue
    for i, sg in enumerate(o.segs):
        if (name, i) not in place: continue
        f0, b0, lin0 = place[(name, i)]
        for fx in sg.fix:
            tm, ti = fx['tm'], fx['ti']
            if tm == 0:
                if (name, ti) not in place: unres['<own seg %s>' % o.segs[ti].name] += 1; continue
                tf, toff, tlin = place[(name, ti)]
            elif tm == 1:
                tf, toff, tlin = DGF, 0, DG
            else:
                n = o.exts[ti].upper()
                if n in sym: tf, toff = sym[n]
                elif n in extra:
                    if len(extra[n]) > 1: conflict.add(n)
                    tf, toff = sorted(extra[n])[0]; fromextra.add(n)
                    absolute = tf == 0 or tf >= len(orig) // 16   # CTOS magic CS:IP, never relocated
                else: unres[n] += 1; continue
                tlin = (tf * 16 + toff) & 0xfffff
            tlin += fx['disp']            # target displacement (ASM86 'sym+n')
            if not (tm == 2 and n in extra and n not in sym): absolute = False
            fm = fx['fm']
            F = DGF if fm == 1 else f0 if fm == 4 else place[(name, fx['fi'])][0] if fm == 0 else tf
            p = lin0 + fx['off']; loc = fx['loc']; offv = (tlin - F * 16) & 0xffff
            if fx['M'] == 0:
                here = p + (2 if loc == 1 else 1)
                struct.pack_into('<H', img, p, (struct.unpack_from('<H', img, p)[0] + tlin - here) & 0xffff); continue
            if loc in (1, 3): struct.pack_into('<H', img, p, (struct.unpack_from('<H', img, p)[0] + offv) & 0xffff)
            if loc == 0: img[p] = (img[p] + offv) & 0xff
            if loc == 2: struct.pack_into('<H', img, p, F)
            if loc == 3: struct.pack_into('<H', img, p + 2, F)
            if loc in (2, 3) and not absolute:   # relocatable paragraph (not an absolute CTOS address)
                q = p + (2 if loc == 3 else 0)
                relocs.append((name, f0, q - f0 * 16))
if unres: print('UNRESOLVED:', dict(unres))
if conflict: print('conflicting blob symbols:', sorted(conflict))
print('symbols taken from .sym/symbols.txt (blob-defined):', len(fromextra))

n = len(orig); diffs = [i for i in range(n) if img[i] != orig[i]]
src = own.count(1); blb = own.count(2)
print(f'image {n:#x}: from source {src:#x} ({100*src/n:.1f}%), blob {blb:#x}, uncovered {n-src-blb:#x}; differing bytes {len(diffs)}')
if diffs:
    runs = []; s = pv = diffs[0]
    for x in diffs[1:]:
        if x != pv + 1: runs.append((s, pv)); s = x
        pv = x
    runs.append((s, pv))
    for a, b in runs[:40]:
        where = 'DG+%04x' % (a - DG) if a >= DG else '%05x' % a
        print(f'  diff {where} len {b-a+1} own={own[a]} new={bytes(img[a:min(b+1,a+8)]).hex()} orig={orig[a:min(b+1,a+8)].hex()}')
open(R + '/work/relinked.img', 'wb').write(img)

# ---- relocation table and run file
mine = [(ra, sa) for _, sa, ra in relocs]
print(f'relocations: {len(mine)} (original {len(origrel)}); same set: {sorted(mine) == sorted(origrel)}; same order: {mine == origrel}')
if sorted(mine) != sorted(origrel):
    a, b = set(mine), set(origrel)
    print('  only new :', sorted((hex(sa), hex(ra)) for ra, sa in a - b)[:10])
    print('  only orig:', sorted((hex(sa), hex(ra)) for ra, sa in b - a)[:10])
elif mine != origrel:
    k = next(i for i in range(len(mine)) if mine[i] != origrel[i])
    print(f'  first order difference at entry {k}: new {[(hex(s), hex(r)) for r, s in mine[k:k+3]]} orig {[(hex(s), hex(r)) for r, s in origrel[k:k+3]]}')
# header: the original's fields (stack, entry, version string ...), new relocation table, checksum
hdr = bytearray(d[:h[4] * 16])
struct.pack_into('<H', hdr, 6, len(mine))
for i, (ra, sa) in enumerate(mine): struct.pack_into('<2H', hdr, h[12] + 4 * i, ra, sa)
run = bytearray(hdr + img)
struct.pack_into('<H', run, 18, 0)
s = sum(struct.unpack_from('<H', run, 2 * i)[0] for i in range(len(run) // 2)) & 0xffff
struct.pack_into('<H', run, 18, (0xffff - s) & 0xffff)
open(R + '/work/Rats.run', 'wb').write(run)
print(f'wrote work/Rats.run ({len(run)} bytes): identical to original Rats.run: {bytes(run) == d}')
