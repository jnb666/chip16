package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/jnb666/chip16/asm"
	"github.com/jnb666/chip16/sdl"
	"github.com/jnb666/chip16/vm"
	log "github.com/sirupsen/logrus"
)

type Opts struct {
	novsync bool
	vivid   bool
	volume  int
	seed    int
	speed   int
	scale   int
	debug   int
	core    bool
}

var opts Opts

func init() {
	flag.BoolVar(&opts.novsync, "novsync", false, "disable renderer vertical sync")
	flag.BoolVar(&opts.vivid, "vivid", false, "use vivid colormap")
	flag.BoolVar(&opts.core, "core", false, "write core dump on error or halt")
	flag.IntVar(&opts.volume, "volume", 128, "volume level in range from 0-255 or -1 to disable sound")
	flag.IntVar(&opts.seed, "seed", 0, "random number seed - default is auto randomised")
	flag.IntVar(&opts.speed, "speed", 1000, "cpu clock cycle time in nanoseconds")
	flag.IntVar(&opts.scale, "scale", defaultScale(), "window scaling factor")
	flag.IntVar(&opts.debug, "debug", 0, "1=debug logging, 2=verbose debug logging")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:\n  chip16 [options] file")
		fmt.Fprintln(os.Stderr, "\tfile suffix is .asm or .s for assembler else .c16 rom format or raw machine code")
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}
	if opts.debug > 0 {
		log.SetLevel(log.InfoLevel + log.Level(opts.debug))
	}

	rom, err := getROM(flag.Arg(0))
	check(err)

	app, err := sdl.New(!opts.novsync, opts.scale, opts.volume)
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
	err = loadROM(v, rom)
	check(err)
	go run(v)
	for app.PollEvents(v) {
		app.Present()
	}
}

func run(v *vm.VM) {
	err := v.Run()
	if err == nil {
		return
	}
	if opts.core {
		if e := os.WriteFile("core", v.Mem[:], 0644); e != nil {
			log.Error(e)
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
	a.BaseDir, _ = os.Getwd()
	err = a.Assemble(r)
	if err == nil {
		log.Infof("assembled %d bytes from %s\n", len(a.Code), file)
	}
	return a.Code, err
}

func loadROM(v *vm.VM, data []byte) error {
	if string(data[:4]) != "CH16" {
		copy(v.Mem[:], data)
		return nil
	}
	size := binary.LittleEndian.Uint32(data[0x06:])
	copy(v.Mem[:], data[16:16+size])
	v.PC = binary.LittleEndian.Uint16(data[0x0A:])
	return nil
}

func isAsm(file string) bool {
	file = strings.ToLower(file)
	return strings.HasSuffix(file, ".asm") || strings.HasSuffix(file, ".s")
}

func defaultScale() int {
	if runtime.GOOS == "darwin" {
		return 2
	} else {
		return 4
	}
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
