package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jnb666/chip16/asm"
	log "github.com/sirupsen/logrus"
)

type Opts struct {
	outfile string
	debug   int
}

var opts Opts

func init() {
	flag.StringVar(&opts.outfile, "o", "rom.bin", "output file name - adds 16 byte header if suffix is .c16")
	flag.IntVar(&opts.debug, "debug", 0, "1=debug logging, 2=verbose debug logging")
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
	if opts.debug > 0 {
		log.SetLevel(log.InfoLevel + log.Level(opts.debug))
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
	a.BaseDir, _ = os.Getwd()
	err = a.Assemble(input)
	check(err)

	log.Infof("writing %d bytes to %s\n", len(a.Code), opts.outfile)
	f, err := os.Create(opts.outfile)
	check(err)
	defer f.Close()
	if strings.HasSuffix(opts.outfile, ".c16") {
		_, err = f.Write(asm.C16Header(a.Code, 0))
	}
	_, err = f.Write(a.Code)
	check(err)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
