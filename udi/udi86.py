#!/usr/bin/env python3
"""udi86 - run Intel OMF-86 LTL (".86") UDI programs (e.g. PL/M-86 V2.3) on Linux.

The program is loaded into a unicorn 8086 real-mode machine.  Programs reach the
UDI (DQ$xxx) system services through INT 0C3h with AX = function number; the
stack at the INT holds a far return address followed by the PL/M parameters
(last parameter at the lowest address).  This mirrors the dispatcher found in
CTOS udi.run ("x10.1 (17May88)").
"""
import os, sys, struct, time, tempfile, argparse
from unicorn import Uc, UC_ARCH_X86, UC_MODE_16, UC_HOOK_INTR, UcError
from unicorn.x86_const import *

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from omf86 import Module

# ---------------------------------------------------------------- exceptions
E_OK = 0
E_MEM = 2
E_FEXIST = 0x20
E_FNEXIST = 0x21
E_SUPPORT = 0x23   # udi.run returns 23h for bad connection / bad parameters
E_FACCESS = 0x26

# ---------------------------------------------------------------- layout
CGROUP_PARA = 0x0100
DGROUP_PARA = 0x1100
OGROUP_PARA = 0x2100
HEAP_PARA = 0x2200
HEAP_END_PARA = 0xA000

# function number -> (name, parameter types in declaration order)
#   w = WORD/BYTE (one stack word), p = POINTER (far, 2 words)
DQ = {
    0: ('DQ$ALLOCATE', 'wp'),
    1: ('DQ$FREE', 'wp'),
    2: ('DQ$GET$SIZE', 'wp'),
    6: ('DQ$DECODE$EXCEPTION', 'wpp'),
    7: ('DQ$GET$SYSTEM$ID', 'pp'),
    8: ('DQ$GET$TIME', 'pp'),
    9: ('DQ$DELETE', 'pp'),
    11: ('DQ$ATTACH', 'pp'),
    12: ('DQ$CREATE', 'pp'),
    13: ('DQ$OPEN', 'wwwp'),
    14: ('DQ$CLOSE', 'wp'),
    15: ('DQ$DETACH', 'wp'),
    16: ('DQ$READ', 'wpwp'),
    17: ('DQ$WRITE', 'wpwp'),
    18: ('DQ$SEEK', 'wwwwp'),
    20: ('DQ$GET$CONNECTION$STATUS', 'wpp'),
    21: ('DQ$CHANGE$EXTENSION', 'ppp'),
    22: ('DQ$OVERLAY', 'pp'),
    23: ('DQ$TRAP$CC', 'p'),
    24: ('DQ$GET$ARGUMENT', 'pp'),
    25: ('DQ$SWITCH$BUFFER', 'pp'),
    26: ('DQ$EXIT', 'w'),
    28: ('DQ$RESERVE$IO$MEMORY', 'wwp'),
}


class Exit(Exception):
    def __init__(self, code):
        self.code = code


class Conn:
    def __init__(self, kind, path=None, writable=False):
        self.kind = kind          # 'file', 'ci', 'co', 'null'
        self.path = path
        self.writable = writable
        self.fh = None
        self.open = False
        self.access = 0
        self.mode = 1 if not writable else 3


class UDI:
    def __init__(self, prog, cmdline, drives, search, write_root, date=None,
                 heap_kb=None, sysid=b'x10.1 (17May88)', trace=False, workdir=None, ff_char=None):
        self.mod = Module(open(prog, 'rb').read())
        self.cmdline = cmdline
        self.drives = drives          # 'F0' -> [dirs]
        self.search = search          # dirs for plain names (read)
        self.write_root = [os.path.realpath(r) for r in write_root]
        self.date = date
        self.sysid = sysid
        self.trace = trace
        self.own_workdir = workdir is None
        self._workdir = workdir
        self.conns = {}
        self.next_conn = 1
        self.ci_data = cmdline
        self.ci_pos = 0
        self.ff_char = ff_char
        self.arg_buf = None           # None = the command line itself
        self.arg_pos = 0
        self.arg_len = 0
        while self.arg_len < len(cmdline) and cmdline[self.arg_len] >= 0x20:
            self.arg_len += 1
        self.heap_end = HEAP_END_PARA if heap_kb is None else HEAP_PARA + heap_kb * 64
        self.mtab = []                # udi memory table: (size, seg, in_use)
        self.brk = HEAP_PARA
        self.ovl_loaded = None
        self.pending_overlay = None
        self.uc = Uc(UC_ARCH_X86, UC_MODE_16)
        self.uc.mem_map(0, 0x100000)
        self.grp_para = {1: CGROUP_PARA, 2: DGROUP_PARA, 3: OGROUP_PARA}
        for g in self.mod.grps[1:]:
            assert g.index in self.grp_para, g.name
        self.load_overlay(1)
        # udi.run allocates every group (max group length) through its DQ$ALLOCATE
        # table, so DQ$GET$SIZE(DS) works (Link86 uses it to find its MEMORY size)
        for g in self.mod.grps[1:]:
            self.mtab.append((g.maxlen, self.grp_para[g.index], True))
        self.uc.hook_add(UC_HOOK_INTR, self.on_int)

    # ------------------------------------------------------------ loading
    @property
    def workdir(self):
        """scratch dir for :WORK: files, created on first use inside the cwd"""
        if self._workdir is None:
            self._workdir = tempfile.mkdtemp(prefix=".udi86_work_", dir=os.getcwd())
        return self._workdir

    def seg_frame(self, si):
        s = self.mod.segs[si]
        assert s.goff % 16 == 0
        return self.grp_para[s.grp] + s.goff // 16

    def load_overlay(self, ovl):
        m = self.mod
        for gi, si, off, data, fixups in m.overlay_records(ovl):
            s = m.segs[si]
            lin = self.grp_para[s.grp] * 16 + s.goff + off
            buf = bytearray(data)
            for fx in fixups:
                assert fx['loc'] == 2 and fx['m'] == 1 and fx['frame'] == 5, fx
                if fx['targ'] == 0:
                    v = self.seg_frame(fx['tdatum'])
                elif fx['targ'] == 1:
                    v = self.grp_para[fx['tdatum']]
                else:
                    raise ValueError('fixup %r' % fx)
                o = fx['off']
                old = struct.unpack_from('<H', buf, o)[0]
                struct.pack_into('<H', buf, o, (old + v) & 0xffff)
            self.uc.mem_write(lin, bytes(buf))
        self.ovl_loaded = ovl

    # ------------------------------------------------------------ memory helpers
    def rd(self, seg, off, n):
        return bytes(self.uc.mem_read(seg * 16 + off, n))

    def wr(self, seg, off, data):
        self.uc.mem_write(seg * 16 + off, bytes(data))

    def rw(self, seg, off):
        return struct.unpack('<H', self.rd(seg, off, 2))[0]

    def ww(self, seg, off, v):
        self.wr(seg, off, struct.pack('<H', v & 0xffff))

    def rstr(self, p):
        seg, off = p
        n = self.rd(seg, off, 1)[0]
        return self.rd(seg, off + 1, n)

    # ------------------------------------------------------------ paths
    def _find_ci(self, d, name):
        """case-insensitive lookup of name in dir d"""
        p = os.path.join(d, name)
        if os.path.exists(p):
            return p
        try:
            low = name.lower()
            for e in os.listdir(d):
                if e.lower() == low:
                    return os.path.join(d, e)
        except OSError:
            pass
        return None

    def split_path(self, path):
        s = path.decode('latin1').strip()
        if len(s) >= 4 and s[0] == ':' and s[3] == ':':
            return s[1:3].upper(), s[4:]
        if s.upper().startswith(':WORK:'):
            return 'WORK', s[6:]
        return None, s

    def resolve_read(self, path):
        dev, name = self.split_path(path)
        if dev == 'WORK':
            return self._find_ci(self.workdir, name)
        dirs = self.drives.get(dev, []) if dev else self.search
        if dev and dev not in self.drives:
            dirs = []
        for d in dirs:
            p = self._find_ci(d, name)
            if p:
                return p
        return None

    def resolve_write(self, path):
        dev, name = self.split_path(path)
        if dev == 'WORK':
            if not name:
                name = 'udi%d.tmp' % len(self.conns)
            return os.path.join(self.workdir, name)
        if dev:
            dirs = self.drives.get(dev)
            if not dirs:
                return None
            d = dirs[0]
        else:
            d = os.getcwd()
        p = self._find_ci(d, name) or os.path.join(d, name)
        return p

    def write_ok(self, p):
        rp = os.path.realpath(p)
        if self._workdir and rp.startswith(os.path.realpath(self._workdir) + os.sep):
            return True
        for r in self.write_root:
            if rp == r or rp.startswith(r + os.sep):
                return True
        return False

    def device(self, path):
        dev, name = self.split_path(path)
        if dev in ('CI',):
            return 'ci'
        if dev in ('CO', 'TO'):
            return 'co'
        if dev in ('BB',):
            return 'null'
        if dev in ('LP',):
            return 'co'
        return None

    # ------------------------------------------------------------ the DQ calls
    def on_int(self, uc, intno, user):
        if intno not in (0xC1, 0xC3):   # udi.run hooks both vectors to one dispatcher
            print('udi86: unexpected INT %02Xh at %04X:%04X' % (intno, uc.reg_read(UC_X86_REG_CS),
                  uc.reg_read(UC_X86_REG_IP)), file=sys.stderr)
            self.exit_code = 99
            uc.emu_stop()
            return
        fn = uc.reg_read(UC_X86_REG_AX)
        ss = uc.reg_read(UC_X86_REG_SS)
        sp = uc.reg_read(UC_X86_REG_SP)
        if fn not in DQ:
            print('udi86: unimplemented UDI function %d' % fn, file=sys.stderr)
            self.exit_code = 98
            uc.emu_stop()
            return
        name, sig = DQ[fn]
        args = []
        o = sp + 4
        for t in reversed(sig):
            if t == 'w':
                args.append(self.rw(ss, o))
                o += 2
            else:
                off = self.rw(ss, o)
                seg = self.rw(ss, o + 2)
                args.append((seg, off))
                o += 4
        args.reverse()
        try:
            res, exc = getattr(self, 'dq_' + name[3:].replace('$', '_').lower())(*args)
        except Exit as e:
            self.exit_code = e.code
            uc.emu_stop()
            return
        if sig.endswith('p') and name not in ('DQ$TRAP$CC',):
            ep = args[-1]
            self.ww(ep[0], ep[1], exc)
        if self.trace:
            pa = ', '.join(('%04X:%04X' % a) if isinstance(a, tuple) else '%X' % a for a in args)
            print('[udi] %04X:%04X ovl%d %s(%s) -> %s exc=%X' % (self.rw(ss, sp + 2), self.rw(ss, sp), self.ovl_loaded, name, pa, res, exc), file=sys.stderr)
        if res is not None:
            uc.reg_write(UC_X86_REG_AX, res & 0xffff)

    # memory
    # Memory: modelled on udi.run's allocator.  It keeps a table of at most 80
    # (size, segment, in-use) entries.  ALLOCATE re-uses the best fitting freed
    # entry, otherwise it gets new memory from the OS (here: a bump allocator).
    # FREE marks entries free and gives trailing free entries back to the OS.
    # Memory is NOT cleared (programs may rely on that).
    def dq_allocate(self, size, ep):
        best = None
        for i, (sz, seg, used) in enumerate(self.mtab):
            if not used and sz >= size and (best is None or sz <= self.mtab[best][0]):
                best = i
        if best is not None:
            sz, seg, used = self.mtab[best]
            self.mtab[best] = (sz, seg, True)
            return seg, E_OK
        if len(self.mtab) >= 80:
            return 0xFFFF, E_MEM
        nbytes = size if size else 0x10000
        paras = (nbytes + 15) // 16
        if self.brk + paras > self.heap_end:
            return 0xFFFF, E_MEM
        seg = self.brk
        self.brk += paras
        self.mtab.append((size, seg, True))
        return seg, E_OK

    def dq_free(self, seg, ep):
        for i, (sz, s, used) in enumerate(self.mtab):
            if s == seg:
                self.mtab[i] = (sz, s, False)
        while self.mtab and not self.mtab[-1][2]:
            sz, s, used = self.mtab.pop()
            if s + ((sz if sz else 0x10000) + 15) // 16 == self.brk:
                self.brk = s
        return None, E_OK

    def dq_get_size(self, seg, ep):
        for sz, s, used in self.mtab:
            if s == seg and used:
                return sz, E_OK
        return 0, E_SUPPORT

    def dq_reserve_io_memory(self, nfiles, nbufs, ep):
        return None, E_OK

    def dq_trap_cc(self, p):
        return None, E_OK

    # command line parsing (semantics copied from udi.run)
    DELIMS = b',()=!$%\\~+-&|[]<>;'

    def _argbuf(self, i):
        if self.arg_buf is None:
            return self.cmdline[i]
        seg, off = self.arg_buf
        return self.rd(seg, off + i, 1)[0]

    def dq_get_argument(self, argp, ep):
        arg = bytearray()
        start = self.arg_pos
        n = self.arg_len

        def ch():
            return self._argbuf(self.arg_pos)
        while self.arg_pos < n and ch() == 0x20:
            self.arg_pos += 1
        res = 0x0D
        while self.arg_pos < n:
            c = ch()
            if c <= 0x20 or c in self.DELIMS:
                self.arg_pos += 1
                if c == 0x20:
                    while self.arg_pos < n and ch() == 0x20:
                        self.arg_pos += 1
                    if self.arg_pos < n:
                        c2 = ch()
                        if c2 > 0x20 and c2 in self.DELIMS:
                            c = c2
                            self.arg_pos += 1
                res = c
                break
            if c == 0x27:
                self.arg_pos += 1
                while self.arg_pos < n and ch() != 0x27:
                    arg.append(ch())
                    self.arg_pos += 1
                self.arg_pos += 1
                continue
            if 0x61 <= c <= 0x7a:
                c -= 0x20
            arg.append(c)
            self.arg_pos += 1
        self.wr(argp[0], argp[1], bytes([len(arg) & 0xff]) + bytes(arg))
        return res, E_OK

    def dq_switch_buffer(self, bufp, ep):
        old = self.arg_pos
        self.arg_buf = bufp
        self.arg_pos = 0
        n = 0
        while self._argbuf(n) >= 0x20:
            n += 1
        self.arg_len = n
        return old, E_OK

    # misc
    def dq_get_system_id(self, p, ep):
        self.wr(p[0], p[1], bytes([len(self.sysid)]) + self.sysid)
        return None, E_OK

    def dq_get_time(self, p, ep):
        if self.date:
            s = self.date
        else:
            s = time.strftime('%m/%d/%y%H:%M:%S')
        s = s.replace(' ', '')
        self.wr(p[0], p[1], s.encode()[:16].ljust(16, b'0'))
        return None, E_OK

    def dq_decode_exception(self, code, p, ep):
        buf = bytearray(b' ' * 0x51)
        buf[0:10] = b'\x11Exception'
        i = 0x11
        while True:
            buf[i] = 0x30 + code % 10
            code //= 10
            if not code:
                break
            i -= 1
        self.wr(p[0], p[1], buf)
        return None, E_OK

    def dq_change_extension(self, pathp, extp, ep):
        path = bytearray(self.rstr(pathp))
        ext = self.rd(extp[0], extp[1], 3)
        i = path.rfind(b'.')
        if i < 0:
            i = len(path)
        path = path[:i]
        if ext[0:1] != b' ':
            path += b'.'
            for c in ext:
                if c == 0x20:
                    break
                path.append(c + 0x20 if 0x41 <= c <= 0x5a else c)
        self.wr(pathp[0], pathp[1], bytes([len(path)]) + path)
        return None, E_OK

    def dq_exit(self, code):
        raise Exit(code)

    def dq_overlay(self, namep, ep):
        name = self.rstr(namep).decode('latin1').strip().upper()
        for o in self.mod.ovls[1:]:
            if o.name.upper() == name:
                # loaded outside emu_start so unicorn's translation cache can be flushed
                self.pending_overlay = o.index
                self.uc.emu_stop()
                return None, E_OK
        print('udi86: overlay %s not found' % name, file=sys.stderr)
        return None, E_FNEXIST

    # files
    def new_conn(self, c):
        # udi.run hands out the lowest free slot of 12
        n = 0
        while n in self.conns:
            n += 1
        self.conns[n] = c
        return n

    def dq_attach(self, pathp, ep):
        path = self.rstr(pathp)
        dev = self.device(path)
        if dev:
            return self.new_conn(Conn(dev, path, writable=(dev != 'ci'))), E_OK
        p = self.resolve_read(path)
        if self.trace:
            print('[udi]   attach %s -> %s' % (path, p), file=sys.stderr)
        if not p or not os.path.isfile(p):
            return 0, E_FNEXIST
        c = Conn('file', p, writable=False)
        c.mode = 1
        return self.new_conn(c), E_OK

    def dq_create(self, pathp, ep):
        path = self.rstr(pathp)
        dev = self.device(path)
        if dev:
            return self.new_conn(Conn(dev, path, writable=True)), E_OK
        p = self.resolve_write(path)
        if self.trace:
            print('[udi]   create %s -> %s' % (path, p), file=sys.stderr)
        if not p or not self.write_ok(p):
            print('udi86: refusing to create %s (%s): outside write root' % (path.decode('latin1'), p),
                  file=sys.stderr)
            return 0, E_FACCESS
        open(p, 'wb').close()
        c = Conn('file', p, writable=True)
        c.mode = 3
        return self.new_conn(c), E_OK

    def dq_delete(self, pathp, ep):
        path = self.rstr(pathp)
        dev, name = self.split_path(path)
        # resolve like an output file (cwd / first dir of the drive), never via the read search path
        p = self.resolve_write(path)
        if p and not os.path.isfile(p):
            p = None
        if not p:
            return None, E_FNEXIST
        if not self.write_ok(p):
            print('udi86: refusing to delete %s' % p, file=sys.stderr)
            return None, E_FACCESS
        os.remove(p)
        return None, E_OK

    def get_conn(self, n):
        return self.conns.get(n)

    def dq_open(self, n, access, nbuf, ep):
        c = self.get_conn(n)
        if not c or c.open:
            return None, E_SUPPORT
        access &= 0xff
        if access not in (1, 2, 3):
            return None, E_SUPPORT
        if c.kind == 'file':
            if access & 2 and not c.writable:
                return None, E_FACCESS
            c.fh = open(c.path, 'r+b' if c.writable else 'rb')
        c.access = access
        c.open = True
        return None, E_OK

    def dq_close(self, n, ep):
        c = self.get_conn(n)
        if not c or not c.open:
            return None, E_SUPPORT
        if c.fh:
            c.fh.close()
            c.fh = None
        c.open = False
        return None, E_OK

    def dq_detach(self, n, ep):
        c = self.conns.pop(n, None)
        if not c:
            return None, E_SUPPORT
        if c.fh:
            c.fh.close()
        return None, E_OK

    def dq_read(self, n, bufp, count, ep):
        c = self.get_conn(n)
        if not c or not c.open or not (c.access & 1):
            return 0, E_SUPPORT
        if c.kind == 'ci':
            data = self.ci_data[self.ci_pos:self.ci_pos + count]
            self.ci_pos += len(data)
        elif c.kind == 'file':
            data = c.fh.read(count)
            if self.ff_char is not None:   # udi.run maps form feed -> LF for PL/M
                data = data.replace(b"\x0c", bytes([self.ff_char]))
        else:
            data = b''
        self.wr(bufp[0], bufp[1], data)
        return len(data), E_OK

    def dq_write(self, n, bufp, count, ep):
        c = self.get_conn(n)
        if not c or not c.open or not (c.access & 2):
            return None, E_SUPPORT
        data = self.rd(bufp[0], bufp[1], count)
        if c.kind == 'co':
            sys.stdout.buffer.write(data.replace(b'\r', b''))
            sys.stdout.flush()
        elif c.kind == 'file':
            c.fh.write(data)
        return None, E_OK

    def dq_seek(self, n, mode, high, low, ep):
        c = self.get_conn(n)
        if not c or not c.open or not (1 <= mode <= 4):
            return None, E_SUPPORT
        if c.kind != 'file':
            return None, E_OK
        off = (high << 16) | low
        cur = c.fh.tell()
        if mode == 1:
            new = max(0, cur - off)
        elif mode == 2:
            new = off
        elif mode == 3:
            new = cur + off
        else:
            c.fh.seek(0, 2)
            new = max(0, c.fh.tell() - off)
        c.fh.seek(new)
        return None, E_OK

    def dq_get_connection_status(self, n, infop, ep):
        c = self.get_conn(n)
        if not c:
            return None, E_SUPPORT
        acc = [0, 2, 5, 0xf][c.mode & 3]
        pos = c.fh.tell() if c.fh else 0
        seek = 3 if c.kind == 'file' else 0
        self.wr(infop[0], infop[1], struct.pack('<BBBI', 0xff if c.open else 0, acc, seek, pos))
        return None, E_OK

    # ------------------------------------------------------------ run
    def run(self):
        try:
            return self._run()
        finally:
            for c in self.conns.values():
                if c.fh:
                    c.fh.close()
            if self.own_workdir and self._workdir:
                import shutil
                shutil.rmtree(self._workdir, ignore_errors=True)

    def _run(self):
        m = self.mod
        uc = self.uc
        gi, si, off = m.regint['CSIP']
        cs = self.grp_para[m.segs[si].grp]
        ip = m.segs[si].goff + off
        gi, si, off = m.regint['SSSP']
        ss = self.grp_para[m.segs[si].grp]
        sp = m.segs[si].goff + off
        gi, si, _ = m.regint['DS']
        ds = self.grp_para[m.segs[si].grp]
        for r, v in ((UC_X86_REG_SS, ss), (UC_X86_REG_SP, sp), (UC_X86_REG_DS, ds),
                     (UC_X86_REG_ES, ds), (UC_X86_REG_CS, cs)):
            uc.reg_write(r, v)
        self.exit_code = None
        begin = cs * 16 + ip
        while True:
            try:
                uc.emu_start(begin, 0xFFFFF + 1)
            except UcError as e:
                cs = uc.reg_read(UC_X86_REG_CS)
                ip = uc.reg_read(UC_X86_REG_IP)
                print('udi86: CPU fault %s at %04X:%04X' % (e, cs, ip), file=sys.stderr)
                return 97
            if self.exit_code is not None:
                break
            if self.pending_overlay is not None:
                self.load_overlay(self.pending_overlay)
                self.pending_overlay = None
                uc.ctl_flush_tb()
            cs = uc.reg_read(UC_X86_REG_CS)
            ip = uc.reg_read(UC_X86_REG_IP)
            begin = cs * 16 + ip
        for c in self.conns.values():
            if c.fh:
                c.fh.close()
        if self.own_workdir and self._workdir:
            import shutil
            shutil.rmtree(self._workdir, ignore_errors=True)
        return self.exit_code


def main():
    ap = argparse.ArgumentParser(description='Run an Intel UDI .86 program')
    ap.add_argument('program')
    ap.add_argument('cmdline', help='command line given to the program on :CI:')
    ap.add_argument('-D', '--drive', action='append', default=[],
                    help='map drive, e.g. F1=/some/dir (may repeat; several dirs are searched in order)')
    ap.add_argument('-I', '--include', action='append', default=[],
                    help='search dir for names without a :Fn: prefix (after cwd)')
    ap.add_argument('--write-root', action='append', default=[])
    ap.add_argument('--date', help='fixed DQ$GET$TIME value "MM/DD/YY HH:MM:SS"')
    ap.add_argument('--heap-kb', type=int, default=None)
    ap.add_argument('--trace', action='store_true')
    ap.add_argument('--ff-to-lf', action='store_true',
                    help='map form feeds to LF when reading files (udi.run does this for PL/M)')
    a = ap.parse_args()
    drives = {}
    for d in a.drive:
        k, v = d.split('=', 1)
        drives.setdefault(k.upper(), []).append(v)
    search = [os.getcwd()] + a.include
    udi = UDI(a.program, a.cmdline.encode('latin1') + b'\r\n', drives, search,
              a.write_root or [os.getcwd()], date=a.date, heap_kb=a.heap_kb, trace=a.trace,
              ff_char=0x0A if a.ff_to_lf else None)
    rc = udi.run()
    sys.exit(rc or 0)


if __name__ == '__main__':
    main()
