package asm

import (
	"bytes"
	"embed"
	"image"
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/jnb666/chip16/vm"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/*
var fs embed.FS

func TestSimple(t *testing.T) {
	v, _ := runTest(t, "simple.asm")
	assert.Equal(t, int16(7), v.R[0])
	assert.Equal(t, int16(6), v.R[1])
	assert.Equal(t, int16(42), v.R[2])
}

func TestFibonacci(t *testing.T) {
	v, _ := runTest(t, "fib.asm")
	for addr := uint16(0x0100); addr <= 0x0120; addr += 2 {
		t.Logf("%04X : %d", addr, v.Load(addr))
	}
	assert.Equal(t, int16(1597), v.Load(0x120))
}

func TestTimer(t *testing.T) {
	v, elapsed := runTest(t, "vblnk.asm")
	assert.Equal(t, uint16(0x0014), v.PC)
	assert.Equal(t, 250*time.Millisecond, elapsed.Round(time.Millisecond))
}

func TestMaze(t *testing.T) {
	v, _ := runTest(t, "maze.asm")
	assert.Equal(t, uint16(0x0060), v.PC)
	assert.Greater(t, v.Cycles, 9500)
	t.Log("writing screenshot to maze.png")
	err := screenshot(v.ScreenImage(), "maze.png")
	assert.NoError(t, err)
}

func TestLife(t *testing.T) {
	code, logs, err := assembleTest("life.asm")
	t.Logf("\n%s", logs)
	require.NoError(t, err)

	v := vm.New(nil)
	v.RNG.Seed(42)
	v.CycleTime = 0
	copy(v.Mem[:], code)

	start := time.Now()
	for range 100 {
		err = v.Run()
		assert.Equal(t, true, err.(vm.Error).Halted)
		assert.Equal(t, uint16(0x0038), v.PC)
	}
	t.Logf("\n%s\nElapsed: %s", v, time.Since(start))

	t.Log("writing screenshot to life.png")
	err = screenshot(v.ScreenImage(), "life.png")
	assert.NoError(t, err)
}

func BenchmarkLife(b *testing.B) {
	code, _, err := assembleTest("life.asm")
	if err != nil {
		b.Fatal(err)
	}
	v := vm.New(nil)
	v.RNG.Seed(42)
	v.CycleTime = 0
	copy(v.Mem[:], code)
	v.Run()
	v.Cycles = 0
	for b.Loop() {
		v.Run()
	}
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "frames/sec")
	b.ReportMetric(float64(v.Cycles)/float64(b.N), "cycles/frame")
}

func runTest(t *testing.T, file string) (v *vm.VM, elapsed time.Duration) {
	code, logs, err := assembleTest(file)
	t.Logf("\n%s", logs)
	require.NoError(t, err)

	v = vm.New(nil)
	copy(v.Mem[:], code)
	start := time.Now()
	err = v.Run()
	elapsed = time.Since(start)
	t.Logf("\n%s\nElapsed: %s", v, elapsed)
	require.IsType(t, vm.Error{}, err)
	assert.Equal(t, true, err.(vm.Error).Halted)
	return v, elapsed
}

func assembleTest(file string) (code []byte, logs string, err error) {
	src, err := fs.Open("testdata/" + file)
	if err != nil {
		return nil, "", err
	}
	defer src.Close()

	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	var buf bytes.Buffer
	log.SetOutput(&buf)

	a := New()
	err = a.Assemble(src)
	return a.Code, buf.String(), err
}

func screenshot(img image.Image, file string) error {
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return os.WriteFile(file, buf.Bytes(), 0644)
}
