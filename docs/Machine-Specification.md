Contents:
* [1. CPU](#cpu)
  * [1.1. Word size](#word-size)
  * [1.2. Registers](#registers)
  * [1.3. Instructions](#instructions)
  * [1.4. Memory](#memory)
  * [1.5. Flags](#flags)
* [2. Graphics](#graphics)
  * [2.1. State registers](#state-registers)
  * [2.2. Screen details](#screen-details)
  * [2.3. Display format](#display-format)
  * [2.4. Sprites](#sprites)
  * [2.5. Palette](#palette)
* [3. Sound](#sound)
  * [3.1. Wave types](#wave-types)
  * [3.2. Constants](#constants)
* [4. Input](#input)
  * [4.1. Controller layout](#controller-layout)

Appendix:
* [A. File format](#file-format)
  * [A.1. Raw format](#raw-format)
  * [A.2. Chip16 format](#chip16-format)

***

<a name="cpu"></a>
## 1. CPU
<a name="word-size"></a>
### 1.1. Word Size
Chip16 uses 16 bit words, as the name implies.
Hence the registers are 16 bits wide, and memory is read 16 bits at a time.

Chip16 is a _little-endian_ machine (LSB, Least Significant Byte first).
All values in registers and memory are read and written in this representation.

<a name="registers"></a>
### 1.2. Registers
* 1x 16 bit program counter (`PC`)
* 1x 16 bit stack pointer (`SP`)
* 16x 16 bit general purpose registers (`R0`..`RF`)
* 1x 8 bit flag register (`FLAGS`)

The general purpose registers should be interpreted as signed ([two's complements representation](http://en.wikipedia.org/wiki/Two%27s_complement), unless otherwise stated).
PC, SP, and FLAGS are unsigned.

<a name="instructions"></a>
### 1.3. Instructions
Chip16 operates at a frequency of 1 MHz (1,000,000 cycles/second).

_All_ instructions:
* take 1 cycle to execute
* are stored in 4 bytes

The complete list of instructions is detailed [here](https://github.com/tykel/chip16/wiki/Instructions).

<a name="memory"></a>
### 1.4. Memory
Chip16 has 64 KB (65,536 B) of memory.

When reading word values from memory, they should be interpreted as signed, like registers, unless otherwise stated.

There is no distinction between ROM and RAM; the contents of ROM are simply mapped into RAM, and may be overwritten.

Special addresses:
* `0x0000`: Start of the ROM/RAM
* `0xFDF0`: Start of the stack (512 B)
* `0xFFF0`: Start of I/O ports (4 B)

The I/O ports are controller inputs (more later).

<a name="flags"></a>
### 1.5. Flags
The flags register maps its bits in the following way:

Bit | 0|1|2|3|4|5|6|7
---|---|---|---|---|---|---|---|---
**Function** ||`C` (carry)|`Z` (zero)||||`O` (overflow)|`N` (negative)

For an explanation of how they are set or affect conditional jumps, refer to the [Instructions wiki](Instructions). 

<a name="graphics"></a>
## 2. Graphics
<a name="state-registers"></a>
### 2.1. State registers
Chip16 keeps its graphics state with a number of hidden registers, which are accessed via special instructions.
They are not directly modifiable, so there is scope for flexibility in the implementation of the state; the following names are for reference only.

Register | Type | Function
---|---|---
`bg` | Nibble | Color index of background layer
`spritew` | Unsigned byte | Width of sprite(s) to draw
`spriteh` | Unsigned byte | Height of sprite(s) to draw
`hflip` | Boolean | Flip sprite(s) to draw, horizontally
`vflip` | Boolean | Flip sprite(s) to draw, vertically

<a name="screen-details"></a>
### 2.2. Screen details
Chip16 uses a 320x240 screen resolution.
The screen is updated at a frequency of 60 Hz.
Every frame (~16.67ms), the internal `VBLANK` flag is raised, which can be waited on with the `VBLNK` instruction.

<a name="display-format"></a>
### 2.3. Display format
Colors are 4 bit indexed, mapped to the default palette initially.
The screen is represented using a foreground layer `FG` and background layer `BG`.
`BG` is simply a color index.

<a name="sprites"></a>
### 2.4. Sprites
Sprites are byte coded color indexes, where 1 byte (8 bits) represents 2 neighboring pixels, left to right.

The current sprite width and height can be set.
The sprite width counts the number of bytes to read per row, and the sprite height how many rows the sprite has. 
Effectively, the sprite width is half the width in pixels.

Sprites are drawn with the `DRW` command, which accepts two words for coordinates. These should remain signed, as sprites can be drawn offscreen, or partly onscreen.

There is no wrapping; what is drawn offscreen stays offscreen, and is thrown away.

If a non-transparent part of the sprite overlaps a non-zero element of the screen (not the background, then), the `C` flag is raised (See [Instructions wiki](https://github.com/tykel/chip16/wiki/Instructions)).

<a name="palette"></a>
### 2.5. Palette
The default palette used initially is the following:

Index | Value | Color || Index | Value | Color
--- | --- | --- | --- | --- | --- | ---
0x0 | `#000000` | Black (transp. in `FG`) | | 0x8 | `#EAD979` | Yellow
0x1 | `#000000` | Black | | 0x9 | `#537A3B` | Green
0x2 | `#888888` | Gray | | 0xA | `#ABD54A` | Light green
0x3 | `#BF3932` | Red | | 0xB | `#252E38` | Dark blue
0x4 | `#DE7AAE` | Pink | |  0xC | `#00467F` | Blue
0x5 | `#4C3D21` | Dark brown | | 0xD | `#68ABCC` | Light blue
0x6 | `#905F25` | Brown | | 0xE | `#BCDEE4` | Sky blue
0x7 | `#E49452` | Orange | | 0xF | `#FFFFFF` | White

The palette can be changed at runtime with the `PAL` instruction.

Note that a palette change affects _all_ drawing operations for the current frame, including those _before_ the palette load.

Color index 0 is always transparent in `FG`, though.

---

<a name="sound"></a>
## 3.Sound
Two options are available:
* Playing one of 3 fixed tones (500 Hz, 1000 Hz, 1500 Hz) for a given number of milliseconds
* Using sound generation options (`SNG` instruction), play a tone from memory for a given number of milliseconds.

When using the second option, the following applies for timing:

`Total time = Attack + Decay + (Duration - Attack - Decay) + Release`

<a name="wave-types"></a>
### 3.1. Wave types
Sounds may be of 4 types, which affects the sound's wave pattern:
* **Triangle**: Slope up and down linearly between negative and positive amplitude at the given frequency.
* **Sawtooth**: Slope up from negative to positive amplitude, then fall back to negative, at the given frequency.
* **Pulse** (**Square**): Alternate between negative and positive amplitude in a binary fashion, at the given frequency.
* **Noise**: A white noise channel, obtained by creating random samples at the given frequency.

<a name="constants"></a>
### 3.2 Constants
This is the mapping of the constants expected in the `SNG` instruction to their reference values.

`Attack` is the duration it takes to go from intensity 0 to max intensity (volume).

Index | 0|1|2|3|4|5|6|7|8|9|10|11|12|13|14|15
--- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | ---
**Duration (ms)** | 2|8|16|24|38|56|68|80|100|250|500|800|1000|3000|5000|8000

`Decay` is the duration it takes to go from max intensity to sustain intensity.

Index | 0|1|2|3|4|5|6|7|8|9|10|11|12|13|14|15
--- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | ---
**Duration (ms)** | 6|24|48|72|114|168|204|240|300|750|1500|2400|3000|9000|15000|24000

`Release` is the duration it takes to go from sustain intensity to intensity 0.

Index | 0|1|2|3|4|5|6|7|8|9|10|11|12|13|14|15
--- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | ---
**Duration (ms)** | 6|24|48|72|114|168|204|240|300|750|1500|2400|3000|9000|15000|24000

`Volume` is the peak volume of the sound, where 0xF is the maximum, and 0x0 is the minimum.

`Sustain` is the volume the sound will sustain after decay, until release, where 0xF is the max., and 0x0 is the min.

---

<a name="input"></a>
## 4. Input
Chip16 uses up to 2 controllers as its input method.
<a name="controller-layout"></a>
### 4.1. Controller layout
The controllers are very similar in layout to that of the NES:

![Chip16 controller layout](http://i.imgur.com/ExCAKZo.png)

`Bit[0] = Up`
`Bit[1] = Down`
`Bit[2] = Left`
`Bit[3] = Right`
`Bit[4] = Select`

`Bit[5] = Start`
`Bit[6] = A`
`Bit[7] = B`
`Bit[8..15] = Unused (Always zero).`

The state of each controller (for each button, 1 = pressed, 0 = not pressed) is updated at every `VBLANK` event.

Up to 2 controllers are supported for now; controller 1 updates addresses 0xFFF0-0xFFF1, controller 2 updates 0xFFF2-0xFFF3.

---

<a name="file-format"></a>
## A. File format
ROMs can be stored either in raw binary format, or in a Chip16 ROM file.
<a name="raw-format"></a>
### A.1. Raw format (.bin, .c16)
Only the ROM is stored, with no metadata.

Start address is set to 0x0000.
<a name="chip16-format"></a>
### A.2. Chip16 format (.c16)
Here a 16 byte header is present, followed by the raw ROM.

The header is as follows:

Offset | Size | Meaning
---|---|---
0x00 | 4 | Magic number ('CH16')
0x04 | 1 | _Reserved_ (0)
0x05 | 1 | Specifcation version H.L (0xHL)
0x06 | 4 | ROM size in bytes (excluding header)
0x0A | 2 | Start address (initial value of `PC` register)
0x0C | 4 | CRC32 checksum of ROM (excluding header) (polynomial: **0x04C11DB7**)