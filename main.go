package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/yonedash/chip8emu/chip8"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

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

	if false {
		if err := vm.Start(context.Background()); err != nil {
			panic(err)
		}
		return
	}

	rl.InitWindow(900, 450, fmt.Sprintf("CHIP-8: %s", file.Name()))
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	px := int32(12)
	w, h := int32(64), int32(32)
	widthPx, heightPx := w*px, h*px

	go func() {
		time.Sleep(time.Second)
		if err := vm.Start(context.Background()); err != nil {
			panic(err)
		}
	}()

	fontSize := int32(24)
	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		offsetX, offsetY := (int32(rl.GetScreenWidth())/2 - widthPx/2), (int32(rl.GetScreenHeight())/2 - heightPx/2)
		// draw file name
		rl.DrawText(file.Name(), offsetX, offsetY-fontSize-2, fontSize, rl.RayWhite)

		pcInfoText := fmt.Sprintf("PC=0x%3X I=0x%3X DT=%2d ST=%2d", vm.ProgramCounter, vm.IndexAddress, vm.DelayTimer, vm.SoundTimer)
		infoW := rl.MeasureText(pcInfoText, fontSize)
		rl.DrawText(pcInfoText, offsetX+widthPx-infoW, offsetY-fontSize-2, fontSize, rl.RayWhite)

		// draw pixels
		for i := range uint16(64 * 32) {
			pixel := io.Buffer.Read(uint16(i))
			x := int32(i % 64)
			y := int32(i / 64)
			if pixel == 0 {
				continue
			}
			rl.DrawRectangle(offsetX+(x*px), offsetY+(y*px), px, px, rl.White)
		}
		// screen outline
		rl.DrawRectangleLines(offsetX, offsetY, widthPx, heightPx, rl.Gray)

		rl.EndDrawing()
	}

}

type Window struct {
	Buffer chip8.FrameBuffer
}

func NewWindow() (*Window, func(), error) {
	return &Window{}, func() {}, nil
}

func (w Window) Next() uint8 {
	return 0
}

func (w Window) Pressed(k uint8) bool {
	return false
}

func (w Window) Beep() {
}

func (w *Window) Draw(fb chip8.FrameBuffer) {
	copy(w.Buffer[:], fb[:])
}
