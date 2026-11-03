import struct,sys,re,os
from capstone import *
R=os.path.dirname(os.path.abspath(__file__))+'/..'
d=open(R+'/orig/Rats.run','rb').read()
h=struct.unpack_from('<16H',d,0)
img=bytes(d[h[4]*16:])
DGB=0x7a60
rel=[struct.unpack_from('<2H',d,h[12]+4*i) for i in range(h[3])]
fix={}
for ra,sa in rel:
    lin=sa*16+ra; fix[lin]=struct.unpack_from('<H',img,lin)[0]
rq={};os_={}
for L in open(R+'/work/rqnames.txt'):
    n,nm=L.strip().split(',',1); rq.setdefault(int(n),nm)
for L in open(R+'/work/ossub.txt'):
    n,nm=L.strip().split(',',1); os_.setdefault(int(n),nm)
names={}   # code linear -> name
dnames={}  # dgroup offset -> name
if os.path.exists(R+'/work/names.txt'):
    for L in open(R+'/work/names.txt'):
        L=L.split('#')[0].strip()
        if not L: continue
        a,n=L.split()[:2]
        if a.startswith('d'): dnames[int(a[1:],16)]=n
        else: names[int(a,16)]=n
def ctos(cs,ip):
    lin=(cs*16+ip)&0xfffff
    if 0x210<=lin<0x220:
        tmp=(-cs)&0xffff; r=tmp+(((ip-0x210-tmp*16)&0xf)>>1)
        return f'CTOS_RQ{r}_{rq.get(r,"?")}'
    if 0x220<=lin<0x230:
        k=(0xffff-cs)//16; n=8*k+(lin-0x220)//2
        return f'OSSUB{n}_{os_.get(n,"?")}'
def nm(lin): return names.get(lin,f'p_{lin:05x}')
def strat(off):
    s=img[DGB+off:DGB+off+40]
    m=re.match(rb'[\x20-\x7e]{3,}',s)
    if m: return repr(m.group().decode())
    return None
def danno(v):
    if v in dnames: return dnames[v]
    s=strat(v)
    if s: return 'str '+s
    if v>0 and strat(v+1) and 0<img[DGB+v]<80: return f'sb[{img[DGB+v]}] '+strat(v+1)
    return None
# collect call targets for labels
targets=set()
for ra,sa in rel:
    lin=sa*16+ra
    if lin>=2 and img[lin-3]==0x9a: targets.add(fix[lin]*16+struct.unpack_from('<H',img,lin-2)[0])
start=int(sys.argv[1],16); end=int(sys.argv[2],16)
segbase=int(sys.argv[3],16)*16 if len(sys.argv)>3 else (start//16)*16
md=Cs(CS_ARCH_X86,CS_MODE_16); md.skipdata=True
prev_ret=True
for ins in md.disasm(img[start:end],start-segbase):
    lin=ins.address+segbase
    if lin in targets or lin in names or (prev_ret and ins.mnemonic=='push' and ins.op_str=='bp'):
        print(f'\n{nm(lin)}:   ; {lin:05x}' + (' (FAR ENTRY)' if lin in targets else ''))
    prev_ret = ins.mnemonic in ('ret','retf','jmp','iret')
    f=[k for k in range(lin,lin+ins.size) if k in fix]
    op=ins.op_str; ann=[]
    if ins.mnemonic=='lcall' and ins.bytes[0]==0x9a:
        off,cs=struct.unpack_from('<HH',ins.bytes,1)
        if f: op=nm(fix[f[0]]*16+off)
        else:
            c=ctos(cs,off); op=c if c else op
        f=[]
    if f: ann.append('SEG '+','.join(hex(fix[k]) for k in f))
    for v in re.findall(r'0x([0-9a-f]+)',op):
        v=int(v,16)
        if v>=0x40:
            a=danno(v)
            if a: ann.append(a)
    print(f'  {lin:05x} {ins.bytes.hex():<12} {ins.mnemonic:6} {op}' + ('   ; '+' | '.join(ann) if ann else ''))
