// Chip16 virtual machine implementation.
//
// See https://github.com/chip16/chip16/wiki/Machine-Specification for specs.
package vm

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
	"unsafe"
)

//go:generate stringer -type Condition -trimprefix C_

const (
	NumRegs   = 16
	MemSize   = 65536
	StackBase = 0xFDF0
	IOBase    = 0xFFF0
	CycleTime = time.Microsecond
)

const (
	F_Carry    = 1 << 1
	F_Zero     = 1 << 2
	F_Overflow = 1 << 6
	F_Negative = 1 << 7
)

const (
	C_Z Condition = iota
	C_NZ
	C_N
	C_NN
	C_P
	C_O
	C_NO
	C_A
	C_AE
	C_B
	C_BE
	C_G
	C_GE
	C_L
	C_LE
	NumConditions
)

type Condition int

type Error struct {
	Message string
	Stack   string
	Halted  bool
}

func (e Error) Error() string {
	return e.Message
}

// Virtual machine registers, memory etc. If IdleWait is set then VBLNK op will sleep to save CPU load.
type VM struct {
	Machine
	Mem       [MemSize]byte
	PC        uint16
	SP        uint16
	R         [NumRegs]int16
	UR        []uint16
	Carry     bool
	Zero      bool
	Overflow  bool
	Negative  bool
	Op        [4]byte
	CycleTime time.Duration
	Cycles    int
	RNG       rand.Rand
}

// Initialise new VM
func New(m Machine) *VM {
	v := new(VM)
	if m != nil {
		v.Machine = m
	} else {
		v.Machine = NewMachine(true, true)
	}
	v.UR = unsafe.Slice((*uint16)(unsafe.Pointer(&v.R)), NumRegs)
	v.SP = StackBase
	v.CycleTime = CycleTime
	v.RNG = *rand.New(rand.NewSource(time.Now().UnixNano()))
	return v
}

// Current VM state
func (v *VM) String() string {
	if v == nil {
		return "<nil>"
	}
	s := new(strings.Builder)
	for row := range 2 {
		for col := range 8 {
			i := row*8 + col
			fmt.Fprintf(s, "R%X: %04X  ", i, v.UR[i])
		}
		s.WriteByte('\n')
	}
	fmt.Fprintf(s, "SP: %04X  PC: %04X  OP: %08X  Cycles: %d  Flags: C=%d Z=%d O=%d N=%d\n",
		v.SP, v.PC, v.Op, v.Cycles, flag(v.Carry), flag(v.Zero), flag(v.Overflow), flag(v.Negative))
	if v.SP > StackBase {
		fmt.Fprintf(s, "ST: %s\n", strings.Join(v.dumpStack(14), " "))
	}
	fmt.Fprintf(s, "%s", v.Machine)
	return s.String()
}

func (v *VM) dumpStack(n int) (stk []string) {
	for addr := v.SP - 2; addr >= StackBase; addr -= 2 {
		stk = append(stk, fmt.Sprintf("%04X", uint16(v.Load(addr))))
		if int(v.SP-addr) >= 2*n {
			stk = append(stk, "...")
			break
		}
	}
	return
}

// Run until halt instruction. Captures and returns any runtime errors.
func (v *VM) Run() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = Error{
				Message: fmt.Sprintf("runtime error at %04X: %02X %v", v.PC, v.Op, e),
				Stack:   stackTrace(),
			}
		}
	}()
	start := time.Now()
	halt := false
	for !halt {
		halt = v.Step()
		// enforce CPU clock timing
		if v.CycleTime != 0 && v.Cycles%1000 == 0 {
			target := start.Add(time.Duration(v.Cycles) * v.CycleTime)
			for time.Now().Before(target) {
				runtime.Gosched()
			}
		}
	}
	return Error{
		Message: fmt.Sprintf("system halted at %04X: %02X", v.PC, v.Op),
		Halted:  true,
	}
}

// Read instruction at PC, increment PC and execute instruction. Returns true on halt.
func (v *VM) Step() bool {
	v.Cycles++
	copy(v.Op[:], v.Mem[v.PC:v.PC+4])
	v.PC += 4
	x, y, z := v.Op[1]&0xF, v.Op[1]>>4, v.Op[2]&0xF
	imm := uint16(v.Op[2]) + uint16(v.Op[3])<<8
	smm := int16(imm)
	switch v.Op[0] {
	case 0x00: // NOP
	case 0x01: // CLS
		v.ClearScreen()
	case 0x02: // VBLNK
		if !v.VBlank() {
			v.PC -= 4
		}
	case 0x03: // BGC
		v.SetBackground(v.Op[2])
	case 0x04: // SPR HHLL
		v.SetSize(v.Op[2], v.Op[3])
	case 0x05: // DRW RX, RY, HHLL
		v.Carry = v.Draw(v.R[x], v.R[y], &v.Mem[imm])
	case 0x06: // DRW RX, RY, RZ
		v.Carry = v.Draw(v.R[x], v.R[y], &v.Mem[v.UR[z]])
	case 0x07: // RND RX, HHLL
		v.R[x] = int16(v.RNG.Intn(int(imm) + 1))
	case 0x08: // FLIP h, v
		v.SetFlip(v.Op[3]&2 != 0, v.Op[3]&1 != 0)
	case 0x09: // SND0
		v.StopSound()
	case 0x0A: // SND1 HHLL
		v.StartSound(500, smm, false)
	case 0x0B: // SND2 HHLL
		v.StartSound(1000, smm, false)
	case 0x0C: // SND3 HHLL
		v.StartSound(1500, smm, false)
	case 0x0D: // SNP RX, HHLL
		v.StartSound(v.R[x], smm, true)
	case 0x0E: // SNG AD, VTSR
		env := Envelope{Attack: v.Op[1] >> 4, Decay: v.Op[1] & 0xf, Sustain: v.Op[2] >> 4, Release: v.Op[2] & 0xf}
		v.SetSoundParams(v.Op[3]&0xf, v.Op[3]>>4, env)
	case 0x0F: // HALT
		return true

	case 0x10: // JMP HHLL
		v.PC = imm
	case 0x11: // JMC HHLL
		if v.Carry {
			v.PC = imm
		}
	case 0x12: // JMx HHLL
		if v.cond(x) {
			v.PC = imm
		}
	case 0x13: // JME RX, RY, HHLL
		if v.R[x] == v.R[y] {
			v.PC = imm
		}
	case 0x14: // CALL HHLL
		v.push(int16(v.PC))
		v.PC = imm
	case 0x15: // RET
		v.PC = uint16(v.pop())
	case 0x16: // JMP RX
		v.PC = uint16(v.R[x])
	case 0x17: // Cx HHLL
		if v.cond(x) {
			v.push(int16(v.PC))
			v.PC = imm
		}
	case 0x18: // CALL RX
		v.push(int16(v.PC))
		v.PC = v.UR[x]

	case 0x20: // LDI RX, HHLL
		v.R[x] = smm
	case 0x21: // LDI SP, HHLL
		v.SP = imm
	case 0x22: // LDM RX, HHLL
		v.R[x] = v.Load(imm)
	case 0x23: // LDM RX, RY
		v.R[x] = v.Load(v.UR[y])
	case 0x24: // MOV RX, RY
		v.R[x] = v.R[y]

	case 0x30: // STM RX, HHLL
		v.Store(imm, v.R[x])
	case 0x31: // STM RX, RY
		v.Store(v.UR[y], v.R[x])

	case 0x40: // ADDI RX, HHLL
		v.R[x] = v.add(v.R[x], smm)
	case 0x41: // ADD RX, RY
		v.R[x] = v.add(v.R[x], v.R[y])
	case 0x42: // ADD RX, RY, RZ
		v.R[z] = v.add(v.R[x], v.R[y])

	case 0x50: // SUBI RX, HHLL
		v.R[x] = v.sub(v.R[x], smm)
	case 0x51: // SUB RX, RY
		v.R[x] = v.sub(v.R[x], v.R[y])
	case 0x52: // SUB RX, RY, RZ
		v.R[z] = v.sub(v.R[x], v.R[y])
	case 0x53: // CMPI RX, HHLL
		v.sub(v.R[x], smm)
	case 0x54: // CMP RX, RY
		v.sub(v.R[x], v.R[y])

	case 0x60: // ANDI RX, HHLL
		v.R[x] = v.setnz(v.R[x] & smm)
	case 0x61: // AND RX, RY
		v.R[x] = v.setnz(v.R[x] & v.R[y])
	case 0x62: // AND RX, RY, RZ
		v.R[z] = v.setnz(v.R[x] & v.R[y])
	case 0x63: // TSTI RX, HHLL
		v.setnz(v.R[x] & smm)
	case 0x64: // TST RX, RY
		v.setnz(v.R[x] & v.R[y])

	case 0x70: // ORI RX, HHLL
		v.R[x] = v.setnz(v.R[x] | smm)
	case 0x71: // OR RX, RY
		v.R[x] = v.setnz(v.R[x] | v.R[y])
	case 0x72: // OR RX, RY, RZ
		v.R[z] = v.setnz(v.R[x] | v.R[y])

	case 0x80: // XORI RX, HHLL
		v.R[x] = v.setnz(v.R[x] ^ smm)
	case 0x81: // XOR RX, RY
		v.R[x] = v.setnz(v.R[x] ^ v.R[y])
	case 0x82: // XOR RX, RY, RZ
		v.R[z] = v.setnz(v.R[x] ^ v.R[y])

	case 0x90: // MULI RX, HHLL
		v.R[x] = v.mul(v.R[x], smm)
	case 0x91: // MUL RX, RY
		v.R[x] = v.mul(v.R[x], v.R[y])
	case 0x92: // MUL RX, RY, RZ
		v.R[z] = v.mul(v.R[x], v.R[y])

	case 0xA0: // DIVI RX, HHLL
		v.R[x] = v.div(v.R[x], smm)
	case 0xA1: // DIV RX, RY
		v.R[x] = v.div(v.R[x], v.R[y])
	case 0xA2: // DIV RX, RY, RZ
		v.R[z] = v.div(v.R[x], v.R[y])

	case 0xA3: // MODI RX, HHLL
		v.R[x] = v.setnz(mod(v.R[x], smm))
	case 0xA4: // MOD RX, RY
		v.R[x] = v.setnz(mod(v.R[x], v.R[y]))
	case 0xA5: // MOD RX, RY, RZ
		v.R[z] = v.setnz(mod(v.R[x], v.R[y]))
	case 0xA6: // REMI RX, HHLL
		v.R[x] = v.setnz(v.R[x] % smm)
	case 0xA7: // REM RX, RY
		v.R[x] = v.setnz(v.R[x] % v.R[y])
	case 0xA8: // REM RX, RY, RZ
		v.R[z] = v.setnz(v.R[x] % v.R[y])

	case 0xB0: // SHL RX, N
		v.R[x] = v.setnz(v.R[x] << (imm & 0xf))
	case 0xB1: // SHR RX, N
		v.R[x] = v.setnz(int16(v.UR[x] >> (imm & 0xf)))
	case 0xB2: // SAR RX, N
		v.R[x] = v.setnz(v.R[x] >> (imm & 0xf))
	case 0xB3: // SHL RX, RY
		v.R[x] = v.setnz(v.R[x] << v.R[y])
	case 0xB4: // SHR RX, RY
		v.R[x] = v.setnz(int16(v.UR[x] >> v.R[y]))
	case 0xB5: // SAR RX, RY
		v.R[x] = v.setnz(v.R[x] >> v.R[y])

	case 0xC0: // PUSH RX
		v.push(v.R[x])
	case 0xC1: // POP RX
		v.R[x] = v.pop()
	case 0xC2: // PUSHALL
		for i := 0; i < NumRegs; i++ {
			v.push(v.R[i])
		}
	case 0xC3: // POPALL
		for i := NumRegs - 1; i >= 0; i-- {
			v.R[i] = v.pop()
		}
	case 0xC4: // PUSHF
		v.push(int16(v.getFlags()))
	case 0xC5: // POPF
		v.setFlags(uint8(v.pop()))
	case 0xD0: // PAL HHLL
		v.LoadPalette(&v.Mem[imm])
	case 0xD1: // PAL RX
		v.LoadPalette(&v.Mem[v.UR[x]])

	case 0xE0: // NOTI RX, HHLL
		v.R[x] = v.setnz(^smm)
	case 0xE1: // NOT RX
		v.R[x] = v.setnz(^v.R[x])
	case 0xE2: // NOT RX, RY
		v.R[x] = v.setnz(^v.R[y])
	case 0xE3: // NEGI RX, HHLL
		v.R[x] = v.setnz(-smm)
	case 0xE4: // NEG RX
		v.R[x] = v.setnz(-v.R[x])
	case 0xE5: // NEG RX, RY
		v.R[x] = v.setnz(-v.R[y])

	default:
		panic("invalid opcode")
	}
	return false
}

func (v *VM) getFlags() uint8 {
	return uint8(flag(v.Carry)<<1 | flag(v.Zero)<<2 | flag(v.Overflow)<<6 | flag(v.Negative)<<7)
}

func (v *VM) setFlags(f uint8) {
	v.Carry, v.Zero, v.Overflow, v.Negative = f&F_Carry != 0, f&F_Zero != 0, f&F_Overflow != 0, f&F_Negative != 0
}

func (v *VM) push(x int16) {
	v.Store(v.SP, x)
	v.SP += 2
}

func (v *VM) pop() int16 {
	v.SP -= 2
	return v.Load(v.SP)
}

func (v *VM) Store(addr uint16, x int16) {
	binary.LittleEndian.PutUint16(v.Mem[addr:], uint16(x))
}

func (v *VM) Load(addr uint16) int16 {
	return int16(binary.LittleEndian.Uint16(v.Mem[addr:]))
}

func (v *VM) cond(typ uint8) (ok bool) {
	switch typ {
	case 0:
		ok = v.Zero
	case 1:
		ok = !v.Zero
	case 2:
		ok = v.Negative
	case 3:
		ok = !v.Negative
	case 4:
		ok = !v.Negative && !v.Zero
	case 5:
		ok = v.Overflow
	case 6:
		ok = !v.Overflow
	case 7:
		ok = !v.Carry && !v.Zero
	case 8:
		ok = !v.Carry
	case 9:
		ok = v.Carry
	case 10:
		ok = v.Carry || v.Zero
	case 11:
		ok = v.Overflow == v.Negative && !v.Zero
	case 12:
		ok = v.Overflow == v.Negative
	case 13:
		ok = v.Overflow != v.Negative
	case 14:
		ok = v.Overflow != v.Negative || v.Zero
	}
	return
}

func (v *VM) setnz(r int16) int16 {
	v.Zero = r == 0
	v.Negative = r < 0
	return r
}

func (v *VM) add(x, y int16) int16 {
	r := x + y
	ux, uy := ucast(x), ucast(y)
	v.Carry = (uint32(ux)+uint32(uy))&0x10000 != 0
	v.Overflow = r >= 0 && x < 0 && y < 0 || r < 0 && x >= 0 && y >= 0
	return v.setnz(r)
}

func (v *VM) sub(x, y int16) int16 {
	r := x - y
	ux, uy := ucast(x), ucast(y)
	v.Carry = (uint32(ux)-uint32(uy))&0x10000 != 0
	v.Overflow = r >= 0 && x < 0 && y >= 0 || r < 0 && x >= 0 && y < 0
	return v.setnz(r)
}

func (v *VM) mul(x, y int16) int16 {
	ux, uy := ucast(x), ucast(y)
	v.Carry = (uint32(ux)+uint32(uy))&0x10000 != 0
	return v.setnz(x * y)
}

func (v *VM) div(x, y int16) int16 {
	v.Carry = x%y != 0
	return v.setnz(x / y)
}

func mod(x, y int16) int16 {
	return (x%y + y) % y
}

func flag(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

func ucast(x int16) uint16 {
	return *(*uint16)(unsafe.Pointer(&x))
}

func stackTrace() string {
	trace := string(debug.Stack())
	lines := strings.Split(trace, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "panic(") {
			return strings.Join(lines[i:], "\n")
		}
	}
	return trace
}
