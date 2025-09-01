package chip8

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"time"
)

var (
	ErrUndefinedInstruction = errors.New("Undefined instruction")
	ErrIllegalMemoryAccess  = errors.New("Illegal memory access")
)

func newUndefinedInstructionErr(opcode Opcode) error {
	return errors.Join(ErrUndefinedInstruction, fmt.Errorf("opcode=%s prefix=%x hex=%s", opcode.String(), opcode.Prefix(), opcode.Hex()))
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

const (
	AddressStackPointer = 0x0
	AddressStackStart   = AddressStackPointer + 1
	MaxStackSize        = 16
	StackAddressBytes   = 2
	AddressSpritesStart = AddressStackStart + MaxStackSize*2
	AddressProgramStart = 0x200
)

func (m *Memory) Write(idx uint12, v uint8) {
	m[idx] = v
}

func (m *Memory) Read(idx uint16) uint8 {
	return m[idx]
}

// The CHIP-8 has 16 8-bit registers.
// They range from V0 to VF.
type Register [16]uint8

const (
	VF = uint4(15)
)

func (r *Register) Read(n uint4) uint8 {
	return r[int(n)]
}

func (r *Register) Write(n uint4, v uint8) {
	r[int(n)] = v
}

// Usually 12-bits by spec, but limited by Go to represent it with 16 bits.
type uint12 = uint16

// The size of the frame buffer is 64x32.
// Each pixel can ether be on or off.
// Therefore the frame buffer must be
// 2048 bits, or 256 bytes in size
type FrameBuffer [256]uint8

func (fb *FrameBuffer) indexes(idx uint16) (arrIdx, bitIdx uint8) {
	arrIdx = uint8(idx / 8)
	bitIdx = uint8(7 - (idx % 8)) // Reverse bit order: 0→7, 1→6, 2→5, etc.
	return
}

func (fb *FrameBuffer) XOR(idx uint16, spriteBit uint8) {
	arrIdx, bitIdx := fb.indexes(idx)
	currentBit := (fb[arrIdx] >> bitIdx) & 1
	newBit := currentBit ^ spriteBit
	result := (fb[arrIdx] & ^(1 << bitIdx)) | (newBit << bitIdx)
	fb[arrIdx] = result
}

func (fb *FrameBuffer) Read(idx uint16) Bool {
	arrIdx, bitIdx := fb.indexes(idx)
	b := fb[arrIdx]
	if (b & (1 << bitIdx)) != 0 {
		return True
	}
	return False
}

func (fb *FrameBuffer) Clear() {
	for i := range fb {
		fb[i] = 0
	}
}

type Opcode uint16

func (op Opcode) Dump() (d []byte) {
	d = make([]byte, 4)
	d[0] = byte(op.U4(0))
	d[1] = byte(op.U4(1))
	d[2] = byte(op.U4(2))
	d[3] = byte(op.U4(3))
	return
}

func (op Opcode) Hex() string {
	hex := func(b byte) rune {
		enc := fmt.Sprintf("%X", b)
		v := enc[0]
		return rune(v)
	}
	d := op.Dump()
	return fmt.Sprintf("%c%c%c%c", hex(d[0]), hex(d[1]), hex(d[2]), hex(d[3]))
}

func (op Opcode) Prefix() uint4 {
	return uint4(op >> 12 & 0x0F)
}

func (op Opcode) NNN() uint12 {
	return uint12(op & 0xFFF)
}

func (op Opcode) NN() uint8 {
	return uint8(op & 0xFF)
}

func (op Opcode) N() uint4 {
	return uint4(op & 0x0F)
}

func (op Opcode) X() uint4 {
	return uint4(op >> 8 & 0x0F)
}

func (op Opcode) Y() uint4 {
	return uint4(op >> 4 & 0x0F)
}

func (op Opcode) U4(n uint4) uint4 {
	return uint4(op >> ((3 - n) * 4) & 0x0F)
}

func (op Opcode) String() string {
	label := "??"
	switch op.Prefix() {
	case 0x0:
		label = "SYS/CLS/RET"
	case 0x1:
		label = "JP"
	case 0x2:
		label = "CALL"
	case 0x3:
		label = "SE"
	case 0x4:
		label = "SNE"
	case 0x5:
		label = "SE"
	case 0x6:
		label = "LD"
	case 0x7:
		label = "ADD"
	case 0x8:
		label = "LD/OR/AND/XOR/ADD/SUB/SHR/SUBN/SHL"
	case 0x9:
		label = "SNE"
	case 0xA:
		label = "LD"
	case 0xB:
		label = "JP"
	case 0xC:
		label = "RND"
	case 0xD:
		label = "DRW"
	case 0xE:
		label = "SKP/SKNP"
	case 0xF:
		label = "LD/ADD"
	}
	return fmt.Sprintf("%x %s", op.Prefix(), label)
}

type Metrics struct {
	CyclesPerSecond float64
	CPUTime         time.Duration
	DrawTime        time.Duration
}

type VM struct {
	Memory         Memory
	Register       Register
	IndexAddress   uint12
	DelayTimer     uint8
	SoundTimer     uint8
	FrameBuffer    FrameBuffer
	ProgramCounter uint16
	Input          InputAdapter
	Output         OutputAdapter
	Metrics        Metrics
}

func New(i InputAdapter, o OutputAdapter) *VM {
	vm := &VM{
		ProgramCounter: AddressProgramStart,
		Input:          i,
		Output:         o,
	}
	vm.loadSprites()
	return vm
}

func (vm *VM) loadSprites() {
	for idx, s := range Sprites {
		begin := AddressSpritesStart + idx*5
		for offset, u8 := range s {
			vm.Memory[begin+offset] = u8
		}
	}
}

// Loads ROM into memory.
func (vm *VM) Load(r io.Reader) error {
	buf := make([]uint8, 2)
	i := AddressProgramStart
	for {
		n, err := r.Read(buf)
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return err
		}
		if n == 0 {
			break
		}
		if i >= 4096 {
			return newIllegalMemoryAccessErr("memory limit exceeded")
		}

		for j := range n {
			vm.Memory[i+j] = buf[j]
		}
		i += n
	}
	return nil
}

func (vm *VM) Start(ctx context.Context, clockRateHz int) error {
	cpu := time.NewTicker(time.Second / time.Duration(clockRateHz)) // 700Hz
	defer cpu.Stop()
	timer := time.NewTicker(time.Second / 60) // 60Hz
	defer timer.Stop()

	t := time.Now()
	var totalCPU, totalDraw time.Duration
	var cycles int64

	for {
		select {
		case <-cpu.C:
			start := time.Now()
			if err := vm.Cycle(); err != nil {
				return err
			}
			totalCPU += time.Since(start)
			start = time.Now()
			vm.Output.Draw(vm.FrameBuffer)
			totalDraw += time.Since(start)

			cycles++

			if s := time.Since(t); s > time.Second {
				vm.Metrics.CyclesPerSecond = float64(cycles) / s.Seconds()
				vm.Metrics.CPUTime = totalCPU / time.Duration(cycles)
				vm.Metrics.DrawTime = totalDraw / time.Duration(cycles)

				cycles = 0
				totalCPU = 0
				totalDraw = 0
				t = time.Now()
			}
		case <-timer.C:
			beep := vm.Tick()
			if beep {
				vm.Output.Beep()
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (vm *VM) readInstr(pc uint16) Opcode {
	return Opcode(vm.Memory[pc])<<8 | Opcode(vm.Memory[pc+1])
}

// Executes a CPU cycle by executing the instruction pointed at
// by the program counter.
func (vm *VM) Cycle() error {
	pc := vm.ProgramCounter
	if pc < AddressProgramStart {
		return newIllegalMemoryAccessErr("tried to access internal memory")
	}

	op := vm.readInstr(pc)
	if true {
		slog.Info("CPU", "PC", pc, "PCX", fmt.Sprintf("%X", pc), "OP", op.String(), "[0]", op.U4(0), "[1]", op.U4(1), "[2]", op.U4(2), "[3]", op.U4(3), "hex", op.Hex())
		if err := vm.Instr(op); err != nil {
			return err
		}
	} else {
		slog.Info("CPU", "PC", pc, "PCX", fmt.Sprintf("%X", pc), "OP", op.String(), "hex", op.Hex(), "U12", fmt.Sprintf("%X", op.NNN()))
	}

	vm.SkipInstr()
	return nil
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

func (vm *VM) StackSize() uint8 {
	return vm.Memory[AddressStackPointer]
}

func (vm *VM) SetStackSize(u8 uint8) {
	vm.Memory[AddressStackPointer] = u8
}

func (vm *VM) StackPush(u12 uint12) error {
	size := vm.StackSize()
	if size >= MaxStackSize {
		return newIllegalMemoryAccessErr("stack overflow")
	}
	top := AddressStackStart + int(size)*2
	vm.Memory[top+1] = uint8(u12)
	vm.Memory[top+2] = uint8(u12 >> 8)
	vm.SetStackSize(size + 1)
	slog.Info("STACK push", "size", vm.StackSize(), "v", int(u12))
	return nil
}

func (vm *VM) StackPop() (uint12, error) {
	size := vm.StackSize()
	if size == 0 {
		return 0, newIllegalMemoryAccessErr("stack is empty")
	}
	top := AddressStackStart + int(size)*2
	v1 := vm.Memory[top-1]
	v2 := vm.Memory[top-0]
	vm.SetStackSize(size - 1)
	v := uint12(v1) | uint12(v2)<<8
	slog.Info("STACK pop", "size", vm.StackSize(), "v", int(v))
	return v, nil
}

func (vm *VM) rand() uint8 {
	return uint8(rand.UintN(255))
}

func (vm *VM) Instr(op Opcode) error {
	switch op.Prefix() {
	case 0x0:
		return vm.instr0(op)
	case 0x1:
		return vm.instr1(op)
	case 0x2:
		return vm.instr2(op)
	case 0x3:
		return vm.instr3(op)
	case 0x4:
		return vm.instr4(op)
	case 0x5:
		return vm.instr5(op)
	case 0x6:
		return vm.instr6(op)
	case 0x7:
		return vm.instr7(op)
	case 0x8:
		return vm.instr8(op)
	case 0x9:
		return vm.instr9(op)
	case 0xA:
		return vm.instrA(op)
	case 0xB:
		return vm.instrB(op)
	case 0xC:
		return vm.instrC(op)
	case 0xD:
		return vm.instrD(op)
	case 0xE:
		return vm.instrE(op)
	case 0xF:
		return vm.instrF(op)
	default:
		return newUndefinedInstructionErr(op)
	}
}

func (vm *VM) instr0(op Opcode) error {
	if op.U4(1) == 0 && op.U4(2) == 0xE {
		switch op.U4(3) {
		case 0x0:
			// Clear display
			vm.FrameBuffer.Clear()
		case 0xE:
			// Return from sub routine
			ptr, err := vm.StackPop()
			if err != nil {
				return err
			}
			vm.Jump(ptr + 2)
		default:
			return newUndefinedInstructionErr(op)
		}
		return nil
	}
	u12 := op.NNN()
	slog.Info("CALL MACHINE", "u12", u12, "dump", op.Dump())
	return nil
}

func (vm *VM) Jump(addr uint16) {
	// After this CPU cycle, the PC will be incremented by two.
	// Therefore addr must be subtracted by two here.
	vm.ProgramCounter = addr - 2
}

func (vm *VM) SkipInstr() {
	vm.ProgramCounter += 2
}

// Jump to address
func (vm *VM) instr1(op Opcode) error {
	addr := op.NNN()
	vm.Jump(addr)
	return nil
}

// Call subroutine at address
func (vm *VM) instr2(op Opcode) error {
	addr := op.NNN()
	slog.Info("CALL", "addr", op.NNN(), "addr_X", fmt.Sprintf("%X", op.NNN()))
	returnAddr := vm.ProgramCounter
	if err := vm.StackPush(returnAddr); err != nil {
		return err
	}
	vm.Jump(addr)
	return nil
}

// Skips next instruction if VX equals NN
func (vm *VM) instr3(op Opcode) error {
	x := op.U4(1)
	nn := op.NN()

	if vm.Register.Read(x) == nn {
		vm.SkipInstr()
	}

	return nil
}

// Skips next instruction if VX not equals NN
func (vm *VM) instr4(op Opcode) error {
	x := op.X()
	nn := op.NN()

	if vm.Register.Read(x) != nn {
		vm.SkipInstr()
	}

	return nil
}

// Skips next instruction if VX is equals VY
func (vm *VM) instr5(op Opcode) error {
	if z := op.U4(3); z != 0 {
		return newUndefinedInstructionErr(op)
	}

	x := op.X()
	y := op.Y()

	if vm.Register.Read(x) == vm.Register.Read(y) {
		vm.SkipInstr()
	}

	return nil
}

// Sets VX to NN
func (vm *VM) instr6(op Opcode) error {
	x := op.X()
	nn := op.NN()
	slog.Debug("6XNN: Set X to NN", "X", x, "NN", nn)

	vm.Register.Write(x, nn)
	return nil
}

// Adds NN to VX
func (vm *VM) instr7(op Opcode) error {
	x := op.X()
	nn := op.NN()

	valueX := vm.Register.Read(x)
	result := valueX + nn

	slog.Debug("7XNN: Add NN to VX", "X", x, "NN", nn, "valueX", valueX, "result", result)

	vm.Register.Write(x, result)

	return nil
}

// Register ops
func (vm *VM) instr8(op Opcode) error {
	x := op.X()
	y := op.Y()
	cmd := op.U4(3)

	valueY := vm.Register.Read(y)
	if cmd == 0x0 {
		// assign VX to value of VY
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
	x := op.X()
	y := op.Y()
	z := op.U4(3)

	if z != 0 {
		return newUndefinedInstructionErr(op)
	}

	valueX := vm.Register.Read(x)
	valueY := vm.Register.Read(y)

	if valueX != valueY {
		vm.SkipInstr()
	}

	return nil
}

// Sets I to address NNN
func (vm *VM) instrA(op Opcode) error {
	addr := op.NNN()
	slog.Debug("ANNN: Set I to NNN", "NNN", addr)
	vm.IndexAddress = addr
	return nil
}

// Jumps to address NNN + V0
func (vm *VM) instrB(op Opcode) error {
	v0 := uint12(vm.Register.Read(0))
	addr := op.NNN() + v0
	slog.Debug("BNNN: Jump to NNN + V0", "NNN", op.NNN(), "V0", v0)
	vm.Jump(addr)
	return nil
}

// Sets VX to result of bitwise on rand number and NN
func (vm *VM) instrC(op Opcode) error {
	x := op.X()
	nn := op.NN()
	result := vm.rand() & nn
	vm.Register.Write(x, result)
	return nil
}

// Draws a sprite at coordinate VX, VY with 8px width
// and height of N. If VF is 1, bits will be XORed.
// Draws a sprite at coordinate VX, VY with 8px width
// and height of N. Each sprite is 8px wide and N pixels tall.
// Each row of 8 pixels is read as bit-coded starting from memory location I.
// VF is set to 1 if any pixels are flipped from set to unset, and 0 otherwise.
func (vm *VM) instrD(op Opcode) error {
	x := op.X()
	y := op.Y()
	n := op.N()
	originX := vm.Register.Read(x) % 64
	originY := vm.Register.Read(y) % 32

	slog.Debug("DXYN: Draw sprite I to X, Y with height N", "I", vm.IndexAddress, "X", x, "Y", y, "N", n, "originX", originX, "originY", originY)

	// Reset collision flag
	vm.Register.Write(VF, 0)

	drawX, drawY := uint8(0), originY
	for i := range uint12(n) {
		spriteByte := vm.Memory[vm.IndexAddress+i]
		drawX = originX

		for j := 7; j >= 0; j-- {
			idx := uint16(drawY)*64 + uint16(drawX)
			pixel := vm.FrameBuffer.Read(idx)
			spriteBit := spriteByte & (1 << j)

			if spriteBit != 0 {
				spriteBit = 1
			}

			// On collision, set carry flag to 1
			if spriteBit != 0 && pixel != 0 {
				vm.Register.Write(VF, 1)
			}

			// XOR
			vm.FrameBuffer.XOR(idx, spriteBit)

			//slog.Debug("DXYN: XOR pixel", "idx", idx, "drawX", drawX, "drawY", drawY, "pixelBit", pixel, "spriteBit", spriteBit, "result", vm.FrameBuffer.Read(idx))

			drawX++
			if drawX >= 64 {
				break
			}
		}

		drawY++
		if drawY >= 32 {
			break
		}
	}

	return nil
}

func (vm *VM) instrE(op Opcode) error {
	x := op.X()
	cmd := op.NN()
	key := Key(vm.Register.Read(x))

	slog.Info("KEY", "X", key, "CMD", cmd)

	switch cmd {
	case 0x9E:
		if vm.Input.Pressed(key) {
			vm.SkipInstr()
		}
	case 0xA1:
		if !vm.Input.Pressed(key) {
			vm.SkipInstr()
		}
	default:
		return newUndefinedInstructionErr(op)
	}

	return nil
}

func (vm *VM) instrF(op Opcode) error {
	x := op.X()
	cmd := op.NN()
	switch cmd {
	case 0x07:
		// Sets VX to value of delay timer
		vm.Register.Write(x, vm.DelayTimer)
	case 0x0A:
		// Wait for next keyboard input and write it to VX
		key := vm.Input.Next()
		vm.Register.Write(x, uint8(key))
	case 0x15:
		// Sets delay timer to VX
		vm.DelayTimer = vm.Register.Read(x)
	case 0x18:
		// Sets sound timer to VX
		vm.SoundTimer = vm.Register.Read(x)
	case 0x1E:
		// Adds VX to I
		vm.IndexAddress += uint12(vm.Register.Read(x))
	case 0x29:
		// Sets I to the location of sprite
		idx := vm.Register.Read(x)
		fmt.Println("REQUEST SPRITE LOCATION", idx)
		if idx > 15 {
			return newIllegalMemoryAccessErr("requested sprite index out of bounds")
		}
		vm.IndexAddress = AddressSpritesStart + uint12(idx*5)
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
		for i := range uint4(x) {
			vm.Memory[vm.IndexAddress+uint12(i)] = vm.Register.Read(i)
		}
	case 0x65:
		// Loads V0 to VX from memory. Starts at address I.
		for i := range uint4(x) {
			vm.Register.Write(i, vm.Memory[vm.IndexAddress+uint12(i)])
		}
	default:
		return newUndefinedInstructionErr(op)
	}
	return nil
}
