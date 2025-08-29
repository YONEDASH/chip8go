package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell"
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
	vm := chip8.New()
	if err := vm.Load(file); err != nil {
		panic(err)
	}
	if err := file.Close(); err != nil {
		panic(err)
	}

	// go func() {
	if err := vm.Start(context.Background(), nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// }()
	if true {
		os.Exit(0)
	}
	screen, err := tcell.NewScreen()
	if err != nil {
		panic(err)
	}

	if err := screen.Init(); err != nil {
		panic(err)
	}
	defer screen.Fini()

	drawFrameBuffer(screen, vm.FrameBuffer)

	screen.Beep()

	time.Sleep(time.Second * 3)
}

func drawFrameBuffer(screen tcell.Screen, fb chip8.FrameBuffer) {
	for y := range 32 {
		for x := range 64 {
			style := tcell.StyleDefault
			if fb.Read(uint8(x), uint8(y)) == chip8.True {
				style = style.Background(tcell.ColorWhite)
			}
			screen.SetContent(x*2, y, ' ', nil, style)
			screen.SetContent(x*2+1, y, ' ', nil, style)
		}
	}

	screen.Show()
}
