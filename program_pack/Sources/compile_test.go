package sources_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jnb666/chip16/asm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sources = []string{
	"AdsrDemo/AdsrDemo.asm",
	"AdsrTest.asm",
	"Anim/Anim.asm",
	"ASCII.asm",
	"Ball.asm",
	"BC_TestRom.asm",
	"Chip1613ST/Chip16Spec1.3.asm",
	"CollisionTest.asm",
	"GB16.asm",
	"Herdle/Herdle.asm",
	"Mandel/mandel.ASM",
	"Maze.asm",
	"MusicMaker/MusicMaker.asm",
	"Ninja/ninja.asm",
	"PadTest.asm",
	"PaletteFlip/PaletteFlip.asm",
	"PCBIOS/Chip16.asm",
	"Pong.asm",
	"Reflection/Reflection.asm",
	"SFX.s",
	"Sokoban_src/Sokoban.asm",
	"SongOfStorms.asm",
	"SoundTest.asm",
	"Starfield.asm",
	"Stopwatch/Stopwatch.asm",
	"triangle.ASM",
}

// check all examples compile
func TestCompile(t *testing.T) {
	dir := thisDir()

	for _, src := range sources {
		file := filepath.Join(dir, src)
		r, err := os.Open(file)
		require.NoError(t, err)
		a := asm.New()
		a.BaseDir = filepath.Dir(file)
		err = a.Assemble(r)
		t.Logf("%s: %d bytes", src, len(a.Code))
		assert.NoError(t, err)
		r.Close()
	}
}

func thisDir() string {
	_, file, _, _ := runtime.Caller(1)
	return filepath.Dir(file)
}
