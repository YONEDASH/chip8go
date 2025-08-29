package chip8

type InputAdapter interface {
	// Blocks until key is pressed. Returns that key.
	Next() uint8
	// Return true, if key k is pressed.
	Pressed(k uint8) bool
}

type DisplayAdapter interface {
	Draw(FrameBuffer)
}
