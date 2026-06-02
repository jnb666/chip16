package vm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func write(v *VM, at uint16, bytes ...byte) {
	copy(v.Mem[at:], bytes)
}

func i16(v uint16) int16 {
	return int16(v)
}

func TestLDI(t *testing.T) {
	v := New(nil)
	write(v, 0, 0x20, 0x01, 0xFE, 0xFF) // LDI R1, 0xFFFE
	v.Step()
	assert.Equal(t, i16(0xFFFE), v.R[1])
}

func TestLDM(t *testing.T) {
	v := New(nil)
	write(v, 0, 0x22, 0x01, 0xFF, 0xAA) // LDM R1, 0xAAFF
	v.Store(0xAAFF, 0x7BCC)
	v.Step()
	assert.Equal(t, int16(0x7BCC), v.R[1])
}

func TestSTM(t *testing.T) {
	v := New(nil)
	v.R[1] = i16(0xBEEF)
	v.R[2] = 0x1234
	write(v, 0, 0x31, 0x21) // STM R1, R2
	v.Step()
	assert.Equal(t, i16(0xBEEF), v.Load(0x1234))
}

func TestADDI(t *testing.T) {
	v := New(nil)
	v.R[1] = 4
	write(v, 0, 0x40, 0x01, 0x03) // ADDI R1, 3
	v.Step()
	assert.Equal(t, int16(7), v.R[1])
	assert.Equal(t, uint8(0), v.getFlags())
}

func TestADD(t *testing.T) {
	v := New(nil)
	v.R[1] = 4
	v.R[2] = -3
	write(v, 0, 0x41, 0x21) // ADD R1, R2
	v.Step()
	assert.Equal(t, int16(1), v.R[1])
	assert.Equal(t, uint8(F_Carry), v.getFlags())
}

func TestADD2(t *testing.T) {
	v := New(nil)
	v.R[1] = 32760
	v.R[2] = 10
	write(v, 0, 0x42, 0x21, 0x03) // ADD R1, R2, R3
	v.Step()
	assert.Equal(t, i16(32770), v.R[3])
	assert.Equal(t, uint8(F_Overflow|F_Negative), v.getFlags())
}

func TestSUBI(t *testing.T) {
	v := New(nil)
	v.R[1] = 1
	write(v, 0, 0x50, 0x01, 0x02) // SUBI R1, 2
	v.Step()
	assert.Equal(t, int16(-1), v.R[1])
	assert.Equal(t, uint8(F_Negative|F_Carry), v.getFlags())
}

func TestSUB(t *testing.T) {
	v := New(nil)
	v.R[1] = 2
	v.R[2] = 2
	write(v, 0, 0x51, 0x21) // SUBI R1, R2
	v.Step()
	assert.Equal(t, int16(0), v.R[1])
	assert.Equal(t, uint8(F_Zero), v.getFlags())
}

func TestJMP(t *testing.T) {
	v := New(nil)
	write(v, 0, 0x12, 0x00, 0xB0, 0x10) // JMPZ 10B0
	write(v, 4, 0x12, 0x01, 0xB0, 0x10) // JMPNZ 10B0
	v.Step()
	v.Step()
	assert.Equal(t, uint16(0x10B0), v.PC)
}

func TestCALL(t *testing.T) {
	v := New(nil)
	write(v, 0, 0x14, 0x00, 0xB0, 0x10) // CALL 10B0
	v.Step()
	assert.Equal(t, uint16(StackBase+2), v.SP)
	assert.Equal(t, uint16(0x10B0), v.PC)
	assert.Equal(t, int16(4), v.Load(StackBase))
}

func TestRET(t *testing.T) {
	v := New(nil)
	write(v, 0, 0x14, 0x00, 0xB0, 0x10) // CALL 10B0
	write(v, 0x10B0, 0x15)              // RET
	v.Step()
	assert.Equal(t, uint16(StackBase+2), v.SP)
	assert.Equal(t, uint16(0x10B0), v.PC)
	v.Step()
	assert.Equal(t, uint16(0x4), v.PC)
	assert.Equal(t, uint16(StackBase), v.SP)
}

func TestRND(t *testing.T) {
	v := New(nil)
	v.RNG.Seed(1)
	write(v, 0, 0x07, 0x01, 0x05, 0x00) // RND RX, 5
	write(v, 4, 0x10)                   // JMP 0
	var gen [10]int16
	for i := range gen {
		v.Step()
		gen[i] = v.R[1]
	}
	assert.Equal(t, [10]int16{5, 5, 3, 3, 5, 5, 5, 5, 1, 1}, gen)
}
