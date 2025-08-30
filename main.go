package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gdamore/tcell"
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

	go func() {
		if err := vm.Start(context.Background()); err != nil {
			panic(err)
		}
	}()

	rl.InitWindow(800, 450, "chip8")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()

		rl.ClearBackground(rl.Black)
		// rl.DrawText("Congrats! You created your first window!", 190, 200, 20, rl.LightGray)

		px := int32(8)
		for x := range int32(48) {
			for y := range int32(32) {
				if vm.FrameBuffer.Read(uint8(x), uint8(y)) == chip8.True {
					rl.DrawRectangle(x*px, y*px, x+px, y+px, rl.RayWhite)
				}
			}
		}
		rl.DrawRectangleLines(0, 0, 48*px, 32*px, rl.Gray)

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

type Terminal struct {
	s tcell.Screen
}

func NewTerminal() (*Terminal, func(), error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, nil, err
	}
	err = s.Init()
	return &Terminal{s}, s.Fini, err
}

func (t Terminal) Next() uint8 {
	for {
		switch t.s.PollEvent().(type) {
		case *tcell.EventKey:
			return 0
		}
	}
}

func (t Terminal) Pressed(k uint8) bool {
	return false
	// return t.s.HasKey(k)
}

func (t Terminal) Beep() {
	if err := t.s.Beep(); err != nil {
		slog.Error("Terminal failed beep", "err", err)
	}
}

func (t Terminal) Draw(fb chip8.FrameBuffer) {
	for y := range 32 {
		for x := range 64 {
			style := tcell.StyleDefault
			if fb.Read(uint8(x), uint8(y)) == chip8.True {
				style = style.Background(tcell.ColorWhite)
			}
			t.s.SetContent(x*2, y, ' ', nil, style)
			t.s.SetContent(x*2+1, y, ' ', nil, style)
		}
	}

	t.s.Show()
}
