package chip8

import (
	"errors"
	"fmt"
)

var (
	ErrUndefinedInstruction = errors.New("Undefined instruction")
)

func newUndefinedInstructionErr(opcode Opcode) error {
	return errors.Join(ErrUndefinedInstruction, fmt.Errorf("opcode=%x"))
}

type Bool = uint8

const (
	False = Bool(0)
	True  = Bool(1)
)

// 0-15
type uint4 = uint8

// The CHIP-8 interpreter itself occupies the first 512 bytes of memory.
// Therefore, most programs written for CHIP-8 do not access memory at
// locations below 512.
// The uppermost 256 bytes are reserved for display refresh. The 96 bytes
// below these, are reserved for the call stack, internal use and other
// variables.
//
// "In modern CHIP-8 implementations, where the interpreter is running
// natively outside the 4K memory space, there is no need to avoid the lower
// 512 bytes of memory (0x000-0x1FF), and it is common to store font data
// there."
//
// Source: https://en.wikipedia.org/wiki/CHIP-8
type Memory [4096]uint8

// The CHIP-8 has 16 8-bit registers.
// They range from V0 to VF.
type Register [16]uint8

func (r Register) Read(n uint4) uint8 {
	return r[int(n)]
}

func (r Register) Write(n uint4, v uint8) {
	r[int(n)] = v
}

type IndexRegister uint16

type Stack [64]uint8

// The size of the frame buffer is 64x32.
// Each pixel can ether be on or off.
// Therefore the frame buffer must be
// 2048 bits, or 256 bytes in size
type FrameBuffer [256]uint8

// Usually 12-bits by spec, but limited by Go to represent it with 16 bits.
type Address = uint16

func (fb *FrameBuffer) getIndexes(x, y uint8) (arrIdx, bitIdx uint8) {
	idx := x%64 + y*64
	arrIdx = idx / 8
	bitIdx = idx % 8
	return
}

func (fb *FrameBuffer) Write(x, y uint8, bit Bool) {
	arrIdx, bitIdx := fb.getIndexes(x, y)
	b := fb[arrIdx]
	b &= ^(1 << bitIdx)
	b |= ((bit & 1) << bitIdx)
	fb[arrIdx] = b
}

func (fb *FrameBuffer) Read(x, y uint8) Bool {
	arrIdx, bitIdx := fb.getIndexes(x, y)
	b := fb[arrIdx]
	return b >> bitIdx & True
}

func (fb *FrameBuffer) Clear() {
	for i := range 256 {
		fb[i] = 0
	}
}

// Dumps the byte at the target location and
// the byte to the left and right (if available).
func (fb *FrameBuffer) Dump(x, y uint8) (d []uint8, arrIdx, bitIdx uint8) {
	arrIdx, bitIdx = fb.getIndexes(x, y)

	if arrIdx > 0 {
		d = append(d, fb[arrIdx-1])
	}
	d = append(d, fb[arrIdx])
	if arrIdx < uint8(len(fb)-1) {
		d = append(d, fb[arrIdx+1])
	}
	return
}

type Opcode uint16

// Checks whether the n-th 4 bit number of the big
// endian encoded Opcode op matches v.
func (op Opcode) matchU4(n, v uint16) bool {
	u4 := (op >> (n * 4)) & 0xF
	return uint16(u4) == v
}

func (op Opcode) matchU16(v uint16) bool {
	return uint16(op) == v
}

func (op Opcode) U12(n uint16) Address {
	u12 := (op >> (n * 4))
	return Address(u12)
}

func (op Opcode) U8(n uint16) uint8 {
	u8 := (op >> (n * 4)) & 0xFF
	return uint8(u8)
}

func (op Opcode) U4(n uint16) uint4 {
	u8 := (op >> (n * 4)) & 0xFF
	return uint4(u8)
}

type VM struct {
	Memory         Memory
	Register       Register
	IndexRegister  IndexRegister
	StackPointer   uint8
	DelayTimer     uint8
	SoundTimer     uint8
	FrameBuffer    FrameBuffer
	ProgramCounter uint16
}

func New() *VM {
	return &VM{}
}

// Tick decreases the delay and sound timer by one.
// Should be called at 60Hz to meet the specification.
// If beep is true, a beep sound should be played.
func (vm *VM) Tick() (beep bool) {
	if vm.DelayTimer > 0 {
		vm.DelayTimer--
	}
	if vm.SoundTimer > 0 {
		vm.SoundTimer--
		beep = true
	}

	return
}

func (vm *VM) Instr(op Opcode) error {
	prefix := (op >> 1) & 0xF

	switch prefix {
	case 0:
		return vm.instr0(op)
	case 1:
		return vm.instr1(op)
	case 2:
		return vm.instr2(op)
	case 3:
		return vm.instr3(op)
	}
	return nil
}

func (vm *VM) instr0(op Opcode) error {
	switch op {
	case 0x00E0: // Clear Display
		vm.FrameBuffer.Clear()
	// case 0x00EE: // Return
	// 	vm.Return()
	default:
		return newUndefinedInstructionErr(op)
	}

	return nil
}

// Jump to address
func (vm *VM) instr1(op Opcode) error {
	addr := op.U12(1)
	vm.ProgramCounter = addr
	return nil
}

// Call subroutine at address
func (vm *VM) instr2(op Opcode) error {

	return nil
}

// Skips next instruction if VX equals NN
func (vm *VM) instr3(op Opcode) error {
	x := op.U4(1)
	nn := op.U8(2)

	if vm.Register.Read(x) == nn {
		vm.ProgramCounter++
	}

	return nil
}
