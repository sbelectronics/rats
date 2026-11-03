import struct,sys,collections
d=open(sys.argv[1],'rb').read()
h=struct.unpack_from('<16H',d,0)
names='sig ver cpnRes irleMax cparDir cparMin cparMax saStack raStack chk raStart saStart rbrgrle iovMax allFs verAlt'.split()
for n,v in zip(names,h): print(f'{n:8} {v:#06x}')
print('w32..', [hex(x) for x in struct.unpack_from('<2H',d,32)])
print(repr(d[0x24:0x50]))
rb=h[12]; n=h[3]
rel=[struct.unpack_from('<2H',d,rb+4*i) for i in range(n)]
c=collections.Counter(sa for ra,sa in rel)
img=d[h[4]*16:]
print('img size',hex(len(img)))
for sa in sorted(c): 
    # value stored at fixup location
    vals=collections.Counter(struct.unpack_from('<H',img,sa*16+ra)[0] for ra,s in rel if s==sa)
    print(f'sa {sa:#06x} count {c[sa]} targets {sorted(vals.items())}')
