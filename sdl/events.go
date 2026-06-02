package sdl

import (
	"time"

	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/jnb666/chip16/vm"
)

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

// Poll for input events - returns false if should quit.
func (a *App) PollEvents(v *vm.VM) bool {
	var ev sdl.Event
	for sdl.PollEvent(&ev) {
		switch ev.Type {
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
	left := keyMask(keys, Controller1Keys) | keyMask(keys, Controller1AltKeys)
	right := keyMask(keys, Controller2Keys)
	v.Store(vm.IOBase, left)
	v.Store(vm.IOBase+2, right)
	return true
}

func keyMask(keys []bool, mapping []sdl.Scancode) (mask int16) {
	for i, code := range mapping {
		if keys[code] {
			mask |= 1 << i
		}
	}
	return mask
}
