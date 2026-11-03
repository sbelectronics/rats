#!/usr/bin/env python3
"""plm86 - run Intel PL/M-86 V2.3 (V2.3PLM86.86) on Linux via the udi86 emulator.

usage: plm86 [options] source.plm [PL/M controls ...]

The .obj and .lst are written to the current directory (named after the source
as given, e.g. bndutl.plm -> bndutl.obj / bndutl.lst), exactly as the compiler
names them.  Nothing is written outside the current directory.
"""
import os, sys, argparse

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import udi86

# The compiler binary is not in the repo; tools/getorig.sh copies it into orig/.
DEFAULT_COMPILER = os.path.join(HERE, '..', 'orig', 'V2.3PLM86.86')


def main(argv=None):
    ap = argparse.ArgumentParser(
        prog='plm86', usage='%(prog)s [options] source.plm [controls ...]',
        description='Compile PL/M-86 with the original Intel PL/M-86 V2.3 compiler (emulated).')
    ap.add_argument('source')
    ap.add_argument('controls', nargs='*',
                    help='compiler controls, e.g. CODE XREF NOLIST NOTYPE OPTIMIZE(3) LARGE')
    ap.add_argument('-I', '--include', action='append', default=[], metavar='DIR',
                    help='directory searched for file names without a :Fn: prefix '
                         '(after the current dir and the source dir); repeatable')
    ap.add_argument('-F', '--drive', action='append', default=[], metavar='Fn=DIR',
                    help='map an ISIS drive, e.g. -F f0=/x/edf -F f1=/x/linker; repeatable '
                         '(several dirs for one drive are searched in order). Unmapped '
                         'drives fall back to the -I search path')
    ap.add_argument('--date', metavar='"MM/DD/YY HH:MM:SS"',
                    help='fixed date/time the compiler sees (goes into the .obj COMENT and .lst header)')
    ap.add_argument('--heap-kb', type=int, default=157,
                    help='memory given to the compiler (default 157, as on the original system)')
    ap.add_argument('--name', help='file name to put on the compiler command line '
                                   '(default: basename of SOURCE as given)')
    ap.add_argument('--compiler', default=os.environ.get('PLM86_COMPILER', DEFAULT_COMPILER))
    ap.add_argument('--trace', action='store_true', help='trace UDI calls to stderr')
    a = ap.parse_intermixed_args(argv)

    src = a.source
    srcdir = os.path.dirname(os.path.abspath(src))
    name = a.name or os.path.basename(src)
    search = [os.getcwd()]
    if srcdir not in search:
        search.append(srcdir)
    search += [os.path.abspath(d) for d in a.include]
    drives = {}
    for d in a.drive:
        k, v = d.split('=', 1)
        k = k.strip().strip(':').upper()
        drives.setdefault(k, []).append(os.path.abspath(v))
    for n in range(10):
        drives.setdefault('F%d' % n, search)
    # The CTOS front end passes the file name quoted, so its case is preserved
    # (DQ$GET$ARGUMENT upper-cases unquoted text).
    cmd = "V2.3PLM86.86 '%s'" % name
    if a.controls:
        cmd += ' ' + ' '.join(a.controls)
    u = udi86.UDI(a.compiler, cmd.encode('latin1') + b'\r\n', drives, search,
                  [os.getcwd()], date=a.date, heap_kb=a.heap_kb, trace=a.trace,
                  ff_char=0x0A)
    return u.run() or 0


if __name__ == '__main__':
    sys.exit(main())
