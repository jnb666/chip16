package sdl

import (
	"bytes"
	_ "embed"
	"image"
	"image/draw"
	"image/png"
	"time"

	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/jnb666/chip16/vm"
	log "github.com/sirupsen/logrus"
)

const (
	TouchPanelWidth = 80
	NumButtons      = 8
	ButtonSize      = 32
)

//go:embed sprites.png
var spriteData []byte

// keys: up, down, left, right, select, start, A, B
var Controller1Keys = []sdl.Scancode{
	sdl.SCANCODE_W, sdl.SCANCODE_S, sdl.SCANCODE_A, sdl.SCANCODE_D,
	sdl.SCANCODE_1, sdl.SCANCODE_2, sdl.SCANCODE_3, sdl.SCANCODE_4,
}
var Controller1AltKeys = []sdl.Scancode{
	sdl.SCANCODE_UP, sdl.SCANCODE_DOWN, sdl.SCANCODE_LEFT, sdl.SCANCODE_RIGHT,
	sdl.SCANCODE_SPACE, sdl.SCANCODE_RETURN, sdl.SCANCODE_Z, sdl.SCANCODE_X,
}
var Controller2Keys = []sdl.Scancode{
	sdl.SCANCODE_I, sdl.SCANCODE_K, sdl.SCANCODE_J, sdl.SCANCODE_L,
	sdl.SCANCODE_7, sdl.SCANCODE_8, sdl.SCANCODE_9, sdl.SCANCODE_0,
}

// optional on screen touch buttons
var buttonPos = []image.Point{{24, 10}, {24, 82}, {2, 46}, {46, 46}, {4, 135}, {44, 135}, {4, 190}, {44, 190}}

// Poll for input events - returns false if should quit.
func (a *App) PollEvents(v *vm.VM) bool {
	var ev sdl.Event
	for sdl.PollEvent(&ev) {
		switch ev.Type {
		case sdl.EVENT_JOYSTICK_ADDED:
			e := ev.JoyDeviceEvent()
			a.joystick = must2(e.Which.OpenJoystick())
			log.Infof("joystick %d added - %s", e.Which, must2(a.joystick.Name()))
		case sdl.EVENT_JOYSTICK_REMOVED:
			e := ev.JoyDeviceEvent()
			if a.joystick != nil && must2(a.joystick.ID()) == e.Which {
				log.Infof("joystick %d removed", e.Which)
				a.joystick.Close()
				a.joystick = nil
			}
		case sdl.EVENT_QUIT:
			return false
		case sdl.EVENT_WINDOW_MINIMIZED:
			a.minimized = true
		case sdl.EVENT_WINDOW_RESTORED:
			a.minimized = false
			a.changed = time.Now()
		}
	}
	keys := sdl.GetKeyboardState()
	if keys[sdl.SCANCODE_ESCAPE] {
		return false
	}
	a.controller[0] = keyMask(keys, Controller1Keys) | keyMask(keys, Controller1AltKeys)
	a.controller[1] = keyMask(keys, Controller2Keys)
	if a.buttons != nil {
		a.controller[0] |= a.buttonMask()
	}
	if a.joystick != nil {
		// hardcoded mapping for https://thepihut.com/products/nes-style-raspberry-pi-compatible-usb-gamepad-controller
		// axis 1 = up, down, axis 0 = left, right
		if vaxis := must2(a.joystick.Axis(1)) / 256; vaxis < 0 {
			a.controller[0] |= 1 << 0
		} else if vaxis > 0 {
			a.controller[0] |= 1 << 1
		}
		if haxis := must2(a.joystick.Axis(0)) / 256; haxis < 0 {
			a.controller[0] |= 1 << 2
		} else if haxis > 0 {
			a.controller[0] |= 1 << 3
		}
		// buttons B=0,  A=1, select=8 start=9
		if a.joystick.Button(8) {
			a.controller[0] |= 1 << 4
		}
		if a.joystick.Button(9) {
			a.controller[0] |= 1 << 5
		}
		if a.joystick.Button(1) {
			a.controller[0] |= 1 << 6
		}
		if a.joystick.Button(0) {
			a.controller[0] |= 1 << 7
		}
	}
	v.Store(vm.IOBase, a.controller[0])
	v.Store(vm.IOBase+2, a.controller[1])
	return true
}

func (a *App) buttonMask() (mask int16) {
	flags, x, y := sdl.GetMouseState()
	if flags&1 == 0 { // left button
		return
	}
	for i := range a.buttons {
		r := a.buttonRect(i)
		if x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.W {
			mask |= 1 << i
		}
	}
	return mask
}

func keyMask(keys []bool, mapping []sdl.Scancode) (mask int16) {
	for i, code := range mapping {
		if keys[code] {
			mask |= 1 << i
		}
	}
	return mask
}

func (a *App) initButtons() error {
	sprites, err := png.Decode(bytes.NewReader(spriteData))
	if err != nil {
		return err
	}
	img := image.NewNRGBA(image.Rect(0, 0, ButtonSize, ButtonSize))
	for i := range NumButtons {
		draw.Draw(img, img.Rect, sprites, image.Pt(i*ButtonSize, 0), draw.Src)
		tex, err := a.renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_STATIC, ButtonSize, ButtonSize)
		if err != nil {
			return err
		}
		tex.SetScaleMode(sdl.SCALEMODE_NEAREST)
		err = tex.Update(nil, img.Pix, int32(img.Stride))
		if err != nil {
			return err
		}
		a.buttons = append(a.buttons, tex)
	}
	return nil
}

func (a *App) drawButtons() {
	for i, tex := range a.buttons {
		if a.controller[0]&(1<<i) != 0 {
			tex.SetColorMod(0, 255, 0)
		} else {
			tex.SetColorMod(128, 128, 128)
		}
		must(a.renderer.RenderTexture(tex, nil, a.buttonRect(i)))
	}
}

func (a *App) buttonRect(i int) *sdl.FRect {
	return &sdl.FRect{
		X: float32(buttonPos[i].X)*a.scale + a.viewport.W,
		Y: float32(buttonPos[i].Y) * a.scale,
		W: ButtonSize * a.scale,
		H: ButtonSize * a.scale,
	}
}
