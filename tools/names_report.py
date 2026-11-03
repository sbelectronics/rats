#!/usr/bin/env python3
"""names_report.py - list every external symbol referenced by every module, the address it resolves
to (via the module's own .sym / tools/symbols.txt), and the PUBLIC defined at that address (if any).
Reports (a) externals whose name differs from the public at the same address, (b) one address with
several names, (c) one name with several addresses."""
import sys, os, collections
T = os.path.dirname(os.path.abspath(__file__)); sys.path.insert(0, T)
import omf2, importlib.util
spec = importlib.util.spec_from_file_location('link', T + '/link.py')
R = T + '/..'; B = R + '/work/build'
# reuse link.py's module list without running it: parse the M list textually
src = open(T + '/link.py').read()
M = eval(src[src.index('M = [') + 4: src.index(']\n', src.index('M = [')) + 1], {'B': B, 'R': R, 'dict': dict})
def loadsym(path):
    s = {}
    if os.path.exists(path):
        for L in open(path):
            L = L.split('#')[0].split()
            if len(L) >= 3: s[L[0].upper()] = (int(L[1], 16), int(L[2], 16))
    return s
glob = loadsym(T + '/symbols.txt')
# publics: need placement -> run link.py's layout cheaply: import it as module would execute; instead
# take addresses from the .sym files of the defining module is unreliable, so recompute via link.py output
import subprocess, json
pubs = {}
exec_ns = {'__file__': T + '/link.py', '__name__': 'linkmod'}
code = src.replace("open(R + '/work/relinked.img', 'wb').write(img)", "")
code = code.replace("if unres: print", "if False: print").replace("print(", "(lambda *a, **k: None)(")
exec(compile(code, 'link', 'exec'), exec_ns)
sym = exec_ns['sym']
addr2pub = collections.defaultdict(set)
for n, (f, o) in sym.items(): addr2pub[(f * 16 + o) & 0xfffff].add(n)
uses = collections.defaultdict(lambda: collections.defaultdict(set))   # lin -> name -> modules
name2lin = collections.defaultdict(set)
for name, path, bl in M:
    if not path or not os.path.exists(path): continue
    o = omf2.parse(open(path, 'rb').read())
    msym = dict(glob); base = os.path.splitext(os.path.basename(path))[0]
    msym.update(loadsym(R + '/src/' + base + '.sym'))
    for e in o.exts[1:]:
        n = e.upper()
        if n in msym: lin = (msym[n][0] * 16 + msym[n][1]) & 0xfffff
        elif n in sym: lin = (sym[n][0] * 16 + sym[n][1]) & 0xfffff
        else: lin = None
        uses[lin][n].add(base); name2lin[n].add(lin)
print('== externals whose name differs from the PUBLIC at that address')
for lin in sorted(x for x in uses if x is not None):
    if lin in addr2pub:
        for n, ms in uses[lin].items():
            if n not in addr2pub[lin]:
                print(f'  {lin:05x} {n:22} used by {",".join(sorted(ms)):30} PUBLIC = {",".join(sorted(addr2pub[lin]))}')
print('== addresses (no PUBLIC yet) with several names')
for lin in sorted(x for x in uses if x is not None):
    if lin not in addr2pub and len(uses[lin]) > 1:
        print(f'  {lin:05x} ' + '  '.join(f'{n}({",".join(sorted(ms))})' for n, ms in uses[lin].items()))
print('== names with several addresses')
for n, ls in sorted(name2lin.items()):
    if len(ls) > 1: print(f'  {n}: ' + ' '.join('%05x' % l if l is not None else '?' for l in ls))
print('== unresolved externals')
for n, ms in uses.get(None, {}).items(): print(f'  {n} used by {",".join(sorted(ms))}')
