package chip8

type DisplayAdapter interface {
	Draw(FrameBuffer)
}
