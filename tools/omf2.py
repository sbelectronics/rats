"""Full OMF-86 object parser with resolved fixup records (for single .obj files)."""
import struct
def idx(b,p):
    v=b[p]
    if v&0x80: return ((v&0x7f)<<8)|b[p+1], p+2
    return v,p+1
def name(b,p):
    n=b[p]; return b[p+1:p+1+n].decode('latin1'),p+1+n
class Seg:
    def __init__(s,name,cls,length):
        s.name=name;s.cls=cls;s.length=length;s.data=bytearray(length);s.fix=[]
class Obj:
    pass
def parse(b):
    o=Obj(); o.lnames=[None]; o.segs=[None]; o.grps=[None]; o.exts=[None]; o.pubs=[]
    thr={}  # ('T'|'F', n) -> (method, index)
    last=None; p=0
    while p+3<=len(b):
        t=b[p]; l=struct.unpack_from('<H',b,p+1)[0]; r=b[p+3:p+2+l]; p+=3+l
        if t==0x80: o.name=name(r,0)[0]
        elif t==0x96:
            q=0
            while q<len(r): n,q=name(r,q); o.lnames.append(n)
        elif t==0x98:
            acbp=r[0];q=1
            if (acbp>>5)==0: q+=3
            ln=struct.unpack_from('<H',r,q)[0];q+=2
            if acbp&2: ln=0x10000
            ni,q=idx(r,q);ci,q=idx(r,q)
            o.segs.append(Seg(o.lnames[ni],o.lnames[ci],ln))
        elif t==0x9A:
            ni,q=idx(r,0); o.grps.append(o.lnames[ni])
        elif t==0x8C:
            q=0
            while q<len(r):
                n,q=name(r,q); ti,q=idx(r,q); o.exts.append(n)
        elif t==0x90:
            gi,q=idx(r,0); si,q=idx(r,q)
            if gi==0 and si==0: q+=2
            while q<len(r):
                n,q=name(r,q); off=struct.unpack_from('<H',r,q)[0];q+=2; ti,q=idx(r,q)
                o.pubs.append((n,si,off))
        elif t==0xA0:
            si,q=idx(r,0); off=struct.unpack_from('<H',r,q)[0];q+=2
            s=o.segs[si]; dat=r[q:]; s.data[off:off+len(dat)]=dat; last=(s,off)
        elif t==0xA2:
            si,q=idx(r,0); off=struct.unpack_from('<H',r,q)[0];q+=2
            s=o.segs[si]
            def blk(q,off):
                rep=struct.unpack_from('<H',r,q)[0]; bc=struct.unpack_from('<H',r,q+2)[0]; q+=4
                if bc==0:
                    n=r[q]; dat=r[q+1:q+1+n]; q+=1+n
                    for _ in range(rep): s.data[off:off+n]=dat; off+=n
                    return q,off
                st=q
                for _ in range(rep):
                    q=st
                    for _ in range(bc): q,off=blk(q,off)
                return q,off
            while q<len(r): q,off=blk(q,off)
            last=None
        elif t==0x9C:
            q=0
            while q<len(r):
                c=r[q]
                if c&0x80:
                    lo=((c&3)<<8)|r[q+1]; M=(c>>6)&1; loc=(c>>2)&7; q+=2
                    fd=r[q]; q+=1
                    if fd&0x80: fm,fi=thr[('F',(fd>>4)&3)]
                    else:
                        fm=(fd>>4)&7; fi=None
                        if fm<3: fi,q=idx(r,q)
                    if fd&8: tm,ti=thr[('T',fd&3)]; tm=tm&3|(fd&4)
                    else:
                        tm=fd&7; ti,q=idx(r,q)
                    disp=0
                    if not (fd&4): disp=struct.unpack_from('<H',r,q)[0]; q+=2
                    s,off=last
                    s.fix.append(dict(off=off+lo,M=M,loc=loc,fm=fm,fi=fi,tm=tm&3,ti=ti,disp=disp))
                else:
                    d=(c>>6)&1; meth=(c>>2)&7; n=c&3; q+=1
                    i=None
                    if d==0 or meth<3: i,q=idx(r,q)
                    thr[('F' if d else 'T',n)]=(meth,i)
        elif t==0x8A: break
    return o
