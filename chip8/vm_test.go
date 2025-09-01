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

func TestFrameBufferFalsePositives(t *testing.T) {
	var fb FrameBuffer
	fb.Clear() // Start with a clean buffer

	// Set a specific pattern of pixels
	// We'll set specific pixels and then check that all others remain unset
	testPoints := []struct {
		x, y uint8
	}{
		{0, 0},   // Top-left corner
		{63, 0},  // Top-right corner
		{0, 31},  // Bottom-left corner
		{63, 31}, // Bottom-right corner
		{32, 16}, // Center
		{8, 0},   // Byte boundary
		{16, 0},  // Byte boundary
		{24, 0},  // Byte boundary
	}

	// Set the test pixels
	for _, p := range testPoints {
		fb.Write(p.x, p.y, True)
	}

	// Helper function to check if a point is in our test set
	isTestPoint := func(x, y uint8) bool {
		for _, p := range testPoints {
			if p.x == x && p.y == y {
				return true
			}
		}
		return false
	}

	// Check every pixel on the screen
	falsePositives := 0
	for y := uint8(0); y < 32; y++ {
		for x := uint8(0); x < 64; x++ {
			v := fb.Read(x, y)
			if isTestPoint(x, y) {
				// This should be set
				assert.Equal(t, True, v, fmt.Sprintf("Test point (%d,%d) should be set", x, y))
			} else {
				// This should NOT be set
				if v == True {
					falsePositives++
					t.Logf("False positive at (%d,%d): reported as set but should be unset", x, y)
				}
			}
		}
	}

	assert.Equal(t, 0, falsePositives, "Found false positive pixel reads")
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
	assert.EqualValues(t, 0xBCD, a.NNN())
}
