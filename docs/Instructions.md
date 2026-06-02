# Chip16 Instructions
## Abbrevations used

Abbreviations | Meaning | | Abbreviations | Meaning | | Abbreviations | Meaning
--- | --- | --- | --- | --- | --- | --- | ---
`HH` | _high_ byte of a word  | | `FG` | screen foreground layer | | `[A]` | value in memory at address `A`
`LL` | _low_ byte of a word  | | `BG` | background color index | | `x` | conditional code for jumps/calls
`N` | _nibble_ (4-bit value)  | | `hflip` | horizontal sprite flip | | |
`X, Y, Z` | 4-bit register identifiers | | `vflip` | vertical sprite flip | | |

## Setting flags

For the instructions marked as affecting flags:

* `C` is set:
   * for `ADD` - when the result would have imaginary bit[16] set; else it is cleared.
   * for `SUB` - when the operation needed to borrow from bit[16]; else it is cleared.
   * for `MUL` - when the result is too big to fit into 16 bits; else it is cleared.
   * for `DIV` - when the remainder of the division is non-zero; else it is cleared.
   * for `DRW` - when the new sprite overlaps any existing pixel (other than 0 - transparent); else it is cleared.

* `Z` is set:
   * when the result is 0, else its cleared.

* `O` is set:
   * in general, when the sign of the result differs from what would be expected.
   * for `ADD`: when the result is positive and both operands were negative,
or if the result is negative and both operands were positive; else it is cleared.
   * for `SUB` (X-Y=Z): when Z is positive and X is negative and Y is positive,
or if Z is negative and X is positive and Y is negative; else it is cleared.

* `N` is set:
   * when the result is less than 0 (bit[15] == 1); else it is cleared.

## Conditional branch types
Branch when the flags requirements are met.

Type | Index | Flags requirements | Mnemonic
---|---|---|---
Z | 0x0 | `Z == 1` | Equal (Zero)
NZ | 0x1 | `Z == 0` | Not Equal (Non-Zero)
N | 0x2 | `N == 1` | Negative
NN | 0x3 | `N == 0` | Not-Negative (Positive or Zero)
P | 0x4 | `N == 0` and `Z==0` | Positive
O | 0x5 | `O == 1` | Overflow
NO | 0x6 | `O == 0` | No Overflow
A | 0x7 | `C == 0` and `Z == 0` | Above (Unsigned Greater Than)
AE | 0x8 | `C == 0` | Above Equal (Unsigned Greater Than or Equal)
B | 0x9 | `C == 1` | Below       (Unsigned Less Than)
BE | 0xA | `C == 1` or `Z == 1` | Below Equal (Unsigned Less Than or Equal)
G | 0xB | `O == N` and `Z == 0` | Signed Greater Than
GE | 0xC | `O == N` | Signed Greater Than or Equal
L | 0xD | `O != N` | Signed Less Than
LE | 0xE | `O != N` or `Z == 1` | Signed Less Than or Equal
RES | 0xF |  | Reserved for future use

Alternative valid mnemonics:

Type | Index | Flags requirements | Mnemonic
---|---|---|---
C | 0x9 | `C == 1` | Carry (Same as B)
NC | 0x8 | `C == 0` | Not Carry (Same as GE)

***

Note: `PC` is incremented by 4 after the instruction is read, and before it is executed.

## 0x - Misc/Video/Audio
Opcode (Hex) | Mnemonic | Usage | Flags affected | Introduced
--- | --- | --- | --- | ---
00 00 00 00 | NOP | No operation. | | 0.8 |
01 00 00 00 | CLS | Clear `FG`, `BG` = 0. | |0.8 |
02 00 00 00 | VBLNK | Wait for VBlank. If (!vblank) `PC` -= 4; | |0.8 |
03 00 0N 00 | BGC N | Set background color to index `N` (0 is black). | |0.8 |
04 00 LL HH | SPR HHLL | Set sprite width (`LL`) and height (`HH`). | |0.8 |
05 YX LL HH | DRW RX, RY, HHLL | Draw sprite from address `HHLL` at (`RX`, `RY`). | `C` | 0.8 |
06 YX 0Z 00 | DRW RX, RY, RZ | Draw sprite from `[RZ]` at (RX, RY). | `C` | 0.8 |
07 0X LL HH | RND RX, HHLL | Store random number in `RX` (max. `HHLL`). || 0.8 |
08 00 00 00 | FLIP 0, 0 | Set `hflip` = false, `vflip` = false | |0.8 |
08 00 00 01 | FLIP 0, 1 | Set `hflip` = false, `vflip` = true | |0.8 |
08 00 00 02 | FLIP 1, 0 | Set `hflip` = true,  `vflip` = false | |0.8 |
08 00 00 03 | FLIP 1, 1 | Set `hflip` = true,  `vflip` = true | |0.8 |
09 00 00 00 | SND0 | Stop playing sounds. | |0.8 |
0A 00 LL HH | SND1 HHLL | Play 500Hz tone for `HHLL` ms. ||0.8 |
0B 00 LL HH | SND2 HHLL | Play 1000Hz tone for `HHLL` ms. ||0.8 |
0C 00 LL HH | SND3 HHLL | Play 1500Hz tone for `HHLL` ms. ||0.8 |
0D 0X LL HH | SNP RX, HHLL | Play tone from `RX` for `HHLL` ms. ||1.1 |
0E AD SR VT | SNG AD, VTSR | Set sound generation parameters. | |1.1 |
0F 00 00 00 | HALT | Halt VM and raise exception. ||non standard |

## 1x - Jumps (Branches)
Opcode (Hex) | Mnemonic | Usage | Flags affected | Introduced
--- | --- | --- | --- | ---  
10 00 LL HH | JMP HHLL | Set `PC` to `HHLL`. ||0.8 |
11 00 LL HH | JMC HHLL | Jump to the specified address if carry flag is raised. ||0.8 |
12 0x LL HH | Jx  HHLL | If `x`, then perform a _JMP_. ||0.9 |
13 YX LL HH | JME RX, RY, HHLL | Set `PC` to `HHLL` if `RX == RY`. ||0.8 |
14 00 LL HH | CALL HHLL | Store `PC` to `[SP]`, increase `SP` by 2, set `PC` to `HHLL`. ||0.8 |
15 00 00 00 | RET | Decrease `SP` by 2, set `PC` to `[SP]`. ||0.8 |
16 0X 00 00 | JMP RX | Set `PC` to `RX`. ||0.8 |
17 0x LL HH | Cx HHLL | If `x`, then perform a _CALL_. ||0.9 |
18 0X 00 00 | CALL RX | Store `PC` to `[SP]`, increase `SP` by 2, set `PC` to `RX`. ||0.8 |

## 2x - Loads
Loads from memory are always 16-bit.

Opcode (Hex) | Mnemonic | Usage | Flags affected | Introduced
--- | --- | --- | --- | ---
20 0X LL HH | LDI RX, HHLL | Set `RX` to `HHLL`. | |0.8 |
21 00 LL HH | LDI SP, HHLL | Set `SP` to `HHLL`. ||0.8 |
22 0X LL HH | LDM RX, HHLL | Set `RX` to `[HHLL]`. ||0.8 |
23 YX 00 00 | LDM RX, RY | Set `RX` to `[RY]`. ||0.8 |
24 YX 00 00 | MOV RX, RY | Set `RX` to `RY`. ||0.8 |

## 3x - Stores
Opcode (Hex) | Mnemonic | Usage | Flags affected | Introduced
--- | --- | --- | --- | ---  
30 0X LL HH | STM RX, HHLL | Set `[HHLL]` to `RX`. | |0.8 |
31 YX 00 00 | STM RX, RY | Set `[RY]` to `RX`. | |0.8 |

## 4x - Addition
Opcode (Hex) | Mnemonic | Usage | Flags affected | Introduced
--- | --- | --- | --- | ---
40 0X LL HH | ADDI RX, HHLL | Set `RX` to `RX`+`HHLL`. | `C` `Z` `O` `N` | 0.8 |
41 YX 00 00 | ADD RX, RY | Set `RX` to `RX`+`RY`. | `C` `Z` `O` `N` | 0.8 |
42 YX 0Z 00 | ADD RX, RY, RZ | Set `RZ` to `RX`+`RY`. | `C` `Z` `O` `N` | 0.8 |

## 5x - Subtraction
Opcode (Hex) | Mnemonic | Usage | Flags affected | Introduced
--- | --- | --- | --- | --- 
50 0X LL HH | SUBI RX, HHLL | Set `RX` to `RX`-`HHLL`. | `C` `Z` `O` `N` | 0.8 |
51 YX 00 00 | SUB RX, RY | Set `RX` to `RX`-`RY`. | `C` `Z` `O` `N` | 0.8 |
52 YX 0Z 00 | SUB RX, RY, RZ | Set `RZ` to `RX`-`RY`. | `C` `Z` `O` `N` | 0.8 |
53 0X LL HH | CMPI RX, HHLL | Compute `RX`-`HHLL`, discard result. | `C` `Z` `O` `N` | 0.8 |
54 YX 00 00 | CMP RX, RY | Compute `RX`-`RY`, discard result. | `C` `Z` `O` `N` | 0.8 |

## 6x - Bitwise AND (&)
Opcode (Hex) | Mnemonic | Usage | Flags affected | Introduced
--- | --- | --- | --- | ---
60 0X LL HH | ANDI RX, HHLL | Set `RX` to `RX`&`HHLL`. | `Z` `N` | 0.8 |
61 YX 00 00 | AND RX, RY | Set `RX` to `RX`&`RY`. | `Z` `N` | 0.8 |
62 YX 0Z 00 | AND RX, RY, RZ | Set `RZ` to `RX`&`RY`. | `Z` `N` | 0.8 |
63 0X LL HH | TSTI RX, HHLL | Compute `RX`&`HHLL`, discard result. | `Z` `N` | 0.8 |
64 YX 00 00 | TST RX, RY | Compute `RX`&`RY`, discard result. | `Z` `N` | 0.8 |

## 7x - Bitwise OR (|)
Opcode (Hex) | Mnemonic | Usage | Flags affected | Introduced
--- | --- | --- | --- | ---
70 0X LL HH | ORI RX, HHLL | Set `RX` to `RX`&#124;`HHLL`. | `Z` `N` | 0.8 |
71 YX 00 00 | OR RX, RY | Set `RX` to `RX`&#124;`RY`. | `Z` `N` | 0.8 |
72 YX 0Z 00 | OR RX, RY, RZ | Set `RZ` to `RX`&#124;`RY`. | `Z` `N` | 0.8 |

## 8x - Bitwise XOR (^)
Opcode (Hex) | Mnemonic | Usage | Flags affected | Introduced
--- | --- | --- | --- | ---
80 0X LL HH | XORI RX, HHLL | Set `RX` to `RX`^`HHLL`. | `Z` `N` | 0.8 |
81 YX 00 00 | XOR RX, RY | Set `RX` to `RX`^`RY`. | `Z` `N` | 0.8 |
82 YX 0Z 00 | XOR RX, RY, RZ | Set `RZ` to `RX`^`RY`. | `Z` `N` | 0.8 |

## 9x - Multiplication
Opcode (Hex) | Mnemonic | Usage | Flags affected | Introduced
--- | --- | --- | --- | ---
90 0X LL HH | MULI RX, HHLL | Set `RX` to `RX`*`HHLL` | `C` `Z` `N` | 1.1 |
91 YX 00 00 | MUL RX, RY | Set `RX` to `RX`*`RY` | `C` `Z` `N` | 0.8 |
92 YX 0Z 00 | MUL RX, RY, RZ | Set `RZ` to `RX`*`RY` | `C` `Z` `N` | 0.8 |

## Ax - Division
Opcode (Hex) | Mnemonic | Usage | Flags affected | Introduced
--- | --- | --- | --- | ---
A0 0X LL HH | DIVI RX, HHLL | Set `RX` to `RX`\\`HHLL` | `C` `Z` `N` | 0.8 |
A1 YX 00 00 | DIV RX, RY | Set `RX` to `RX`\\`RY` | `C` `Z` `N` | 0.8 |
A2 YX 0Z 00 | DIV RX, RY, RZ | Set `RZ` to `RX`\\`RY` | `C` `Z` `N` | 0.8 |
A3 0X LL HH | MODI RX, HHLL  | Set `RX` to `RX` MOD `HHLL` | `Z` `N` | 1.3 |
A4 YX 00 00 | MOD RX, RY    | Set `RX` to `RX` MOD `RY`   | `Z` `N` | 1.3 |
A5 YX 0Z 00 | MOD RX, RY, RZ | Set `RZ` to `RX` MOD `RY` | `Z` `N` | 1.3 |
A6 0X LL HH | REMI RX, HHLL | Set `RX` to `RX` % `HHLL` | `Z` `N` | 1.3 |
A7 YX 00 00 | REM RX, RY | Set `RX` to `RX` % `RY` | `Z` `N` | 1.3 |
A8 YX 0Z 00 | REM RX, RY, RZ | Set `RZ` to `RX` % `RY` | `Z` `N` | 1.3 |

## Bx - Logical/Arithmetic Shifts
Opcode (Hex) | Mnemonic | Usage | Flags affected | Introduced
--- | --- | --- | --- | ---
B0 0X 0N 00 | SHL RX, N | Set `RX` to `RX` << `N` | `Z` `N` | 0.8 |
B1 0X 0N 00 | SHR RX, N | Set `RX` to `RX` >> `N` | `Z` `N` | 0.8 |
B0 0X 0N 00 | SAL RX, N | Set `RX` to `RX` << `N` | `Z` `N` | 0.8 |
B2 0X 0N 00 | SAR RX, N | Set `RX` to `RX` >> `N`, copying leading bit | `Z` `N` | 0.8 |
B3 YX 00 00 | SHL RX, RY | Set `RX` to `RX` << `RY` | `Z` `N` | 0.8 |
B4 YX 00 00 | SHR RX, RY | Set `RX` to `RX` >> `RY` | `Z` `N` | 0.8 |
B3 YX 00 00 | SAL RX, RY | Set `RX` to `RX` << `RY` | `Z` `N` | 0.8 |
B5 YX 00 00 | SAR RX, RY | Set `RX` to `RX` >> `RY`, copying leading bit | `Z` `N` | 0.8 |

Note that a left arithmetic shift is a left logical shift, since we are not expanding the leading bit. 

Hence `SAL` is syntactic sugar, and maps to its corresponding `SHL` opcode.

## Cx - Push/Pop
Opcode (Hex) | Mnemonic | Usage | Flags affected | Introduced
--- | --- | --- | --- | ---
C0 0X 00 00 | PUSH RX | Set `[SP]` to `RX`, increase `SP` by 2 | | 0.8 |
C1 0X 00 00 | POP  RX | Decrease `SP` by 2, set `RX` to `[SP]` | | 0.8 |
C2 00 00 00 | PUSHALL | Store `R0`..`RF` at `[SP]`, increase SP by 32 | | 0.8 |
C3 00 00 00 | POPALL | Decrease `SP` by 32, load `R0`..`RF` from `[SP]` | | 0.8 |
C4 00 00 00 | PUSHF | Set `[SP]` to `FLAGS`, increase `SP` by 2 | | 1.1 |
C5 00 00 00 | POPF | Decrease `SP` by 2, set `FLAGS` to `[SP]` | | 1.1 |

## Dx - Palette
Opcode (Hex) | Mnemonic | Usage | Flags affected | Introduced
--- | --- | --- | --- | ---
D0 00 LL HH | PAL HHLL | Load palette from `[HHLL]` |  | 1.1 |
D1 0X 00 00 | PAL RX | Load palette from `[RX]` |  | 1.1 |

## Ex - Not/Neg
Opcode | Mnemonic | Meaning | Flags affected | Introduced
---|---|---|---|---
E0 0X LL HH | NOTI RX, HHLL | Set `RX` to NOT `HHLL` | `Z` `N` | 1.3 |
E1 0X 00 00 | NOT RX | Set `RX` to NOT `RX` | `Z` `N` | 1.3 |
E2 YX 00 00 | NOT RX, RY | Set `RX` to NOT `RY` | `Z` `N` | 1.3 |
E3 0X LL HH | NEGI RX, HHLL | Set `RX` to NEG `HHLL` | `Z` `N` | 1.3 |
E4 0X 00 00 | NEG RX | Set `RX` to NEG `RX` | `Z` `N` | 1.3 |
E5 YX 00 00 | NEG RX, RY | Set `RX` to NEG `RY` | `Z` `N` | 1.3 |
