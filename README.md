# A Chip16 Emulator in go

This is a simple retro style 16 bit game console emulator with graphics and sound using the [github.com/chip16/chip16](https://github.com/chip16/chip16) machine specification and instruction set.

To install just run `go install ./cmd/*`. It's tested on macOS and Linux and should work on all platforms supported by go.

To run the emulator: `chip16 file` where file may be either a raw binary, a .c16 binary with header or a .asm assembly file. Type `chip16 -h` for options.

[API documentation](https://pkg.go.dev/github.com/jnb666/chip16) is on pkg.go.dev as per usual.

### In this repo

- [vm](http://github.com/jnb666/tree/master/vm) is the core virtual machine and defines the Machine interface
- [sdl](http://github.com/jnb666/tree/master/sdl) implements vm.Machine to provide graphics and sound using SDL3
- [cmd/chip16](http://github.com/jnb666/tree/cmd/chip16) is the command line interface for the emulator

- [asm](http://github.com/jnb666/tree/master/asm) provides an API to generate and disassemble chip16 machine code
- [cmd/gas16](http://github.com/jnb666/tree/cmd/gas16) is the command line interface for the assember
- [cmd/dis16](http://github.com/jnb666/tree/cmd/dis16) is a simple disassembler to dump binary files

- [docs](http://github.com/jnb666/tree/master/docs) has a copy of the docs from [the Chip16 wiki](https://github.com/chip16/chip16/wiki)
- [examples](http://github.com/jnb666/tree/master/examples) contains some example programs
- [program_pack](http://github.com/jnb666/tree/master/program_pack) is the contents of `Chip16 program pack 09.04.2018.zip` with ROMs and source code

### Benchmarks

Test using game of life simulation and disabling 1 Mhz cycle timing so VM runs at full speed:

```
goos: darwin
goarch: arm64
pkg: github.com/jnb666/chip16/asm
cpu: Apple M2
BenchmarkLife-8         4882       2124615 ns/op           470.7 frames/sec          5.900 ns/instruction

goos: linux
goarch: amd64
pkg: github.com/jnb666/chip16/asm
cpu: AMD Ryzen 9 9900X 12-Core Processor
BenchmarkLife-24        9358       1255118 ns/op           796.7 frames/sec          3.486 ns/instruction

oos: linux
goarch: arm64 (Raspberry Pi 4)
pkg: github.com/jnb666/chip16/asm

BenchmarkLife-4        1142       10319710 ns/op           96.90 frames/sec         28.63 ns/instruction
```

### TODO
- menu screen with list of ROMs in current directory if none selected - return to this on hitting escape
- music example with selection of tunes
- asm: add support for image conversion in importbin
- add debugger
- add a keyboard interface using memory mapped IO
- add other os services - e.g. get time, sleep
- high level programming environment - e.g. a BASIC or FORTH interpreter or compiler
