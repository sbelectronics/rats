"""Parser for Intel OMF-86 LTL (load-time-locatable) overlay modules (.86 files)."""
import struct

def _idx(b, p):
    v = b[p]
    if v & 0x80:
        return ((v & 0x7f) << 8) | b[p+1], p+2
    return v, p+1

def _name(b, p):
    n = b[p]
    return b[p+1:p+1+n].decode('latin1'), p+1+n

class Seg:
    pass

class Grp:
    pass

class Ovl:
    pass

class Module:
    def __init__(self, data):
        self.b = data
        self.lnames = [None]
        self.segs = [None]
        self.grps = [None]
        self.ovls = [None]
        self.regint = {}
        self.parse()

    def records(self, start):
        b = self.b
        p = start
        while p < len(b):
            t = b[p]
            l = struct.unpack_from('<H', b, p+1)[0]
            yield p, t, b[p+3:p+2+l]
            p += 3 + l
            if t in (0x78, 0x8A):
                return

    def parse(self):
        b = self.b
        for p, t, r in self.records(0):
            if t == 0x6E:
                self.name, q = _name(r, 0)
                (self.attr,) = struct.unpack_from('<B', r, q)
                (self.nseg, self.ngrp, self.novl, self.ovloff, self.static, self.maxsize,
                 self.dyn, self.maxdyn) = struct.unpack_from('<HHHIIIII', r, q+1)
            elif t == 0x96:
                q = 0
                while q < len(r):
                    n, q = _name(r, q)
                    self.lnames.append(n)
            elif t == 0x98:
                s = Seg()
                s.acbp = r[0]
                q = 1
                assert s.acbp >> 5 == 6, hex(s.acbp)
                s.ltldat = r[q]
                s.maxlen, s.goff = struct.unpack_from('<HH', r, q+1)
                q += 5
                (s.length,) = struct.unpack_from('<H', r, q)
                q += 2
                ni, q = _idx(r, q)
                ci, q = _idx(r, q)
                oi, q = _idx(r, q)
                s.name = self.lnames[ni]
                s.cls = self.lnames[ci]
                s.ovl = oi
                s.grp = None
                s.index = len(self.segs)
                self.segs.append(s)
            elif t == 0x9A:
                g = Grp()
                ni, q = _idx(r, 0)
                g.name = self.lnames[ni]
                assert r[q] == 0xFB
                g.ltldat = r[q+1]
                g.maxlen, g.length = struct.unpack_from('<HH', r, q+2)
                q += 6
                g.segs = []
                while q < len(r):
                    assert r[q] == 0xFF
                    si, q = _idx(r, q+1)
                    g.segs.append(si)
                    self.segs[si].grp = len(self.grps)
                g.index = len(self.grps)
                self.grps.append(g)
            elif t == 0x76:
                o = Ovl()
                o.name, q = _name(r, 0)
                o.loc, o.attr = struct.unpack_from('<IB', r, q)
                o.index = len(self.ovls)
                self.ovls.append(o)
            elif t == 0x70:
                q = 0
                while q < len(r):
                    rt = r[q]
                    reg = rt >> 6
                    q += 1
                    gi, q = _idx(r, q)
                    si, q = _idx(r, q)
                    if reg in (0, 1):
                        (off,) = struct.unpack_from('<H', r, q)
                        q += 2
                    else:
                        off = None
                    self.regint[('CSIP', 'SSSP', 'DS', 'ES')[reg]] = (gi, si, off)
            elif t == 0x8A:
                break
            if t == 0x70:
                break

    def overlay_records(self, ovl):
        """Yield (kind, group, seg, offset, bytes, fixups) for each data record in overlay."""
        o = self.ovls[ovl]
        last = None
        for p, t, r in self.records(o.loc):
            if t in (0x72, 0x74):
                if last:
                    yield last
                gi, q = _idx(r, 0)
                si, q = _idx(r, q)
                (off,) = struct.unpack_from('<H', r, q)
                q += 2
                if t == 0x72:
                    data = bytes(r[q:])
                else:
                    data = b''
                    while q < len(r):
                        d, q = self._iter(r, q)
                        data += d
                last = [gi, si, off, data, []]
            elif t == 0x9C:
                q = 0
                while q < len(r):
                    fx, q = self._fixup(r, q)
                    if fx is not None:
                        last[4].append(fx)
            elif t in (0x78, 0x8A):
                if last:
                    yield last
                return
            elif t == 0x88:      # COMENT
                continue
            else:
                raise ValueError('unexpected record %02x at %x' % (t, p))

    def _iter(self, r, q):
        rep, cnt = struct.unpack_from('<HH', r, q)
        q += 4
        if cnt == 0:
            n = r[q]
            d = bytes(r[q+1:q+1+n])
            q += 1 + n
        else:
            d = b''
            for i in range(cnt):
                dd, q = self._iter(r, q)
                d += dd
        return d * rep, q

    def _fixup(self, r, q):
        v = r[q]
        if not (v & 0x80):
            raise ValueError('THREAD fixups not supported')
        locat = (v << 8) | r[q+1]
        q += 2
        m = (locat >> 14) & 1
        loc = (locat >> 10) & 7
        doff = locat & 0x3ff
        fixdat = r[q]
        q += 1
        f = fixdat >> 7
        frame = (fixdat >> 4) & 7
        tt = (fixdat >> 3) & 1
        p = (fixdat >> 2) & 1
        targ = fixdat & 3
        assert not f and not tt
        fdatum = None
        if frame < 4:
            fdatum, q = _idx(r, q)
        tdatum, q = _idx(r, q)
        disp = 0
        if not p:
            (disp,) = struct.unpack_from('<H', r, q)
            q += 2
        return dict(m=m, loc=loc, off=doff, frame=frame, fdatum=fdatum, targ=targ,
                    tdatum=tdatum, disp=disp), q

if __name__ == '__main__':
    import sys
    m = Module(open(sys.argv[1], 'rb').read())
    print('RHEADR', m.name, 'attr', m.attr, 'static %x max %x dyn %x maxdyn %x' % (m.static, m.maxsize, m.dyn, m.maxdyn))
    for s in m.segs[1:]:
        print('seg %2d %-20s %-8s ovl %d grp %s goff %05x len %05x max %05x ltldat %02x' % (s.index, s.name, s.cls, s.ovl, s.grp, s.goff, s.length, s.maxlen, s.ltldat))
    for g in m.grps[1:]:
        print('grp %d %s len %x max %x segs %s' % (g.index, g.name, g.length, g.maxlen, g.segs))
    for o in m.ovls[1:]:
        print('ovl %d %s loc %x attr %x' % (o.index, o.name, o.loc, o.attr))
    print(m.regint)
    for o in m.ovls[1:]:
        for gi, si, off, data, fx in m.overlay_records(o.index):
            print('  ovl %d g%d s%d %04x len %04x end %04x %s' % (o.index, gi, si, off, len(data), off+len(data), fx if fx else ''))
