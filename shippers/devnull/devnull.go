package devnull

type DevNull struct{}

func New() *DevNull {
	return &DevNull{}
}

func (_ DevNull) Ship(b []byte) {}
