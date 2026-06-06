// Chip16 graphics and sound implentation using SDL3.
package sdl

import (
	"fmt"
	"image/color"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/jnb666/chip16/vm"
	log "github.com/sirupsen/logrus"
)

const (
	SleepWait  = 100 * time.Millisecond
	SleepAfter = 15 * time.Second
)

var DefaultOptions = Options{Scale: defaultScale(), Volume: 128}

// SDL options for constructing a new app.
type Options struct {
	Scale      float64
	Volume     int
	NoVSync    bool
	Fullscreen bool
	UseTouch   bool
}

func (o Options) String() string {
	var s []string
	v := reflect.ValueOf(o)
	for fld, val := range v.Fields() {
		if !val.IsZero() {
			s = append(s, fmt.Sprintf("%s: %v", fld.Name, val))
		}
	}
	return strings.Join(s, " ")
}

func defaultScale() float64 {
	if runtime.GOOS == "darwin" {
		return 2
	} else {
		return 4
	}
}

// App implements the vm.Machine interface.
type App struct {
	Graphics
	Sound
	Timer
	win        *sdl.Window
	renderer   *sdl.Renderer
	tex        *sdl.Texture
	buttons    []*sdl.Texture
	joystick   *sdl.Joystick
	bgcol      color.RGBA
	viewport   *sdl.FRect
	scale      float32
	controller [2]int16
	changed    time.Time
	minimized  bool
	lastFrame  time.Time
	fpsTicker  *time.Ticker
	frames     int
	secs       int
}

// Initialise SDL application.
func New(opts Options) (*App, error) {
	err := sdl.LoadLibrary(libraryPath())
	if err != nil {
		return nil, err
	}
	err = sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO | sdl.INIT_JOYSTICK)
	if err != nil {
		return nil, err
	}
	a := &App{changed: time.Now()}
	a.Graphics.Init()
	a.Sound.Init(opts.Volume)
	if err = a.initWindow(opts); err != nil {
		return nil, err
	}
	opts.Scale = float64(a.scale)
	log.Infof("SDL version %s - %s", sdl.GetVersion(), opts)
	a.renderer.SetDrawColor(0, 0, 0, 0xFF)
	a.tex, err = a.renderer.CreateTexture(sdl.PIXELFORMAT_INDEX8, sdl.TEXTUREACCESS_STREAMING, vm.ScreenWidth, vm.ScreenHeight)
	if err != nil {
		return nil, err
	}
	if err = a.SetPalette(vm.DefaultPalette); err != nil {
		return nil, err
	}
	a.tex.SetBlendMode(sdl.BLENDMODE_BLEND)
	a.tex.SetScaleMode(sdl.SCALEMODE_NEAREST)
	if err = updateTexture(a.tex, a.FG.Pix); err != nil {
		return nil, err
	}
	if opts.UseTouch {
		err = a.initButtons()
	}
	a.fpsTicker = time.NewTicker(time.Second)
	a.lastFrame = time.Now()
	return a, err
}

func (a *App) initWindow(opts Options) error {
	var err error
	var flags sdl.WindowFlags
	var width, height int
	screenWidth := vm.ScreenWidth
	if opts.UseTouch {
		screenWidth += TouchPanelWidth
	}
	if !opts.Fullscreen {
		a.scale = float32(opts.Scale)
		width, height = int(float32(screenWidth)*a.scale+0.5), int(vm.ScreenHeight*a.scale+0.5)
	} else {
		flags |= sdl.WINDOW_FULLSCREEN
		bounds, err := sdl.GetPrimaryDisplay().Bounds()
		if err != nil {
			return err
		}
		log.Debugf("display bounds: %+v", bounds)
		width, height = int(bounds.W), int(bounds.H)
		a.scale = min(roundScale(width, screenWidth), roundScale(height, vm.ScreenHeight))
		sdl.HideCursor()
	}
	a.viewport = &sdl.FRect{W: vm.ScreenWidth * a.scale, H: vm.ScreenHeight * a.scale}
	if !opts.UseTouch {
		a.viewport.X = (float32(width) - float32(screenWidth)*a.scale) / 2
	}
	a.viewport.Y = (float32(height) - a.viewport.H) / 2
	log.Debugf("window: %dx%d  scale: %g  viewport: %+v", width, height, a.scale, a.viewport)

	a.win, a.renderer, err = sdl.CreateWindowAndRenderer("chip16", width, height, flags)
	if err != nil {
		return err
	}
	if !opts.NoVSync {
		if err := a.renderer.SetVSync(1); err != nil {
			return err
		}
		a.tickRate = vm.TickRate
		a.idleWait = true
	}
	a.win.Raise()
	return nil
}

func roundScale(dsize, fsize int) float32 {
	return float32(2*dsize/fsize) / 2
}

func (a *App) SetPalette(cmap color.Palette) error {
	colors := make([]sdl.Color, vm.PaletteSize)
	for i := range colors {
		colors[i] = sdl.Color(color.RGBAModel.Convert(cmap[i]).(color.RGBA))
	}
	a.Graphics.mu.Lock()
	defer a.Graphics.mu.Unlock()
	return updatePalette(a.tex, colors)
}

// Clean up resources on exit.
func (a *App) Destroy() {
	for _, tex := range a.buttons {
		tex.Destroy()
	}
	a.tex.Destroy()
	a.renderer.Destroy()
	a.win.Destroy()
	sdl.Quit()
}

// Render next frame to screen and wait for vsync.
func (a *App) Present() {
	if a.sndUpdated.Swap(false) {
		a.changed = time.Now()
	}
	if a.loadpal.Swap(false) {
		must(a.SetPalette(a.FG.Palette))
		a.Graphics.mu.Lock()
		a.bgcol = a.RGBA(a.BG)
		a.Graphics.mu.Unlock()
		a.changed = time.Now()
	}
	if a.redrawBG.Swap(false) {
		a.Graphics.mu.Lock()
		a.bgcol = a.RGBA(a.BG)
		a.Graphics.mu.Unlock()
		a.changed = time.Now()
	}
	if a.redrawFG.Swap(false) {
		a.Graphics.mu.Lock()
		must(updateTexture(a.tex, a.FG.Pix))
		a.Graphics.mu.Unlock()
		a.changed = time.Now()
	}
	// draw to back buffer and set vblank flag
	if !a.minimized {
		a.renderer.SetDrawColor(0, 0, 0, 0)
		a.renderer.Clear()
		a.renderer.SetDrawColor(a.bgcol.R, a.bgcol.G, a.bgcol.B, 255)
		a.renderer.RenderFillRect(a.viewport)
		a.renderer.RenderTexture(a.tex, nil, a.viewport)
		if a.buttons != nil {
			a.drawButtons()
		}
	}
	a.setVBlank(time.Now())
	// calc frames per sec and update window title
	a.updateFPS()
	// copy from back buffer to display and wait for next frame vsync
	a.renderer.Present()
	if d := time.Since(a.lastFrame); d < vm.TickRate {
		sdl.DelayPrecise(uint64(vm.TickRate - d))
	}
	a.lastFrame = time.Now()
	// extra delay if inactive for some time
	sleepMode := a.minimized || time.Since(a.changed) > SleepAfter
	if a.sleep != sleepMode {
		log.Debug("set sleep ", sleepMode)
		a.sleep = sleepMode
	}
	if a.sleep {
		time.Sleep(SleepWait)
	}
}

func (a *App) updateFPS() {
	a.frames++
	select {
	case <-a.fpsTicker.C:
	default:
		return
	}
	if a.secs >= 2 {
		a.win.SetTitle(fmt.Sprintf("chip16 @ %d frames/sec", a.frames))
	}
	a.frames = 0
	a.secs++
}

// Timer implements the vm.ITimer interface
type Timer struct {
	vblank   bool
	idleWait bool
	sleep    bool
	tickRate time.Duration
	next     time.Time
	mu       sync.Mutex
}

func (t *Timer) setVBlank(tick time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.vblank = true
	if t.sleep {
		t.next = tick.Add(t.tickRate + SleepWait)
	} else {
		t.next = tick.Add(t.tickRate)
	}
}

func (t *Timer) getVBlank() (vblank bool, next time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	vblank = t.vblank
	t.vblank = false
	return vblank, t.next
}

// Returns true once per frame.
func (t *Timer) VBlank() bool {
	for {
		vblank, next := t.getVBlank()
		if vblank || !(t.sleep || t.idleWait) {
			return vblank
		}
		if d := time.Until(next); d > vm.BusyWait {
			time.Sleep(d - vm.BusyWait)
		}
	}
}

// Graphics implements the vm.IGraphics interface
type Graphics struct {
	vm.GraphicsBase
	redrawBG atomic.Bool
	redrawFG atomic.Bool
	loadpal  atomic.Bool
	mu       sync.Mutex
}

func (g *Graphics) ClearScreen() {
	g.mu.Lock()
	g.GraphicsBase.ClearScreen()
	g.mu.Unlock()
	log.Trace("clear screen")
	g.redrawFG.Store(true)
}

func (g *Graphics) SetBackground(ix uint8) {
	g.mu.Lock()
	g.GraphicsBase.SetBackground(ix)
	g.mu.Unlock()
	log.Tracef("set background %d: %v", g.BG, g.FG.Palette[g.BG])
	g.redrawBG.Store(true)
}

func (g *Graphics) Draw(x, y int16, dptr *byte) bool {
	g.mu.Lock()
	collision := g.GraphicsBase.Draw(x, y, dptr)
	g.mu.Unlock()
	log.Tracef("draw %dx%d sprite at %d,%d -> %v", g.SpriteW, g.SpriteH, x, y, collision)
	g.redrawFG.Store(true)
	return collision
}

func (g *Graphics) LoadPalette(dptr *byte) {
	g.mu.Lock()
	g.GraphicsBase.LoadPalette(dptr)
	g.mu.Unlock()
	g.loadpal.Store(true)
}

// utils
func updateTexture(tex *sdl.Texture, pixels []byte) error {
	buf, _, err := tex.Lock(nil)
	if err != nil {
		return err
	}
	copy(buf, pixels)
	tex.Unlock()
	return nil
}

func updatePalette(tex *sdl.Texture, colors []sdl.Color) error {
	log.Debugf("load palette: %v", colors)
	p, err := sdl.CreatePalette(vm.PaletteSize)
	if err != nil {
		return err
	}
	defer p.Destroy()
	p.SetColors(colors, 0)
	return tex.SetPalette(p)
}

func must2[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func libraryPath() string {
	switch runtime.GOOS {
	case "windows":
		return "SDL3.dll"
	case "linux", "freebsd":
		return "libSDL3.so.0"
	case "darwin":
		return "/usr/local/lib/libSDL3.dylib"
	default:
		return ""
	}
}
