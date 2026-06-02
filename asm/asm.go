// Chip16 assembler used by the gas16 command.
//
// Opcode syntax is as per https://github.com/chip16/chip16/wiki/Instructions.
//
// Additional assembler directives are:
//
//	:label				- define label at current address
//	name EQU Value			- define constant
//	DB vals...			- store bytes
//	DB "string"			- store bytes from string
//	DW vals...			- store 16 bit words
//	IMPORTBIN file,off,len,label	- append file contents from off:off+len to end of binary
package asm

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/jnb666/chip16/vm"
)

type Assembler struct {
	Code       []byte
	Constants  map[string]string
	Labels     map[string]int
	addr2Label map[int]string
	commands   []command
	imports    []importbin
	op         string
	err        error
}

type command struct {
	line int
	op   string
	args []string
	size int
}

type importbin struct {
	label string
	data  []byte
}

func New() *Assembler {
	return &Assembler{
		Constants:  map[string]string{},
		Labels:     map[string]int{},
		addr2Label: map[int]string{},
	}
}

func (a *Assembler) Error() error {
	return a.err
}

func (a *Assembler) Instruction(op string, args []string) error {
	switch op {
	case "JZ", "JNZ", "JN", "JNN", "JP", "JO", "JNO", "JA", "JAE", "JB", "JBE", "JG", "JGE", "JL", "JLE":
		a.Jx(op[1:], args)
	case "CZ", "CNZ", "CN", "CNN", "CP", "CO", "CNO", "CA", "CAE", "CB", "CBE", "CG", "CGE", "CL", "CLE":
		a.Cx(op[1:], args)
	default:
		a.callMethod(op, args)
	}
	return a.err
}

func (a *Assembler) DB(args []string) {
	if code := a.parseDB(args); a.err == nil {
		a.Code = append(a.Code, code...)
	}
}

func (a *Assembler) DW(args []string) {
	if code := a.parseDW(args); a.err == nil {
		a.Code = append(a.Code, code...)
	}
}

func (a *Assembler) NOP(args []string) {
	if a.nargs("NOP", args, 0) {
		a.emit(0x00)
	}
}

func (a *Assembler) CLS(args []string) {
	if a.nargs("CLS", args, 0) {
		a.emit(0x01)
	}
}

func (a *Assembler) VBLNK(args []string) {
	if a.nargs("VBLNK", args, 0) {
		a.emit(0x02)
	}
}

func (a *Assembler) BGC(args []string) {
	if a.nargs("BGC", args, 1) {
		a.emit(0x03 | rz(a.index(args[0])))
	}
}

func (a *Assembler) SPR(args []string) {
	if a.nargs("SPR", args, 1) {
		a.emit(0x04 | imm(a.num(args[0])))
	}
}

func (a *Assembler) DRW(args []string) {
	if a.nargs("DRW", args, 3) {
		x, y := a.reg(args[0]), a.reg(args[1])
		if isReg(args[2]) {
			a.emit(0x06 | rx(x) | ry(y) | rz(a.reg(args[2])))
		} else {
			a.emit(0x05 | rx(x) | ry(y) | imm(a.num(args[2])))
		}
	}
}

func (a *Assembler) RND(args []string) {
	if a.nargs("RND", args, 2) {
		a.emit(0x07 | rx(a.reg(args[0])) | imm(a.num(args[1])))
	}
}

func (a *Assembler) FLIP(args []string) {
	if a.nargs("FLIP", args, 2) {
		a.emit(0x08 | uint32((a.index(args[0])<<1)|a.index(args[1]))<<24)
	}
}

func (a *Assembler) SND0(args []string) {
	if a.nargs("SND0", args, 0) {
		a.emit(0x09)
	}
}

func (a *Assembler) SND1(args []string) {
	if a.nargs("SND1", args, 1) {
		a.emit(0x0A | imm(a.num(args[0])))
	}
}

func (a *Assembler) SND2(args []string) {
	if a.nargs("SND2", args, 1) {
		a.emit(0x0B | imm(a.num(args[0])))
	}
}

func (a *Assembler) SND3(args []string) {
	if a.nargs("SND3", args, 1) {
		a.emit(0x0C | imm(a.num(args[0])))
	}
}

func (a *Assembler) SNP(args []string) {
	if a.nargs("SNP", args, 2) {
		a.emit(0x0D | rx(a.reg(args[0])) | imm(a.num(args[1])))
	}
}

func (a *Assembler) SNG(args []string) {
	if a.nargs("SNG", args, 2) {
		a.emit(0x0E | uint32(a.byte(args[0]))<<8 | imm(a.num(args[1])))
	}
}

func (a *Assembler) HALT(args []string) {
	if a.nargs("HALT", args, 0) {
		a.emit(0x0f)
	}
}

func (a *Assembler) JMP(args []string) {
	if a.nargs("JMP", args, 1) {
		if isReg(args[0]) {
			a.emit(0x16 | rx(a.reg(args[0])))
		} else {
			a.emit(0x10 | imm(a.num(args[0])))
		}
	}
}

func (a *Assembler) JMC(args []string) {
	if a.nargs("JMC", args, 1) {
		a.emit(0x11 | imm(a.num(args[0])))
	}
}

func (a *Assembler) Jx(code string, args []string) {
	if a.nargs("Jx", args, 1) {
		a.emit(0x12 | rx(a.cond(code)) | imm(a.num(args[0])))
	}
}

func (a *Assembler) JME(args []string) {
	if a.nargs("JME", args, 3) {
		a.emit(0x13 | rx(a.reg(args[0])) | ry(a.reg(args[1])) | imm(a.num(args[2])))
	}
}

func (a *Assembler) CALL(args []string) {
	if a.nargs("CALL", args, 1) {
		if isReg(args[0]) {
			a.emit(0x18 | rx(a.reg(args[0])))
		} else {
			a.emit(0x14 | imm(a.num(args[0])))
		}
	}
}

func (a *Assembler) RET(args []string) {
	if a.nargs("RET", args, 0) {
		a.emit(0x15)
	}
}

func (a *Assembler) Cx(code string, args []string) {
	if a.nargs("CALLx", args, 1) {
		a.emit(0x17 | rx(a.cond(code)) | imm(a.num(args[0])))
	}
}

func (a *Assembler) LDI(args []string) {
	if a.nargs("LDI", args, 2) {
		i := a.num(args[1])
		if args[0] == "SP" {
			a.emit(0x21 | imm(i))
		} else {
			a.emit(0x20 | rx(a.reg(args[0])) | imm(i))
		}
	}
}

func (a *Assembler) LDM(args []string) {
	if a.nargs("LDM", args, 2) {
		x := a.reg(args[0])
		if isReg(args[1]) {
			a.emit(0x23 | rx(x) | ry(a.reg(args[1])))
		} else {
			a.emit(0x22 | rx(x) | imm(a.num(args[1])))
		}
	}
}

func (a *Assembler) MOV(args []string) {
	if a.nargs("MOV", args, 2) {
		a.emit(0x24 | rx(a.reg(args[0])) | ry(a.reg(args[1])))
	}
}

func (a *Assembler) STM(args []string) {
	if a.nargs("STM", args, 2) {
		x := a.reg(args[0])
		if isReg(args[1]) {
			a.emit(0x31 | rx(x) | ry(a.reg(args[1])))
		} else {
			a.emit(0x30 | rx(x) | imm(a.num(args[1])))
		}
	}
}

func (a *Assembler) ADDI(args []string) {
	if a.nargs("ADDI", args, 2) {
		a.emit(0x40 | rx(a.reg(args[0])) | imm(a.num(args[1])))
	}
}

func (a *Assembler) ADD(args []string) {
	if a.nargs("ADD", args, 2, 3) {
		x, y := a.reg(args[0]), a.reg(args[1])
		if len(args) == 2 {
			a.emit(0x41 | rx(x) | ry(y))
		} else {
			a.emit(0x42 | rx(x) | ry(y) | rz(a.reg(args[2])))
		}
	}
}

func (a *Assembler) SUBI(args []string) {
	if a.nargs("SUBI", args, 2) {
		a.emit(0x50 | rx(a.reg(args[0])) | imm(a.num(args[1])))
	}
}

func (a *Assembler) SUB(args []string) {
	if a.nargs("SUB", args, 2, 3) {
		x, y := a.reg(args[0]), a.reg(args[1])
		if len(args) == 2 {
			a.emit(0x51 | rx(x) | ry(y))
		} else {
			a.emit(0x52 | rx(x) | ry(y) | rz(a.reg(args[2])))
		}
	}
}

func (a *Assembler) CMPI(args []string) {
	if a.nargs("CMPI", args, 2) {
		a.emit(0x53 | rx(a.reg(args[0])) | imm(a.num(args[1])))
	}
}

func (a *Assembler) CMP(args []string) {
	if a.nargs("CMP", args, 2) {
		a.emit(0x54 | rx(a.reg(args[0])) | ry(a.reg(args[1])))
	}
}

func (a *Assembler) ANDI(args []string) {
	if a.nargs("ANDI", args, 2) {
		a.emit(0x60 | rx(a.reg(args[0])) | imm(a.num(args[1])))
	}
}

func (a *Assembler) AND(args []string) {
	if a.nargs("AND", args, 2, 3) {
		x, y := a.reg(args[0]), a.reg(args[1])
		if len(args) == 2 {
			a.emit(0x61 | rx(x) | ry(y))
		} else {
			a.emit(0x62 | rx(x) | ry(y) | rz(a.reg(args[2])))
		}
	}
}

func (a *Assembler) TSTI(args []string) {
	if a.nargs("TSTI", args, 2) {
		a.emit(0x63 | rx(a.reg(args[0])) | imm(a.num(args[1])))
	}
}

func (a *Assembler) TST(args []string) {
	if a.nargs("TST", args, 2) {
		a.emit(0x64 | rx(a.reg(args[0])) | ry(a.reg(args[1])))
	}
}

func (a *Assembler) ORI(args []string) {
	if a.nargs("ORI", args, 2) {
		a.emit(0x70 | rx(a.reg(args[0])) | imm(a.num(args[1])))
	}
}

func (a *Assembler) OR(args []string) {
	if a.nargs("OR", args, 2, 3) {
		x, y := a.reg(args[0]), a.reg(args[1])
		if len(args) == 2 {
			a.emit(0x71 | rx(x) | ry(y))
		} else {
			a.emit(0x72 | rx(x) | ry(y) | rz(a.reg(args[2])))
		}
	}
}

func (a *Assembler) XORI(args []string) {
	if a.nargs("XORI", args, 2) {
		a.emit(0x80 | rx(a.reg(args[0])) | imm(a.num(args[1])))
	}
}

func (a *Assembler) XOR(args []string) {
	if a.nargs("XOR", args, 2, 3) {
		x, y := a.reg(args[0]), a.reg(args[1])
		if len(args) == 2 {
			a.emit(0x81 | rx(x) | ry(y))
		} else {
			a.emit(0x82 | rx(x) | ry(y) | rz(a.reg(args[2])))
		}
	}
}

func (a *Assembler) MULI(args []string) {
	if a.nargs("MULI", args, 2) {
		a.emit(0x90 | rx(a.reg(args[0])) | imm(a.num(args[1])))
	}
}

func (a *Assembler) MUL(args []string) {
	if a.nargs("MUL", args, 2, 3) {
		x, y := a.reg(args[0]), a.reg(args[1])
		if len(args) == 2 {
			a.emit(0x91 | rx(x) | ry(y))
		} else {
			z := a.reg(args[2])
			a.emit(0x92 | rx(x) | ry(y) | rz(z))
		}
	}
}

func (a *Assembler) DIVI(args []string) {
	if a.nargs("DIVI", args, 2) {
		a.emit(0xA0 | rx(a.reg(args[0])) | imm(a.num(args[1])))
	}
}

func (a *Assembler) DIV(args []string) {
	if a.nargs("DIV", args, 2, 3) {
		x, y := a.reg(args[0]), a.reg(args[1])
		if len(args) == 2 {
			a.emit(0xA1 | rx(x) | ry(y))
		} else {
			z := a.reg(args[2])
			a.emit(0xA2 | rx(x) | ry(y) | rz(z))
		}
	}
}

func (a *Assembler) MODI(args []string) {
	if a.nargs("MODI", args, 2) {
		a.emit(0xA3 | rx(a.reg(args[0])) | imm(a.num(args[1])))
	}
}

func (a *Assembler) MOD(args []string) {
	if a.nargs("MOD", args, 2, 3) {
		x, y := a.reg(args[0]), a.reg(args[1])
		if len(args) == 2 {
			a.emit(0xA4 | rx(x) | ry(y))
		} else {
			z := a.reg(args[2])
			a.emit(0xA5 | rx(x) | ry(y) | rz(z))
		}
	}
}

func (a *Assembler) REMI(args []string) {
	if a.nargs("REMI", args, 2) {
		a.emit(0xA6 | rx(a.reg(args[0])) | imm(a.num(args[1])))
	}
}

func (a *Assembler) REM(args []string) {
	if a.nargs("REM", args, 2, 3) {
		x, y := a.reg(args[0]), a.reg(args[1])
		if len(args) == 2 {
			a.emit(0xA7 | rx(x) | ry(y))
		} else {
			z := a.reg(args[2])
			a.emit(0xA8 | rx(x) | ry(y) | rz(z))
		}
	}
}

func (a *Assembler) SHL(args []string) {
	if a.nargs("SHL", args, 2) {
		x := a.reg(args[0])
		if isReg(args[1]) {
			a.emit(0xB3 | rx(x) | ry(a.reg(args[1])))
		} else {
			a.emit(0xB0 | rx(x) | rz(a.index(args[1])))
		}
	}
}

func (a *Assembler) SHR(args []string) {
	if a.nargs("SHR", args, 2) {
		x := a.reg(args[0])
		if isReg(args[1]) {
			a.emit(0xB4 | rx(x) | ry(a.reg(args[1])))
		} else {
			a.emit(0xB1 | rx(x) | rz(a.index(args[1])))
		}
	}
}

func (a *Assembler) SAR(args []string) {
	if a.nargs("SAR", args, 2) {
		x := a.reg(args[0])
		if isReg(args[1]) {
			a.emit(0xB5 | rx(x) | ry(a.reg(args[1])))
		} else {
			a.emit(0xB2 | rx(x) | rz(a.index(args[1])))
		}
	}
}

func (a *Assembler) PUSH(args []string) {
	if a.nargs("PUSH", args, 1) {
		a.emit(0xC0 | rx(a.reg(args[0])))
	}
}

func (a *Assembler) POP(args []string) {
	if a.nargs("POP", args, 1) {
		a.emit(0xC1 | rx(a.reg(args[0])))
	}
}

func (a *Assembler) PUSHALL(args []string) {
	if a.nargs("PUSHALL", args, 0) {
		a.emit(0xC2)
	}
}

func (a *Assembler) POPALL(args []string) {
	if a.nargs("POPALL", args, 0) {
		a.emit(0xC3)
	}
}

func (a *Assembler) PUSHF(args []string) {
	if a.nargs("PUSHF", args, 0) {
		a.emit(0xC4)
	}
}

func (a *Assembler) POPF(args []string) {
	if a.nargs("POPF", args, 0) {
		a.emit(0xC5)
	}
}

func (a *Assembler) PAL(args []string) {
	if a.nargs("PAL", args, 1) {
		if isReg(args[0]) {
			a.emit(0xD1 | rx(a.reg(args[0])))
		} else {
			a.emit(0xD0 | imm(a.num(args[0])))
		}
	}
}

func (a *Assembler) NOTI(args []string) {
	if a.nargs("NOTI", args, 2) {
		a.emit(0xE0 | rx(a.reg(args[0])) | imm(a.num(args[1])))
	}
}

func (a *Assembler) NOT(args []string) {
	if a.nargs("NOT", args, 1, 2) {
		x := a.reg(args[0])
		if len(args) == 1 {
			a.emit(0xE1 | rx(x))
		} else {
			a.emit(0xE2 | rx(x) | ry(a.reg(args[1])))
		}
	}
}

func (a *Assembler) NEGI(args []string) {
	if a.nargs("NEGI", args, 2) {
		a.emit(0xE3 | rx(a.reg(args[0])) | imm(a.num(args[1])))
	}
}

func (a *Assembler) NEG(args []string) {
	if a.nargs("NEG", args, 1, 2) {
		x := a.reg(args[0])
		if len(args) == 1 {
			a.emit(0xE4 | rx(x))
		} else {
			a.emit(0xE5 | rx(x) | ry(a.reg(args[1])))
		}
	}
}

func (a *Assembler) callMethod(name string, args []string) {
	var zero reflect.Value
	fn := reflect.ValueOf(a).MethodByName(name)
	typ := fn.Type()
	if fn == zero || typ.Kind() != reflect.Func || typ.NumIn() != 1 || typ.In(0) != reflect.TypeFor[[]string]() {
		a.err = fmt.Errorf("error: opcode %s not defined", name)
		return
	}
	fn.Call([]reflect.Value{reflect.ValueOf(args)})
}

func (a *Assembler) emit(op uint32) {
	a.Code = binary.LittleEndian.AppendUint32(a.Code, op)
}

func (a *Assembler) nargs(op string, args []string, n ...int) bool {
	if a.err == nil {
		a.op = op
		if slices.Contains(n, len(args)) {
			return true
		}
		a.err = fmt.Errorf("%s expects %s arguments", op, join(n))
	}
	return false
}

func (a *Assembler) reg(name string) int {
	if a.err != nil {
		return 0
	}
	if isReg(name) {
		if i, err := atoi(name[1:], 16); err == nil {
			return int(i)
		}
	}
	a.err = fmt.Errorf("%s: invalid register %s", a.op, name)
	return 0
}

func (a *Assembler) index(s string) int {
	if a.err != nil {
		return 0
	}
	if val, ok := a.Constants[s]; ok {
		s = val
	}
	if i, err := atoi(getBase(s)); err == nil && i >= 0 && i < 16 {
		return i
	}
	a.err = fmt.Errorf("%s: invalid index %s", a.op, s)
	return 0
}

func (a *Assembler) num(s string) uint16 {
	if a.err != nil {
		return 0
	}
	if addr, ok := a.Labels[s]; ok {
		return uint16(addr)
	}
	if val, ok := a.Constants[s]; ok {
		s = val
	}
	if i, err := atoi(getBase(s)); err == nil && i >= -32768 && i < 65536 {
		return uint16(i)
	}
	a.err = fmt.Errorf("%s: invalid immediate value %s", a.op, s)
	return 0
}

func (a *Assembler) byte(s string) uint8 {
	if a.err != nil {
		return 0
	}
	if val, ok := a.Constants[s]; ok {
		s = val
	}
	if i, err := atoi(getBase(s)); err == nil && i >= -128 && i < 256 {
		return uint8(i)
	}
	a.err = fmt.Errorf("%s: invalid byte constant %s", a.op, s)
	return 0
}

func (a *Assembler) cond(code string) int {
	if a.err != nil {
		return 0
	}
	code = strings.ToUpper(code)
	for i := range vm.NumConditions {
		if i.String() == code {
			return int(i)
		}
	}
	a.err = fmt.Errorf("%s: invalid condition %s", a.op, code)
	return 0
}

func rx(x int) uint32 {
	return uint32(x&0xf) << 8
}

func ry(y int) uint32 {
	return uint32(y&0xf) << 12
}

func rz(z int) uint32 {
	return uint32(z&0xf) << 16
}

func imm(x uint16) uint32 {
	return uint32(x) << 16
}

func isReg(name string) bool {
	return len(name) == 2 && (name[0] == 'R' || name[0] == 'r') && isHexDigit(name[1])
}

func isHexDigit(r byte) bool {
	return r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F'
}

func atoi(s string, base int) (int, error) {
	i, err := strconv.ParseInt(s, base, 32)
	return int(i), err
}

func getBase(s string) (string, int) {
	if strings.HasPrefix(s, "#") {
		return s[1:], 16
	}
	if strings.HasSuffix(s, "h") {
		return s[:len(s)-1], 16
	}
	if strings.HasPrefix(s, "0x") {
		return s[2:], 16
	}
	return s, 10
}

func join(nums []int) string {
	if len(nums) == 1 {
		return strconv.Itoa(nums[0])
	}
	if len(nums) == 2 {
		return fmt.Sprintf("%d or %d", nums[0], nums[1])
	}
	return "no"
}
