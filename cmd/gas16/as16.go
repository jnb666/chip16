package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/jnb666/chip16/asm"
	log "github.com/sirupsen/logrus"
)

type Opts struct {
	outfile string
	debug   bool
}

var opts Opts

func init() {
	flag.StringVar(&opts.outfile, "o", "rom.bin", "output file name")
	flag.BoolVar(&opts.debug, "debug", false, "enable debug logging")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:\n  gas16 [options] file.asm\nOptions:")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}
	if opts.debug {
		log.SetLevel(log.DebugLevel)
	}

	var err error
	var input io.Reader
	if flag.Arg(0) == "-" {
		input = os.Stdin
	} else {
		input, err = os.Open(flag.Arg(0))
		check(err)
	}

	a := asm.New()
	err = a.Assemble(input)
	check(err)

	log.Infof("writing %d bytes to %s\n", len(a.Code), opts.outfile)
	err = os.WriteFile(opts.outfile, a.Code, 0644)
	check(err)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
