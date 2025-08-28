package chip8

import (
	"errors"
	"fmt"
	"math/rand/v2"
)

var (
	ErrUndefinedInstruction = errors.New("Undefined instruction")
	ErrIllegalMemoryAccess  = errors.New("Illegal memory access")
)

func newUndefinedInstructionErr(opcode Opcode) error {
	return errors.Join(ErrUndefinedInstruction, fmt.Errorf("opcode=%x"))
}

func newIllegalMemoryAccessErr(msg string) error {
	return errors.Join(ErrIllegalMemoryAccess, errors.New(msg))
}

type Bool = uint8

const (
	False = Bool(0)
	True  = Bool(1)
)

// 0-15
type uint4 uint8

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

const (
	VF = uint4(15)
)

func (r Register) Read(n uint4) uint8 {
	return r[int(n)]
}

func (r Register) Write(n uint4, v uint8) {
	r[int(n)] = v
}

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
	IndexAddress   Address
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

func (vm *VM) skipInstr() {
	vm.IndexAddress++
}

func (vm *VM) rand() uint8 {
	return uint8(rand.UintN(255))
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
	case 4:
		return vm.instr4(op)
	case 5:
		return vm.instr5(op)
	case 6:
		return vm.instr6(op)
	case 7:
		return vm.instr7(op)
	case 8:
		return vm.instr8(op)
	case 9:
		return vm.instr9(op)
	case 10:
		return vm.instrA(op)
	case 11:
		return vm.instrB(op)
	case 12:
		return vm.instrC(op)
	case 13:
		return vm.instrD(op)
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
	vm.IndexAddress = addr
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
		vm.skipInstr()
	}

	return nil
}

// Skips next instruction if VX not equals NN
func (vm *VM) instr4(op Opcode) error {
	x := op.U4(1)
	nn := op.U8(2)

	if vm.Register.Read(x) != nn {
		vm.skipInstr()
	}

	return nil
}

// Skips next instruction if VX is equals VY
func (vm *VM) instr5(op Opcode) error {
	if z := op.U4(3); z != 0 {
		return newUndefinedInstructionErr(op)
	}

	x := op.U4(1)
	y := op.U4(2)

	if vm.Register.Read(x) == vm.Register.Read(y) {
		vm.skipInstr()
	}

	return nil
}

// Sets VX to NN
func (vm *VM) instr6(op Opcode) error {
	x := op.U4(1)
	nn := op.U8(2)

	vm.Register.Write(x, nn)

	return nil
}

// Adds NN to VX
func (vm *VM) instr7(op Opcode) error {
	x := op.U4(1)
	nn := op.U8(2)

	result := vm.Register.Read(x)
	result += nn
	vm.Register.Write(x, result)

	return nil
}

// Register ops
func (vm *VM) instr8(op Opcode) error {
	x := op.U4(1)
	y := op.U4(2)
	cmd := op.U4(3)

	valueY := vm.Register.Read(y)
	if cmd == 0x0 {
		vm.Register.Write(x, valueY)
		return nil
	}
	valueX := vm.Register.Read(x)

	var value uint8
	switch cmd {
	case 0x1:
		// bitwise OR
		value = valueX | valueY
	case 0x2:
		// bitwise AND
		value = valueX & valueY
	case 0x3:
		// XOR
		value = valueX ^ valueY
	case 0x4:
		// Add y to x; sets VF to 1 if overflow, otherwise 0
		value = valueX + valueY
		overflowU8 := uint8(0)
		if value < valueX || value < valueY {
			overflowU8 = 1
		}
		vm.Register.Write(VF, overflowU8)
	case 0x5:
		// Subtract y from x; sets to 0 if underflow, otherwise 1
		value = valueX - valueY
		underflowU8 := uint8(0)
		if valueX >= valueY {
			underflowU8 = 1
		}
		vm.Register.Write(VF, underflowU8)
	case 0x6:
		// Shifts x to the right by 1, stores least significant
		// bit of x prior to shift into VF
		leastSignificantBit := valueX & 1
		value = valueX >> 1
		vm.Register.Write(VF, leastSignificantBit)
	case 0x7:
		// Subtract x from y; sets to 0 if underflow, otherwise 1
		value = valueX - valueY
		underflowU8 := uint8(0)
		if valueY >= valueX {
			underflowU8 = 1
		}
		vm.Register.Write(VF, underflowU8)
	case 0xE:
		// Shifts x to the left by 1, sets VF to 1 if most
		// significant bit of x was set or 0 if was unset
		mostSignificantBit := (valueX >> 7) & 1
		value = valueX << 1
		vm.Register.Write(VF, mostSignificantBit)
	default:
		return newUndefinedInstructionErr(op)
	}

	vm.Register.Write(x, value)

	return nil
}

// Skips next instruction if VX not equals VY
func (vm *VM) instr9(op Opcode) error {
	x := op.U4(1)
	y := op.U4(2)
	z := op.U4(3)

	if z != 0 {
		return newUndefinedInstructionErr(op)
	}

	valueX := vm.Register.Read(x)
	valueY := vm.Register.Read(y)

	if valueX != valueY {
		vm.skipInstr()
	}

	return nil
}

// Sets I to address NNN
func (vm *VM) instrA(op Opcode) error {
	addr := op.U12(1)
	vm.IndexAddress = addr
	return nil
}

// Jumps to address NNN + V0
func (vm *VM) instrB(op Opcode) error {
	addr := op.U12(1)
	v0 := vm.Register.Read(0)
	vm.ProgramCounter = uint16(addr) + uint16(v0)
	return nil
}

// Sets VX to result of bitwise on rand number and NN
func (vm *VM) instrC(op Opcode) error {
	x := op.U4(1)
	nn := op.U8(2)
	result := vm.rand() & nn
	vm.Register.Write(x, result)
	return nil
}

// Draws a sprite at coordinate VX, VY with 8px width
// and height of N. If VF is 1, bits will be XORed.
func (vm *VM) instrD(op Opcode) error {
	x := op.U4(1)
	y := op.U4(2)
	n := op.U4(3)
	spriteX := vm.Register.Read(x)
	spriteY := vm.Register.Read(y)

	spriteMemLoc := vm.IndexAddress
	for range n {
		for i := range 8 {
			loc := int(spriteMemLoc+uint16(n)) + i
			if loc >= 4096 {
				return newIllegalMemoryAccessErr(fmt.Sprintf("location %d is out of range", loc))
			}
			bit := vm.Memory[loc]

			// Check if bits should be XORed
			if vm.Register.Read(VF) == 1 {
				v := vm.FrameBuffer.Read(spriteX, spriteY)
				vm.FrameBuffer.Write(spriteX, spriteY, v^bit)
			} else {
				vm.FrameBuffer.Write(spriteX, spriteY, bit)
			}
		}
	}

	return nil
}

func (vm *VM) instrE(op Opcode) error {
	return newUndefinedInstructionErr(op)
}

func (vm *VM) instrF(op Opcode) error {
	x := op.U4(1)
	cmd := op.U8(3)
	switch cmd {
	case 0x07:
		// Sets VX to value of delay timer
		vm.Register.Write(x, vm.DelayTimer)
	case 0x0A:
	// TODO
	case 0x15:
		// Sets delay timer to VX
		vm.DelayTimer = vm.Register.Read(x)
	case 0x18:
		// Sets sound timer to VX
		vm.SoundTimer = vm.Register.Read(x)
	case 0x1E:
		// Adds VX to I
		vm.IndexAddress += Address(vm.Register.Read(x))
	case 0x29:
	// Sets I to the location of sprite
	// TODO
	case 0x33:
		// Stores the binary-coded decimal representation of VX, with the hundreds
		// digit in memory at location in I, the tens digit at location I+1, and the
		// ones digit at location I+2
		valueX := vm.Register.Read(x)
		hundreds := valueX / 100
		tens := (valueX / 10) % 10
		ones := valueX % 10
		vm.Memory[vm.IndexAddress] = hundreds
		vm.Memory[vm.IndexAddress+1] = tens
		vm.Memory[vm.IndexAddress+2] = ones
	case 0x55:
		// Stores from V0 to VX (including VX) in memory. Starts at address I.
		for i := range uint4(16) {
			vm.Memory[vm.IndexAddress+Address(i)] = vm.Register.Read(i)
		}
	case 0x65:
		// Loads V0 to VX from memory. Starts at address I.
		for i := range uint4(16) {
			vm.Register.Write(i, vm.Memory[vm.IndexAddress+Address(i)])
		}
	default:
		return newUndefinedInstructionErr(op)
	}
	return nil
}
