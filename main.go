package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/yonedash/chip8emu/chip8"
)

func main() {
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

	if true {
		if err := vm.Start(context.Background()); err != nil {
			panic(err)
		}
		return
	}

	rl.InitWindow(800, 450, "chip8")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	px := int32(12)
	w, h := int32(48), int32(32)
	widthPx, heightPx := w*px, h*px

	go func() {
		time.Sleep(time.Second)
		if err := vm.Start(context.Background()); err != nil {
			panic(err)
		}
	}()

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		offsetX, offsetY := (int32(rl.GetScreenWidth())/2 - widthPx/2), (int32(rl.GetScreenHeight())/2 - heightPx/2)
		// draw file name
		rl.DrawText(file.Name(), offsetX, offsetY-rl.GetFontDefault().BaseSize-2, rl.GetFontDefault().BaseSize, rl.RayWhite)
		// draw pixels
		for x := range int32(w) {
			for y := range int32(h) {
				if io.Buffer.Read(uint8(x), uint8(y)) == chip8.True {
					rl.DrawRectangle(offsetX+(x*px), offsetY+(y*px), x+px, y+px, rl.RayWhite)
				}
			}
		}
		// screen outline
		rl.DrawRectangleLines(offsetX, offsetY, 48*px, 32*px, rl.Gray)

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
