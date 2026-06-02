// Chip16 graphics and sound implentation using SDL3.
package sdl

import (
	"fmt"
	"image/color"
	"runtime"
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

// App implements the vm.IMachine interface.
type App struct {
	Graphics
	Sound
	Timer
	win       *sdl.Window
	renderer  *sdl.Renderer
	tex       *sdl.Texture
	changed   time.Time
	minimized bool
	fpsTicker *time.Ticker
	frames    int
	secs      int
}

// Initialise SDL application.
func New(vsync bool, scale, volume int) (*App, error) {
	err := sdl.LoadLibrary(libraryPath())
	if err != nil {
		return nil, err
	}
	err = sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO)
	if err != nil {
		return nil, err
	}
	log.Infof("SDL version %s: scale=%d vsync=%v", sdl.GetVersion(), scale, vsync)

	a := &App{changed: time.Now()}
	a.Graphics.Init()
	a.Sound.Init(volume)
	a.win, a.renderer, err = sdl.CreateWindowAndRenderer("chip16", vm.ScreenWidth*scale, vm.ScreenHeight*scale, 0)
	if err != nil {
		return nil, err
	}
	if vsync {
		if err := a.renderer.SetVSync(1); err != nil {
			return nil, err
		}
		a.tickRate = vm.TickRate
		a.idleWait = true
	}
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
	a.fpsTicker = time.NewTicker(time.Second)
	return a, nil
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
		bg := a.RGBA(a.BG)
		a.Graphics.mu.Unlock()
		log.Debugf("set draw color %+v", bg)
		a.renderer.SetDrawColor(bg.R, bg.G, bg.B, 0xFF)
		a.changed = time.Now()
	}
	if a.redrawBG.Swap(false) {
		a.Graphics.mu.Lock()
		bg := a.RGBA(a.BG)
		a.Graphics.mu.Unlock()
		log.Debugf("set draw color %+v", bg)
		a.renderer.SetDrawColor(bg.R, bg.G, bg.B, 0xFF)
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
		a.renderer.Clear()
		a.renderer.RenderTexture(a.tex, nil, nil)
	}
	a.setVBlank(time.Now())
	// calc frames per sec and update window title
	a.updateFPS()
	// copy from back buffer to display - waits for next frame vsync
	a.renderer.Present()
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
	log.Debug("set background ", g.BG)
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
	log.Debug("load palette")
	p, err := sdl.CreatePalette(vm.PaletteSize)
	if err != nil {
		return err
	}
	defer p.Destroy()
	p.SetColors(colors, 0)
	return tex.SetPalette(p)
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
