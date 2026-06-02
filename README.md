# A Chip16 Emulator in go

This is a simple retro style 16 bit game console emulator with graphics and sound using the [github.com/chip16/chip16](https://github.com/chip16/chip16) machine specification and instruction set.

To install just run `go install ./cmd/*`. It's tested on macOS and Linux and should work on all platforms supported by go.

To run the emulator: `chip16 file` where file may be either a raw binary, a .c16 binary with header or a .asm assembly file. Type `chip16 -h` for options.

[API documentation](https://pkg.go.dev/github.com/jnb666/chip16) is on pkg.go.dev as per usual.

### In this repo

- [vm](http://github.com/jnb666/tree/master/vm) is the core virtual machine and defines the Machine interface
- [sdl](http://github.com/jnb666/tree/master/sdl) implements vm.Machine to provide graphics and sound using SDL3
- [cmd/chip16](http://github.com/jnb666/tree/cmd/chip16) is the command line interface for the emulator

- [asm](http://github.com/jnb666/tree/master/asm) provides an API to generate chip16 machine code
- [cmd/gas16](http://github.com/jnb666/tree/cmd/gas16) is the command line interface for the assember

- [docs](http://github.com/jnb666/tree/master/docs) has a copy of the docs from [the Chip16 wiki](https://github.com/chip16/chip16/wiki)
- [examples](http://github.com/jnb666/tree/master/examples) contains some example programs
- [program_pack](http://github.com/jnb666/tree/master/program_pack) is the contents of `Chip16 program pack 09.04.2018.zip` with ROMs and source code

### TODO

- asm: add support for image conversion in importbin and include directive
- add a disassembler and debugger
- add a keyboard interface using memory mapped IO
- add other os services - e.g. get time, sleep
- high level programming environment - e.g. a BASIC or FORTH interpreter or compiler
