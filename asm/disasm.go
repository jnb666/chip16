package asm

import (
	"fmt"

	"github.com/jnb666/chip16/vm"
)

type Opcode [4]uint8

// Opcode disassembly. Returns empty string if not valid.
func (op Opcode) String() string {
	x, y, z := op[1]&0xF, op[1]>>4, op[2]&0xF
	imm := uint16(op[2]) + uint16(op[3])<<8
	smm := int16(imm)
	switch op[0] {
	case 0x00:
		if op[1] == 0 && imm == 0 {
			return "NOP"
		}
	case 0x01:
		if op[1] == 0 && imm == 0 {
			return "CLS"
		}
	case 0x02:
		if op[1] == 0 && imm == 0 {
			return "VBLNK"
		}
	case 0x03:
		if op[1] == 0 && op[3] == 0 && op[2] < 16 {
			return fmt.Sprintf("BGC %d", op[2])
		}
	case 0x04:
		if op[1] == 0 {
			return fmt.Sprintf("SPR #%04X", imm)
		}
	case 0x05:
		return fmt.Sprintf("DRW R%X, R%X, #%04X", x, y, imm)
	case 0x06:
		if op[3] == 0 && op[2] < 16 {
			return fmt.Sprintf("DRW R%X, R%X, R%X", x, y, z)
		}
	case 0x07:
		if op[1] < 16 {
			return fmt.Sprintf("RND R%X, %d", x, smm)
		}
	case 0x08:
		if op[1] == 0 && op[2] == 0 && op[3] < 4 {
			return fmt.Sprintf("FLIP %d, %d", (op[3]&2)>>1, op[3]&1)
		}
	case 0x09:
		if op[1] == 0 && imm == 0 {
			return "SND0"
		}
	case 0x0A:
		if op[1] == 0 {
			return fmt.Sprintf("SND1 #%04X", imm)
		}
	case 0x0B:
		if op[1] == 0 {
			return fmt.Sprintf("SND2 #%04X", imm)
		}
	case 0x0C:
		if op[1] == 0 {
			return fmt.Sprintf("SND3 #%04X", imm)
		}
	case 0x0D:
		if op[1] < 16 {
			return fmt.Sprintf("SNP R%X, #%04X", x, imm)
		}
	case 0x0E:
		return fmt.Sprintf("SNG #%02X, #%04X", op[1], imm)
	case 0x0F:
		if op[1] == 0 && imm == 0 {
			return "HALT"
		}

	case 0x10:
		if op[1] == 0 {
			return fmt.Sprintf("JMP #%04X", imm)
		}
	case 0x11:
		if op[1] == 0 {
			return fmt.Sprintf("JMC #%04X", imm)
		}
	case 0x12:
		if op[1] < 16 {
			return fmt.Sprintf("J%s #%04X", vm.Condition(op[1]), imm)
		}
	case 0x13:
		return fmt.Sprintf("JME R%X, R%X, #%04X", x, y, imm)
	case 0x14:
		if op[1] == 0 {
			return fmt.Sprintf("CALL #%04X", imm)
		}
	case 0x15:
		if op[1] == 0 && imm == 0 {
			return "RET"
		}
	case 0x16:
		if op[2] == 0 && op[3] == 0 && op[1] < 16 {
			return fmt.Sprintf("JMP R%X", x)
		}
	case 0x17:
		if op[1] < 16 {
			return fmt.Sprintf("C%s #%04X", vm.Condition(op[1]), imm)
		}
	case 0x18:
		if op[2] == 0 && op[3] == 0 && op[1] < 16 {
			return fmt.Sprintf("CALL R%X", x)
		}

	case 0x20:
		if op[1] < 16 {
			return fmt.Sprintf("LDI R%X, %d", x, smm)
		}
	case 0x21:
		if op[1] == 0 {
			return fmt.Sprintf("LDI SP, #%04X", imm)
		}
	case 0x22:
		if op[1] < 16 {
			return fmt.Sprintf("LDM R%X, #%04X", x, imm)
		}
	case 0x23:
		if imm == 0 {
			return fmt.Sprintf("LDM R%X, R%X", x, y)
		}
	case 0x24:
		if imm == 0 {
			return fmt.Sprintf("MOV R%X, R%X", x, y)
		}

	case 0x30:
		if op[1] < 16 {
			return fmt.Sprintf("STM R%X, #%04X", x, imm)
		}
	case 0x31:
		if imm == 0 {
			return fmt.Sprintf("STM R%X, R%X", x, y)
		}

	case 0x40:
		if op[1] < 16 {
			return fmt.Sprintf("ADDI R%X, %d", x, smm)
		}
	case 0x41:
		if imm == 0 {
			return fmt.Sprintf("ADD R%X, R%X", x, y)
		}
	case 0x42:
		if imm < 16 {
			return fmt.Sprintf("ADD R%X, R%X, R%X", x, y, z)
		}

	case 0x50:
		if op[1] < 16 {
			return fmt.Sprintf("SUBI R%X, %d", x, smm)
		}
	case 0x51:
		if imm == 0 {
			return fmt.Sprintf("SUB R%X, R%X", x, y)
		}
	case 0x52:
		if imm < 16 {
			return fmt.Sprintf("SUB R%X, R%X, R%X", x, y, z)
		}
	case 0x53:
		if op[1] < 16 {
			return fmt.Sprintf("CMPI R%X, %d", x, smm)
		}
	case 0x54:
		if imm == 0 {
			return fmt.Sprintf("CMP R%X, R%X", x, y)
		}

	case 0x60:
		if op[1] < 16 {
			return fmt.Sprintf("ANDI R%X, #%04X", x, imm)
		}
	case 0x61:
		if imm == 0 {
			return fmt.Sprintf("AND R%X, R%X", x, y)
		}
	case 0x62:
		if imm < 16 {
			return fmt.Sprintf("AND R%X, R%X, R%X", x, y, z)
		}
	case 0x63:
		if op[1] < 16 {
			return fmt.Sprintf("TSTI R%X, #%04X", x, imm)
		}
	case 0x64:
		if imm == 0 {
			return fmt.Sprintf("TST R%X, R%X", x, y)
		}

	case 0x70:
		if op[1] < 16 {
			return fmt.Sprintf("ORI R%X, #%04X", x, imm)
		}
	case 0x71:
		if imm == 0 {
			return fmt.Sprintf("OR R%X, R%X", x, y)
		}
	case 0x72:
		if imm < 16 {
			return fmt.Sprintf("OR R%X, R%X, R%X", x, y, z)
		}

	case 0x80:
		if op[1] < 16 {
			return fmt.Sprintf("XORI R%X, #%04X", x, imm)
		}
	case 0x81:
		if imm == 0 {
			return fmt.Sprintf("XOR R%X, R%X", x, y)
		}
	case 0x82:
		if imm < 16 {
			return fmt.Sprintf("XOR R%X, R%X, R%X", x, y, z)
		}

	case 0x90:
		if op[1] < 16 {
			return fmt.Sprintf("MULI R%X, %d", x, smm)
		}
	case 0x91:
		if imm == 0 {
			return fmt.Sprintf("MUL R%X, R%X", x, y)
		}
	case 0x92:
		if imm < 16 {
			return fmt.Sprintf("MUL R%X, R%X, R%X", x, y, z)
		}

	case 0xA0:
		if op[1] < 16 {
			return fmt.Sprintf("DIVI R%X, %d", x, smm)
		}
	case 0xA1:
		if imm == 0 {
			return fmt.Sprintf("DIV R%X, R%X", x, y)
		}
	case 0xA2:
		if imm < 16 {
			return fmt.Sprintf("DIV R%X, R%X, R%X", x, y, z)
		}
	case 0xA3:
		if op[1] < 16 {
			return fmt.Sprintf("MODI R%X, %d", x, smm)
		}
	case 0xA4:
		if imm == 0 {
			return fmt.Sprintf("MOD R%X, R%X", x, y)
		}
	case 0xA5:
		if imm < 16 {
			return fmt.Sprintf("MOD R%X, R%X, R%X", x, y, z)
		}
	case 0xA6:
		if op[1] < 16 {
			return fmt.Sprintf("REMI R%X, %d", x, smm)
		}
	case 0xA7:
		if imm == 0 {
			return fmt.Sprintf("REM R%X, R%X", x, y)
		}
	case 0xA8:
		if imm < 16 {
			return fmt.Sprintf("REM R%X, R%X, R%X", x, y, z)
		}

	case 0xB0:
		if op[1] < 16 && imm < 16 {
			return fmt.Sprintf("SHL R%X, %d", x, z)
		}
	case 0xB1:
		if op[1] < 16 && imm < 16 {
			return fmt.Sprintf("SHR R%X, %d", x, z)
		}
	case 0xB2:
		if op[1] < 16 && imm < 16 {
			return fmt.Sprintf("SAR R%X, %d", x, z)
		}
	case 0xB3:
		if imm == 0 {
			return fmt.Sprintf("SHL R%X, R%X", x, y)
		}
	case 0xB4:
		if imm == 0 {
			return fmt.Sprintf("SHR R%X, R%X", x, y)
		}
	case 0xB5:
		if imm == 0 {
			return fmt.Sprintf("SAR R%X, R%X", x, y)
		}

	case 0xC0:
		if op[1] < 16 && imm == 0 {
			return fmt.Sprintf("PUSH R%X", x)
		}
	case 0xC1:
		if op[1] < 16 && imm == 0 {
			return fmt.Sprintf("POP R%X", x)
		}
	case 0xC2:
		if op[1] == 0 && imm == 0 {
			return "PUSHALL"
		}
	case 0xC3:
		if op[1] == 0 && imm == 0 {
			return "POPALL"
		}
	case 0xC4:
		if op[1] == 0 && imm == 0 {
			return "PUSHF"
		}
	case 0xC5:
		if op[1] == 0 && imm == 0 {
			return "POPF"
		}

	case 0xD0:
		if op[1] == 0 {
			return fmt.Sprintf("PAL #%04X", imm)
		}
	case 0xD1: // PAL RX
		if op[1] < 16 && imm == 0 {
			return fmt.Sprintf("PAL R%X", x)
		}

	case 0xE0:
		if op[1] < 16 {
			return fmt.Sprintf("NOTI R%X, #%04X", x, imm)
		}
	case 0xE1:
		if op[1] < 16 && imm == 0 {
			return fmt.Sprintf("NOT R%X", x)
		}
	case 0xE2:
		if imm == 0 {
			return fmt.Sprintf("NOT R%X, R%X", x, y)
		}
	case 0xE3:
		if op[1] < 16 {
			return fmt.Sprintf("NEGI R%X, #%04X", x, imm)
		}
	case 0xE4:
		if op[1] < 16 && imm == 0 {
			return fmt.Sprintf("NEG R%X", x)
		}
	case 0xE5:
		if imm == 0 {
			return fmt.Sprintf("NEG R%X, R%X", x, y)
		}

	}
	return ""
}
