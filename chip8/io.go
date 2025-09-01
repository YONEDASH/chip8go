package chip8

type Key uint4

const (
	Key0 Key = iota
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
	KeyA
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
)

type Number interface{ int | int32 | int64 }

type Keymap[T interface{ Number }] map[T]Key

func (km Keymap[T]) Mapped(key Key) T {
	for k, v := range km {
		if v == key {
			return k
		}
	}
	return 0
}

type InputAdapter interface {
	// Blocks until key is pressed. Returns that key.
	Next() Key
	// Return true, if key k is pressed.
	Pressed(k Key) bool
}

type OutputAdapter interface {
	Beep(seconds float64)
	Draw(fb FrameBuffer)
}
