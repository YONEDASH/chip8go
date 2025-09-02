package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/yonedash/chip8emu/chip8"
)

func main() {
	const (
		debug        = true
		screenBorder = true
		pixelBorder  = false
	)

	if debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if len(os.Args) != 2 {
		fmt.Println(os.Args[0], "<rom>")
		os.Exit(1)
	}
	filename := os.Args[1]
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	io, fini, err := NewWindow()
	if err != nil {
		panic(err)
	}
	defer fini()

	go func() {
		var stopChan = make(chan os.Signal, 1)
		signal.Notify(stopChan, syscall.SIGTERM, syscall.SIGINT)
		<-stopChan
		fini()
		os.Exit(0)
	}()

	vm := chip8.New(io, io)
	if err := vm.Load(file); err != nil {
		panic(err)
	}
	if err := file.Close(); err != nil {
		panic(err)
	}

	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagVsyncHint)
	rl.InitWindow(800, 450, fmt.Sprintf("CHIP-8: %s", file.Name()))
	defer rl.CloseWindow()

	rl.SetTargetFPS(int32(rl.GetMonitorRefreshRate(rl.GetCurrentMonitor())))

	rl.InitAudioDevice()
	defer rl.CloseAudioDevice()

	go func() {
		if err := vm.Start(context.Background(), 2000); err != nil {
			panic(err)
		}
	}()

	for !rl.WindowShouldClose() {
		w, h := int32(64), int32(32)
		px := min(int32(rl.GetScreenWidth())/w, int32(rl.GetScreenHeight())/h)
		widthPx, heightPx := w*px, h*px
		fontSize := min(int32(max(2, int32(rl.GetScreenHeight())-heightPx)/4), 30)
		offsetX, offsetY := (int32(rl.GetScreenWidth())/2 - widthPx/2), (int32(rl.GetScreenHeight())/2 - heightPx/2)

		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		if debug {
			rl.DrawText(fmt.Sprintf("FPS=%d CPS=%.1f CPU=%.3fms Draw=%.3fms", rl.GetFPS(), vm.Metrics.CyclesPerSecond, float64(vm.Metrics.CPUTime.Nanoseconds())/1_000_000, float64(vm.Metrics.DrawTime.Nanoseconds())/1_000_000), offsetX, offsetY-fontSize-2, fontSize, rl.RayWhite)
			pcInfoText := fmt.Sprintf("PC=0x%3X I=0x%3X DT=%2d ST=%2d", vm.ProgramCounter, vm.IndexAddress, vm.DelayTimer, vm.SoundTimer)
			rl.DrawText(pcInfoText, offsetX, offsetY+heightPx+2, fontSize, rl.RayWhite)
		}

		// draw pixels
		for i := range uint16(64 * 32) {
			pixel := io.Buffer.Read(uint16(i))
			x := int32(i % 64)
			y := int32(i / 64)
			if pixel == 0 {
				continue
			}
			rl.DrawRectangle(offsetX+(x*px), offsetY+(y*px), px, px, rl.White)
			if pixelBorder {
				rl.DrawRectangleLines(offsetX+(x*px), offsetY+(y*px), px, px, rl.DarkGray)
			}
		}
		// screen outline
		if screenBorder {
			rl.DrawRectangleLines(offsetX, offsetY, widthPx, heightPx, rl.DarkGray)
		}

		rl.EndDrawing()
	}

}

var ModernKeymap = chip8.Keymap[int32]{
	rl.KeyOne:   chip8.Key1,
	rl.KeyTwo:   chip8.Key2,
	rl.KeyThree: chip8.Key3,
	rl.KeyQ:     chip8.Key4,
	rl.KeyW:     chip8.Key5,
	rl.KeyE:     chip8.Key6,
	rl.KeyA:     chip8.Key7,
	rl.KeyS:     chip8.Key8,
	rl.KeyD:     chip8.Key9,
	rl.KeyFour:  chip8.KeyC,
	rl.KeyR:     chip8.KeyD,
	rl.KeyF:     chip8.KeyE,
	rl.KeyV:     chip8.KeyF,
	rl.KeyZ:     chip8.KeyA,
	rl.KeyX:     chip8.Key0,
	rl.KeyC:     chip8.KeyB,
}

type Window struct {
	Buffer  chip8.FrameBuffer
	beeping bool
}

func NewWindow() (*Window, func(), error) {
	return &Window{}, func() {}, nil
}

func (w *Window) Next() chip8.Key {
	for k, v := range ModernKeymap {
		if rl.IsKeyReleased(k) {
			return v
		}
	}
	return chip8.KeyInvalid
}

func (w *Window) Pressed(k chip8.Key) bool {
	m := ModernKeymap.Mapped(k)
	return rl.IsKeyDown(m) || rl.GetKeyPressed() == m
}

func (w *Window) Beep(seconds float64) {
	go w.beep(seconds)
}

func (w *Window) beep(seconds float64) {
	if w.beeping || seconds <= 0 {
		return
	}
	w.beeping = true

	const (
		freq       = 440.0
		sampleRate = 44100
		channels   = 1
		sampleSize = 16 // bits per sample
	)
	dur := float64(seconds)

	sampleCount := int(dur * sampleRate)
	bufSize := sampleCount * channels * (sampleSize / 8)
	data := make([]byte, bufSize)

	// fill buffer with signed 16-bit PCM little endian
	for i := range sampleCount {
		t := float64(i) / float64(sampleRate)
		s := int16(math.Sin(2*math.Pi*freq*t) * 32767)
		binary.LittleEndian.PutUint16(data[i*2:], uint16(s))
	}

	// build wave from Go slice
	wave := rl.NewWave(
		uint32(sampleCount),
		uint32(sampleRate),
		uint32(sampleSize),
		uint32(channels),
		data,
	)

	// load and play sound
	sound := rl.LoadSoundFromWave(wave)
	rl.PlaySound(sound)

	// wait for playback
	time.Sleep(time.Duration(float64(time.Second)*dur) + 100*time.Millisecond)

	rl.UnloadSound(sound)
	w.beeping = false
}

func (w *Window) Draw(fb chip8.FrameBuffer) {
	copy(w.Buffer[:], fb[:])
}
