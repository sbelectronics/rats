package main

// platform_kbdtab.go - CTOS keyboard translation table rgInfoIkey, transcribed
// from CTOS Kbd_All.asm: per raw key code 00h..7Fh the
// (unshifted value, shifted value, attributes).  Attributes: 80h = responds to LOCK;
// bits 5-6 = typematic delay (00 none, 20h 0.2 s, 40h 0.4 s, 60h 0.7 s); bits 3-4 =
// repeat rate (00 1/s, 08h 10/s, 10h 20/s, 18h 30/s).

const (
	pKAttrControl    = 0x00
	pKAttrAlpha      = 0x80 + 0x60 + 0x10
	pKAttrRepeat     = 0x00 + 0x60 + 0x10
	pKAttrRepeatFast = 0x00 + 0x40 + 0x18
)

var pRgInfoIkey = [128 * 3]byte{
	0, 0xC0, pKAttrControl, // 000  help
	0x01, 0xC1, pKAttrRepeatFast, // 001  up-arrow
	0x02, 0xC2, pKAttrControl, // 002  mark
	0x0D, 0xC3, pKAttrControl, // 003  bound
	0x04, 0xC4, pKAttrControl, // 004  finish
	0x05, 0xC5, pKAttrControl, // 005  prev page
	0x06, 0x10, pKAttrRepeat, // 006  1/2-1/4
	0x07, 0xC7, pKAttrControl, // 007  cancel
	0x08, 0x08, pKAttrRepeat, // 008  backspace
	0x09, 0x09, pKAttrControl, // 009  tab
	0x0A, 0x0A, pKAttrRepeat, // 00A  return
	0x0B, 0xCB, pKAttrRepeatFast, // 00B  down-arrow
	0x0C, 0xCC, pKAttrControl, // 00C  next page
	0x0A, 0x0A, pKAttrRepeat, // 00D  next
	0x0E, 0xCE, pKAttrRepeatFast, // 00E  left-arrow
	0x12, 0xD2, pKAttrRepeatFast, // 00F  right-arrow
	0x80, 0x81, pKAttrControl, // 010  (SH-L)
	0x11, 0xD1, pKAttrRepeat, // 011  scroll-up
	0x0F, 0xCF, pKAttrControl, // 012  move
	0x13, 0xD3, pKAttrRepeat, // 013  scroll-down
	0x14, 0xD4, pKAttrControl, // 014  copy
	0x15, 0xD5, pKAttrControl, // 015  f1
	0x16, 0xD6, pKAttrControl, // 016  f2
	0x17, 0xD7, pKAttrControl, // 017  f3
	0x18, 0xD8, pKAttrControl, // 018  f4
	0x19, 0xD9, pKAttrControl, // 019  f5
	0x1A, 0xDA, pKAttrControl, // 01A  f6
	0x1B, 0xC9, pKAttrControl, // 01B  GO
	0x1C, 0xDC, pKAttrControl, // 01C  f7
	0x1D, 0xCD, pKAttrControl, // 01D  f8
	0x1E, 0xCA, pKAttrControl, // 01E  f9
	0x1F, 0xDF, pKAttrControl, // 01F  f10
	0x20, 0x20, pKAttrRepeat, // 020  space
	0x39, 0x39, pKAttrRepeat, // 021  9 9
	0x82, 0x83, pKAttrControl, // 022  (SH-L)
	0x84, 0x85, pKAttrControl, // 023  (0)
	0x86, 0x87, pKAttrControl, // 024  (NEXT)
	0, 0, 0, // 025
	0, 0, 0, // 026
	0x27, 0x22, pKAttrRepeat, // 027   "
	0, 0, 0, // 028
	0, 0, 0, // 029
	0, 0, 0, // 02A
	0x3D, 0x2B, pKAttrRepeat, // 02B  + =
	0x2C, 0x2C, pKAttrRepeat, // 02C  , ,
	0x2D, 0x5F, pKAttrRepeat, // 02D  - _
	0x2E, 0x2E, pKAttrRepeat, // 02E  . .
	0x2F, 0x3F, pKAttrRepeat, // 02F  / ?
	0x30, 0x29, pKAttrRepeat, // 030  0 )
	0x31, 0x21, pKAttrRepeat, // 031  1 !
	0x32, 0x40, pKAttrRepeat, // 032  2 @
	0x33, 0x23, pKAttrRepeat, // 033  3 #
	0x34, 0x24, pKAttrRepeat, // 034  4 $
	0x35, 0x25, pKAttrRepeat, // 035  5 percent
	0x36, 0x03, pKAttrRepeat, // 036  6 cent
	0x37, 0x26, pKAttrRepeat, // 037  7 &
	0x38, 0x2A, pKAttrRepeat, // 038  8 *
	0x39, 0x28, pKAttrRepeat, // 039  9 (
	0, 0, 0, // 03A
	0x3B, 0x3A, pKAttrRepeat, // :
	0, 0, 0, // 03C
	0, 0, 0, // 03D
	0, 0, 0, // 03E
	0, 0, 0, // 03F  3F = iKeyInvalid
	0, 0, 0, // 040  40 = iKeyNoKeys
	0x36, 0x36, pKAttrRepeat, // 041  6 6
	0x2D, 0x2D, pKAttrRepeat, // 042  - -
	0, 0, 0, // 043  reset
	0xD0, 0xD0, pKAttrControl, // 044  overtype
	0, 0, pKAttrControl, // 045  lock
	0x32, 0x32, pKAttrRepeat, // 046  2 2
	0x33, 0x33, pKAttrRepeat, // 047  3 3
	0, 0, 0, // 048  shift
	0, 0, 0, // 049  shift
	0x30, 0x30, pKAttrRepeat, // 04A  0 0
	0x2E, 0x2E, pKAttrRepeat, // 04B  . .
	0, 0, 0, // 04C  code
	0, 0, 0, // 04D  code
	0, 0, 0, // 04E
	0, 0, 0, // 04F
	0, 0, 0, // 050
	0, 0, 0, // 051
	0, 0, 0, // 052
	0, 0, 0, // 053
	0, 0, 0, // 054
	0, 0, 0, // 055
	0, 0, 0, // 056
	0, 0, 0, // 057
	0, 0, 0, // 058
	0, 0, 0, // 059
	0, 0, 0, // 05A
	0x5B, 0x3C, pKAttrRepeat, // 05B  [ <
	0x37, 0x7C, pKAttrRepeat, // 05C  7 |
	0x5D, 0x3E, pKAttrRepeat, // 05D  ] >
	0x5E, 0x7E, pKAttrRepeat, // 05E  ^ ~
	0, 0, 0, // 05F
	0x31, 0x60, pKAttrRepeat, // 060  1 `
	0x61, 0x41, pKAttrAlpha, // 061  a A
	0x62, 0x42, pKAttrAlpha, // 062  b B
	0x63, 0x43, pKAttrAlpha, // 063  c C
	0x64, 0x44, pKAttrAlpha, // 064  d D
	0x65, 0x45, pKAttrAlpha, // 065  e E
	0x66, 0x46, pKAttrAlpha, // 066  f F
	0x67, 0x47, pKAttrAlpha, // 067  g G
	0x68, 0x48, pKAttrAlpha, // 068  h H
	0x69, 0x49, pKAttrAlpha, // 069  i I
	0x6A, 0x4A, pKAttrAlpha, // 06A  j J
	0x6B, 0x4B, pKAttrAlpha, // 06B  k K
	0x6C, 0x4C, pKAttrAlpha, // 06C  l L
	0x6D, 0x4D, pKAttrAlpha, // 06D  m M
	0x6E, 0x4E, pKAttrAlpha, // 06E  n N
	0x6F, 0x4F, pKAttrAlpha, // 06F  o O
	0x70, 0x50, pKAttrAlpha, // 070  p P
	0x71, 0x51, pKAttrAlpha, // 071  q Q
	0x72, 0x52, pKAttrAlpha, // 072  r R
	0x73, 0x53, pKAttrAlpha, // 073  s S
	0x74, 0x54, pKAttrAlpha, // 074  t T
	0x75, 0x55, pKAttrAlpha, // 075  u U
	0x76, 0x56, pKAttrAlpha, // 076  v V
	0x77, 0x57, pKAttrAlpha, // 077  w W
	0x78, 0x58, pKAttrAlpha, // 078  x X
	0x79, 0x59, pKAttrAlpha, // 079  y Y
	0x7A, 0x5A, pKAttrAlpha, // 07A  z Z
	0x34, 0x7B, pKAttrRepeat, // 07B  4 {
	0x38, 0x5C, pKAttrRepeat, // 07C  8 backslash
	0x35, 0x7D, pKAttrRepeat, // 07D  5 }
	0, 0, 0, // 07E
	0x7F, 0xC8, pKAttrRepeat, // 07F  del
}
