#!/usr/bin/env python3
"""cmpx.py <file.obj> <code_start> [--const OFF] [--data OFF] [--hunks N]
Resolve the obj's fixups against tools/symbols.txt (NAME FRAME OFFSET, hex) and the
given placement, then compare byte-exact with the original image (instruction diff)."""
import sys,os,struct,difflib,argparse
T=os.path.dirname(os.path.abspath(__file__)); sys.path.insert(0,T)
import omf2
from capstone import *
ap=argparse.ArgumentParser(); ap.add_argument('obj'); ap.add_argument('start')
ap.add_argument('--const',default=None); ap.add_argument('--data',default=None)
ap.add_argument('--hunks',type=int,default=5); ap.add_argument('--syms',action='append',default=[]); ap.add_argument('--quiet',action='store_true')
a=ap.parse_args()
R=T+'/..'
d=open(R+'/orig/Rats.run','rb').read(); h=struct.unpack_from('<16H',d,0); img=d[h[4]*16:]
DGF=0x7a6
sym={}
for sf in [T+'/symbols.txt']+a.syms:
  for L in open(sf):
    L=L.split('#')[0].split()
    if len(L)>=3: sym[L[0].upper()]=(int(L[1],16),int(L[2],16))
o=omf2.parse(open(a.obj,'rb').read())
start=int(a.start,16)
place={}
for i,s in enumerate(o.segs):
    if s is None: continue
    if s.cls=='CODE': place[i]=(start>>4,start&15)
    elif s.name=='CONST' and a.const: place[i]=(DGF,int(a.const,16))
    elif s.name=='DATA' and a.data: place[i]=(DGF,int(a.data,16))
    elif s.cls in('CONST','DATA','STACK','MEMORY'): place[i]=(DGF,None)
code=[s for s in o.segs if s and s.cls=='CODE'][0]
ci=o.segs.index(code)
buf=bytearray(code.data); unres=set(); unk=[]
def tgt(tm,ti):
    if tm==0:
        f,off=place[ti]
        if off is None: raise KeyError('placement for seg '+o.segs[ti].name)
        return f,off
    if tm==1:
        if o.grps[ti]=='DGROUP': return DGF,0
    if tm==2:
        n=o.exts[ti].upper()
        if n in sym: return sym[n]
        unres.add(n); return None
    raise KeyError((tm,ti))
for fx in code.fix:
    try: t=tgt(fx['tm'],fx['ti'])
    except KeyError as e: unk.append(str(e)); continue
    if t is None: continue
    tf,toff=t; tlin=((tf*16+toff)&0xfffff)+fx['disp']
    fm=fx['fm']
    if fm==5: F=tf if fx['tm']!=0 or o.segs[fx['ti']].cls!='CODE' else start>>4
    elif fm==1: F=DGF
    elif fm==4: F=start>>4
    elif fm==0: F=place[fx['fi']][0]
    elif fm==2:
        F=sym[o.exts[fx['fi']].upper()][0]
    else: F=tf
    p=fx['off']; loc=fx['loc']
    if fx['tm']==0 and o.segs[fx['ti']].cls=='CODE':  # own code
        tlin=start+toff-(start&15) if False else (start-(start&15))+ (start&15)  # unused
    if fx['tm']==0 and o.segs[fx['ti']].cls=='CODE':
        offv=toff+fx['disp']  # own code: toff = segment base offset within its frame
    else:
        offv=(tlin-F*16)&0xffff
    if fx['M']==0:  # self relative
        here=start+p+(2 if loc==1 else 1)
        v=(struct.unpack_from('<H',buf,p)[0]+tlin-here)&0xffff; struct.pack_into('<H',buf,p,v); continue
    if loc in(1,3):
        v=(struct.unpack_from('<H',buf,p)[0]+offv)&0xffff; struct.pack_into('<H',buf,p,v)
    if loc==0:
        buf[p]=(buf[p]+offv)&0xff
    if loc==2: struct.pack_into('<H',buf,p,F)
    if loc==3: struct.pack_into('<H',buf,p+2,F)
if unres: print('UNRESOLVED:',' '.join(sorted(unres)))
if unk: print('UNPLACED:',sorted(set(unk)))
n=len(buf); orig=img[start:start+n]
bad=[i for i in range(n) if buf[i]!=orig[i]]
print(f'{code.name}: len {n:#x} -> orig {start:#05x}-{start+n:#05x}; differing bytes {len(bad)}')
if not bad or a.quiet: sys.exit(0 if not bad else 1)
md=Cs(CS_ARCH_X86,CS_MODE_16)
def lst(bb,base):
    out=[]
    for i in md.disasm(bytes(bb),base):
        k=i.bytes.hex()
        if i.mnemonic.startswith('j') or i.mnemonic in ('loop','jcxz'): k=i.mnemonic+str(i.size)
        elif i.mnemonic=='lcall' and i.bytes[0]==0x9a: k='lcall'+i.bytes[3:5].hex()
        out.append((k,i.address,f'{i.bytes.hex():12} {i.mnemonic} {i.op_str}'))
    return out
A=lst(buf,start); B=lst(img[start:start+n+64],start)
sm=difflib.SequenceMatcher(None,[x[0] for x in A],[x[0] for x in B],autojunk=False)
ops=[x for x in sm.get_opcodes() if x[0]!='equal']
for tag,i1,i2,j1,j2 in ops[:a.hunks]:
    print(f'--- {tag}')
    for x in A[max(0,i1-2):i2+1][:14]: print('  NEW ',f'{x[1]:05x}',x[2])
    for x in B[max(0,j1-2):j2+1][:14]: print('  ORIG',f'{x[1]:05x}',x[2])
sys.exit(1)
