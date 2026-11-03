#!/usr/bin/env python3
"""Recompile the CTOS Linker PL/M modules and compare with the shipped .obj/.lst.
Run from an empty scratch directory, with
  CTOS_LINKER = directory of the CTOS Linker sources and their shipped .obj/.lst
  CTOS_EDF    = directory of the CTOS .edf include files (mapped to :F0:)"""
import os, re, sys, glob, subprocess
L = os.environ['CTOS_LINKER']
E = os.environ['CTOS_EDF']
PLM = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'plm86')
want = sys.argv[1:]
ok = bad = 0
for lst in sorted(glob.glob(L + '/*.lst')):
    base = os.path.basename(lst)[:-4]
    if want and base not in want:
        continue
    obj = os.path.join(L, base + '.obj')
    t = open(lst, 'rb').read().decode('latin1')
    m = re.search(r'COMPILER INVOKED BY:\s+V2.3PLM86.86 (\S+)(.*?)\r?\n', t)
    if not m or not os.path.exists(obj):
        continue
    d = re.search(r'(\d\d/\d\d/\d\d) (\d\d:\d\d:\d\d)  PAGE', t)
    src = m.group(1)
    ctl = m.group(2).split()
    for f in glob.glob(base + '.*'):
        os.remove(f)
    r = subprocess.run([PLM, '-F', 'f1=' + L, '-F', 'f0=' + E, '--date', d.group(1) + ' ' + d.group(2),
                        os.path.join(L, src)] + ctl, capture_output=True)
    mine = base + '.obj'
    if not os.path.exists(mine):
        print('%-10s FAILED rc=%d %s' % (base, r.returncode, r.stdout.decode()[-200:]))
        bad += 1
        continue
    same = open(mine, 'rb').read() == open(obj, 'rb').read()
    mylst = open(base + '.lst', 'rb').read().decode('latin1').replace('\r', '').splitlines()
    ref = t.replace('\r', '').splitlines()
    ldiff = [(a, b) for a, b in zip(mylst, ref) if a != b] + ([('<len>', '%d vs %d' % (len(mylst), len(ref)))] if len(mylst) != len(ref) else [])
    print('%-10s obj %s  lst %s' % (base, 'IDENTICAL' if same else 'DIFFERENT',
          'identical' if not ldiff else 'diff: ' + '; '.join('%s | %s' % (a.strip(), b.strip()) for a, b in ldiff[:3])))
    ok += same
    bad += not same
print('obj identical: %d, different/failed: %d' % (ok, bad))
