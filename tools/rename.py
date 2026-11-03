#!/usr/bin/env python3
"""rename.py - apply simultaneous, case-insensitive whole-identifier renames to module sources.
usage: rename.py <Module> OLD=NEW [OLD=NEW ...]
Edits src/<Module>.plm (or .asm) and src/<Module>.sym. Identifier chars: A-Z a-z 0-9 _ $.
Skips text inside PL/M string literals ('...') and ASM comments are renamed too (harmless)."""
import sys, re, os
R = os.path.dirname(os.path.abspath(__file__)) + '/../src/'
mod = sys.argv[1]
mp = {}
for a in sys.argv[2:]:
    o, n = a.split('='); mp[o.upper()] = n
pat = re.compile(r"/\*.*?\*/|'[^'\n]*'|[A-Za-z_$][A-Za-z0-9_$]*", re.S)
idp = re.compile(r'[A-Za-z_$][A-Za-z0-9_$]*')
def sub(m):
    t = m.group(0)
    if t.startswith("'"): return t
    if t.startswith('/*'): return idp.sub(lambda m: mp.get(m.group(0).upper(), m.group(0)), t)
    return mp.get(t.upper(), t)
for ext in ('.plm', '.asm', '.sym'):
    p = R + mod + ext
    if not os.path.exists(p): continue
    s = open(p, newline='').read()
    s2 = pat.sub(sub, s)
    if ext == '.sym':   # sym files hold upper-case names
        s2 = re.sub(r'^(\S+)', lambda m: m.group(1).upper(), s2, flags=re.M)
    if s2 != s:
        open(p, 'w', newline='').write(s2); print('updated', mod + ext)
