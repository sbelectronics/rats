package main

// main.go - program entry: Ebiten window "Rats 9.2" showing the 80 x 34 cell
// screen (10 x 15 px cells = 800 x 510) at an integer scale, keyboard feed, and the
// game (GameMainFn, set in game_*.go) in its own goroutine.  Without GameMainFn a test
// pattern is shown.  -snapshot out.png renders the screen to a PNG without a window.

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	pScrW = 80 * pCellW
	pScrH = pMaxLines * pCellH
)

// pKeyMap: PC key -> CTOS key (see PORT_GUIDE.md and Kbd_All.asm rgInfoIkey).
var pKeyMap = map[ebiten.Key]pKeyDef{
	ebiten.KeyArrowUp:        {raw: 0x01},
	ebiten.KeyArrowDown:      {raw: 0x0B},
	ebiten.KeyArrowLeft:      {raw: 0x0E},
	ebiten.KeyArrowRight:     {raw: 0x0F},
	ebiten.KeyNumpad8:        {raw: 0x7C},
	ebiten.KeyNumpad5:        {raw: 0x7D},
	ebiten.KeyNumpad4:        {raw: 0x7B},
	ebiten.KeyNumpad6:        {raw: 0x41},
	ebiten.KeyNumpad7:        {raw: 0x5C},
	ebiten.KeyNumpad9:        {raw: 0x21},
	ebiten.KeyNumpad1:        {raw: 0x60},
	ebiten.KeyNumpad2:        {raw: 0x46},
	ebiten.KeyNumpad3:        {raw: 0x47},
	ebiten.KeyNumpad0:        {raw: 0x4A},
	ebiten.KeyNumpadDecimal:  {raw: 0x4B},
	ebiten.KeyNumpadSubtract: {raw: 0x42},
	ebiten.KeyNumpadEnter:    {raw: 0x1B},
	ebiten.KeyShiftLeft:      {raw: 0x48},
	ebiten.KeyShiftRight:     {raw: 0x49},
	ebiten.KeyCapsLock:       {raw: 0x45},
	ebiten.KeySpace:          {raw: 0x12, chUn: ' ', chSh: ' ', explicitChar: true}, // MOVE raw, space char
	ebiten.KeyControlLeft:    {raw: 0x07},
	ebiten.KeyEscape:         {raw: 0x07},
	ebiten.KeyEnter:          {raw: 0x1B},
	ebiten.KeyF10:            {raw: 0x1B},
	ebiten.KeyF1:             {raw: 0x00},
	ebiten.KeyF2:             {raw: 0x16},
	ebiten.KeyF3:             {raw: 0x17},
	ebiten.KeyF4:             {raw: 0x18},
	ebiten.KeyF5:             {raw: 0x19},
	ebiten.KeyF6:             {raw: 0x1A},
	ebiten.KeyF7:             {raw: 0x1C},
	ebiten.KeyF8:             {raw: 0x1D},
	ebiten.KeyF9:             {raw: 0x1E},
	ebiten.KeyF12:            {raw: 0x04},
	ebiten.KeyPause:          {raw: 0x04},
	ebiten.KeyTab:            {raw: 0x0A},
	ebiten.KeyBackspace:      {raw: 0x08},
	ebiten.KeyDelete:         {raw: 0x7F},
	ebiten.KeyInsert:         {raw: 0x44},
	ebiten.KeyPageUp:         {raw: 0x05},
	ebiten.KeyPageDown:       {raw: 0x0C},
	ebiten.KeyHome:           {raw: 0x02},
	ebiten.KeyMinus:          {raw: 0x2D, chUn: '-', chSh: '_', explicitChar: true},
	ebiten.KeyEqual:          {raw: 0x2B, chUn: '=', chSh: '+', explicitChar: true},
	ebiten.KeyComma:          {raw: 0x2C, chUn: ',', chSh: '<', explicitChar: true},
	ebiten.KeyPeriod:         {raw: 0x2E, chUn: '.', chSh: '>', explicitChar: true},
	ebiten.KeySlash:          {raw: 0x2F, chUn: '/', chSh: '?', explicitChar: true},
	ebiten.KeySemicolon:      {raw: 0x3B, chUn: ';', chSh: ':', explicitChar: true},
	ebiten.KeyQuote:          {raw: 0x27, chUn: '\'', chSh: '"', explicitChar: true},
	ebiten.KeyBracketLeft:    {raw: 0x5B, chUn: '[', chSh: '{', explicitChar: true},
	ebiten.KeyBracketRight:   {raw: 0x5D, chUn: ']', chSh: '}', explicitChar: true},
	ebiten.KeyBackslash:      {raw: pRawNone, chUn: '\\', chSh: '|', explicitChar: true},
	ebiten.KeyBackquote:      {raw: pRawNone, chUn: '`', chSh: '~', explicitChar: true},
}

func init() {
	for k := ebiten.KeyA; k <= ebiten.KeyZ; k++ {
		pKeyMap[k] = pKeyDef{raw: byte(0x61 + int(k-ebiten.KeyA))}
	}
	shifted := ")!@#$%^&*("
	for k := ebiten.KeyDigit0; k <= ebiten.KeyDigit9; k++ {
		d := int(k - ebiten.KeyDigit0)
		pKeyMap[k] = pKeyDef{raw: byte(0x30 + d), chUn: byte('0' + d), chSh: shifted[d], explicitChar: true}
	}
}

// pPlatformInit sets the DGROUP and system-segment state the original finds at start.
func pPlatformInit() {
	pPlatformVideoBoot()
	pAscbInit()
}

type pGame struct {
	scale int
	img   *ebiten.Image
	pix   []byte
	st    pScreenState
	last  uint64
	blink bool
	drawn bool
	keys  []ebiten.Key
}

func (g *pGame) Update() error {
	now := time.Now()
	g.keys = inpututil.AppendJustPressedKeys(g.keys[:0])
	for _, k := range g.keys {
		if d, ok := pKeyMap[k]; ok {
			pKbdPress(int(k), d, now)
		}
	}
	g.keys = inpututil.AppendJustReleasedKeys(g.keys[:0])
	for _, k := range g.keys {
		if d, ok := pKeyMap[k]; ok {
			pKbdRelease(int(k), d)
		}
	}
	pKbdTypematic(now, func(key int) (pKeyDef, bool) {
		k := ebiten.Key(key)
		d, ok := pKeyMap[k]
		return d, ok && ebiten.IsKeyPressed(k)
	})
	return nil
}

func pBlinkPhase() bool { return time.Now().UnixMilli()/500%2 == 0 }

func (g *pGame) Draw(screen *ebiten.Image) {
	pSnapScreen(&g.st)
	blink := pBlinkPhase()
	if !g.drawn || g.st.serial != g.last || blink != g.blink {
		pRenderScreen(&g.st, g.pix, pScrW, blink)
		g.img.WritePixels(g.pix)
		g.last, g.blink, g.drawn = g.st.serial, blink, true
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(g.scale), float64(g.scale))
	screen.DrawImage(g.img, op)
}

func (g *pGame) Layout(w, h int) (int, int) { return pScrW * g.scale, pScrH * g.scale }

// pTestPattern fills the screen with all glyphs and every attribute.
func pTestPattern() {
	pVidInit(0x102, 1, 0, TRUE) // the game's InitVideo call
	put := func(col, row uint16, s string, attr byte) {
		for i := 0; i < len(s); i++ {
			pVidA4(col+uint16(i), row, uint16(attr)<<8|uint16(s[i]))
		}
	}
	put(0, 0, "Rats 9.2 platform test pattern: 256 glyphs, attributes 0-F (8+ blink)", 0)
	for i := 0; i < 256; i++ {
		pVidA4(uint16(i%32)*2+8, uint16(2+i/32), uint16(i))
	}
	for r := 0; r < 8; r++ {
		put(0, uint16(2+r), fmt.Sprintf("%02X:", r*32), 0)
	}
	for a := 0; a < 16; a++ {
		row := uint16(11 + a)
		put(0, row, fmt.Sprintf("attr %X", a), 0)
		put(8, row, "ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrstuvwxyz 0123456789", byte(a))
	}
	put(0, 28, "odd-start fill:", 0)
	pVidA5(17, 28, 9, 0x0400|'#') // reverse, starting at an odd column
	put(0, 30, "keys: ", 0)
	pPosFrameCursor(0, 6, 30)
}

func pTestLoop() {
	pTestPattern()
	col := uint16(6)
	for {
		b := pKbdWait()
		s := fmt.Sprintf("%02X ", b)
		for i := 0; i < len(s); i++ {
			pVidA4(col, 30, uint16(s[i]))
			col++
		}
		if col > 76 {
			col = 6
			pClearRect(6, 30, 74, 1)
		}
		pPosFrameCursor(0, col, 30)
	}
}

// pSnapKeys parses "-snapkeys" items "<duration>:<hexbyte>[:<hexbyte>...]" (comma separated)
// and pushes the bytes into the keyboard queue at those times (for scripted snapshots).
func pSnapKeys(spec string) {
	start := time.Now()
	for _, item := range strings.Split(spec, ",") {
		parts := strings.Split(strings.TrimSpace(item), ":")
		if len(parts) < 2 {
			continue
		}
		at, err := time.ParseDuration(parts[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "snapkeys:", err)
			continue
		}
		time.Sleep(time.Until(start.Add(at)))
		for _, h := range parts[1:] {
			v, err := strconv.ParseUint(h, 16, 8)
			if err == nil {
				pKbdPush(byte(v))
			}
		}
	}
}

func pSnapshot(path string, delay time.Duration, keys string) error {
	if GameMainFn != nil {
		go GameMainFn()
		if keys != "" {
			go pSnapKeys(keys)
		}
		time.Sleep(delay)
	} else {
		pTestPattern()
	}
	var st pScreenState
	pSnapScreen(&st)
	img := image.NewRGBA(image.Rect(0, 0, pScrW, pScrH))
	pRenderScreen(&st, img.Pix, pScrW, true)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func main() {
	scale := flag.Int("scale", 2, "integer window scale factor")
	snapshot := flag.String("snapshot", "", "render the screen to this PNG file and exit (no window)")
	snapDelay := flag.Duration("snapdelay", 2*time.Second, "with -snapshot: how long to run the game first")
	snapKeys := flag.String("snapkeys", "", "with -snapshot: scripted keys, e.g. \"1s:1b,2s:12:0f\" (time:hexbytes,...)")
	phosphor := flag.String("phosphor", "white", "screen colour: white, green or amber")
	dataDir := flag.String("datadir", "", "directory for score files (default %APPDATA%\\Rats)")
	flag.Parse()
	if *scale < 1 {
		*scale = 1
	}
	pDataDir = *dataDir
	pSetPhosphor(*phosphor)
	pPlatformInit()

	if *snapshot != "" {
		if err := pSnapshot(*snapshot, *snapDelay, *snapKeys); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	pAudioInit()
	g := &pGame{scale: *scale, img: ebiten.NewImage(pScrW, pScrH), pix: make([]byte, pScrW*pScrH*4)}
	ebiten.SetWindowTitle("Rats 9.2")
	ebiten.SetWindowSize(pScrW**scale, pScrH**scale)
	pWindowUp = true
	if GameMainFn != nil {
		go GameMainFn()
	} else {
		go pTestLoop()
	}
	if err := ebiten.RunGame(g); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0) // window closed
}
