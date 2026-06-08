package vm

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"time"
	"unsafe"
)

//go:generate stringer -type Color,Waveform

// Peripheral interface definitions
type Machine interface {
	Graphics
	Sound
	Input
	Timer
}

type Timer interface {
	VBlank() bool
}

type Graphics interface {
	ClearScreen()
	SetBackground(ix uint8)
	Draw(x, y int16, data *byte) bool
	SetSize(width, height uint8)
	SetFlip(horiz, vert bool)
	LoadPalette(data *byte)
	ScreenImage() image.Image
}

type Sound interface {
	StartSound(freq, msec int16, useEnvelope bool)
	StopSound()
	SetSoundParams(typ, vol uint8, env Envelope)
}

type Input interface {
	Events() chan Event
}

const (
	ScreenWidth  = 320
	ScreenHeight = 240
	PaletteSize  = 16
	TickRate     = time.Second / 60
	BusyWait     = 2 * time.Millisecond
	InputQueue   = 256
)

const (
	ButtonUp = 1 << iota
	ButtonDown
	ButtonLeft
	ButtonRight
	ButtonSelect
	ButtonStart
	ButtonA
	ButtonB
)

const (
	Controller1 Device = iota
	Controller2
)

const (
	Transparent Color = iota
	Black
	Gray
	Red
	Pink
	DarkBrown
	Brown
	Orange
	Yellow
	Green
	LightGreen
	DarkBlue
	Blue
	LightBlue
	SkyBlue
	White
)

type Color uint8

// As defined in chip16 spec
var DefaultPalette = color.Palette{
	color.RGBA{},
	rgb(0x000000),
	rgb(0x888888),
	rgb(0xBF3932),
	rgb(0xDE7AAE),
	rgb(0x4C3D41),
	rgb(0x905F25),
	rgb(0xE49452),
	rgb(0xEAD979),
	rgb(0x537A3B),
	rgb(0xABD54A),
	rgb(0x252E38),
	rgb(0x00467F),
	rgb(0x68ABCC),
	rgb(0xBCDEE4),
	rgb(0xFFFFFF),
}

// With more vivid colours
var AltPalette = color.Palette{
	color.RGBA{},
	rgb(0x000000),
	rgb(0x808080),
	rgb(0xFF0000),
	rgb(0xFF69B4),
	rgb(0x8B4513),
	rgb(0xD2691E),
	rgb(0xFFA500),
	rgb(0xFFFF00),
	rgb(0x008000),
	rgb(0x00FF00),
	rgb(0x000080),
	rgb(0x0000FF),
	rgb(0x00FFFF),
	rgb(0x87CEEB),
	rgb(0xFFFFFF),
}

const (
	Triangle Waveform = iota
	Sawtooth
	Pulse
	Noise
)

type Waveform uint8

type Envelope struct {
	Attack  uint8
	Decay   uint8
	Sustain uint8
	Release uint8
}

type Event struct {
	Device Device
	State  uint16
}

type Device uint8

func NewMachine(vsync, idleWait bool) Machine {
	m := new(machine)
	m.InputBase.Init()
	m.GraphicsBase.Init()
	if vsync {
		m.timer.setVsync(idleWait)
	}
	return m
}

// Base machine implementation
type machine struct {
	GraphicsBase
	InputBase
	timer
}

func (machine) StopSound() {}

func (machine) StartSound(freq, msec int16, useEnvelope bool) {}

func (machine) SetSoundParams(typ, vol uint8, env Envelope) {}

// Default standalone timer
type timer struct {
	vsync    bool
	idleWait bool
	ticker   *time.Ticker
	next     time.Time
}

var _ Timer = &timer{}

func (t *timer) setVsync(idleWait bool) {
	t.vsync = true
	t.idleWait = idleWait
	t.ticker = time.NewTicker(TickRate)
	t.next = time.Now().Add(TickRate)

}

// Returns true once every TickRate if vsync is set. Will sleep for up to TickRate-BusyWait if idleWait is set.
func (t *timer) VBlank() bool {
	if !t.vsync {
		return true
	}
	if t.idleWait {
		if d := time.Until(t.next); d > BusyWait {
			time.Sleep(d - BusyWait)
		}
	}
	select {
	case tick := <-t.ticker.C:
		t.next = tick.Add(TickRate)
		return true
	default:
		return false
	}
}

// Default input implementation
type InputBase struct {
	queue chan Event
}

func (i *InputBase) Init() {
	i.queue = make(chan Event, InputQueue)
}

func (i *InputBase) Events() chan Event {
	return i.queue
}

// Default standalone graphics implementation
type GraphicsBase struct {
	FG      image.Paletted
	BG      Color
	SpriteW int
	SpriteH int
	HFlip   bool
	VFlip   bool
}

var _ Graphics = &GraphicsBase{}

func (g *GraphicsBase) Init() {
	g.BG = Transparent
	g.FG = *image.NewPaletted(image.Rect(0, 0, ScreenWidth, ScreenHeight), DefaultPalette)
}

func (g *GraphicsBase) RGBA(ix Color) color.RGBA {
	return g.FG.Palette[ix].(color.RGBA)
}

func (g *GraphicsBase) String() string {
	return fmt.Sprintf("BG: %s  Sprite_W: %02X  Sprite_H: %02X  HFlip: %v  VFlip: %v",
		g.BG, g.SpriteW, g.SpriteH, g.HFlip, g.VFlip)
}

func (g *GraphicsBase) SetBackground(ix uint8) {
	g.BG = Color(ix & (PaletteSize - 1))
}

func (g *GraphicsBase) SetSize(width, height uint8) {
	g.SpriteW, g.SpriteH = int(width), int(height)
}

func (g *GraphicsBase) SetFlip(horiz, vert bool) {
	g.HFlip, g.VFlip = horiz, vert
}

func (g *GraphicsBase) ClearScreen() {
	g.BG = 0
	clear(g.FG.Pix)
}

func (g *GraphicsBase) Draw(x, y int16, dptr *byte) bool {
	collision := false
	x0, y0 := int(x), int(y)
	w, h := g.SpriteW, g.SpriteH
	data := unsafe.Slice(dptr, w*h)
	for iy := 0; iy < h; iy++ {
		if !g.VFlip {
			drawLine(&g.FG, data[iy*w:(iy+1)*w], x0, y0+iy, g.HFlip, &collision)
		} else {
			drawLine(&g.FG, data[(h-iy-1)*w:(h-iy)*w], x0, y0+iy, g.HFlip, &collision)
		}
	}
	return collision
}

func (g *GraphicsBase) LoadPalette(dptr *byte) {
	data := unsafe.Slice(dptr, PaletteSize*3)
	// color 0 is always transparent
	for i := 1; i < PaletteSize; i++ {
		g.FG.Palette[i] = color.RGBA{R: data[i*3], G: data[i*3+1], B: data[i*3+2], A: 255}
	}
}

func (g *GraphicsBase) ScreenImage() image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, ScreenWidth, ScreenHeight))
	draw.Draw(dst, dst.Rect, image.NewUniform(g.FG.Palette[g.BG]), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Rect, &g.FG, image.Point{}, draw.Over)
	return dst
}

// Utils
func drawLine(img *image.Paletted, data []byte, x, y int, flip bool, collision *bool) {
	if y < 0 || y >= ScreenHeight {
		return
	}
	if !flip {
		for ix, pix := range data {
			drawPixels(img, x+2*ix, y, (pix&0xF0)>>4, pix&0xF, collision)
		}
	} else {
		for ix := range data {
			pix := data[len(data)-ix-1]
			drawPixels(img, x+2*ix, y, pix&0xF, (pix&0xF0)>>4, collision)
		}
	}
}

func drawPixels(img *image.Paletted, x, y int, i1, i2 uint8, collision *bool) {
	off := y*img.Stride + x
	if i1 != 0 && x >= 0 && x < ScreenWidth {
		*collision = *collision || img.Pix[off] != 0
		img.Pix[off] = i1
	}
	if i2 != 0 && x+1 >= 0 && x+1 < ScreenWidth {
		*collision = *collision || img.Pix[off+1] != 0
		img.Pix[off+1] = i2
	}
}

func rgb(c uint32) color.RGBA {
	return color.RGBA{R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c), A: 255}
}
