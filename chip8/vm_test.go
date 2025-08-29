package chip8_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "github.com/yonedash/chip8emu/chip8"
)

func stringifyByte(b uint8) string {
	return fmt.Sprintf("[%08b]", b)
}

func stringifyDump(b []uint8) string {
	var d string
	for _, s := range b {
		d += stringifyByte(s)
	}
	return d
}

func TestFrameBuffer(t *testing.T) {
	var fb FrameBuffer

	assertWrite := func(x, y uint8, bit Bool) {
		t.Helper()
		fb.Write(x, y, bit)
		v := fb.Read(x, y)

		dumped, arrIdx, bitIdx := fb.Dump(x, y)
		assert.Equal(t, bit, v, fmt.Sprintf("write x=%d y=%d bit=%d arrIdx=%d bitIdx=%d dump=%s", x, y, bit, arrIdx, bitIdx, stringifyDump(dumped)))
	}

	assertWrite(0, 0, True)
	assertWrite(7, 0, True)
	assertWrite(7, 0, False)
	assertWrite(8, 0, True)
	assertWrite(60, 0, True)
	assertWrite(0, 1, True)
	assertWrite(0, 1, False)
}

func TestStack(t *testing.T) {
	vm := New()

	for i := range uint16(12) {
		require.NoError(t, vm.StackPush(i), "stack push")
	}
	err := vm.StackPush(42)
	assert.Error(t, err, "stack overflow")

	for i := range uint16(12) {
		v, err := vm.StackPop()
		assert.NoError(t, err, "stack pop")
		assert.Equal(t, v, 11-i, "stack popped value")
	}

	_, err = vm.StackPop()
	assert.Error(t, err, "stack empty")
}
