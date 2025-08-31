package chip8_test

import (
	"fmt"
	"math"
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
	vm := New(nil, nil)

	for i := range uint16(MaxStackSize) {
		require.NoError(t, vm.StackPush(i), "stack push")
	}
	err := vm.StackPush(42)
	require.Error(t, err, "stack overflow")

	for i := range uint16(MaxStackSize) {
		v, err := vm.StackPop()
		require.NoError(t, err, "stack pop")
		require.Equal(t, int(MaxStackSize-1-i), int(v), "stack popped value")
	}

	_, err = vm.StackPop()
	require.Error(t, err, "stack empty")

	// Test big numbers
	m := uint16(math.MaxUint16)
	require.NoError(t, vm.StackPush(m), "push big number")
	v, err := vm.StackPop()
	require.NoError(t, err, "pop big number")
	require.Equal(t, m, v, "big number")
}

func TestOpcodeHelpers(t *testing.T) {
	a := Opcode(0xABCD)
	assert.EqualValues(t, 0xA, a.U4(0))
	assert.EqualValues(t, 0xB, a.U4(1))
	assert.EqualValues(t, 0xC, a.U4(2))
	assert.EqualValues(t, 0xD, a.U4(3))
	assert.EqualValues(t, uint8(0xAB), a.U8(0))
	assert.EqualValues(t, uint8(0xBC), a.U8(1))
	assert.EqualValues(t, uint8(0xCD), a.U8(2))
	assert.EqualValues(t, 0xBCD, a.U12())
}
