package main

// platform_font.go - the character fonts, built into the program.
// Format: 256 glyphs x 32 bytes; each glyph is 16 rows of little-endian 16-bit
// words, 10 pixels wide: bit 9 = leftmost pixel, bit 0 = rightmost.  The displayed
// cell is 10 x 15 pixels (font rows 1..15; see the constants below).
//
// Like the real workstation, the screen starts with the standard system font
// ([sys]<sys>Sys.font, "Sys.Font VERSION 7.0", glyph part only).  The game shows its options form in that font and only then
// calls LoadFont("[sys]<games>Rats.font") (RatsC: InitVideo, GetOptions, SelectFont),
// which replaces the font RAM with the Rats glyphs (rats, walls, people ...).

import (
	_ "embed"
	"strings"
)

//go:embed Rats.font
var pRatsFont []byte

//go:embed Sys.font
var pSysFont []byte

// pFont is the font currently in the (emulated) font RAM.  Guarded by pVidMu.
var pFont = pSysFont

// The character cell is 15 scan lines: font rows 1..15 are displayed.  Row 0 is
// blank in every glyph of both Rats.font and Sys.font; drawing it would put a dark
// gap between the vertical wall pieces, which on the real screen join up.
// So the screen is 80 x 34 cells = 800 x 510 pixels.
const (
	pCellW    = 10
	pCellH    = 15
	pFirstRow = 1 // first font row shown
)

// pFontByName maps a CTOS font file spec to a built-in font (nil if unknown).
func pFontByName(spec string) []byte {
	s := strings.ToLower(spec)
	switch {
	case strings.HasSuffix(s, "rats.font"): // Rats.font (T1Rats.font is the NGEN variant; not emulated)
		return pRatsFont
	case strings.HasSuffix(s, "sys.font"):
		return pSysFont
	}
	return nil
}

// pGlyphRow returns the pixel bits of displayed row r (0..14, = font row r+1) of glyph ch
// (bit 9 = leftmost, bit 0 = rightmost).
func pGlyphRow(ch byte, r int) uint16 {
	o := int(ch)*32 + (r+pFirstRow)*2
	return uint16(pFont[o]) | uint16(pFont[o+1])<<8
}

// pGlyphPixel reports whether pixel (x, y) of glyph ch is set.
func pGlyphPixel(ch byte, x, y int) bool {
	return pGlyphRow(ch, y)>>uint(pCellW-1-x)&1 != 0
}
