package main

import (
	"fmt"
	"os"

	"github.com/jnb666/chip16/asm"
	log "github.com/sirupsen/logrus"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: dis16 rom.c16")
		os.Exit(1)
	}
	data, err := os.ReadFile(os.Args[1])
	check(err)
	code, start, err := asm.ReadC16Header(data)
	if err != nil {
		log.Warn(err)
	}

	addr := 0
	if start > 0 {
		dump(addr, code[:start])
		addr = int(start)
		fmt.Println("_start:")
	}
	for addr < len(code) {
		dumpOp(addr, code[addr:])
		addr += 4
	}
}

func dumpOp(addr int, data []byte) {
	if len(data) >= 4 {
		text := asm.Opcode(data[:4]).String()
		fmt.Printf("%04X : %02X  %s\n", addr, data[:4], text)
	}
}

func dump(addr int, data []byte) {
	for len(data) > 0 {
		n := min(len(data), 4)
		fmt.Printf("%04X : %02X\n", addr, data[:n])
		data = data[n:]
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
