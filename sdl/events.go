package sdl

import (
	"bytes"
	_ "embed"
	"image"
	"image/draw"
	"image/png"
	"slices"
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

// keyboard mapping for controller 1 - controller 2 is same but with shift pressed
var ControllerKeys = map[sdl.Scancode]uint16{
	sdl.SCANCODE_UP:     vm.ButtonUp,
	sdl.SCANCODE_DOWN:   vm.ButtonDown,
	sdl.SCANCODE_LEFT:   vm.ButtonLeft,
	sdl.SCANCODE_RIGHT:  vm.ButtonRight,
	sdl.SCANCODE_SPACE:  vm.ButtonSelect,
	sdl.SCANCODE_RETURN: vm.ButtonStart,
	sdl.SCANCODE_Z:      vm.ButtonA,
	sdl.SCANCODE_X:      vm.ButtonB,
}

// joystick mapping for https://thepihut.com/products/nes-style-raspberry-pi-compatible-usb-gamepad-controller
var JoystickButtons = map[uint8]uint16{
	8: vm.ButtonSelect,
	9: vm.ButtonStart,
	1: vm.ButtonA,
	0: vm.ButtonB,
}

// optional on screen touch buttons
var buttonPos = []image.Point{{24, 10}, {24, 82}, {2, 46}, {46, 46}, {4, 135}, {44, 135}, {4, 190}, {44, 190}}

// Input implements the vm.Input interface
type Input struct {
	vm.InputBase
	joysticks []Joystick
	state     [2]uint16
	buttons   []*sdl.Texture
}

type Joystick struct {
	id  sdl.JoystickID
	dev *sdl.Joystick
}

// Poll for input events - returns false if should quit.
func (a *App) PollEvents(events chan vm.Event) bool {
	var event sdl.Event
	update := [2]bool{}
	for sdl.PollEvent(&event) {
		switch event.Type {
		case sdl.EVENT_KEY_DOWN, sdl.EVENT_KEY_UP:
			ev := event.KeyboardEvent()
			if ev.Repeat {
				continue
			}
			if mask, ok := ControllerKeys[ev.Scancode]; ok {
				controller := shiftModifier()
				a.setBit(controller, ev.Down, mask)
				update[controller] = true
			} else if ev.Scancode == sdl.SCANCODE_ESCAPE && ev.Down {
				return false
			}

		case sdl.EVENT_MOUSE_BUTTON_DOWN:
			ev := event.MouseButtonEvent()
			if ev.Button == 1 {
				for i := range a.buttons {
					r := a.buttonRect(i)
					if ev.X >= r.X && ev.Y >= r.Y && ev.X < r.X+r.W && ev.Y < r.Y+r.H {
						a.state[0] = 1 << i
						update[0] = true
					}
				}
			}

		case sdl.EVENT_MOUSE_BUTTON_UP:
			ev := event.MouseButtonEvent()
			if ev.Button == 1 && len(a.buttons) > 0 {
				a.state[0] = 0
				update[0] = true
			}

		case sdl.EVENT_JOYSTICK_AXIS_MOTION:
			ev := event.JoyAxisEvent()
			if controller, ok := a.joystick(ev.Which); ok {
				if ev.Axis == 1 {
					a.setAxis(controller, int(ev.Value)/256, vm.ButtonUp, vm.ButtonDown)
				} else {
					a.setAxis(controller, int(ev.Value)/256, vm.ButtonLeft, vm.ButtonRight)
				}
				update[controller] = true
			}

		case sdl.EVENT_JOYSTICK_BUTTON_DOWN, sdl.EVENT_JOYSTICK_BUTTON_UP:
			ev := event.JoyButtonEvent()
			if controller, ok := a.joystick(ev.Which); ok {
				if mask, ok := JoystickButtons[ev.Button]; ok {
					a.setBit(controller, ev.Down, mask)
					update[controller] = true
				}
			}

		case sdl.EVENT_JOYSTICK_ADDED:
			ev := event.JoyDeviceEvent()
			js := must2(ev.Which.OpenJoystick())
			log.Infof("added joystick %d: %s", ev.Which, must2(js.Name()))
			a.joysticks = append(a.joysticks, Joystick{id: ev.Which, dev: js})

		case sdl.EVENT_JOYSTICK_REMOVED:
			ev := event.JoyDeviceEvent()
			ix := slices.IndexFunc(a.joysticks, func(j Joystick) bool { return j.id == ev.Which })
			if ix > 0 {
				a.joysticks[ix].dev.Close()
				log.Infof("removed joystick %d", ev.Which)
				a.joysticks = slices.Delete(a.joysticks, ix, ix+1)
			} else {
				log.Warnf("got joystick removed event for %d but it was not previously added", ev.Which)
			}

		case sdl.EVENT_WINDOW_MINIMIZED:
			a.minimized = true
		case sdl.EVENT_WINDOW_RESTORED:
			a.minimized = false
			a.changed = time.Now()
		case sdl.EVENT_QUIT:
			return false
		}
	}

	for i, changed := range update {
		if changed {
			ev := vm.Event{Device: vm.Device(i), State: a.state[i]}
			select {
			case events <- ev:
				log.Debugf("sent event: device=%d state=%08b", i, a.state[i])
			default:
				log.Error("dropping input event: channel full")
			}
		}
	}
	return true
}

func (a *App) setBit(controller vm.Device, on bool, mask uint16) {
	if on {
		a.state[controller] |= mask
	} else {
		a.state[controller] &^= mask
	}
}

func (a *App) setAxis(controller vm.Device, value int, maskNeg, maskPos uint16) {
	if value > 0 {
		a.state[controller] |= maskPos
	} else if value < 0 {
		a.state[controller] |= maskNeg
	} else {
		a.state[controller] &^= (maskPos | maskNeg)
	}
}

func (a *App) joystick(id sdl.JoystickID) (vm.Device, bool) {
	ix := slices.IndexFunc(a.joysticks, func(j Joystick) bool { return j.id == id })
	if ix >= 0 && ix < 2 {
		return vm.Device(ix), true
	}
	log.Warnf("Joystick event with invalid ID %d - ignoring", id)
	return 0, false
}

func shiftModifier() vm.Device {
	keys := sdl.GetKeyboardState()
	if keys[sdl.SCANCODE_LSHIFT] || keys[sdl.SCANCODE_RSHIFT] {
		return vm.Controller2
	} else {
		return vm.Controller1
	}
}

// code to draw on screen touch buttons
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
		if a.state[0]&(1<<i) != 0 {
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
