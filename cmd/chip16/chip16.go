package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jnb666/chip16/asm"
	"github.com/jnb666/chip16/sdl"
	"github.com/jnb666/chip16/vm"
	log "github.com/sirupsen/logrus"
)

type Opts struct {
	sdl.Options
	vivid  bool
	seed   int
	speed  int
	debug  int
	core   bool
	screen bool
	nosum  bool
}

var opts = Opts{Options: sdl.DefaultOptions}

func getopts() {
	driver, err := sdl.Init()
	check(err)
	flag.BoolVar(&opts.Fullscreen, "fullscreen", driver == "kmsdrm", "use fullscreen instead of windowed mode")
	flag.BoolVar(&opts.NoVSync, "novsync", false, "disable renderer vertical sync")
	flag.BoolVar(&opts.UseTouch, "touch", driver == "kmsdrm", "onscreen buttons for gamepad controls")
	flag.BoolVar(&opts.vivid, "vivid", false, "use vivid colormap")
	flag.BoolVar(&opts.core, "coredump", false, "write core dump on error or halt")
	flag.BoolVar(&opts.screen, "screendump", false, "write screen.png image on halt")
	flag.BoolVar(&opts.nosum, "nochecksum", false, "ignore invalid .c16 checksum")
	flag.IntVar(&opts.Volume, "volume", 128, "volume level in range from 0-255 or -1 to disable sound")
	flag.IntVar(&opts.seed, "seed", 0, "random number seed - default is auto randomised")
	flag.IntVar(&opts.speed, "speed", 1000, "cpu clock cycle time in nanoseconds")
	flag.IntVar(&opts.debug, "debug", 0, "1=debug logging, 2=verbose debug logging")
	flag.Float64Var(&opts.Scale, "scale", opts.Scale, "set window scaling factor")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:\n  chip16 [options] file")
		fmt.Fprintln(os.Stderr, "\tfile suffix is .asm or .s for assembler else .c16 rom format or raw machine code")
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}
	if opts.debug > 0 {
		log.SetLevel(log.InfoLevel + log.Level(opts.debug))
	}
}

func main() {
	getopts()

	rom, err := getROM(flag.Arg(0))
	check(err)

	app, err := sdl.New(opts.Options)
	check(err)
	defer app.Destroy()
	if opts.vivid {
		app.SetPalette(vm.AltPalette)
	}
	v := vm.New(app)
	if opts.seed != 0 {
		v.RNG.Seed(int64(opts.seed))
	}
	v.CycleTime = time.Duration(opts.speed) * time.Nanosecond
	code, start, err := asm.ReadC16Header(rom)
	if opts.nosum && err != nil {
		log.Warn(err)
	} else {
		check(err)
	}
	copy(v.Mem[:], code)
	v.PC = start

	go run(v)
	frames := 0
	startTime := time.Now()
	for app.PollEvents(v) {
		app.Present()
		frames++
	}
	elapsed := time.Since(startTime).Seconds()
	log.Infof("Average frame rate = %.1f frames/sec", float64(frames)/elapsed)
}

func run(v *vm.VM) {
	err := v.Run()
	if err == nil {
		return
	}
	if opts.core {
		log.Info("writing core dump")
		if e := os.WriteFile("core", v.Mem[:], 0644); e != nil {
			log.Error(e)
		}
	}
	if opts.screen {
		log.Info("writing screen to screen.png")
		var buf bytes.Buffer
		e := png.Encode(&buf, v.ScreenImage())
		if e == nil {
			e = os.WriteFile("screen.png", buf.Bytes(), 0644)
		}
		if e != nil {
			log.Error(err)
		}
	}
	if e, ok := err.(vm.Error); ok {
		fmt.Fprintf(os.Stderr, "%s\n", v)
		log.Fatalf("%s\n\n%s", e.Message, e.Stack)
	} else {
		log.Fatal(err)
	}
}

func getROM(file string) (rom []byte, err error) {
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	if !isAsm(file) {
		return io.ReadAll(r)
	}
	a := asm.New()
	a.BaseDir = filepath.Dir(file)
	err = a.Assemble(r)
	if err == nil {
		log.Infof("assembled %d bytes from %s\n", len(a.Code), file)
	}
	return a.Code, err
}

func isAsm(file string) bool {
	file = strings.ToLower(file)
	return strings.HasSuffix(file, ".asm") || strings.HasSuffix(file, ".s")
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
