package examples_test

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
	"AdsrDemo.asm",
	"bgTest.asm",
	"bounce.asm",
	"flipoffscreen.asm",
	"life.asm",
	"music.asm",
	"rectTest.asm",
}

// check all examples compile
func TestCompile(t *testing.T) {
	dir := thisDir()
	for _, file := range sources {
		r, err := os.Open(filepath.Join(dir, file))
		require.NoError(t, err)
		a := asm.New()
		err = a.Assemble(r)
		t.Logf("%s: %d bytes", file, len(a.Code))
		assert.Greater(t, len(a.Code), 0)
		assert.NoError(t, err)
		r.Close()
	}
}

func thisDir() string {
	_, file, _, _ := runtime.Caller(1)
	return filepath.Dir(file)
}
